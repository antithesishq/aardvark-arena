package game

import "fmt"

// StateViolationError indicates an action was attempted in an invalid state.
// Examples: not your turn, game already over, setup not received.
type StateViolationError struct {
	// Message describes the state violation.
	Message string
}

// Error implements the error interface.
func (e StateViolationError) Error() string {
	return fmt.Sprintf("state violation: %s", e.Message)
}

// IllegalMoveError indicates a move that violates game rules.
// Examples: out of bounds, cell occupied, invalid piece movement.
type IllegalMoveError struct {
	// Message describes the illegal move.
	Message string
}

// Error implements the error interface.
func (e IllegalMoveError) Error() string {
	return fmt.Sprintf("illegal move: %s", e.Message)
}
