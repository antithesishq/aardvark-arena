package gameserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
)

// StateOrErr holds either a game state or an error message.
type StateOrErr struct {
	State json.RawMessage
	Error string
}

type inboxMsg struct {
	pid internal.PlayerID

	// one of move or conn must be nil
	move json.RawMessage
	conn chan<- StateOrErr
}

type playerConn struct {
	player game.Player
	conn   chan<- StateOrErr
}

// resultMsg represents the outcome of a game session.
type resultMsg struct {
	sid    internal.SessionID
	status game.Status // the final status of the game
}

// Protocol handles the game session communication protocol.
type Protocol[Move any, Shared any] struct {
	inbox  <-chan inboxMsg
	result chan<- resultMsg

	sid         internal.SessionID
	deadline    time.Time
	turnTimeout time.Duration
	turnTimer   *time.Timer
	players     map[internal.PlayerID]playerConn
	state       game.State[Shared]
	session     game.Session[Move, Shared]
}

// NewProtocol creates a Protocol that manages a game session.
func NewProtocol[M any, S any](
	inbox <-chan inboxMsg,
	result chan<- resultMsg,
	sid internal.SessionID,
	deadline time.Time,
	turnTimeout time.Duration,
	state game.State[S],
	session game.Session[M, S],
) Protocol[M, S] {
	return Protocol[M, S]{
		inbox:       inbox,
		result:      result,
		sid:         sid,
		deadline:    deadline,
		turnTimeout: turnTimeout,
		// double the initial turnTimeout as a connection grace period
		turnTimer: time.NewTimer(turnTimeout * 2),
		players:   make(map[internal.PlayerID]playerConn),
		state:     state,
		session:   session,
	}
}

func (p *Protocol[M, S]) report(status game.Status) {
	p.result <- resultMsg{
		sid:    p.sid,
		status: status,
	}
}

// RunToCompletion runs the game session until it ends or the deadline is reached.
func (p *Protocol[M, S]) RunToCompletion() {
	timer := time.NewTimer(time.Until(p.deadline))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			p.report(game.Cancelled)
			return
		case <-p.turnTimer.C:
			p.report(p.state.CurrentPlayer.Opponent().Wins())
			return
		case msg, ok := <-p.inbox:
			if !ok {
				return
			}
			if msg.conn != nil && msg.move != nil {
				log.Fatal("BUG: both move and conn are set")
			} else if msg.conn != nil {
				p.handleConn(msg.pid, msg.conn)
			} else if msg.move != nil {
				p.handleMove(msg.pid, msg.move)
			}
			if p.state.Status.IsTerminal() {
				p.report(p.state.Status)
				return
			}
		}
	}
}

func (p *Protocol[M, S]) handleConn(pid internal.PlayerID, conn chan<- StateOrErr) {
	if existing, ok := p.players[pid]; ok {
		// if existing, replace
		close(existing.conn)
		existing.conn = conn
	} else if len(p.players) == 0 {
		// first player to connect is P1
		p.players[pid] = playerConn{
			player: game.P1,
			conn:   conn,
		}
	} else if len(p.players) == 1 {
		// second player to connect is P2
		p.players[pid] = playerConn{
			player: game.P2,
			conn:   conn,
		}
	} else {
		conn <- StateOrErr{Error: "too many players connected"}
	}

	// make sure the new connection sees the latest state
	p.SendState(pid)
}

func (p *Protocol[M, S]) handleMove(pid internal.PlayerID, rawMove json.RawMessage) {
	playerConn, ok := p.players[pid]
	if !ok {
		log.Fatal("BUG: move from disconnected player")
	}

	var move M
	if err := json.Unmarshal(rawMove, &move); err != nil {
		p.SendErr(pid, fmt.Errorf("invalid move: %w", err))
		return
	}

	var err error
	p.state, err = p.session.MakeMove(p.state, playerConn.player, move)
	if err != nil {
		p.SendErr(pid, fmt.Errorf("invalid move: %w", err))
		return
	}
	// reset turn timer after each valid move
	p.turnTimer.Reset(p.turnTimeout)
	p.BroadcastState()
}

// TrySend attempts to send a message to a player if they are connected.
func (p *Protocol[M, S]) TrySend(pid internal.PlayerID, msg StateOrErr) {
	playerConn, ok := p.players[pid]
	if ok {
		playerConn.conn <- msg
	}
}

// BroadcastState sends the current game state to all connected players.
func (p *Protocol[M, S]) BroadcastState() {
	for pid := range p.players {
		p.SendState(pid)
	}
}

// SendState sends the current game state to a specific player.
func (p *Protocol[M, S]) SendState(pid internal.PlayerID) {
	encodedState, err := json.Marshal(p.state)
	if err != nil {
		log.Panicf("failed to marshal state: %v", err)
	}
	p.TrySend(pid, StateOrErr{State: encodedState})
}

// SendErr sends an error message to a specific player.
func (p *Protocol[M, S]) SendErr(pid internal.PlayerID, err error) {
	p.TrySend(pid, StateOrErr{Error: err.Error()})
}
