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
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const reconnectInterval = time.Second

// Session manages a websocket connection to the game server,
// transparently handling reconnection on failure.
type Session struct {
	client    *http.Client
	serverURL url.URL
	sid       internal.SessionID
	pid       internal.PlayerID

	// communication channels between protocol and session
	protocolRx chan gameserver.PlayerMsg
	protocolTx chan json.RawMessage
}

// NewSession creates a Session that will connect to the given game server.
func NewSession(
	serverURL url.URL,
	sid internal.SessionID,
	pid internal.PlayerID,
) *Session {
	return &Session{
		client:     internal.NewHttpClient(),
		serverURL:  serverURL,
		sid:        sid,
		pid:        pid,
		protocolRx: make(chan gameserver.PlayerMsg, 1),
		protocolTx: make(chan json.RawMessage, 1),
	}
}

// Run manages the websocket connection lifecycle.
func (s *Session) Run(ctx context.Context) {
	defer close(s.protocolRx)

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

// dial the server, retrying on temporary failure
func (s *Session) dial(ctx context.Context) (*websocket.Conn, error) {
	for {
		u := s.serverURL.JoinPath("session", s.sid.String(), s.pid.String())
		conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{HTTPClient: s.client})
		if internal.HttpIsTemporary(err) {
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
			case move := <-s.protocolTx:
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
