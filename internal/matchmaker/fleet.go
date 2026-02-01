package matchmaker

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
	"github.com/google/uuid"
)

// FailureTimeout is the grace period we give game servers to recover after we
// see a network error.
var FailureTimeout = time.Minute

// ErrNoServersAvailable is returned when no game servers can accept a session.
var ErrNoServersAvailable = fmt.Errorf("no gameservers available")

// Fleet monitors a collection of GameServers and handles session creation.
type Fleet struct {
	servers        []*server
	client         *http.Client
	token          internal.Token
	sessionTimeout time.Duration
}

// Keep track of a single game server.
type server struct {
	url url.URL
	// retryAt is set when we either see a network error, or the gameserver is
	// full. When this time passes, we will resume attempting to create Sessions
	// on this gameserver.
	retryAt *time.Time
}

// NewFleet creates a Fleet from the given server URLs.
func NewFleet(urls []*url.URL, token internal.Token, sessionTimeout time.Duration) *Fleet {
	var servers []*server
	for _, url := range urls {
		servers = append(servers, &server{url: *url})
	}
	return &Fleet{
		servers:        servers,
		client:         internal.NewHttpClient(),
		token:          token,
		sessionTimeout: sessionTimeout,
	}
}

// SessionInfo describes an active game session on a server.
type SessionInfo struct {
	Server    url.URL
	SessionID internal.SessionID
	Game      game.Kind
	Timeout   time.Duration
}

// CreateSession creates a new game session on an available server.
func (f *Fleet) CreateSession(kind game.Kind) (*SessionInfo, error) {
	// gather candidates
	var candidates []*server
	now := time.Now()
	for _, server := range f.servers {
		if server.retryAt == nil || server.retryAt.Before(now) {
			candidates = append(candidates, server)
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoServersAvailable
	}

	// randomly shuffle candidates
	rand.Shuffle(len(candidates), func(i, j int) {
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
		if internal.HttpIsTemporary(err) {
			retryAt := time.Now().Add(FailureTimeout)
			server.retryAt = &retryAt
			continue
		}
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			return &SessionInfo{
				Server:    server.url,
				SessionID: sid,
				Game:      kind,
				Timeout:   f.sessionTimeout,
			}, nil
		} else if resp.StatusCode == http.StatusServiceUnavailable {
			retryAfterSecs, err := strconv.Atoi(resp.Header.Get("Retry-After"))
			if err == nil {
				retryAt := time.Now().Add(time.Second * time.Duration(retryAfterSecs))
				server.retryAt = &retryAt
			} else {
				retryAt := time.Now().Add(FailureTimeout)
				server.retryAt = &retryAt
			}
			continue
		}
		// all other statuses are unexpected errors
		return nil, fmt.Errorf("unexpected response from gameserver: %s", resp.Status)
	}

	// all servers unavailable
	return nil, ErrNoServersAvailable
}
