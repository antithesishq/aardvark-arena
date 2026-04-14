// Package gameserver implements the game session server.
package gameserver

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

// Config holds server configuration.
type Config struct {
	ID            uuid.UUID
	TurnTimeout   time.Duration
	MaxSessions   int
	Token         internal.Token
	MatchmakerURL *url.URL
	SelfURL       *url.URL
}

// Server manages game sessions.
type Server struct {
	cfg      Config
	mux      *http.ServeMux
	sessions *SessionManager
	reporter *Reporter
	client   *http.Client
}

// New creates a new Server. Background goroutines are tied to the given context
// and will stop when it is cancelled.
func New(ctx context.Context, cfg Config) *Server {
	resultCh := make(chan resultMsg, cfg.MaxSessions)
	s := &Server{
		cfg:      cfg,
		mux:      http.NewServeMux(),
		sessions: NewSessionManager(cfg, resultCh),
		reporter: NewReporter(resultCh, cfg.Token, cfg.MatchmakerURL),
		client:   internal.NewHTTPClient(),
	}
	s.routes()
	s.reporter.StartReporter(ctx)
	s.startRegistrationLoop(ctx)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	internal.CORS(s.mux).ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("PUT /session/{sid}", internal.TokenAuth(s.cfg.Token, s.handleCreateSession))
	s.mux.HandleFunc("/session/{sid}/{pid}", s.handleSessionConnect)
	s.mux.HandleFunc("GET /sessions", s.handleListSessions)
	s.mux.HandleFunc("GET /session/{sid}/watch", s.handleWatchSession)
	s.mux.HandleFunc("DELETE /session/{sid}", s.handleCancelSession)
	s.mux.HandleFunc("DELETE /sessions", s.handleCancelAllSessions)
	s.mux.HandleFunc("GET /watch", s.handleWatch)
	s.mux.HandleFunc("POST /drain", s.handleDrain)
	s.mux.HandleFunc("POST /activate", s.handleActivate)
}

// HealthResponse contains the server health status.
type HealthResponse struct {
	ActiveSessions int
	MaxSessions    int
	Full           bool
	Active         bool
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	active := s.sessions.ActiveSessions()
	health := HealthResponse{
		ActiveSessions: active,
		MaxSessions:    s.cfg.MaxSessions,
		Full:           active >= s.cfg.MaxSessions,
		Active:         s.sessions.IsActive(),
	}
	assert.Always(
		health.ActiveSessions <= health.MaxSessions,
		"gameserver active sessions never exceed max sessions",
		map[string]any{"active": health.ActiveSessions, "max": health.MaxSessions},
	)
	if err := internal.RespondJSON(w, health); err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
	}
}

// CreateSessionRequest is the request body for creating a new game session.
type CreateSessionRequest struct {
	Game    game.Kind
	Timeout time.Duration
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	body, err := internal.BindJSON[CreateSessionRequest](r.Body)
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	deadline := time.Now().Add(body.Timeout)
	err = s.sessions.CreateSession(sid, body.Game, deadline)
	if e, ok := err.(*ErrMaxSessions); ok {
		assert.Reachable(
			"gameserver sometimes reaches max session capacity",
			map[string]any{"sid": sid.String()},
		)
		retrySeconds := strconv.Itoa(int(math.Ceil(time.Until(e.RetryAt).Seconds())))
		w.Header().Add("Retry-After", retrySeconds)
		internal.WriteError(w, http.StatusServiceUnavailable, err)
		return
	} else if _, ok := err.(*ErrNotActive); ok {
		w.Header().Add("Retry-After", "300")
		internal.WriteError(w, http.StatusServiceUnavailable, err)
		return
	} else if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSessionConnect(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pid, err := internal.PathUUID(r, "pid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	log.Printf("player %s connecting to session %s", pid, sid)
	err = s.sessions.JoinSession(pid, sid, conn)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, fmt.Sprintf("failed to join session: %v", err))
		return
	}
}

func (s *Server) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	if err := internal.RespondJSON(w, s.sessions.ListSessions()); err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) handleWatchSession(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("spectator websocket upgrade failed: %v", err)
		return
	}
	ch, err := s.sessions.WatchSession(sid)
	if err != nil {
		_ = conn.Close(websocket.StatusNormalClosure, err.Error())
		return
	}
	for msg := range ch {
		if err := wsjson.Write(context.Background(), conn, msg); err != nil {
			_ = conn.Close(websocket.StatusInternalError, "write failed")
			return
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "session ended")
}

func (s *Server) handleCancelSession(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.sessions.CancelSession(sid); err != nil {
		internal.WriteError(w, http.StatusNotFound, err)
		return
	}
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("watch websocket upgrade failed: %v", err)
		return
	}
	// Discard reads so the server detects client-initiated close promptly.
	conn.CloseRead(r.Context())

	ch := s.sessions.RegisterWatcher()
	defer s.sessions.UnregisterWatcher(ch)
	for evt := range ch {
		if err := wsjson.Write(r.Context(), conn, evt); err != nil {
			_ = conn.Close(websocket.StatusInternalError, "write failed")
			return
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "done")
}

func (s *Server) handleCancelAllSessions(w http.ResponseWriter, _ *http.Request) {
	s.sessions.CancelAllSessions()
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleDrain(w http.ResponseWriter, _ *http.Request) {
	s.sessions.SetActive(false)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleActivate(w http.ResponseWriter, _ *http.Request) {
	s.sessions.SetActive(true)
	go s.register()
	_, _ = w.Write([]byte("ok"))
}

// canRegister reports whether this server has the configuration needed to
// register with a matchmaker.
func (s *Server) canRegister() bool {
	return s.cfg.MatchmakerURL != nil && s.cfg.MatchmakerURL.Host != "" &&
		s.cfg.SelfURL != nil && s.cfg.SelfURL.Host != ""
}

// register sends a registration request to the matchmaker. Returns true on success.
func (s *Server) register() bool {
	if !s.canRegister() {
		return false
	}
	type registerRequest struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	body, err := internal.EncodeJSON(registerRequest{
		ID:  s.cfg.ID.String(),
		URL: s.cfg.SelfURL.String(),
	})
	if err != nil {
		log.Printf("register: failed to encode request: %v", err)
		return false
	}
	reqURL := s.cfg.MatchmakerURL.JoinPath("servers", "register")
	req, err := http.NewRequest("POST", reqURL.String(), bytes.NewReader(body))
	if err != nil {
		log.Printf("register: failed to create request: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if !s.cfg.Token.IsNil() {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.cfg.Token.String()))
	}
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("register: failed to contact matchmaker: %v", err)
		return false
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		log.Printf("registered with matchmaker as %s at %s", internal.ShortID(s.cfg.ID), s.cfg.SelfURL)
		return true
	}
	log.Printf("register: matchmaker returned %s", resp.Status)
	return false
}

// startRegistrationLoop periodically registers with the matchmaker. It retries
// with exponential backoff until the first success, then re-registers at a
// fixed interval to act as a heartbeat (the matchmaker stores fleet state
// in memory, so re-registration is needed after matchmaker restarts).
func (s *Server) startRegistrationLoop(ctx context.Context) {
	if !s.canRegister() {
		return
	}
	go func() {
		backoff := time.Second
		const maxBackoff = 10 * time.Second
		const heartbeat = 30 * time.Second
		registered := false
		for {
			if s.register() {
				if !registered {
					registered = true
				}
				backoff = heartbeat
			} else if registered {
				// Lost contact after a successful registration — reset backoff.
				backoff = time.Second
				registered = false
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if !registered {
				backoff = min(backoff*2, maxBackoff)
			}
		}
	}()
}
