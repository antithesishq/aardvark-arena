# draw-outcome-reached — Draw Outcomes Are Reachable

## Evidence

- **TicTacToe**: Draw when board is full with no winner. `isFull()` check after win check (`tictactoe.go:118-120`). AI strategy favors winning/blocking, but draws are common in optimal play.
- **Connect4**: Draw when board is full with no 4-in-a-row. `isFull()` check after win check (`connect4.go:111-114`). Draws are rarer than in TTT due to board size.
- **Battleship**: No draw possible — game ends when all opponent ships are sunk.

## Relevant Code Paths
- `internal/game/tictactoe.go:118-120` — TTT draw check
- `internal/game/connect4.go:111-114` — Connect4 draw check
- `internal/gameserver/protocol.go:95-111` — `report()` with Draw status

## SUT Instrumentation
- **Missing**: `Reachable` assertion when `state.Status == Draw` in Protocol.report().
