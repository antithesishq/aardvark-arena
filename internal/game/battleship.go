package game

import (
	"fmt"
	"math/rand"
	"slices"
)

// Battleship board dimensions.
var battleshipBounds = Bounds{Width: 10, Height: 10}

// ShipType identifies different ships in Battleship.
type ShipType int

// Ship types for Battleship.
const (
	Carrier        ShipType = iota
	BattleshipShip          // Avoid collision with game.Battleship Kind
	Cruiser
	Submarine
	Destroyer
)

// shipSizes maps ship types to their lengths.
var shipSizes = map[ShipType]int{
	Carrier:        5,
	BattleshipShip: 4,
	Cruiser:        3,
	Submarine:      3,
	Destroyer:      2,
}

// totalShipCells is the sum of all ship sizes.
const totalShipCells = 5 + 4 + 3 + 3 + 2 // 17

// allShipTypes is the ordered list of ships to place.
var allShipTypes = []ShipType{Carrier, BattleshipShip, Cruiser, Submarine, Destroyer}

// Orientation for ship placement.
type Orientation int

// Orientation values for ship placement.
const (
	Horizontal Orientation = iota
	Vertical
)

// ShipPlacement describes a single ship's position and orientation.
type ShipPlacement struct {
	Ship        ShipType
	Position    Position
	Orientation Orientation
}

// positions returns all positions this ship occupies.
func (p ShipPlacement) positions() []Position {
	size := shipSizes[p.Ship]
	result := make([]Position, size)
	for i := range size {
		if p.Orientation == Horizontal {
			result[i] = Position{X: p.Position.X + i, Y: p.Position.Y}
		} else {
			result[i] = Position{X: p.Position.X, Y: p.Position.Y + i}
		}
	}
	return result
}

// BattleshipMoveKind discriminates setup vs attack moves.
type BattleshipMoveKind int

// Move kinds for Battleship.
const (
	SetupMoveKind BattleshipMoveKind = iota
	AttackMoveKind
)

// BattleshipMove is a tagged union for Battleship moves.
type BattleshipMove struct {
	Kind       BattleshipMoveKind
	Placements []ShipPlacement // For SetupMoveKind
	Target     Position        // For AttackMoveKind
}

// AttackResult represents the outcome of an attack at a position.
type AttackResult int

// Attack results.
const (
	Miss AttackResult = iota
	Hit
)

// BattleshipSharedState is the game state visible to both players.
type BattleshipSharedState struct {
	// Attacks contains attacks made on each player.
	// Maps position to result (Hit or Miss).
	Attacks PlayerMap[map[Position]AttackResult]
}

// NewBattleshipSharedState creates initial shared state.
func NewBattleshipSharedState() BattleshipSharedState {
	return BattleshipSharedState{
		Attacks: PlayerMap[map[Position]AttackResult]{
			P1: make(map[Position]AttackResult),
			P2: make(map[Position]AttackResult),
		},
	}
}

// hitCount returns how many hits a player has landed (on their opponent).
func (s BattleshipSharedState) hitCount(player Player) int {
	count := 0
	for _, result := range s.Attacks.Get(player.Opponent()) {
		if result == Hit {
			count++
		}
	}
	return count
}

// BattleshipSession keeps track of a Battleship game.
type BattleshipSession struct {
	// shipCells stores the set of cells occupied by each player's ships.
	shipCells PlayerMap[[]Position]
}

// NewBattleshipSession creates a new session.
func NewBattleshipSession() *BattleshipSession {
	return &BattleshipSession{}
}

// MakeMove processes a move for the given player.
func (s *BattleshipSession) MakeMove(state State[BattleshipSharedState], player Player, move BattleshipMove) (State[BattleshipSharedState], error) {
	if err := state.CanMakeMove(player); err != nil {
		return state, err
	}

	if s.inSetup() {
		return s.handleSetup(state, player, move)
	}
	return s.handleAttack(state, player, move)
}

// inSetup returns true if the game is still in the setup phase.
func (s *BattleshipSession) inSetup() bool {
	return s.shipCells.P1 == nil || s.shipCells.P2 == nil
}

func (s *BattleshipSession) handleSetup(state State[BattleshipSharedState], player Player, move BattleshipMove) (State[BattleshipSharedState], error) {
	if move.Kind != SetupMoveKind {
		return state, fmt.Errorf("must submit setup move during setup phase")
	}

	if s.shipCells.Get(player) != nil {
		return state, fmt.Errorf("already completed setup")
	}

	// Validate placements
	providedShips := make(map[ShipType]bool)
	occupied := make([]Position, 0, totalShipCells)
	for _, placement := range move.Placements {
		if providedShips[placement.Ship] {
			return state, fmt.Errorf("duplicate ship placement")
		}
		providedShips[placement.Ship] = true
		for _, pos := range placement.positions() {
			if !pos.InBounds(battleshipBounds) {
				return state, fmt.Errorf("ship placement out of bounds")
			}
			if slices.Contains(occupied, pos) {
				return state, fmt.Errorf("ships overlap")
			}
			occupied = append(occupied, pos)
		}
	}

	// Validate all required ships are provided exactly once
	for _, ship := range allShipTypes {
		if !providedShips[ship] {
			return state, fmt.Errorf("missing required ship")
		}
	}

	// Store placements privately in session
	s.shipCells.Set(player, occupied)

	state.CurrentPlayer = state.CurrentPlayer.Opponent()
	return state, nil
}

func (s *BattleshipSession) handleAttack(state State[BattleshipSharedState], player Player, move BattleshipMove) (State[BattleshipSharedState], error) {
	if move.Kind != AttackMoveKind {
		return state, fmt.Errorf("must submit attack move after setup phase")
	}
	if !move.Target.InBounds(battleshipBounds) {
		return state, fmt.Errorf("target out of bounds")
	}

	// Check if hit or miss using private ship data
	opponent := player.Opponent()
	if slices.Contains(s.shipCells.Get(opponent), move.Target) {
		state.Shared.Attacks.Get(opponent)[move.Target] = Hit
	} else {
		state.Shared.Attacks.Get(opponent)[move.Target] = Miss
		// Switch turns on miss
		state.CurrentPlayer = state.CurrentPlayer.Opponent()
	}

	// Check for win (all opponent ship cells hit)
	if state.Shared.hitCount(player) == totalShipCells {
		state.Status = player.Wins()
		return state, nil
	}

	return state, nil
}

// BattleshipAi implements Ai for Battleship.
type BattleshipAi struct {
	setup bool
}

// NewBattleshipAi creates a new Battleship AI for the given player.
func NewBattleshipAi() *BattleshipAi {
	return &BattleshipAi{}
}

// GetMove returns the AI's chosen move for the current state.
func (ai *BattleshipAi) GetMove(player Player, state BattleshipSharedState) (BattleshipMove, error) {
	if !ai.setup {
		ai.setup = true
		return ai.getSetupMove()
	}
	return ai.getAttackMove(player, state)
}

func (ai *BattleshipAi) getSetupMove() (BattleshipMove, error) {
	placements := make([]ShipPlacement, 0, len(allShipTypes))
	occupied := make(map[Position]bool)

	for _, ship := range allShipTypes {
		placed := false
		for attempts := 0; attempts < 1000 && !placed; attempts++ {
			orientation := Orientation(rand.Intn(2))
			var maxX, maxY int
			if orientation == Horizontal {
				maxX = battleshipBounds.Width - shipSizes[ship]
				maxY = battleshipBounds.Height - 1
			} else {
				maxX = battleshipBounds.Width - 1
				maxY = battleshipBounds.Height - shipSizes[ship]
			}

			if maxX < 0 || maxY < 0 {
				continue
			}

			pos := Position{X: rand.Intn(maxX + 1), Y: rand.Intn(maxY + 1)}
			placement := ShipPlacement{Ship: ship, Position: pos, Orientation: orientation}

			// Check no overlap with already placed ships
			valid := true
			positions := placement.positions()
			for _, p := range positions {
				if occupied[p] {
					valid = false
					break
				}
			}

			if valid {
				for _, p := range positions {
					occupied[p] = true
				}
				placements = append(placements, placement)
				placed = true
			}
		}

		if !placed {
			return BattleshipMove{}, fmt.Errorf("failed to place ships")
		}
	}

	return BattleshipMove{Kind: SetupMoveKind, Placements: placements}, nil
}

func (ai *BattleshipAi) getAttackMove(player Player, state BattleshipSharedState) (BattleshipMove, error) {
	// Attacks on the opponent represent our previous attacks
	opponentBoard := state.Attacks.Get(player.Opponent())

	// First, look for untargeted cells adjacent to previous hits
	adjacentOffsets := []Position{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	var adjacentTargets []Position
	for pos, result := range opponentBoard {
		if result != Hit {
			continue
		}
		for _, offset := range adjacentOffsets {
			adj := Position{X: pos.X + offset.X, Y: pos.Y + offset.Y}
			if !adj.InBounds(battleshipBounds) {
				continue
			}
			if _, exists := opponentBoard[adj]; exists {
				continue
			}
			adjacentTargets = append(adjacentTargets, adj)
		}
	}

	if len(adjacentTargets) > 0 {
		target := adjacentTargets[rand.Intn(len(adjacentTargets))]
		return BattleshipMove{Kind: AttackMoveKind, Target: target}, nil
	}

	// Fall back to random targeting
	var targets []Position
	for x := range battleshipBounds.Width {
		for y := range battleshipBounds.Height {
			pos := Position{X: x, Y: y}
			if _, exists := opponentBoard[pos]; !exists {
				targets = append(targets, pos)
			}
		}
	}

	if len(targets) == 0 {
		return BattleshipMove{}, fmt.Errorf("no valid targets")
	}

	target := targets[rand.Intn(len(targets))]
	return BattleshipMove{Kind: AttackMoveKind, Target: target}, nil
}
