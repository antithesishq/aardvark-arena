// Package game provides shared game types and logic.
package game

import (
	"fmt"
	"slices"
)

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

// In returns true if the status is one of the given statuses.
func (s Status) In(statuses ...Status) bool {
	return slices.Contains(statuses, s)
}

// IsTerminal returns true if the game has ended.
func (s Status) IsTerminal() bool {
	return s != Active
}

func (s Status) String() string {
	switch s {
	case Active:
		return "Active"
	case P1Win:
		return "P1Win"
	case P2Win:
		return "P2Win"
	case Draw:
		return "Draw"
	case Cancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

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

// AllGames lists every supported game kind.
var AllGames = [...]Kind{Battleship, TicTacToe, Connect4}

// MarshalText implements encoding.TextMarshaler.
func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (k *Kind) UnmarshalText(raw []byte) error {
	unverified := Kind(raw)
	switch unverified {
	case Battleship, TicTacToe, Connect4:
		*k = unverified
		return nil
	default:
		return fmt.Errorf("invalid game kind: %q", raw)
	}
}

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

// MarshalText implements encoding.TextMarshaler so Position can be used as
// a JSON map key (e.g. map[Position]AttackResult in Battleship).
func (p Position) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%d,%d", p.X, p.Y), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (p *Position) UnmarshalText(text []byte) error {
	n, err := fmt.Sscanf(string(text), "%d,%d", &p.X, &p.Y)
	if n != 2 {
		return fmt.Errorf("invalid position format")
	}
	return err
}

// InBounds returns true if the position is within the given bounds.
func (p Position) InBounds(b Bounds) bool {
	return p.X >= 0 && p.X < b.Width && p.Y >= 0 && p.Y < b.Height
}

// State holds the current state of a game session.
type State[Shared any] struct {
	// CurrentPlayer is the player whose turn it is.
	CurrentPlayer Player
	// Status is the current game status.
	Status Status
	// Shared is the game-specific shared state.
	Shared Shared
}

// NewState initializes the Game state object.
func NewState[Shared any](shared Shared) State[Shared] {
	return State[Shared]{
		CurrentPlayer: P1,
		Status:        Active,
		Shared:        shared,
	}
}

// CanMakeMove validates that the specified player is allowed to make a move.
// Returns an error if not.
func (s State[Shared]) CanMakeMove(player Player) error {
	if s.Status != Active {
		return fmt.Errorf("game is over")
	}
	if player != s.CurrentPlayer {
		return fmt.Errorf("not your turn")
	}
	return nil
}

// Session defines the interface for a game session that can process moves.
type Session[Move any, Shared any] interface {
	MakeMove(State[Shared], Player, Move) (State[Shared], error)
}

// Ai defines the interface for an AI that can generate moves.
type Ai[Move any, Shared any] interface {
	GetMove(Player, Shared) (Move, error)
}

// PlayerMap is a generic map-like structure for storing values per player.
type PlayerMap[V any] struct {
	P1 V
	P2 V
}

// Get retrieves the value for the specified player.
func (pm *PlayerMap[V]) Get(player Player) V {
	if player == P1 {
		return pm.P1
	}
	return pm.P2
}

// Set assigns the value for the specified player.
func (pm *PlayerMap[V]) Set(player Player, value V) {
	if player == P1 {
		pm.P1 = value
	} else {
		pm.P2 = value
	}
}
