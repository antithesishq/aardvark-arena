# all-game-types-played

## Evidence

Player game selection happens in `chooseGamePreference` (`internal/player/loop.go:125-144`). With `SpecificGameSelectionRate = 0.20`, ~20% of queue requests specify a game; the rest accept any game. When no preference is specified, `selectMatchGame` (`match_queue.go:138-156`) picks randomly from `game.AllGames` (Battleship, TicTacToe, Connect4).

Three `Reachable` assertions (R21, R22, R23) confirm each game type is played.

## Instrumentation Status

**FULLY COVERED** — R21, R22, R23 cover all three game types.
