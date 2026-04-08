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
	"github.com/antithesishq/antithesis-sdk-go/assert"
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

// SessionManager manages active game sessions.
type SessionManager struct {
	// mu protects reads/writes to the sessions map
	mu       sync.Mutex
	sessions map[internal.SessionID]sessionHandle
	resultCh chan resultMsg
	cfg      Config

	// watchMu protects watchers and lastEvents. Lock ordering: mu before watchMu.
	watchMu    sync.Mutex
	watchers   map[chan WatchEvent]struct{}
	lastEvents map[internal.SessionID]*WatchEvent
}

// NewSessionManager creates a new SessionManager with the given config.
func NewSessionManager(cfg Config, resultCh chan resultMsg) *SessionManager {
	return &SessionManager{
		sessions:   make(map[internal.SessionID]sessionHandle),
		resultCh:   resultCh,
		cfg:        cfg,
		watchers:   make(map[chan WatchEvent]struct{}),
		lastEvents: make(map[internal.SessionID]*WatchEvent),
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
			assert.Reachable(
				"session creation sometimes fails because the server is full",
				map[string]any{"active_sessions": len(s.sessions), "max": s.cfg.MaxSessions},
			)
			// find min deadline among active sessions
			var minDeadline time.Time
			for _, session := range s.sessions {
				if !session.IsFinished() && (minDeadline.IsZero() || session.deadline.Before(minDeadline)) {
					minDeadline = session.deadline
				}
			}
			if minDeadline.IsZero() {
				// No active sessions found — retry soon
				minDeadline = time.Now().Add(5 * time.Second)
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
			cleanup: func() {
				s.mu.Lock()
				delete(s.sessions, sid)
				s.mu.Unlock()
				log.Printf("session %s removed", sid)

				s.watchMu.Lock()
				delete(s.lastEvents, sid)
				s.fanoutWatch(WatchEvent{Type: WatchEventSessionEnd, SessionID: sid})
				s.watchMu.Unlock()

				s.broadcastHealth()
			},
			onWatch: func(players map[string]int, state json.RawMessage) {
				evt := WatchEvent{
					Type:      WatchEventSession,
					SessionID: sid,
					Game:      game,
					Players:   players,
					State:     state,
					Deadline:  &deadline,
				}
				s.watchMu.Lock()
				s.lastEvents[sid] = &evt
				s.fanoutWatch(evt)
				s.watchMu.Unlock()
			},
		}
		go handle.RunToCompletion(game, deadline, s.cfg.TurnTimeout)
		s.sessions[sid] = handle
		log.Printf("session %s created", sid)

		// Broadcast health inline — mu is already held, so we can't call
		// broadcastHealth() which would re-acquire it.
		evt := s.healthEvent()
		s.watchMu.Lock()
		s.fanoutWatch(evt)
		s.watchMu.Unlock()
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
	cleanup  func()
	onWatch  func(players map[string]int, state json.RawMessage)
}

func (h *sessionHandle) IsFinished() bool {
	return h.ctx.Err() != nil
}

// Starts a goroutine.
func (h *sessionHandle) RunToCompletion(g game.Kind, deadline time.Time, turnTimeout time.Duration) {
	defer h.cancel()
	defer h.cleanup()

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
			h.onWatch,
		)
		protocol.RunToCompletion(h.ctx.Done())
	case game.Connect4:
		protocol := NewProtocol(
			h.inbox,
			h.result,
			h.sid,
			deadline,
			turnTimeout,
			game.NewState(game.NewConnect4Board()),
			&game.Connect4Session{},
			h.onWatch,
		)
		protocol.RunToCompletion(h.ctx.Done())
	case game.Battleship:
		protocol := NewProtocol(
			h.inbox,
			h.result,
			h.sid,
			deadline,
			turnTimeout,
			game.NewState(game.NewBattleshipSharedState()),
			&game.BattleshipSession{},
			h.onWatch,
		)
		protocol.RunToCompletion(h.ctx.Done())
	default:
		assert.Unreachable(
			"session manager should only run supported game kinds",
			map[string]any{"game": string(g)},
		)
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

// CancelSession cancels a session by its ID.
func (s *SessionManager) CancelSession(sid internal.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	handle, ok := s.sessions[sid]
	if !ok || handle.IsFinished() {
		return fmt.Errorf("session %s not found or already finished", sid)
	}
	handle.cancel()
	return nil
}

// SessionSummary is a brief view of an active session.
type SessionSummary struct {
	SessionID internal.SessionID `json:"session_id"`
	Game      game.Kind          `json:"game"`
}

// ListSessions returns all currently active (unfinished) sessions.
func (s *SessionManager) ListSessions() []SessionSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]SessionSummary, 0, len(s.sessions))
	for sid, handle := range s.sessions {
		if !handle.IsFinished() {
			result = append(result, SessionSummary{
				SessionID: sid,
				Game:      handle.game,
			})
		}
	}
	return result
}

// RegisterWatcher adds a server-level watcher. It sends a snapshot of current
// health and all active sessions, then streams subsequent events until the
// caller calls UnregisterWatcher.
func (s *SessionManager) RegisterWatcher() chan WatchEvent {
	ch := make(chan WatchEvent, 64)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.watchMu.Lock()
	defer s.watchMu.Unlock()

	s.watchers[ch] = struct{}{}

	// Send health snapshot.
	sendLatest(ch, s.healthEvent())

	// Send last known state for every active session.
	for sid := range s.sessions {
		if evt, ok := s.lastEvents[sid]; ok {
			sendLatest(ch, *evt)
		}
	}
	return ch
}

// UnregisterWatcher removes a server-level watcher and closes its channel.
func (s *SessionManager) UnregisterWatcher(ch chan WatchEvent) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	delete(s.watchers, ch)
	close(ch)
}

// fanoutWatch sends an event to all registered watchers. Caller must hold watchMu.
func (s *SessionManager) fanoutWatch(event WatchEvent) {
	for ch := range s.watchers {
		sendLatest(ch, event)
	}
}

// broadcastHealth sends the current health to all watchers.
// Acquires mu then watchMu (correct lock ordering).
func (s *SessionManager) broadcastHealth() {
	s.mu.Lock()
	evt := s.healthEvent()
	s.mu.Unlock()

	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	s.fanoutWatch(evt)
}

// healthEvent returns a WatchEvent with current health info. Caller must hold mu.
func (s *SessionManager) healthEvent() WatchEvent {
	return WatchEvent{
		Type:           WatchEventHealth,
		ActiveSessions: len(s.sessions),
		MaxSessions:    s.cfg.MaxSessions,
	}
}

// WatchSession registers a spectator channel on the given session.
func (s *SessionManager) WatchSession(sid internal.SessionID) (chan PlayerMsg, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	handle, ok := s.sessions[sid]
	if !ok || handle.IsFinished() {
		return nil, fmt.Errorf("session %s not found or already finished", sid)
	}
	ch := make(chan PlayerMsg, 1)
	select {
	case handle.inbox <- inboxMsg{spectatorCh: ch}:
	case <-handle.ctx.Done():
		return nil, fmt.Errorf("session %s ended before spectator could join", sid)
	}
	return ch, nil
}
