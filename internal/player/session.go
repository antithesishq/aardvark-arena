package player

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

const reconnectInterval = time.Second
const dialTimeout = 30 * time.Second

// ErrGameServerUnavailable is returned by Run when the game server cannot be
// reached within dialTimeout.
var ErrGameServerUnavailable = errors.New("game server unavailable")

// Session manages a websocket connection to the game server,
// transparently handling reconnection on failure.
type Session struct {
	client    *http.Client
	serverURL url.URL
	sid       internal.SessionID
	pid       internal.PlayerID
	behavior  Behavior

	// communication channels between protocol and session
	protocolRx chan gameserver.PlayerMsg
	protocolTx chan json.RawMessage
}

// NewSession creates a Session that will connect to the given game server.
func NewSession(
	serverURL url.URL,
	sid internal.SessionID,
	pid internal.PlayerID,
	behavior Behavior,
) *Session {
	return &Session{
		client:     internal.NewHTTPClient(),
		serverURL:  serverURL,
		sid:        sid,
		pid:        pid,
		behavior:   behavior,
		protocolRx: make(chan gameserver.PlayerMsg, 1),
		protocolTx: make(chan json.RawMessage, 1),
	}
}

// Run manages the websocket connection lifecycle. It returns
// ErrGameServerUnavailable when the server cannot be reached within
// dialTimeout, nil on a clean session close, or the underlying error otherwise.
func (s *Session) Run(ctx context.Context) error {
	defer close(s.protocolRx)
	if s.behavior.Evil {
		go s.runExtraConnectChaos(ctx)
	}

	for {
		conn, err := s.dial(ctx)
		if err != nil {
			log.Printf("player %s: dial error: %v", s.pid, err)
			return err
		}

		err = s.bridge(ctx, conn)
		if status := websocket.CloseStatus(err); status >= 0 || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			// websocket closed by server, finished
			log.Printf("player %s: connection closed", s.pid)
			break
		}
		if err != nil {
			log.Printf("player %s: connection error: %v", s.pid, err)
		}
	}
	return nil
}

func (s *Session) runExtraConnectChaos(ctx context.Context) {
	rng := internal.NewRand()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.behavior.doExtraConnect(rng) {
				continue
			}
			assert.Reachable(
				"evil players sometimes attempt random-id background joins",
				map[string]any{"sid": s.sid.String()},
			)
			pid := uuid.New()
			dialCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
			u := s.serverURL.JoinPath("session", s.sid.String(), pid.String())
			conn, _, err := websocket.Dial(dialCtx, u.String(), &websocket.DialOptions{HTTPClient: s.client})
			cancel()
			if err != nil {
				continue
			}
			// Keep the connection very brief; we only need to exercise join logic.
			_ = conn.Close(websocket.StatusNormalClosure, "chaos join probe")
		}
	}
}

// dial the server, retrying on temporary failure up to dialTimeout. If the
// timeout elapses with no successful connection, returns ErrGameServerUnavailable.
func (s *Session) dial(ctx context.Context) (*websocket.Conn, error) {
	timer := time.NewTimer(dialTimeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return nil, ErrGameServerUnavailable
		default:
		}

		u := s.serverURL.JoinPath("session", s.sid.String(), s.pid.String())
		conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{HTTPClient: s.client})
		if internal.HTTPIsTemporary(err) {
			time.Sleep(reconnectInterval)
			continue
		}
		return conn, err
	}
}

// bridge runs the read/write loops for a single websocket connection.
// Returns when the connection fails or the context is cancelled.
func (s *Session) bridge(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case move, ok := <-s.protocolTx:
				if !ok {
					return
				}
				if err := wsjson.Write(ctx, conn, move); err != nil {
					log.Printf("player %s: write error: %v", s.pid, err)
					return
				}
			}
		}
	}()

	for {
		var msg gameserver.PlayerMsg
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return err
		}
		select {
		case s.protocolRx <- msg:
		case <-ctx.Done():
			return nil
		}
	}
}
