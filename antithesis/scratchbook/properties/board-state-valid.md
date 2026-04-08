# board-state-valid — Board Configuration Always Legal

## Evidence

Board state is modified only through `MakeMove`, which validates moves. The state is serialized via `json.Marshal(p.state)` and broadcast to all players and spectators.

### Structural Invariants per Game

- **TicTacToe**: `|count(P1) - count(P2)| <= 1` (at most one more piece for the current player's opponent, since they just moved). Board is 3x3.
- **Connect4**: For each column, pieces are contiguous from the bottom (no floating pieces). Board is 7x6.
- **Battleship**: `Attacks` map entries are within 10x10 bounds. Hit count never exceeds `totalShipCells` (17).

## Failure Scenario

State is only mutated in the Protocol's single goroutine, so concurrent mutation isn't a concern. The risk is in serialization: `json.Marshal` is called while the state is a local variable in the goroutine, so there's no concurrent access issue. The more realistic failure path is a bug in `MakeMove` that violates a structural invariant despite passing validation.

## Relevant Code Paths
- `internal/gameserver/protocol.go:229-242` — `BroadcastState` serializes and sends
- `internal/game/tictactoe.go:96-126` — TTT MakeMove
- `internal/game/connect4.go:88-118` — Connect4 MakeMove
- `internal/game/battleship.go:136-219` — Battleship MakeMove

## SUT Instrumentation
- **Missing**: `Always` assertion after each `BroadcastState` that validates structural invariants on the serialized state.
- Alternative: Workload-side assertion in the player/spectator that validates each received state message.
