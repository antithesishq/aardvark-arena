package game

import (
	"fmt"
	"math/rand"
)

// lines contains all winning lines: 3 rows, 3 columns, 2 diagonals.
// Each line is an array of 3 positions.
var lines = [8][3]Position{
	// Rows
	{{0, 0}, {1, 0}, {2, 0}},
	{{0, 1}, {1, 1}, {2, 1}},
	{{0, 2}, {1, 2}, {2, 2}},
	// Columns
	{{0, 0}, {0, 1}, {0, 2}},
	{{1, 0}, {1, 1}, {1, 2}},
	{{2, 0}, {2, 1}, {2, 2}},
	// Diagonals
	{{0, 0}, {1, 1}, {2, 2}}, // top-left to bottom-right
	{{0, 2}, {1, 1}, {2, 0}}, // bottom-left to top-right
}

var ticTacToeBounds = Bounds{Width: 3, Height: 3}

// TicTacToeBoard is the shared state for a TicTacToe game.
// Each cell is nil (empty) or points to the player who occupies it.
type TicTacToeBoard struct {
	// Cells is a 3x3 grid where each cell is nil or points to the occupying player.
	Cells [3][3]*Player
}

// NewTicTacToeBoard creates an empty TicTacToe board.
func NewTicTacToeBoard() TicTacToeBoard {
	return TicTacToeBoard{}
}

func (b TicTacToeBoard) countOccupiedBy(line [3]Position, player Player) int {
	count := 0
	for _, pos := range line {
		cell := b.Cells[pos.X][pos.Y]
		if cell != nil && *cell == player {
			count++
		}
	}
	return count
}

func (b TicTacToeBoard) firstEmpty(line [3]Position) *Position {
	for _, pos := range line {
		if b.Cells[pos.X][pos.Y] == nil {
			return &pos
		}
	}
	return nil
}

func (b TicTacToeBoard) empties() []Position {
	var empty []Position
	for x := range 3 {
		for y := range 3 {
			if b.Cells[x][y] == nil {
				empty = append(empty, Position{X: x, Y: y})
			}
		}
	}
	return empty
}

func (b TicTacToeBoard) checkWinFor(player Player) bool {
	for _, line := range lines {
		if b.countOccupiedBy(line, player) == 3 {
			return true
		}
	}
	return false
}

func (b TicTacToeBoard) isFull() bool {
	for x := range 3 {
		for y := range 3 {
			if b.Cells[x][y] == nil {
				return false
			}
		}
	}
	return true
}

// TicTacToeSession implements GameSession for TicTacToe.
type TicTacToeSession struct{}

// MakeMove processes a move for the given player at the specified position.
func (s *TicTacToeSession) MakeMove(state State[TicTacToeBoard], player Player, move Position) (State[TicTacToeBoard], error) {
	if err := state.CanMakeMove(player); err != nil {
		return state, err
	}
	if !move.InBounds(ticTacToeBounds) {
		return state, fmt.Errorf("position out of bounds")
	}
	if state.Shared.Cells[move.X][move.Y] != nil {
		return state, fmt.Errorf("cell is already occupied")
	}

	// Place the player's mark
	state.Shared.Cells[move.X][move.Y] = &player

	// Check for win
	if state.Shared.checkWinFor(player) {
		state.Status = player.Wins()
		return state, nil
	}

	// Check for draw
	if state.Shared.isFull() {
		state.Status = Draw
		return state, nil
	}

	// Switch to next player
	state.CurrentPlayer = state.CurrentPlayer.Opponent()

	return state, nil
}

// TicTacToeAi implements GameAi for TicTacToe.
// Simple strategy: play into a line with only own tokens, or block a line with 2 opponent tokens, otherwise random.
type TicTacToeAi struct{}

// NewTicTacToeAi creates a new TicTacToe AI for the given player.
func NewTicTacToeAi() *TicTacToeAi {
	return &TicTacToeAi{}
}

// GetMove returns the AI's chosen move for the current board state.
func (ai *TicTacToeAi) GetMove(player Player, board TicTacToeBoard) (Position, error) {
	opponent := player.Opponent()

	for _, line := range lines {
		myCount := board.countOccupiedBy(line, player)
		oppCount := board.countOccupiedBy(line, opponent)

		emptyPos := board.firstEmpty(line)
		if emptyPos != nil {
			// Line contains only my tokens - continue building
			if myCount > 0 && oppCount == 0 {
				return *emptyPos, nil
			}

			// Line has 2 opponent tokens and one empty - block
			if oppCount == 2 && myCount == 0 {
				return *emptyPos, nil
			}
		}
	}

	// Play randomly
	empties := board.empties()
	if len(empties) == 0 {
		return Position{}, fmt.Errorf("no valid moves available")
	}
	return empties[rand.Intn(len(empties))], nil
}
