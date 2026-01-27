// Package game provides shared game types and logic.
package game

// Player represents which player is acting.
type Player int

const (
	// P1 represents the first player.
	P1 Player = iota
	// P2 represents the second player.
	P2
)

// Opponent returns the other player.
func (p Player) Opponent() Player {
	if p == P1 {
		return P2
	}
	return P1
}

// Wins returns either P1Win or P2Win depending on the player.
func (p Player) Wins() Status {
	if p == P1 {
		return P1Win
	}
	return P2Win
}

// Status represents the current state of a game.
type Status int

const (
	// Active indicates the game is still in progress.
	Active Status = iota
	// P1Win indicates player 1 has won.
	P1Win
	// P2Win indicates player 2 has won.
	P2Win
	// Draw indicates the game ended in a draw.
	Draw
	// Cancelled indicates the game was cancelled before completion.
	Cancelled
)

// Kind identifies the type of game.
type Kind string

const (
	// Battleship is the game kind for Battleship.
	Battleship Kind = "battleship"
	// TicTacToe is the game kind for Tic-Tac-Toe.
	TicTacToe Kind = "tictactoe"
	// Connect4 is the game kind for Connect Four.
	Connect4 Kind = "connect4"
)

// Bounds represents the dimensions of a 2D board.
type Bounds struct {
	// Width is the number of columns.
	Width int
	// Height is the number of rows.
	Height int
}

// Position represents a position on a 2D board.
type Position struct {
	// X is the column index.
	X int
	// Y is the row index.
	Y int
}

// InBounds returns true if the position is within the given bounds.
func (p Position) InBounds(b Bounds) bool {
	return p.X >= 0 && p.X < b.Width && p.Y >= 0 && p.Y < b.Height
}

// State holds the current state of a game session.
type State[Shared any] struct {
	// NextPlayer is the player whose turn it is.
	NextPlayer Player
	// Status is the current game status.
	Status Status
	// Shared is the game-specific shared state.
	Shared Shared
}

// CanMakeMove validates that the specified player is allowed to make a move.
// Returns an error if not.
func (s State[Shared]) CanMakeMove(player Player) error {
	if s.Status != Active {
		return StateViolationError{"game is over"}
	}
	if player != s.NextPlayer {
		return StateViolationError{"not your turn"}
	}
	return nil
}

// Session defines the interface for a game session that can process moves.
type Session[Move any, Shared any] interface {
	MakeMove(Player, Move) (State[Shared], error)
}

// Ai defines the interface for an AI that can generate moves.
type Ai[Move any, Shared any] interface {
	GetMove(Shared) (Move, error)
}
