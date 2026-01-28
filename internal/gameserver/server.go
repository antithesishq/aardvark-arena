// Package gameserver implements the game session server.
package gameserver

import (
	"log"
	"net/http"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/coder/websocket"
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
	s.mux.HandleFunc("PUT /session/{sid}", s.handleCreateSession)
	s.mux.HandleFunc("/session/{sid}/{pid}", s.handleSessionConnect)
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
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	// TODO: parse body { Game, Deadline }
	// TODO: create session or return 503 if at capacity
	log.Printf("create session: %s", sid)
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
