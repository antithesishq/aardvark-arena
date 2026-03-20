package gameserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/google/uuid"
)

// PlayerMsg holds either a game state or an error message.
type PlayerMsg struct {
	Player game.Player
	State  json.RawMessage
	Error  string
}

type inboxMsg struct {
	pid internal.PlayerID

	// one of move or conn must be nil
	move        json.RawMessage
	conn        chan PlayerMsg
	spectatorCh chan PlayerMsg
}

type playerConn struct {
	player game.Player
	conn   chan PlayerMsg
}

// resultMsg represents the outcome of a game session.
type resultMsg struct {
	sid       internal.SessionID
	cancelled bool
	winner    internal.PlayerID
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
	spectators  []chan PlayerMsg
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
		turnTimer:  time.NewTimer(turnTimeout * 2),
		players:    make(map[internal.PlayerID]playerConn),
		spectators: nil,
		state:      state,
		session:    session,
	}
}

func (p *Protocol[M, S]) playerToID(player game.Player) internal.PlayerID {
	for pid, p := range p.players {
		if p.player == player {
			return pid
		}
	}
	return uuid.Nil
}

func (p *Protocol[M, S]) report(status game.Status) {
	var winner internal.PlayerID
	switch status {
	case game.P1Win:
		winner = p.playerToID(game.P1)
	case game.P2Win:
		winner = p.playerToID(game.P2)
	}
	log.Printf("session %s ended: status=%s winner=%s", p.sid, status, winner)
	p.state.Status = status
	p.BroadcastState()
	p.result <- resultMsg{
		sid:       p.sid,
		cancelled: status == game.Cancelled,
		winner:    winner,
	}
}

// RunToCompletion runs the game session until it ends, the deadline is reached, or done is closed.
func (p *Protocol[M, S]) RunToCompletion(done <-chan struct{}) {
	timer := time.NewTimer(time.Until(p.deadline))
	defer timer.Stop()

outer:
	for {
		select {
		case <-done:
			p.report(game.Cancelled)
			break outer
		case <-timer.C:
			p.report(game.Cancelled)
			break outer
		case <-p.turnTimer.C:
			if len(p.players) == 0 {
				// No one is connected yet; keep waiting for a connection or deadline.
				p.turnTimer.Reset(p.turnTimeout * 2)
				continue
			}
			p.report(p.state.CurrentPlayer.Opponent().Wins())
			break outer
		case msg, ok := <-p.inbox:
			if !ok {
				break outer
			}
			if msg.spectatorCh != nil {
				p.spectators = append(p.spectators, msg.spectatorCh)
				encoded, err := json.Marshal(p.state)
				if err == nil {
					sendLatest(msg.spectatorCh, PlayerMsg{State: encoded})
				}
				continue
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
				break outer
			}
		}
	}

	// close player conns before finishing the protocol
	for _, pc := range p.players {
		close(pc.conn)
	}
	for _, ch := range p.spectators {
		close(ch)
	}
}

func (p *Protocol[M, S]) handleConn(pid internal.PlayerID, conn chan PlayerMsg) {
	if existing, ok := p.players[pid]; ok {
		assert.Reachable(
			"players sometimes reconnect to an in-progress session",
			map[string]any{"sid": p.sid.String(), "pid": pid.String()},
		)
		// if existing, replace
		close(existing.conn)
		p.players[pid] = playerConn{player: existing.player, conn: conn}
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
		assert.Reachable(
			"extra player connections are sometimes rejected",
			map[string]any{"sid": p.sid.String(), "pid": pid.String()},
		)
		conn <- PlayerMsg{Error: "too many players connected"}
	}

	// make sure the new connection sees the latest state
	p.SendState(pid)
}

func (p *Protocol[M, S]) handleMove(pid internal.PlayerID, rawMove json.RawMessage) {
	playerConn, ok := p.players[pid]
	if !ok {
		assert.Unreachable(
			"moves should only arrive from connected players",
			map[string]any{"sid": p.sid.String(), "pid": pid.String()},
		)
		log.Fatal("BUG: move from disconnected player")
	}

	var move M
	if err := json.Unmarshal(rawMove, &move); err != nil {
		assert.Reachable(
			"sessions sometimes receive invalid move payloads",
			map[string]any{"sid": p.sid.String(), "pid": pid.String()},
		)
		p.SendErr(pid, fmt.Errorf("invalid move: %w", err))
		return
	}

	var err error
	p.state, err = p.session.MakeMove(p.state, playerConn.player, move)
	if err != nil {
		assert.Reachable(
			"sessions sometimes receive invalid semantic moves",
			map[string]any{"sid": p.sid.String(), "pid": pid.String()},
		)
		p.SendErr(pid, fmt.Errorf("invalid move: %w", err))
		return
	}
	// reset turn timer after each valid move
	p.turnTimer.Reset(p.turnTimeout)
	p.BroadcastState()
}

// BroadcastState sends the current game state to all connected players.
func (p *Protocol[M, S]) BroadcastState() {
	for pid := range p.players {
		p.SendState(pid)
	}
	encoded, err := json.Marshal(p.state)
	if err != nil {
		return
	}
	for _, ch := range p.spectators {
		sendLatest(ch, PlayerMsg{State: encoded})
	}
}

// sendLatest sends msg on ch without blocking. If the channel is full,
// it drains the stale value and retries so the channel holds the newest message.
func sendLatest[T any](ch chan T, msg T) {
	select {
	case ch <- msg:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- msg:
		default:
		}
	}
}

// SendState sends the current game state to a specific player.
func (p *Protocol[M, S]) SendState(pid internal.PlayerID) {
	encodedState, err := json.Marshal(p.state)
	if err != nil {
		log.Panicf("failed to marshal state: %v", err)
	}
	playerConn, ok := p.players[pid]
	if ok {
		sendLatest(playerConn.conn, PlayerMsg{Player: playerConn.player, State: encodedState})
	}
}

// SendErr sends an error message to a specific player.
// Drops the message if the channel is full to avoid blocking.
func (p *Protocol[M, S]) SendErr(pid internal.PlayerID, err error) {
	playerConn, ok := p.players[pid]
	if ok {
		msg := PlayerMsg{Player: playerConn.player, Error: err.Error()}
		select {
		case playerConn.conn <- msg:
		default:
		}
	}
}
