package internal

import (
	"math"
	"time"
)

// ELO rating constants.
const (
	DefaultElo       = 1000
	KFactor          = 32
	MaxEloDiff       = 200
	EloDiffRelaxRate = 50 // per second waiting
)

// CalcElo computes the new ELO ratings after a match.
// Returns (newWinnerElo, newLoserElo). If draw is true, the result is a tie.
func CalcElo(winnerElo, loserElo int, draw bool) (int, int) {
	expected := 1.0 / (1.0 + math.Pow(10, float64(loserElo-winnerElo)/400.0))

	score := 1.0
	if draw {
		score = 0.5
	}

	newWinner := int(math.Round(float64(winnerElo) + KFactor*(score-expected)))
	newLoser := int(math.Round(float64(loserElo) + KFactor*((1-score)-(1-expected))))

	return newWinner, newLoser
}

// MatchElo determines if two players should match by comparing their Elo while
// taking their wait time into account. Every second a player is in the queue
// relaxes the acceptable difference in Elo between the players.
func MatchElo(a, b int, entryA, entryB time.Time) bool {
	eloDiff := math.Abs(float64(a - b))
	waitTime := math.Max(time.Since(entryA).Seconds(), time.Since(entryB).Seconds())
	relaxedDiff := eloDiff - (EloDiffRelaxRate * math.Max(0, waitTime))
	return relaxedDiff <= MaxEloDiff
}
