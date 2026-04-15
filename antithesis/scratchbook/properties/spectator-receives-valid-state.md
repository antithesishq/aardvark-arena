# spectator-receives-valid-state — Broadcast State Is Structurally Valid

## Evidence

`BroadcastState` (`internal/gameserver/protocol.go:229-242`) serializes the protocol's state and sends it to all connected players and spectators. The state is a goroutine-local variable, so there's no concurrent mutation risk.

This property replaces the workload-side `spectator-state-consistency` with a SUT-side check that doesn't require a spectator workload client. By placing the assertion in `BroadcastState`, every state broadcast is validated regardless of who receives it.

## Structural Invariants per Game

- **TicTacToe**: `|count(P1) - count(P2)| <= 1`. Board is 3x3. No cell contains an invalid value.
- **Connect4**: For each column, pieces are contiguous from the bottom. Board is 7x6.
- **Battleship**: Attack map entries are within 10x10 bounds. Hit count per player <= totalShipCells (17).

## Failure Scenario

A bug in `MakeMove` could produce an invalid board that passes move validation but violates structural invariants. For example, a Connect4 move could somehow place a piece in a column without filling from the bottom (a bug in `lowestEmpty`). The move-level check wouldn't catch this because it validates the move input, not the resulting board structure.

## Relevant Code Paths
- `internal/gameserver/protocol.go:229-242` — `BroadcastState`
- Game-specific state types: `TicTacToeBoard`, `Connect4Board`, `BattleshipSharedState`

## SUT Instrumentation
- **Missing**: `Always` assertion in `BroadcastState` (or a game-specific validation function called from there) that validates the serialized state against structural invariants.
