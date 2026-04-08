package internal

import (
	"math/rand"
)

// NewRand returns a new math/rand RNG.
func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(rand.Int63()))
}
