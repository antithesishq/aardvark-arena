# game-rules-enforced — Game Rules Never Violated

## Evidence

Each game implements the `Session[Move, Shared]` interface with a `MakeMove` method that validates moves before mutating state:

- **TicTacToe** (`internal/game/tictactoe.go:96-126`): Checks bounds, cell occupancy, turn order via `CanMakeMove`.
- **Connect4** (`internal/game/connect4.go:88-118`): Checks column bounds, column fullness, turn order.
- **Battleship** (`internal/game/battleship.go:136-219`): Phase-aware validation — setup validates all ship placements (no duplicates, in-bounds, no overlap), attack validates bounds and phase.

The Protocol (`internal/gameserver/protocol.go:206-227`) deserializes the raw JSON into the typed move, then calls `MakeMove`. If `MakeMove` returns an error, the move is rejected and the state is not modified.

## Failure Scenario

A bug in move validation could allow an invalid move to mutate state. This is most likely in the Battleship setup phase, which has the most complex validation (5 ships, overlap checking, bounds across two orientations). An evil player sending carefully crafted placements could potentially exploit an edge case.

## Relevant Code Paths
- `internal/game/game.go:167-175` — `CanMakeMove` (turn order + game-over check)
- `internal/gameserver/protocol.go:206-227` — `handleMove` (deserialize + call MakeMove)
- `internal/player/behavior.go` — Evil behavior rates and corruption methods

## SUT Instrumentation
- **Missing**: `Always` assertion after each successful `MakeMove` that validates the resulting board is structurally valid (e.g., TTT piece counts balanced, Connect4 gravity respected, Battleship ship counts correct).
- **Missing**: `Always` assertion that rejected moves don't modify state (verify state before == state after on error).
