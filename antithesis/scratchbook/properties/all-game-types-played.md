# all-game-types-played — All Three Game Types Are Exercised

## Evidence

Game selection happens in two places:
1. **Player preference** (`internal/player/loop.go:109-120`): 20% chance of selecting a specific game. Games chosen uniformly from `game.AllGames` (Battleship, TicTacToe, Connect4).
2. **Match queue** (`internal/matchmaker/match_queue.go:134-152`): If both players have no preference, a random game is selected from `game.AllGames`.

With 80% "any game" rate and 20% specific-game rate spread across 3 games, all three types should be exercised.

## Relevant Code Paths
- `internal/player/loop.go:109-120` — `chooseGamePreference`
- `internal/matchmaker/match_queue.go:134-152` — `selectMatchGame`
- `internal/game/game.go:91` — `AllGames` array

## SUT Instrumentation
- **Missing**: Three `Reachable` assertions — one per game type — in `SessionManager.CreateSession` or `sessionHandle.RunToCompletion`.
