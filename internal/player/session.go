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

// Run manages the websocket connection lifecycle.
func (s *Session) Run(ctx context.Context) {
	defer close(s.protocolRx)
	if s.behavior.Evil {
		go s.runExtraConnectChaos(ctx)
	}

	for {
		conn, err := s.dial(ctx)
		if err != nil {
			log.Printf("player %s: dial error: %v", s.pid, err)
			return
		}

		err = s.bridge(ctx, conn)
		if status := websocket.CloseStatus(err); status >= 0 || errors.Is(err, io.EOF) {
			// websocket closed by server, finished
			log.Printf("player %s: connection closed", s.pid)
			break
		}
		if err != nil {
			log.Printf("player %s: connection error: %v", s.pid, err)
		}
	}
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

// dial the server, retrying on temporary failure.
func (s *Session) dial(ctx context.Context) (*websocket.Conn, error) {
	for {
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
