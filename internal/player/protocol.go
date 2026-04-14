package player

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
)

// Completion holds the result of a finished protocol run.
type Completion struct {
	Status      game.Status
	Interrupted bool
}

// Protocol handles the player-side game communication protocol.
type Protocol[Move any, Shared any] struct {
	rx        <-chan gameserver.PlayerMsg
	tx        chan<- json.RawMessage
	ai        game.Ai[Move, Shared]
	behavior  Behavior
	rng       *rand.Rand
	moveDelay time.Duration
}

// NewProtocol creates a Protocol that communicates over the given channels.
func NewProtocol[M any, S any](
	rx <-chan gameserver.PlayerMsg,
	tx chan<- json.RawMessage,
	ai game.Ai[M, S],
	behavior Behavior,
	moveDelay time.Duration,
) *Protocol[M, S] {
	return &Protocol[M, S]{
		rx:        rx,
		tx:        tx,
		ai:        ai,
		behavior:  behavior,
		rng:       internal.NewRand(),
		moveDelay: moveDelay,
	}
}

// RunToCompletion executes the protocol to completion.
// It returns the final game status and any error encountered.
func (p *Protocol[M, S]) RunToCompletion() (Completion, error) {
	defer close(p.tx)

	for msg := range p.rx {
		if msg.Error != "" {
			log.Printf("server error: %s", msg.Error)
			continue
		}
		var state game.State[S]
		if err := json.Unmarshal(msg.State, &state); err != nil {
			return Completion{Interrupted: true}, fmt.Errorf("decode state: %w", err)
		}
		if state.Status.IsTerminal() {
			assert.Sometimes(state.Status == game.Draw, "games sometimes end in draws", nil)
			assert.Sometimes(state.Status == game.Cancelled, "games sometimes end due to cancellation", nil)
			assert.Sometimes(
				state.Status == game.P1Win || state.Status == game.P2Win,
				"games sometimes end with a winner",
				map[string]any{"status": state.Status.String()},
			)
			return Completion{Status: state.Status}, nil
		}
		if state.CurrentPlayer != msg.Player && p.behavior.doOutOfTurn(p.rng) {
			assert.Reachable(
				"evil players sometimes send nuisance out-of-turn moves",
				nil,
			)
			err := p.makeMove(msg.Player, state.Shared, true)
			if err != nil {
				return Completion{Interrupted: true}, err
			}
			continue
		}
		if state.CurrentPlayer == msg.Player {
			err := p.makeMove(msg.Player, state.Shared, false)
			if err != nil {
				return Completion{Interrupted: true}, err
			}
		}
	}

	return Completion{Interrupted: true}, nil
}

func (p *Protocol[M, S]) makeMove(player game.Player, shared S, forceChaos bool) error {
	if p.moveDelay > 0 {
		time.Sleep(p.moveDelay)
	}
	move, err := p.ai.GetMove(player, shared)
	if err != nil {
		return fmt.Errorf("ai failed: %w", err)
	}
	raw, err := json.Marshal(move)
	if err != nil {
		return fmt.Errorf("encode move: %w", err)
	}
	if forceChaos || p.behavior.doChaos(p.rng) {
		assert.Reachable(
			"evil players sometimes submit intentionally bad moves",
			nil,
		)
		raw = p.corruptMove(raw)
	}
	p.tx <- raw
	return nil
}

func (p *Protocol[M, S]) corruptMove(raw []byte) []byte {
	if p.behavior.toMalformed(p.rng) {
		// Deliberately malformed JSON.
		return []byte(`{"evil":`)
	}
	// Valid JSON but often semantically nonsensical for concrete game move types.
	if len(raw) > 0 && raw[0] == '[' {
		return []byte(`[999999,999999]`)
	}
	if len(raw) > 0 && raw[0] == '{' {
		return []byte(`{"evil":true,"x":999999,"y":999999}`)
	}
	return []byte(`null`)
}
