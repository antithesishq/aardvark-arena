// Package matchmaker implements the matchmaking server.
package matchmaker

import (
	"net/http"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
)

// Config holds server configuration.
type Config struct {
	// SessionTimeout is the duration after which unfinished sessions are cancelled.
	SessionTimeout time.Duration

	// How frequently will the match queue be checked for matches
	MatchInterval time.Duration

	// GameServers is the list of available game server URLs to route to.
	GameServers []*url.URL

	// Token authenticates requests to/from game servers.
	Token internal.Token

	// DatabasePath is the path to the SQLite database file.
	DatabasePath string
}

// Server manages matchmaking.
type Server struct {
	cfg   Config
	mux   *http.ServeMux
	queue *MatchQueue
	db    *DB
}

// New creates a new Server.
func New(cfg Config) (*Server, error) {
	db, err := NewDB(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	fleet := NewFleet(cfg.GameServers, cfg.Token, cfg.SessionTimeout)
	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		queue: NewMatchQueue(fleet, db),
		db:    db,
	}
	s.routes()
	s.queue.StartMatcher(cfg.MatchInterval)
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("PUT /queue/{pid}", s.handleQueue)
	s.mux.HandleFunc("DELETE /queue/{pid}", s.handleUnqueue)
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

type ResultRequest struct {
	Status game.Status
}

func (s *Server) handleResult(w http.ResponseWriter, r *http.Request) {
	sid, err := internal.PathUUID(r, "sid")
	if err != nil {
		internal.WriteError(w, http.StatusBadRequest, err)
		return
	}
	_, _ = w.Write([]byte(sid.String()))
}
