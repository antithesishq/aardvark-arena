package internal

import "testing"

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
			checkFunc: func(t *testing.T, newWinner, newLoser int) {
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
			checkFunc: func(t *testing.T, newWinner, newLoser int) {
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
