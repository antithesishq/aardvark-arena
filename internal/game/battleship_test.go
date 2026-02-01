package game

import (
	"testing"
)

func TestAiHuntsAdjacentToHits(t *testing.T) {
	ai := NewBattleshipAi()
	ai.setup = true // Skip setup phase

	// Create state with a single hit in the middle of the board
	state := NewBattleshipSharedState()
	hitPos := Position{X: 5, Y: 5}
	state.Attacks.Get(P2)[hitPos] = Hit // P1's attacks are stored on P2's board

	adjacent := map[Position]bool{
		{X: 5, Y: 4}: true,
		{X: 5, Y: 6}: true,
		{X: 4, Y: 5}: true,
		{X: 6, Y: 5}: true,
	}

	// The AI should always pick one of the adjacend empty cells
	move, err := ai.GetMove(P1, state)
	if err != nil {
		t.Fatalf("GetMove failed: %v", err)
	}
	if move.Kind != AttackMoveKind {
		t.Fatalf("expected attack move, got %v", move.Kind)
	}
	if !adjacent[move.Target] {
		t.Errorf("AI chose %v which is not adjacent to hit at %v", move.Target, hitPos)
	}
}

func TestAiFallsBackToRandomWhenNoAdjacentUnknown(t *testing.T) {
	ai := NewBattleshipAi()
	ai.setup = true

	// Create state with a hit and all adjacent cells explored
	state := NewBattleshipSharedState()
	hitPos := Position{X: 5, Y: 5}
	opponentBoard := state.Attacks.Get(P2)
	opponentBoard[hitPos] = Hit
	opponentBoard[Position{X: 5, Y: 4}] = Miss
	opponentBoard[Position{X: 5, Y: 6}] = Miss
	opponentBoard[Position{X: 4, Y: 5}] = Miss
	opponentBoard[Position{X: 6, Y: 5}] = Miss

	// AI should pick a random cell (not the hit or adjacent misses)
	explored := map[Position]bool{
		hitPos:       true,
		{X: 5, Y: 4}: true,
		{X: 5, Y: 6}: true,
		{X: 4, Y: 5}: true,
		{X: 6, Y: 5}: true,
	}

	move, err := ai.GetMove(P1, state)
	if err != nil {
		t.Fatalf("GetMove failed: %v", err)
	}
	if explored[move.Target] {
		t.Errorf("AI chose already explored cell %v", move.Target)
	}
	if !move.Target.InBounds(battleshipBounds) {
		t.Errorf("AI chose out of bounds cell %v", move.Target)
	}
}
