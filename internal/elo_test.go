package internal

import (
	"math"
	"testing"
	"time"

	"hegel.dev/go/hegel"
)

func TestCalcElo(t *testing.T) {
	tests := []struct {
		name      string
		winner    int
		loser     int
		draw      bool
		checkFunc func(t *testing.T, newWinner, newLoser int)
	}{
		{
			name:   "equal players win",
			winner: 1000, loser: 1000, draw: false,
			checkFunc: func(t *testing.T, newWinner, newLoser int) {
				if newWinner <= 1000 {
					t.Errorf("winner should increase, got %d", newWinner)
				}
				if newLoser >= 1000 {
					t.Errorf("loser should decrease, got %d", newLoser)
				}
			},
		},
		{
			name:   "equal players draw",
			winner: 1000, loser: 1000, draw: true,
			checkFunc: func(t *testing.T, newWinner, newLoser int) {
				if newWinner != 1000 || newLoser != 1000 {
					t.Errorf("should not change, got %d and %d", newWinner, newLoser)
				}
			},
		},
		{
			name:   "underdog wins",
			winner: 800, loser: 1200, draw: false,
			checkFunc: func(t *testing.T, newWinner, _ int) {
				if newWinner <= 800 {
					t.Errorf("underdog should increase, got %d", newWinner)
				}
				if newWinner-800 <= 16 {
					t.Errorf("underdog should gain more than equal match, got %d", newWinner-800)
				}
			},
		},
		{
			name:   "favorite wins",
			winner: 1200, loser: 800, draw: false,
			checkFunc: func(t *testing.T, newWinner, _ int) {
				if newWinner <= 1200 {
					t.Errorf("favorite should increase, got %d", newWinner)
				}
				if newWinner-1200 >= 16 {
					t.Errorf("favorite should gain less than equal match, got %d", newWinner-1200)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newWinner, newLoser := CalcElo(tt.winner, tt.loser, tt.draw)
			tt.checkFunc(t, newWinner, newLoser)

			// all cases should be zero sum
			delta := (newWinner - tt.winner) + (newLoser - tt.loser)
			if delta != 0 {
				t.Errorf("should be zero sum, got %d", delta)
			}
		})
	}
}

func TestCalcEloProperties(t *testing.T) {
	t.Run("zero-sum: total ELO is conserved", hegel.Case(func(ht *hegel.T) {
		winnerElo := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		loserElo := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		draw := hegel.Draw(ht, hegel.Booleans())

		newWinner, newLoser := CalcElo(winnerElo, loserElo, draw)
		delta := (newWinner - winnerElo) + (newLoser - loserElo)
		if delta != 0 {
			ht.Fatalf("not zero-sum: winner %d->%d, loser %d->%d, delta=%d",
				winnerElo, newWinner, loserElo, newLoser, delta)
		}
	}))

	t.Run("winner never loses ELO on a win", hegel.Case(func(ht *hegel.T) {
		winnerElo := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		loserElo := hegel.Draw(ht, hegel.Integers[int](0, 4000))

		newWinner, _ := CalcElo(winnerElo, loserElo, false)
		if newWinner < winnerElo {
			ht.Fatalf("winner lost ELO: %d -> %d", winnerElo, newWinner)
		}
	}))

	t.Run("loser never gains ELO on a loss", hegel.Case(func(ht *hegel.T) {
		winnerElo := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		loserElo := hegel.Draw(ht, hegel.Integers[int](0, 4000))

		_, newLoser := CalcElo(winnerElo, loserElo, false)
		if newLoser > loserElo {
			ht.Fatalf("loser gained ELO: %d -> %d", loserElo, newLoser)
		}
	}))

	t.Run("draw between equal players changes nothing", hegel.Case(func(ht *hegel.T) {
		elo := hegel.Draw(ht, hegel.Integers[int](0, 4000))

		newA, newB := CalcElo(elo, elo, true)
		if newA != elo || newB != elo {
			ht.Fatalf("draw between equal players changed ELO: %d -> (%d, %d)", elo, newA, newB)
		}
	}))

	t.Run("underdog gains at least as much as favorite on win", hegel.Case(func(ht *hegel.T) {
		low := hegel.Draw(ht, hegel.Integers[int](0, 3000))
		gap := hegel.Draw(ht, hegel.Integers[int](1, 1000))
		high := low + gap

		// Underdog (low) wins
		underdogNew, _ := CalcElo(low, high, false)
		underdogGain := underdogNew - low

		// Favorite (high) wins
		favoriteNew, _ := CalcElo(high, low, false)
		favoriteGain := favoriteNew - high

		if underdogGain < favoriteGain {
			ht.Fatalf("underdog gain (%d) should be >= favorite gain (%d) [low=%d high=%d]",
				underdogGain, favoriteGain, low, high)
		}
	}))
}

func TestMatchEloProperties(t *testing.T) {
	t.Run("symmetric: order of players does not matter", hegel.Case(func(ht *hegel.T) {
		a := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		b := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		// Use fixed past times so time.Since is stable within a single test case
		baseTime := time.Now().Add(-10 * time.Second)
		offsetA := hegel.Draw(ht, hegel.Integers[int](0, 30))
		offsetB := hegel.Draw(ht, hegel.Integers[int](0, 30))
		entryA := baseTime.Add(-time.Duration(offsetA) * time.Second)
		entryB := baseTime.Add(-time.Duration(offsetB) * time.Second)

		ab := MatchElo(a, b, entryA, entryB)
		ba := MatchElo(b, a, entryB, entryA)
		if ab != ba {
			ht.Fatalf("MatchElo not symmetric: (%d,%d)=%v but (%d,%d)=%v",
				a, b, ab, b, a, ba)
		}
	}))

	t.Run("close ELO always matches", hegel.Case(func(ht *hegel.T) {
		base := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		diff := hegel.Draw(ht, hegel.Integers[int](0, MaxEloDiff))
		a := base
		b := base + diff
		now := time.Now()

		if !MatchElo(a, b, now, now) {
			ht.Fatalf("players within MaxEloDiff should match: a=%d b=%d diff=%d", a, b, diff)
		}
	}))

	t.Run("ELO difference is non-negative", hegel.Case(func(ht *hegel.T) {
		a := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		b := hegel.Draw(ht, hegel.Integers[int](0, 4000))
		now := time.Now()

		// If difference is within MaxEloDiff, should match
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		result := MatchElo(a, b, now, now)
		if diff <= MaxEloDiff && !result {
			ht.Fatalf("should match: a=%d b=%d diff=%d <= MaxEloDiff=%d", a, b, diff, MaxEloDiff)
		}
	}))

	t.Run("longer wait relaxes threshold", hegel.Case(func(ht *hegel.T) {
		base := hegel.Draw(ht, hegel.Integers[int](500, 3500))
		gap := hegel.Draw(ht, hegel.Integers[int](MaxEloDiff+1, 1000))
		a := base
		b := base + gap

		// With zero wait, should NOT match
		now := time.Now()
		if MatchElo(a, b, now, now) {
			ht.Fatalf("should not match with zero wait: a=%d b=%d gap=%d", a, b, gap)
		}

		// With enough wait, should match
		// relaxedDiff = gap - EloDiffRelaxRate * waitTime <= MaxEloDiff
		// waitTime >= (gap - MaxEloDiff) / EloDiffRelaxRate
		neededWait := float64(gap-MaxEloDiff)/float64(EloDiffRelaxRate) + 1
		longAgo := now.Add(-time.Duration(neededWait) * time.Second)
		if !MatchElo(a, b, longAgo, longAgo) {
			ht.Fatalf("should match with long wait: a=%d b=%d gap=%d neededWait=%.1fs",
				a, b, gap, neededWait)
		}
	}))
}

// Verify that CalcElo does not produce extreme values or panic on extreme ELO inputs.
func TestCalcEloRobustness(t *testing.T) {
	t.Run("no crash on extreme ELO values", hegel.Case(func(ht *hegel.T) {
		winnerElo := hegel.Draw(ht, hegel.Integers[int](math.MinInt32, math.MaxInt32))
		loserElo := hegel.Draw(ht, hegel.Integers[int](math.MinInt32, math.MaxInt32))
		draw := hegel.Draw(ht, hegel.Booleans())

		// Should not panic
		CalcElo(winnerElo, loserElo, draw)
	}))
}
