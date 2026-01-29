package gameserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// ErrMaxSessions is returned when the server has reached its maximum session capacity.
var ErrMaxSessions = errors.New("max sessions reached")

type SessionManager struct {
	// mu protects reads/writes to the sessions map
	mu       sync.Mutex
	sessions map[internal.SessionID]sessionHandle
	resultCh chan resultMsg
	cfg      Config
}

func NewSessionManager(cfg Config) *SessionManager {
	return &SessionManager{
		sessions: make(map[internal.SessionID]sessionHandle),
		resultCh: make(chan resultMsg, cfg.MaxSessions),
		cfg:      cfg,
	}
}

func (s *SessionManager) ActiveSessions() int {
	return len(s.sessions)
}

// CreateSession idempotently creates a session, returning an error if the
// session is finished.
func (s *SessionManager) CreateSession(sid internal.SessionID, game game.Kind, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sid]; !ok {
		if len(s.sessions) >= s.cfg.MaxSessions {
			return ErrMaxSessions
		}
		inboxCh := make(chan inboxMsg, 2)
		ctx, cancel := context.WithCancel(context.Background())
		handle := sessionHandle{
			ctx:    ctx,
			cancel: cancel,
			inbox:  inboxCh,
			result: s.resultCh,
		}
		go handle.RunToCompletion(game, deadline)
		s.sessions[sid] = handle
	}

	handle := s.sessions[sid]
	if handle.IsFinished() {
		return fmt.Errorf("session %s is already finished", sid)
	}
	return nil
}

type sessionHandle struct {
	ctx    context.Context
	cancel context.CancelFunc
	inbox  chan inboxMsg
	result chan resultMsg
}

func (h *sessionHandle) IsFinished() bool {
	select {
	case <-h.ctx.Done():
		return true
	default:
		return false
	}
}

// Starts a goroutine
func (h *sessionHandle) RunToCompletion(g game.Kind, deadline time.Time) {
	defer h.cancel()

	switch g {
	case game.TicTacToe:
		protocol := Protocol[game.Position, game.TicTacToeBoard]{
			inbox:    h.inbox,
			result:   h.result,
			deadline: deadline,
			players:  make(map[internal.PlayerID]playerConn),
			state:    game.NewState(game.NewTicTacToeBoard()),
			session:  &game.TicTacToeSession{},
		}
		protocol.RunToCompletion()
	case game.Connect4:
		protocol := Protocol[int, game.Connect4Board]{
			inbox:    h.inbox,
			result:   h.result,
			deadline: deadline,
			players:  make(map[internal.PlayerID]playerConn),
			state:    game.NewState(game.NewConnect4Board()),
			session:  &game.Connect4Session{},
		}
		protocol.RunToCompletion()
	case game.Battleship:
		protocol := Protocol[game.BattleshipMove, game.BattleshipSharedState]{
			inbox:    h.inbox,
			result:   h.result,
			deadline: deadline,
			players:  make(map[internal.PlayerID]playerConn),
			state:    game.NewState(game.NewBattleshipSharedState()),
			session:  game.NewBattleshipSession(),
		}
		protocol.RunToCompletion()
	}
}

// Join a player to a session
func (h *sessionHandle) Join(pid internal.PlayerID, conn *websocket.Conn) {
	// Send connection request to session protocol.
	stateCh := make(chan StateOrErr, 1)
	h.inbox <- inboxMsg{
		pid:  pid,
		conn: stateCh,
	}

	// Create another cancellable context, to ensure that when either connection
	// goroutine exits, the other one also exits.
	// This goroutine will also be cancelled when the session finishes.
	ctx, cancel := context.WithCancel(h.ctx)

	// This goroutine reads messages from the ws
	go func() {
		defer cancel()
		var move json.RawMessage
		err := wsjson.Read(ctx, conn, &move)
		if err != nil {
			_ = conn.Close(4000, "failed to read JSON")
			return
		}
		h.inbox <- inboxMsg{
			pid:  pid,
			move: move,
		}
	}()

	// This goroutine writes messages to the ws
	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case state, ok := <-stateCh:
				if !ok {
					return
				}
				err := wsjson.Write(ctx, conn, state)
				if err != nil {
					_ = conn.Close(websocket.StatusInternalError, "failed to write JSON")
					return
				}
			}
		}
	}()
}
