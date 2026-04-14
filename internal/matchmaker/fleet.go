package matchmaker

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/google/uuid"
)

// FailureTimeout is the grace period we give game servers to recover after we
// see a network error.
var FailureTimeout = time.Minute

// ErrNoServersAvailable is returned when no game servers can accept a session.
var ErrNoServersAvailable = fmt.Errorf("no gameservers available")

// Fleet monitors a collection of GameServers and handles session creation.
type Fleet struct {
	mu             sync.Mutex
	servers        []*server
	client         *http.Client
	token          internal.Token
	sessionTimeout time.Duration
	rng            *rand.Rand
}

// Keep track of a single game server.
type server struct {
	id  uuid.UUID
	url url.URL
	// retryAt is set when we either see a network error, or the gameserver is
	// full. When this time passes, we will resume attempting to create Sessions
	// on this gameserver.
	retryAt *time.Time
}

// NewFleet creates an empty Fleet. Game servers join via Register.
func NewFleet(token internal.Token, sessionTimeout time.Duration) *Fleet {
	return &Fleet{
		client:         internal.NewHTTPClient(),
		token:          token,
		sessionTimeout: sessionTimeout,
		rng:            internal.NewRand(),
	}
}

// Register adds or updates a game server in the fleet. If a server with the
// same ID already exists, its URL is updated and its retryAt is cleared.
func (f *Fleet) Register(id uuid.UUID, serverURL url.URL) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.servers {
		if s.id == id {
			s.url = serverURL
			s.retryAt = nil
			return
		}
	}
	f.servers = append(f.servers, &server{id: id, url: serverURL})
}

// ServerInfo describes a registered game server.
type ServerInfo struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// Servers returns a snapshot of all registered servers.
func (f *Fleet) Servers() []ServerInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]ServerInfo, len(f.servers))
	for i, s := range f.servers {
		result[i] = ServerInfo{ID: s.id.String(), URL: s.url.String()}
	}
	return result
}

// SessionInfo describes an active game session on a server.
type SessionInfo struct {
	ServerID  uuid.UUID
	Server    url.URL
	SessionID internal.SessionID
	Game      game.Kind
	Timeout   time.Duration
}

// ResetRetry clears the retryAt on the server matching the given ID, allowing
// it to be used as a candidate immediately. This should be called when a session
// on the server ends, since the server likely has capacity again.
func (f *Fleet) ResetRetry(serverID uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.servers {
		if s.id == serverID {
			s.retryAt = nil
			return
		}
	}
}

// CreateSession creates a new game session on an available server.
func (f *Fleet) CreateSession(kind game.Kind) (*SessionInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// gather candidates
	var candidates []*server
	now := time.Now()
	for _, server := range f.servers {
		if server.retryAt == nil || server.retryAt.Before(now) {
			candidates = append(candidates, server)
		}
	}

	if len(candidates) == 0 {
		assert.Reachable(
			"fleet sometimes has no currently available gameserver candidates",
			map[string]any{"total_servers": len(f.servers)},
		)
		return nil, ErrNoServersAvailable
	}

	// randomly shuffle candidates
	f.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	body, err := internal.EncodeJSON(gameserver.CreateSessionRequest{
		Game:    kind,
		Timeout: f.sessionTimeout,
	})
	if err != nil {
		return nil, err
	}

	// try to create a session on each candidate
	for _, server := range candidates {
		sid := uuid.New()
		reqURL := server.url.JoinPath("session", sid.String())
		req, err := http.NewRequest("PUT", reqURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if !f.token.IsNil() {
			req.Header.Set("Authorization", "Bearer "+f.token.String())
		}
		resp, err := f.client.Do(req)
		if internal.HTTPIsTemporary(err) {
			assert.Reachable(
				"fleet sometimes encounters temporary transport failures",
				map[string]any{"server": server.url.String()},
			)
			retryAt := time.Now().Add(FailureTimeout)
			server.retryAt = &retryAt
			continue
		}
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			log.Printf("session %s created on gameserver %s", sid, internal.ShortID(server.id))
			return &SessionInfo{
				ServerID:  server.id,
				Server:    server.url,
				SessionID: sid,
				Game:      kind,
				Timeout:   f.sessionTimeout,
			}, nil
		} else if resp.StatusCode == http.StatusServiceUnavailable {
			assert.Reachable(
				"gameservers sometimes reject session creation due to capacity",
				map[string]any{"server": server.url.String()},
			)
			retryAfterSecs, err := strconv.Atoi(resp.Header.Get("Retry-After"))
			if err == nil {
				retryAt := time.Now().Add(time.Second * time.Duration(retryAfterSecs))
				server.retryAt = &retryAt
			} else {
				retryAt := time.Now().Add(FailureTimeout)
				server.retryAt = &retryAt
			}
			_ = resp.Body.Close()
			continue
		}
		_ = resp.Body.Close()
		assert.Unreachable(
			"gameserver should only return 200 or 503 for session creation",
			map[string]any{
				"server": server.url.String(),
				"status": resp.StatusCode,
			},
		)
		return nil, fmt.Errorf("unexpected response from gameserver: %s", resp.Status)
	}

	// all servers unavailable
	return nil, ErrNoServersAvailable
}
