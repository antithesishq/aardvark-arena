package gameserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// ErrMaxSessions is returned when the server has reached its maximum session capacity.
type ErrMaxSessions struct {
	RetryAt time.Time
}

func (e ErrMaxSessions) Error() string {
	return "max session reached"
}

// https://stackoverflow.com/questions/25065055/what-is-the-maximum-time-time-in-go
var maxTime = time.Unix(0, 0).Add(1<<63 - 1)

// SessionManager manages active game sessions.
type SessionManager struct {
	// mu protects reads/writes to the sessions map
	mu       sync.Mutex
	sessions map[internal.SessionID]sessionHandle
	resultCh chan resultMsg
	cfg      Config
}

// NewSessionManager creates a new SessionManager with the given config.
func NewSessionManager(cfg Config, resultCh chan resultMsg) *SessionManager {
	return &SessionManager{
		sessions: make(map[internal.SessionID]sessionHandle),
		resultCh: resultCh,
		cfg:      cfg,
	}
}

// ActiveSessions returns the number of currently active sessions.
func (s *SessionManager) ActiveSessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// CreateSession idempotently creates a session, returning an error if the
// session is finished or the requested game is incorrect.
func (s *SessionManager) CreateSession(sid internal.SessionID, game game.Kind, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sid]; !ok {
		if len(s.sessions) >= s.cfg.MaxSessions {
			// find min deadline
			var minDeadline = maxTime
			for _, session := range s.sessions {
				if !session.IsFinished() && session.deadline.Before(minDeadline) {
					minDeadline = session.deadline
				}
			}
			return &ErrMaxSessions{RetryAt: minDeadline}
		}
		inboxCh := make(chan inboxMsg, 2)
		ctx, cancel := context.WithCancel(context.Background())
		handle := sessionHandle{
			sid:      sid,
			ctx:      ctx,
			cancel:   cancel,
			game:     game,
			deadline: deadline,
			inbox:    inboxCh,
			result:   s.resultCh,
		}
		go handle.RunToCompletion(game, deadline, s.cfg.TurnTimeout)
		s.sessions[sid] = handle
		log.Printf("session %s created", sid)
	}

	handle := s.sessions[sid]
	if handle.IsFinished() {
		return fmt.Errorf("session %s is already finished", sid)
	}
	if handle.game != game {
		return fmt.Errorf("session %s already exists with different game kind", sid)
	}
	return nil
}

// JoinSession adds a player to an existing session over the given websocket.
func (s *SessionManager) JoinSession(pid internal.PlayerID, sid internal.SessionID, conn *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	handle, ok := s.sessions[sid]
	if !ok {
		return fmt.Errorf("session %s does not exist", sid)
	}
	if handle.IsFinished() {
		return fmt.Errorf("session %s is already finished", sid)
	}

	handle.Join(pid, conn)
	return nil
}

type sessionHandle struct {
	sid      internal.SessionID
	ctx      context.Context
	cancel   context.CancelFunc
	game     game.Kind
	deadline time.Time
	inbox    chan inboxMsg
	result   chan resultMsg
}

func (h *sessionHandle) IsFinished() bool {
	return h.ctx.Err() != nil
}

// Starts a goroutine.
func (h *sessionHandle) RunToCompletion(g game.Kind, deadline time.Time, turnTimeout time.Duration) {
	defer h.cancel()

	switch g {
	case game.TicTacToe:
		protocol := NewProtocol(
			h.inbox,
			h.result,
			h.sid,
			deadline,
			turnTimeout,
			game.NewState(game.NewTicTacToeBoard()),
			&game.TicTacToeSession{},
		)
		protocol.RunToCompletion()
	case game.Connect4:
		protocol := NewProtocol(
			h.inbox,
			h.result,
			h.sid,
			deadline,
			turnTimeout,
			game.NewState(game.NewConnect4Board()),
			&game.Connect4Session{},
		)
		protocol.RunToCompletion()
	case game.Battleship:
		protocol := NewProtocol(
			h.inbox,
			h.result,
			h.sid,
			deadline,
			turnTimeout,
			game.NewState(game.NewBattleshipSharedState()),
			&game.BattleshipSession{},
		)
		protocol.RunToCompletion()
	default:
		log.Fatal("unsupported game kind")
	}
}

// Join a player to a session.
func (h *sessionHandle) Join(pid internal.PlayerID, conn *websocket.Conn) {
	// Send connection request to session protocol.
	stateCh := make(chan PlayerMsg, 1)
	h.inbox <- inboxMsg{
		pid:  pid,
		conn: stateCh,
	}

	// Write goroutine: drains all state messages from the protocol then
	// closes the websocket. Uses context.Background() so writes are not
	// interrupted when the session context is cancelled.
	go func() {
		for state := range stateCh {
			if err := wsjson.Write(context.Background(), conn, state); err != nil {
				_ = conn.Close(websocket.StatusInternalError, "failed to write JSON")
				return
			}
		}
		_ = conn.Close(websocket.StatusNormalClosure, "finished")
	}()

	// Read goroutine: reads moves from the websocket until the connection
	// is closed (by the write goroutine above, or by the player).
	go func() {
		for {
			var move json.RawMessage
			if err := wsjson.Read(context.Background(), conn, &move); err != nil {
				return
			}
			select {
			case h.inbox <- inboxMsg{pid: pid, move: move}:
			case <-h.ctx.Done():
				return
			}
		}
	}()
}
