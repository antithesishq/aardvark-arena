// Package gameserver implements the game session server.
package gameserver

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/coder/websocket"
)

// Config holds server configuration.
type Config struct {
	TurnTimeout time.Duration
	MaxSessions int
}

// Server manages game sessions.
type Server struct {
	mux      *http.ServeMux
	sessions *SessionManager
}

// New creates a new Server.
func New(cfg Config) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		sessions: NewSessionManager(cfg),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("PUT /session/{sid}", s.handleCreateSession)
	s.mux.HandleFunc("/session/{sid}/{pid}", s.handleSessionConnect)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type HealthResponse struct {
	ActiveSessions int
	MaxSessions    int
	Full           bool
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	health := HealthResponse{
		ActiveSessions: s.sessions.ActiveSessions(),
		MaxSessions:    s.sessions.cfg.MaxSessions,
		Full:           s.sessions.ActiveSessions() >= s.sessions.cfg.MaxSessions,
	}
	if err := internal.RespondJSON(w, health); err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
	}
}

type CreateSessionRequest struct {
	Game     game.Kind
	Deadline time.Time
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
	err = s.sessions.CreateSession(sid, body.Game, body.Deadline)
	if errors.Is(err, ErrMaxSessions) {
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

	// TODO: connect websocket to session handle
	log.Printf("player %s connecting to session %s", pid, sid)
	_ = conn.Close(websocket.StatusNormalClosure, "not implemented")
}
