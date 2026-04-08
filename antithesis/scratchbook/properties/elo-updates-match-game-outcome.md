# elo-updates-match-game-outcome

## Evidence

`CalcElo` in `internal/elo.go:18-30` uses score=1.0 for wins and score=0.5 for draws. The expected value is based on the rating difference. For a win:
- Winner: `newWinner = winnerElo + K*(1 - expected)`, where `expected < 1` always, so `K*(1-expected) > 0`, meaning ELO increases.
- Loser: `newLoser = loserElo + K*(0 - (1-expected))` = `loserElo - K*(1-expected)`, so ELO decreases.

For a draw with equal ratings: both change by `K*(0.5 - 0.5) = 0`.
For a draw with unequal ratings: the higher-rated player loses points, lower gains.

## Code Paths

- `CalcElo` — `elo.go:18-30` — computes new ratings
- `ReportSessionResult` — `db.go:316-339` — applies to both players

## Subtlety

The `CalcElo` function is called with `(player.Elo, opponent.Elo, draw)` where `player` is the winner (or either player for draws). The function names its parameters `winnerElo, loserElo` but for draws, the "winner" designation is arbitrary. For draws, `score=0.5` and the function is symmetric up to rounding.

The loop at `db.go:316-339` iterates players and finds the winner. For a draw (`winner == uuid.Nil`), `player.PlayerID == winner` is false for all players, so the first player is treated as the "winner" in the `CalcElo` call. This is correct because the draw case is symmetric.

## Instrumentation Status

**NOT COVERED** — An `Always(newWinnerElo >= winnerElo)` for wins and symmetry check for draws would verify directional correctness.
