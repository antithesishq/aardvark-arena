package game

import (
	"math"
	"testing"

	"hegel.dev/go/hegel"
)

func TestPlayerOpponentInvolution(t *testing.T) {
	t.Run("opponent of opponent is self", hegel.Case(func(ht *hegel.T) {
		p := hegel.Draw(ht, hegel.SampledFrom([]Player{P1, P2}))
		if p.Opponent().Opponent() != p {
			ht.Fatalf("Opponent is not an involution for %v", p)
		}
	}))

	t.Run("opponent is different from self", hegel.Case(func(ht *hegel.T) {
		p := hegel.Draw(ht, hegel.SampledFrom([]Player{P1, P2}))
		if p.Opponent() == p {
			ht.Fatalf("Opponent returned self for %v", p)
		}
	}))
}

func TestPositionMarshalRoundtrip(t *testing.T) {
	t.Run("MarshalText -> UnmarshalText recovers original", hegel.Case(func(ht *hegel.T) {
		x := hegel.Draw(ht, hegel.Integers[int](math.MinInt32, math.MaxInt32))
		y := hegel.Draw(ht, hegel.Integers[int](math.MinInt32, math.MaxInt32))
		original := Position{X: x, Y: y}

		text, err := original.MarshalText()
		if err != nil {
			ht.Fatalf("MarshalText failed: %v", err)
		}

		var decoded Position
		if err := decoded.UnmarshalText(text); err != nil {
			ht.Fatalf("UnmarshalText failed for %q: %v", text, err)
		}

		if decoded != original {
			ht.Fatalf("roundtrip failed: %v -> %q -> %v", original, text, decoded)
		}
	}))
}

func TestKindMarshalRoundtrip(t *testing.T) {
	t.Run("MarshalText -> UnmarshalText recovers original", hegel.Case(func(ht *hegel.T) {
		original := hegel.Draw(ht, hegel.SampledFrom(AllGames[:]))

		text, err := original.MarshalText()
		if err != nil {
			ht.Fatalf("MarshalText failed: %v", err)
		}

		var decoded Kind
		if err := decoded.UnmarshalText(text); err != nil {
			ht.Fatalf("UnmarshalText failed for %q: %v", text, err)
		}

		if decoded != original {
			ht.Fatalf("roundtrip failed: %v -> %q -> %v", original, text, decoded)
		}
	}))
}

func TestKindUnmarshalRejectsInvalid(t *testing.T) {
	t.Run("invalid text is rejected", hegel.Case(func(ht *hegel.T) {
		s := hegel.Draw(ht, hegel.Text(0, 50))
		// Skip if it happens to be a valid kind
		for _, k := range AllGames {
			if s == string(k) {
				ht.Skip()
			}
		}
		var k Kind
		if err := k.UnmarshalText([]byte(s)); err == nil {
			ht.Fatalf("expected error for invalid kind %q, got nil", s)
		}
	}))
}

func TestInBoundsCorrectness(t *testing.T) {
	t.Run("matches manual check", hegel.Case(func(ht *hegel.T) {
		x := hegel.Draw(ht, hegel.Integers[int](-10, 20))
		y := hegel.Draw(ht, hegel.Integers[int](-10, 20))
		w := hegel.Draw(ht, hegel.Integers[int](1, 15))
		h := hegel.Draw(ht, hegel.Integers[int](1, 15))

		pos := Position{X: x, Y: y}
		bounds := Bounds{Width: w, Height: h}

		expected := x >= 0 && x < w && y >= 0 && y < h
		got := pos.InBounds(bounds)
		if got != expected {
			ht.Fatalf("InBounds(%v, %v) = %v, expected %v", pos, bounds, got, expected)
		}
	}))
}

func TestStatusIsTerminal(t *testing.T) {
	t.Run("only Active is non-terminal", hegel.Case(func(ht *hegel.T) {
		s := hegel.Draw(ht, hegel.SampledFrom([]Status{Active, P1Win, P2Win, Draw, Cancelled}))
		if s == Active && s.IsTerminal() {
			ht.Fatal("Active should not be terminal")
		}
		if s != Active && !s.IsTerminal() {
			ht.Fatalf("%v should be terminal", s)
		}
	}))
}

func TestTicTacToeMoveProperties(t *testing.T) {
	t.Run("valid move occupies exactly one cell", hegel.Case(func(ht *hegel.T) {
		state := NewState(NewTicTacToeBoard())
		session := &TicTacToeSession{}

		// Make a few random valid moves to get to an interesting state
		nMoves := hegel.Draw(ht, hegel.Integers[int](0, 8))
		for i := 0; i < nMoves; i++ {
			if state.Status != Active {
				break
			}
			empties := state.Shared.empties()
			if len(empties) == 0 {
				break
			}
			idx := hegel.Draw(ht, hegel.Integers[int](0, len(empties)-1))
			move := empties[idx]
			var err error
			state, err = session.MakeMove(state, state.CurrentPlayer, move)
			if err != nil {
				ht.Fatalf("valid move failed: %v", err)
			}
		}

		if state.Status != Active {
			return
		}

		empties := state.Shared.empties()
		emptyBefore := len(empties)
		idx := hegel.Draw(ht, hegel.Integers[int](0, len(empties)-1))
		move := empties[idx]
		player := state.CurrentPlayer

		newState, err := session.MakeMove(state, player, move)
		if err != nil {
			ht.Fatalf("valid move failed: %v", err)
		}

		emptyAfter := len(newState.Shared.empties())
		if emptyAfter != emptyBefore-1 {
			ht.Fatalf("expected %d empties after move, got %d", emptyBefore-1, emptyAfter)
		}

		// The cell should now be occupied by the player who moved
		if newState.Shared.Cells[move.X][move.Y] == nil || *newState.Shared.Cells[move.X][move.Y] != player {
			ht.Fatalf("cell (%d,%d) not occupied by player %v after move", move.X, move.Y, player)
		}
	}))

	t.Run("non-terminal move switches current player", hegel.Case(func(ht *hegel.T) {
		state := NewState(NewTicTacToeBoard())
		session := &TicTacToeSession{}

		nMoves := hegel.Draw(ht, hegel.Integers[int](0, 7))
		for i := 0; i < nMoves; i++ {
			if state.Status != Active {
				break
			}
			empties := state.Shared.empties()
			if len(empties) == 0 {
				break
			}
			idx := hegel.Draw(ht, hegel.Integers[int](0, len(empties)-1))
			var err error
			state, err = session.MakeMove(state, state.CurrentPlayer, empties[idx])
			if err != nil {
				ht.Fatalf("valid move failed: %v", err)
			}
		}

		if state.Status != Active {
			return
		}

		empties := state.Shared.empties()
		idx := hegel.Draw(ht, hegel.Integers[int](0, len(empties)-1))
		playerBefore := state.CurrentPlayer

		newState, err := session.MakeMove(state, playerBefore, empties[idx])
		if err != nil {
			ht.Fatalf("valid move failed: %v", err)
		}

		if newState.Status == Active && newState.CurrentPlayer == playerBefore {
			ht.Fatal("current player should switch after a non-terminal move")
		}
	}))

	t.Run("invalid moves are rejected", hegel.Case(func(ht *hegel.T) {
		state := NewState(NewTicTacToeBoard())
		session := &TicTacToeSession{}

		// Out of bounds move
		x := hegel.Draw(ht, hegel.Integers[int](-10, 10))
		y := hegel.Draw(ht, hegel.Integers[int](-10, 10))
		pos := Position{X: x, Y: y}

		if !pos.InBounds(ticTacToeBounds) {
			_, err := session.MakeMove(state, state.CurrentPlayer, pos)
			if err == nil {
				ht.Fatalf("out-of-bounds move (%d,%d) should be rejected", x, y)
			}
		}
	}))
}

func TestConnect4GravityProperty(t *testing.T) {
	t.Run("piece lands at lowest empty row", hegel.Case(func(ht *hegel.T) {
		state := NewState(NewConnect4Board())
		session := &Connect4Session{}

		// Make a sequence of random valid moves
		nMoves := hegel.Draw(ht, hegel.Integers[int](0, 30))
		for i := 0; i < nMoves; i++ {
			if state.Status != Active {
				break
			}
			cols := state.Shared.validColumns()
			if len(cols) == 0 {
				break
			}
			idx := hegel.Draw(ht, hegel.Integers[int](0, len(cols)-1))
			var err error
			state, err = session.MakeMove(state, state.CurrentPlayer, cols[idx])
			if err != nil {
				ht.Fatalf("valid move failed: %v", err)
			}
		}

		if state.Status != Active {
			return
		}

		cols := state.Shared.validColumns()
		if len(cols) == 0 {
			return
		}
		idx := hegel.Draw(ht, hegel.Integers[int](0, len(cols)-1))
		col := cols[idx]

		expectedRow := state.Shared.lowestEmpty(col)
		player := state.CurrentPlayer

		newState, err := session.MakeMove(state, player, col)
		if err != nil {
			ht.Fatalf("valid move to col %d failed: %v", col, err)
		}

		// Verify the piece landed at the expected row
		if newState.Shared.Cells[col][expectedRow] == nil {
			ht.Fatalf("piece did not land at row %d in col %d", expectedRow, col)
		}
		if *newState.Shared.Cells[col][expectedRow] != player {
			ht.Fatalf("wrong player at (%d,%d): got %v, expected %v",
				col, expectedRow, *newState.Shared.Cells[col][expectedRow], player)
		}

		// All rows below should be occupied
		for r := 0; r < expectedRow; r++ {
			if newState.Shared.Cells[col][r] == nil {
				ht.Fatalf("row %d below piece in col %d is empty (gravity violated)", r, col)
			}
		}
	}))

	t.Run("full column is rejected", hegel.Case(func(ht *hegel.T) {
		state := NewState(NewConnect4Board())
		session := &Connect4Session{}

		// Fill a random column
		col := hegel.Draw(ht, hegel.Integers[int](0, 6))
		for row := 0; row < 6; row++ {
			p := Player(row % 2)
			state.Shared.Cells[col][row] = &p
		}

		_, err := session.MakeMove(state, state.CurrentPlayer, col)
		if err == nil {
			ht.Fatalf("move to full column %d should be rejected", col)
		}
	}))
}
