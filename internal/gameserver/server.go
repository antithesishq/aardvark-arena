// Package gameserver implements the game session server.
package gameserver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
)

// Config holds server configuration.
type Config struct {
	TurnTimeout time.Duration
	MaxSessions int
}

// Server manages game sessions.
type Server struct {
	cfg Config
	mux *http.ServeMux
}

// New creates a new Server.
func New(cfg Config) *Server {
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("PUT /session/{sessionID}", s.handleCreateSession)
	s.mux.HandleFunc("/session/{sessionID}/{playerID}", s.handleSessionConnect)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	type Health struct {
		ActiveSessions int
		MaxSessions    int
	}
	health := Health{}
	// TODO: set health stats
	if err := internal.WriteJSON(w, health); err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := internal.PathUUID(r, "sessionID")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	// TODO: parse body { Game, Deadline }
	// TODO: create session or return 503 if at capacity
	log.Printf("create session: %s", sessionID)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSessionConnect(w http.ResponseWriter, r *http.Request) {
	sessionID, err := internal.PathUUID(r, "sessionID")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	playerID, err := internal.PathUUID(r, "playerID")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// TODO: lookup session, fail if not exists
	// TODO: upgrade to websocket
	// TODO: handle HELLO, MOVE messages
	// TODO: send STATE, ERROR, QUIT messages
	log.Printf("player %s connecting to session %s", playerID, sessionID)
	internal.WriteError(w, http.StatusNotImplemented, fmt.Errorf("not implemented"))
}
