package matchmaker

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
	"github.com/google/uuid"
)

// FailureTimeout is the grace period we give game servers to recover after we
// see a network error
var FailureTimeout = time.Minute

var NoServersAvailable = fmt.Errorf("no gameservers available")

// Fleet monitors a collection of GameServers and handles session creation
type Fleet struct {
	servers        []*server
	client         *http.Client
	sessionTimeout time.Duration
}

// Keep track of a single game server
type server struct {
	url url.URL
	// retryAt is set when we either see a network error, or the gameserver is
	// full. When this time passes, we will resume attempting to create Sessions
	// on this gameserver.
	retryAt *time.Time
}

func NewFleet(urls []*url.URL, sessionTimeout time.Duration) *Fleet {
	var servers []*server
	for _, url := range urls {
		servers = append(servers, &server{url: *url})
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	return &Fleet{
		servers:        servers,
		client:         client,
		sessionTimeout: sessionTimeout,
	}
}

type SessionInfo struct {
	Server    url.URL
	SessionID internal.SessionID
	Game      game.Kind
}

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
		return nil, NoServersAvailable
	}

	// randomly shuffle candidates
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	body, err := internal.EncodeJSON(gameserver.CreateSessionRequest{
		Game:     kind,
		Deadline: time.Now().Add(f.sessionTimeout),
	})
	if err != nil {
		return nil, err
	}

	// try to create a session on each candidate
	for _, server := range candidates {
		sid := uuid.New()
		req := &http.Request{
			Method:        "PUT",
			URL:           server.url.JoinPath("session", sid.String()),
			Header:        http.Header{"Content-Type": {"application/json"}},
			Body:          io.NopCloser(bytes.NewReader(body)),
			ContentLength: int64(len(body)),
		}
		resp, err := f.client.Do(req)
		if urlerr, ok := err.(*url.Error); ok {
			if urlerr.Temporary() {
				*server.retryAt = time.Now().Add(FailureTimeout)
				continue
			}
		}
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			return &SessionInfo{
				Server:    server.url,
				SessionID: sid,
				Game:      kind,
			}, nil
		} else if resp.StatusCode == http.StatusServiceUnavailable {
			*server.retryAt = time.Now().Add(FailureTimeout)
			continue
		} else {
			// all other status's are unexpected errors
			return nil, fmt.Errorf("unexpected response from gameserver: %s", resp.Status)
		}
	}

	// all servers unavailable
	return nil, NoServersAvailable
}
