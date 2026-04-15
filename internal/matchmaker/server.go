// Package matchmaker implements the matchmaking server.
package matchmaker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/antithesis-sdk-go/assert"

	"github.com/google/uuid"
)

// Config holds server configuration.
type Config struct {
	// SessionTimeout is the duration after which unfinished sessions are cancelled.
	SessionTimeout time.Duration

	// How frequently will the match queue be checked for matches
	MatchInterval time.Duration

	// How frequently will the database be checked for expired sessions
	SessionMonitorInterval time.Duration

	// DatabasePath is the path to the SQLite database file.
	DatabasePath string
}

// Server manages matchmaking.
type Server struct {
	cfg   Config
	mux   *http.ServeMux
	queue *MatchQueue
	fleet *Fleet
	db    *DB
}

// New creates a new Server. Background goroutines are tied to the given context
// and will stop when it is cancelled.
func New(ctx context.Context, cfg Config) (*Server, error) {
	db, err := NewDB(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	fleet := NewFleet(cfg.SessionTimeout)
	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		queue: NewMatchQueue(fleet, db),
		fleet: fleet,
		db:    db,
	}
	s.routes()
	s.queue.StartMatcher(ctx, cfg.MatchInterval)
	s.db.StartSessionMonitor(ctx, cfg.SessionMonitorInterval, s.queue.Untrack)
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	internal.CORS(s.mux).ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("PUT /queue/{pid}", s.handleQueue)
	s.mux.HandleFunc("DELETE /queue/{pid}", s.handleUnqueue)
	s.mux.HandleFunc("PUT /results/{sid}", s.handleResult)
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("GET /leaderboard", s.handleLeaderboard)
	s.mux.HandleFunc("DELETE /session/{sid}", s.handleCancelSession)
	s.mux.HandleFunc("GET /servers", s.handleServers)
	s.mux.HandleFunc("POST /servers/register", s.handleRegister)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// QueueRequest is the JSON body for the queue endpoint.
type QueueRequest struct {
	Game *game.Kind
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	pid, err := internal.PathUUID(r, "pid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	body, err := internal.BindJSON[QueueRequest](r.Body)
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	player, err := s.db.GetOrCreatePlayer(pid)
	if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	info, err := s.queue.Queue(player, body.Game)
	if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if info == nil {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("queued"))
		return
	}
	_ = internal.RespondJSON(w, info)
}

func (s *Server) handleUnqueue(w http.ResponseWriter, r *http.Request) {
	pid, err := internal.PathUUID(r, "pid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	s.queue.Unqueue(pid)
	_, _ = w.Write([]byte("ok"))
}

// ResultRequest is the payload sent by a game server to report a session outcome.
type ResultRequest struct {
	Cancelled bool
	Winner    internal.PlayerID
}

func (s *Server) handleResult(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	body, err := internal.BindJSON[ResultRequest](r.Body)
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	assert.Always(
		!body.Cancelled || body.Winner == uuid.Nil,
		"cancelled session results never declare a winner",
		map[string]any{"sid": sid.String()},
	)
	err = s.db.ReportSessionResult(sid, body.Cancelled, body.Winner)
	if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	s.queue.Untrack(sid)
	_, _ = w.Write([]byte("ok"))
}

// StatusResponse is the payload for GET /status.
type StatusResponse struct {
	Queue    []QueuedPlayer      `json:"queue"`
	Sessions []ActiveSessionView `json:"sessions"`
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	sessions, err := s.db.ActiveSessions()
	if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	_ = internal.RespondJSON(w, StatusResponse{
		Queue:    s.queue.QueuedPlayers(),
		Sessions: sessions,
	})
}

func (s *Server) handleCancelSession(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	session, err := s.db.GetSession(sid)
	if err != nil {
		internal.WriteError(w, http.StatusNotFound, err)
		return
	}
	if session.CompletedAt.Valid {
		internal.WriteError(w, http.StatusGone, fmt.Errorf("session already completed"))
		return
	}

	// Proxy DELETE to the game server. If it returns 404 the session already
	// ended there — we still mark it cancelled in the DB.
	gsURL, err := url.Parse(session.Server)
	if err == nil {
		reqURL := gsURL.JoinPath("session", sid.String())
		req, err2 := http.NewRequest(http.MethodDelete, reqURL.String(), nil)
		if err2 == nil {
			resp, err2 := internal.NewHTTPClient().Do(req)
			if err2 != nil {
				log.Printf("cancel: could not reach game server %s: %v", session.Server, err2)
			} else {
				_ = resp.Body.Close()
			}
		}
	}

	if err := s.db.ReportSessionResult(sid, true, uuid.Nil); err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	s.queue.Untrack(sid)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleServers(w http.ResponseWriter, _ *http.Request) {
	_ = internal.RespondJSON(w, s.fleet.Servers())
}

// RegisterRequest is the JSON body for POST /servers/register.
type RegisterRequest struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	body, err := internal.BindJSON[RegisterRequest](r.Body)
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if body.ID == "" || body.URL == "" {
		internal.WriteError(w, http.StatusBadRequest, fmt.Errorf("id and url are required"))
		return
	}
	id, err := uuid.Parse(body.ID)
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid id: %w", err))
		return
	}
	u, err := url.Parse(body.URL)
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid url: %w", err))
		return
	}
	s.fleet.Register(id, *u)
	log.Printf("registered game server %s at %s", internal.ShortID(id), body.URL)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, _ *http.Request) {
	players, err := s.db.Leaderboard()
	if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	_ = internal.RespondJSON(w, players)
}
