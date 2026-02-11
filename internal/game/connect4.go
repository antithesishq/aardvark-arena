package game

import (
	"fmt"
	"math/rand"

	"github.com/antithesishq/aardvark-arena/internal"
)

var connect4Bounds = Bounds{Width: 7, Height: 6}

// Connect4Board is the shared state for a Connect4 game.
// Each cell is nil (empty) or points to the player who occupies it.
type Connect4Board struct {
	Cells [7][6]*Player // [column][row], row 0 is bottom
}

// NewConnect4Board creates an empty Connect4 board.
func NewConnect4Board() Connect4Board {
	return Connect4Board{}
}

// lowestEmpty returns the lowest empty row in the given column, or -1 if full.
func (b Connect4Board) lowestEmpty(col int) int {
	for row := 0; row < connect4Bounds.Height; row++ {
		if b.Cells[col][row] == nil {
			return row
		}
	}
	return -1
}

// isFull returns true if the board has no empty cells.
func (b Connect4Board) isFull() bool {
	for col := 0; col < connect4Bounds.Width; col++ {
		if b.lowestEmpty(col) >= 0 {
			return false
		}
	}
	return true
}

// countDirection counts consecutive pieces belonging to player starting from (col, row)
// and moving in direction (dc, dr), not including the starting cell.
func (b Connect4Board) countDirection(col, row, dc, dr int, player Player) int {
	count := 0
	for i := 1; i < 4; i++ {
		c, r := col+dc*i, row+dr*i
		if c < 0 || c >= connect4Bounds.Width || r < 0 || r >= connect4Bounds.Height {
			break
		}
		if b.Cells[c][r] == nil || *b.Cells[c][r] != player {
			break
		}
		count++
	}
	return count
}

// checkWinAt checks if placing a piece at (col, row) creates a 4-in-a-row.
func (b Connect4Board) checkWinAt(col, row int, player Player) bool {
	// Check horizontal, vertical, and both diagonals
	directions := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range directions {
		count := 1 + b.countDirection(col, row, d[0], d[1], player) + b.countDirection(col, row, -d[0], -d[1], player)
		if count >= 4 {
			return true
		}
	}
	return false
}

// validColumns returns all columns that have at least one empty cell.
func (b Connect4Board) validColumns() []int {
	var cols []int
	for col := 0; col < connect4Bounds.Width; col++ {
		if b.lowestEmpty(col) >= 0 {
			cols = append(cols, col)
		}
	}
	return cols
}

// Connect4Session implements Session for Connect4.
type Connect4Session struct{}

// MakeMove processes a move (column selection) for the given player.
func (s *Connect4Session) MakeMove(state State[Connect4Board], player Player, col int) (State[Connect4Board], error) {
	if err := state.CanMakeMove(player); err != nil {
		return state, err
	}
	if col < 0 || col >= connect4Bounds.Width {
		return state, fmt.Errorf("column out of bounds")
	}

	row := state.Shared.lowestEmpty(col)
	if row < 0 {
		return state, fmt.Errorf("column is full")
	}

	// Place the piece
	state.Shared.Cells[col][row] = &player

	// Check for win
	if state.Shared.checkWinAt(col, row, player) {
		state.Status = player.Wins()
		return state, nil
	}

	// Check for draw
	if state.Shared.isFull() {
		state.Status = Draw
		return state, nil
	}

	state.CurrentPlayer = state.CurrentPlayer.Opponent()
	return state, nil
}

// Connect4Ai implements Ai for Connect4.
type Connect4Ai struct {
	rng *rand.Rand
}

// NewConnect4Ai creates a new Connect4 AI for the given player.
func NewConnect4Ai() *Connect4Ai {
	return &Connect4Ai{rng: internal.NewRand()}
}

// GetMove returns the AI's chosen column.
// Strategy: win if possible, block opponent win, otherwise prefer center columns.
func (ai *Connect4Ai) GetMove(player Player, board Connect4Board) (int, error) {
	validCols := board.validColumns()
	if len(validCols) == 0 {
		return 0, fmt.Errorf("no valid moves available")
	}

	// Check for winning move
	for _, col := range validCols {
		row := board.lowestEmpty(col)
		if board.checkWinAt(col, row, player) {
			return col, nil
		}
	}

	// Check for blocking move
	opponent := player.Opponent()
	for _, col := range validCols {
		row := board.lowestEmpty(col)
		if board.checkWinAt(col, row, opponent) {
			return col, nil
		}
	}

	// Prefer center columns (3, 2, 4, 1, 5, 0, 6)
	preferred := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range preferred {
		if board.lowestEmpty(col) >= 0 {
			return col, nil
		}
	}

	// Fallback to random
	return validCols[ai.rng.Intn(len(validCols))], nil
}
