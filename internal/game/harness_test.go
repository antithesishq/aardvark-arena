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
	ais PlayerMap[Ai[Move, Shared]],
) (State[Shared], error) {
	state := initialState
	maxTurns := 1000 // Safety limit to prevent infinite loops

	for range maxTurns {
		if state.Status != Active {
			return state, nil
		}

		// Select the AI for the current player
		ai := ais.Get(state.CurrentPlayer)

		// Get the AI's move
		move, err := ai.GetMove(state.CurrentPlayer, state.Shared)
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
		ais := PlayerMap[Ai[Position, TicTacToeBoard]]{
			P1: NewTicTacToeAi(),
			P2: NewTicTacToeAi(),
		}
		finalState, err := RunGame(initialState, session, ais)

		if err != nil {
			t.Fatalf("game returned error: %v", err)
		}

		if !finalState.Status.In(P1Win, P2Win, Draw) {
			t.Errorf("unexpected terminal status: %v", finalState.Status)
		}
	})
}

// TestBattleshipHarness verifies the harness works with Battleship.
func TestBattleshipHarness(t *testing.T) {
	t.Run("Battleship test", func(t *testing.T) {
		initialState := NewState(NewBattleshipSharedState())
		session := NewBattleshipSession()
		ais := PlayerMap[Ai[BattleshipMove, BattleshipSharedState]]{
			P1: NewBattleshipAi(),
			P2: NewBattleshipAi(),
		}
		finalState, err := RunGame(initialState, session, ais)

		if err != nil {
			t.Fatalf("game returned error: %v", err)
		}

		if !finalState.Status.In(P1Win, P2Win) {
			t.Errorf("unexpected terminal status: %v", finalState.Status)
		}
	})
}

// TestConnect4Harness verifies the harness works with Connect4.
func TestConnect4Harness(t *testing.T) {
	t.Run("Connect4 test", func(t *testing.T) {
		initialState := NewState(NewConnect4Board())
		session := &Connect4Session{}
		ais := PlayerMap[Ai[int, Connect4Board]]{
			P1: NewConnect4Ai(),
			P2: NewConnect4Ai(),
		}
		finalState, err := RunGame(initialState, session, ais)

		if err != nil {
			t.Fatalf("game returned error: %v", err)
		}

		if !finalState.Status.In(P1Win, P2Win, Draw) {
			t.Errorf("unexpected terminal status: %v", finalState.Status)
		}
	})
}
