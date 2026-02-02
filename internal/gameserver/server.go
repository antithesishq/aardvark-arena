// Package gameserver implements the game session server.
package gameserver

import (
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
	"github.com/coder/websocket"
)

// Config holds server configuration.
type Config struct {
	TurnTimeout   time.Duration
	MaxSessions   int
	Token         internal.Token
	MatchmakerURL *url.URL
}

// Server manages game sessions.
type Server struct {
	cfg      Config
	mux      *http.ServeMux
	sessions *SessionManager
	reporter *Reporter
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
	}
	s.routes()
	s.reporter.StartReporter(ctx)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("PUT /session/{sid}", internal.TokenAuth(s.cfg.Token, s.handleCreateSession))
	s.mux.HandleFunc("/session/{sid}/{pid}", s.handleSessionConnect)
}

// HealthResponse contains the server health status.
type HealthResponse struct {
	ActiveSessions int
	MaxSessions    int
	Full           bool
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	health := HealthResponse{
		ActiveSessions: s.sessions.ActiveSessions(),
		MaxSessions:    s.cfg.MaxSessions,
		Full:           s.sessions.ActiveSessions() >= s.cfg.MaxSessions,
	}
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
		retrySeconds := strconv.Itoa(int(math.Ceil(time.Until(e.RetryAt).Seconds())))
		w.Header().Add("Retry-After", retrySeconds)
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
