package player

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
)

type Completion struct {
	Status      game.Status
	Interrupted bool
}

// Protocol handles the player-side game communication protocol.
type Protocol[Move any, Shared any] struct {
	rx <-chan gameserver.PlayerMsg
	tx chan<- json.RawMessage
	ai game.Ai[Move, Shared]

	player game.Player
}

// NewProtocol creates a Protocol that communicates over the given channels.
func NewProtocol[M any, S any](
	rx <-chan gameserver.PlayerMsg,
	tx chan<- json.RawMessage,
	ai game.Ai[M, S],
) *Protocol[M, S] {
	return &Protocol[M, S]{
		rx: rx,
		tx: tx,
		ai: ai,
	}
}

// Run executes the protocol to completion.
// Returns the final game status and any error encountered.
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
			return Completion{Status: state.Status}, nil
		}
		if state.CurrentPlayer == msg.Player {
			err := p.makeMove(msg.Player, state.Shared)
			if err != nil {
				return Completion{Interrupted: true}, err
			}
		}
	}

	return Completion{Interrupted: true}, nil
}

func (p *Protocol[M, S]) makeMove(player game.Player, shared S) error {
	move, err := p.ai.GetMove(player, shared)
	if err != nil {
		return fmt.Errorf("ai failed: %w", err)
	}
	raw, err := json.Marshal(move)
	if err != nil {
		return fmt.Errorf("encode move: %w", err)
	}
	p.tx <- raw
	return nil
}
