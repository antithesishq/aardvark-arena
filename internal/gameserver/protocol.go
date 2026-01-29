package gameserver

import (
	"encoding/json"
	"fmt"
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
	err    error
	status game.Status // the final status of the game
}

// Protocol handles the game session communication protocol.
type Protocol[Move any, Shared any] struct {
	inbox  <-chan inboxMsg
	result chan<- resultMsg

	deadline time.Time
	players  map[internal.PlayerID]playerConn
	state    game.State[Shared]
	session  game.Session[Move, Shared]
}

// RunToCompletion runs the game session until it ends or the deadline is reached.
func (p *Protocol[M, S]) RunToCompletion() {
	timer := time.NewTimer(time.Until(p.deadline))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			p.result <- resultMsg{status: game.Cancelled}
			return
		case msg, ok := <-p.inbox:
			if !ok {
				return
			}
			var err error
			if msg.conn != nil && msg.move != nil {
				panic("BUG: both move and conn are set")
			} else if msg.conn != nil {
				err = p.handleConn(msg.pid, msg.conn)
			} else if msg.move != nil {
				err = p.handleMove(msg.pid, msg.move)
			}
			if err != nil {
				p.result <- resultMsg{err: err}
				return
			}
			if p.state.Status.IsTerminal() {
				p.result <- resultMsg{status: p.state.Status}
				return
			}
		}
	}
}

func (p *Protocol[M, S]) handleConn(pid internal.PlayerID, conn chan<- StateOrErr) error {
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
	return p.SendState(pid)
}

func (p *Protocol[M, S]) handleMove(pid internal.PlayerID, rawMove json.RawMessage) error {
	playerConn, ok := p.players[pid]
	if !ok {
		panic("BUG: move from disconnected player")
	}

	var move M
	if err := json.Unmarshal(rawMove, &move); err != nil {
		p.SendErr(pid, fmt.Errorf("invalid move: %w", err))
		return nil
	}

	var err error
	p.state, err = p.session.MakeMove(p.state, playerConn.player, move)
	if err != nil {
		p.SendErr(pid, fmt.Errorf("invalid move: %w", err))
		return nil
	}
	if err := p.BroadcastState(); err != nil {
		return err
	}
	return nil
}

// TrySend attempts to send a message to a player if they are connected.
func (p *Protocol[M, S]) TrySend(pid internal.PlayerID, msg StateOrErr) {
	playerConn, ok := p.players[pid]
	if ok {
		playerConn.conn <- msg
	}
}

// BroadcastState sends the current game state to all connected players.
func (p *Protocol[M, S]) BroadcastState() error {
	for pid := range p.players {
		if err := p.SendState(pid); err != nil {
			return err
		}
	}
	return nil
}

// SendState sends the current game state to a specific player.
func (p *Protocol[M, S]) SendState(pid internal.PlayerID) error {
	encodedState, err := json.Marshal(p.state)
	if err != nil {
		return err
	}
	p.TrySend(pid, StateOrErr{State: encodedState})
	return nil
}

// SendErr sends an error message to a specific player.
func (p *Protocol[M, S]) SendErr(pid internal.PlayerID, err error) {
	p.TrySend(pid, StateOrErr{Error: err.Error()})
}
