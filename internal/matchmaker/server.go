// Package matchmaker implements the matchmaking server.
package matchmaker

import (
	"context"
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

// New creates a new Server. Background goroutines are tied to the given context
// and will stop when it is cancelled.
func New(ctx context.Context, cfg Config) (*Server, error) {
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
	s.mux.HandleFunc("PUT /results/{sid}", internal.TokenAuth(s.cfg.Token, s.handleResult))
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("GET /leaderboard", s.handleLeaderboard)
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
	assert.Sometimes(body.Game != nil,
		"players sometimes request a specific game kind",
		nil,
	)
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
		assert.Sometimes(true,
			"queue requests sometimes wait before a session is assigned",
			nil,
		)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("queued"))
		return
	}
	assert.Sometimes(true,
		"queue requests sometimes return an immediate session assignment",
		map[string]any{"game": string(info.Game)},
	)
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

func (s *Server) handleLeaderboard(w http.ResponseWriter, _ *http.Request) {
	players, err := s.db.Leaderboard()
	if err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	_ = internal.RespondJSON(w, players)
}
