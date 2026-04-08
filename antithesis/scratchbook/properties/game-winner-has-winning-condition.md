# game-winner-has-winning-condition

## Evidence

Each game implementation has win-detection logic:

1. **TicTacToe** (`internal/game/tictactoe.go:72-79`): `checkWinFor(player)` scans all 8 lines for 3-in-a-row.
2. **Connect4** (`internal/game/connect4.go:61-71`): `checkWinAt(col, row, player)` checks 4 directions from the last placed piece.
3. **Battleship** (`internal/game/battleship.go:113-122`): `hitCount(player) == totalShipCells` (all 17 opponent ship cells hit).

When any `MakeMove` returns a terminal state (P1Win/P2Win), the protocol's `report` function sends the result. But there's no independent verification that the board actually shows a valid winning condition at the time the result is reported.

## Code Paths

- `TicTacToeSession.MakeMove` (`tictactoe.go:110-114`): Checks `checkWinFor(player)` after placing piece
- `Connect4Session.MakeMove` (`connect4.go:103-107`): Checks `checkWinAt` after placing piece
- `BattleshipSession.handleAttack` (`battleship.go:212-215`): Checks hitCount after marking hit
- `protocol.report` (`protocol.go:96-112`): Sends result based on `state.Status`

## What Goes Wrong on Violation

If `MakeMove` incorrectly declares a win, the game ends prematurely. The losing player gets an unearned loss. In Battleship, this could happen if the hit counter is wrong (off-by-one in `totalShipCells` constant, or a hit is counted twice).

## Instrumentation Status

**NOT COVERED** — This is a gap-fill property from the Coverage Balance evaluation. An `Always` in `protocol.report` that re-validates the winning condition on the board when `status` is P1Win or P2Win would close this gap. The assertion would need to be game-type-specific.
