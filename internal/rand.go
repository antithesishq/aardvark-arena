package internal

import (
	"math/rand"

	antirandom "github.com/antithesishq/antithesis-sdk-go/random"
)

type antithesisSource struct{}

// Seed is a no-op because randomness is provided externally by Antithesis.
func (antithesisSource) Seed(_ int64) {}

func (s antithesisSource) Int63() int64 {
	return int64(s.Uint64() >> 1)
}

func (antithesisSource) Uint64() uint64 {
	return antirandom.GetRandom()
}

// NewRand returns a math/rand RNG backed by Antithesis random values.
func NewRand() *rand.Rand {
	return rand.New(antithesisSource{})
}
