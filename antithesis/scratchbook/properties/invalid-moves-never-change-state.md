# invalid-moves-never-change-state

## Evidence

Each game's `MakeMove` implementation follows the pattern:
1. Call `CanMakeMove` — returns error if not player's turn or game is over
2. Validate move-specific constraints
3. Only mutate state after all validation passes

Go passes structs by value, so `MakeMove(state, ...)` receives a copy. If it returns an error, the caller's `state` is unchanged because the return value is the (unmutated or partially-mutated) copy, not a pointer.

However, there's a subtlety: `BattleshipSession.MakeMove` modifies `s.shipCells` (a pointer receiver field) during setup. If `handleSetup` partially processes then errors, `s.shipCells` may already be modified. Let me trace: `handleSetup` validates all placements before storing (`battleship.go:163-185`), then stores at line 188. So the validation-before-mutation pattern holds.

## Code Paths

- `TicTacToeSession.MakeMove` — `tictactoe.go:96-126` — validates before placing
- `Connect4Session.MakeMove` — `connect4.go:88-118` — validates before placing
- `BattleshipSession.MakeMove` — `battleship.go:136-145` -> `handleSetup` / `handleAttack`
- `protocol.handleMove` — `protocol.go:215-248` — on error, calls `SendErr`, does not update `p.state`

In the protocol, `handleMove` at line 236: `p.state, err = p.session.MakeMove(p.state, ...)`. If err != nil, `p.state` is assigned the returned value. Since the game implementations return `state` (the unmodified input copy) on validation errors, `p.state` stays unchanged.

## Instrumentation Status

**NOT COVERED** — A workload-side assertion or SUT-side check comparing state before and after an errored MakeMove would verify this. R11 and R12 confirm invalid moves are received. Medium priority since the code logic is sound but subtle.
