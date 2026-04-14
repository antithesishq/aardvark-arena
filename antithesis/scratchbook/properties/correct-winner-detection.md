# correct-winner-detection — Winner Matches Board State

## Evidence

Win detection is implemented per-game:
- **TicTacToe** (`internal/game/tictactoe.go:72-79`): `checkWinFor` iterates all 8 winning lines, checking if all 3 cells belong to the player.
- **Connect4** (`internal/game/connect4.go:61-71`): `checkWinAt` checks 4 directions from the placed piece, counting consecutive same-player pieces.
- **Battleship** (`internal/game/battleship.go:213-216`): Win when `hitCount(player) == totalShipCells` (17).

The Protocol (`internal/gameserver/protocol.go:95-111`) maps the game status (P1Win/P2Win) to the actual player UUID and sends the result to the Reporter.

## Failure Scenario

The `playerToID` function (`protocol.go:86-93`) iterates `p.players` map to find the UUID matching the game.Player enum. If the map is corrupted (e.g., a reconnection race replaces a player entry), the wrong UUID could be reported as the winner.

## Relevant Code Paths
- `internal/gameserver/protocol.go:86-93` — `playerToID` mapping
- `internal/gameserver/protocol.go:95-111` — `report` sends result
- `internal/gameserver/reporter.go:50-81` — `submitResult` to matchmaker

## SUT Instrumentation
- **Missing**: `Always` assertion in `Protocol.report()` that when status is P1Win/P2Win, the board actually contains a winning configuration for the mapped player.
