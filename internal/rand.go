package internal

import (
	"math/rand"

	antirandom "github.com/antithesishq/antithesis-sdk-go/random"
)

// NewRand returns a math/rand RNG backed by Antithesis random values.
func NewRand() *rand.Rand {
	return rand.New(antirandom.Source())
}
