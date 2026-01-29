// Package matchmaker implements the matchmaking server.
package matchmaker

import (
	"net/http"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
)

// Config holds server configuration.
type Config struct {
	// SessionTimeout is the duration after which unfinished sessions are cancelled.
	SessionTimeout time.Duration

	// GameServers is the list of available game server URLs to route to.
	GameServers []*url.URL

	// Token authenticates requests to/from game servers.
	Token internal.Token
}

// Server manages matchmaking.
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
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// HealthResponse contains the server health status.
type HealthResponse struct {
	Status string
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	health := HealthResponse{
		Status: "ok",
	}
	if err := internal.RespondJSON(w, health); err != nil {
		internal.WriteError(w, http.StatusInternalServerError, err)
	}
}
