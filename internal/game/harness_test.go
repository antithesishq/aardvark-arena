package game

import (
	"fmt"
	"testing"
)

// RunGame drives a game session to completion using the provided AIs.
// It is game-agnostic: it knows nothing about moves, board state, or rules.
// Returns the final state and any error encountered.
func RunGame[Move any, Shared any](
	initialState State[Shared],
	session Session[Move, Shared],
	ais map[Player]Ai[Move, Shared],
) (State[Shared], error) {
	state := initialState
	maxTurns := 1000 // Safety limit to prevent infinite loops

	for range maxTurns {
		if state.Status != Active {
			return state, nil
		}

		// Select the AI for the current player
		ai, ok := ais[state.CurrentPlayer]
		if !ok {
			return state, fmt.Errorf("no AI registered for player %d", state.CurrentPlayer)
		}

		// Get the AI's move
		move, err := ai.GetMove(state.Shared)
		if err != nil {
			return state, fmt.Errorf("player %d AI failed to get move: %w", state.CurrentPlayer, err)
		}

		// Execute the move
		state, err = session.MakeMove(state, state.CurrentPlayer, move)
		if err != nil {
			return state, fmt.Errorf("player %d move failed: %w", state.CurrentPlayer, err)
		}
	}

	return state, fmt.Errorf("game did not terminate within %d turns", maxTurns)
}

// TestTicTacToeHarness verifies the harness works with TicTacToe.
func TestTicTacToeHarness(t *testing.T) {
	t.Run("TicTacToe test", func(t *testing.T) {
		initialState := NewState(NewTicTacToeBoard())
		session := &TicTacToeSession{}
		ais := map[Player]Ai[Position, TicTacToeBoard]{
			P1: NewTicTacToeAi(P1),
			P2: NewTicTacToeAi(P2),
		}
		finalState, err := RunGame(initialState, session, ais)

		if err != nil {
			t.Fatalf("game returned error: %v", err)
		}

		if finalState.Status == Active {
			t.Error("game did not reach terminal state")
		}

		validTerminal := finalState.Status.IsTerminal() && finalState.Status != Cancelled

		if !validTerminal {
			t.Errorf("unexpected terminal status: %v", finalState.Status)
		}
	})
}
