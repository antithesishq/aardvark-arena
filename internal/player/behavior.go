package player

import (
	"math/rand"
)

// Behavior configures how often a player behaves maliciously.
type Behavior struct {
	// Evil enables malicious behavior.
	Evil bool

	// ChaosRate is the probability [0,1] that a move attempt is corrupted.
	ChaosRate float64

	// OutOfTurnRate is the probability [0,1] of sending a nuisance move
	// when it's not this player's turn.
	OutOfTurnRate float64

	// MalformedRate is the probability [0,1] that a chaotic move is malformed
	// JSON; otherwise it is valid JSON that is likely semantically invalid.
	MalformedRate float64

	// ExtraConnectRate is the probability [0,1] of attempting a background
	// websocket join with a random player id to the current session.
	ExtraConnectRate float64

	// QueueAbandonRate is the probability [0,1] of submitting a queue request
	// for a random throwaway player id, and never polling it again.
	QueueAbandonRate float64
}

func (b Behavior) doChaos(rng *rand.Rand) bool {
	return b.Evil && rng.Float64() < b.ChaosRate
}

func (b Behavior) doOutOfTurn(rng *rand.Rand) bool {
	return b.Evil && rng.Float64() < b.OutOfTurnRate
}

func (b Behavior) toMalformed(rng *rand.Rand) bool {
	return b.Evil && rng.Float64() < b.MalformedRate
}

func (b Behavior) doExtraConnect(rng *rand.Rand) bool {
	return b.Evil && rng.Float64() < b.ExtraConnectRate
}

func (b Behavior) doQueueAbandon(rng *rand.Rand) bool {
	return b.Evil && rng.Float64() < b.QueueAbandonRate
}
