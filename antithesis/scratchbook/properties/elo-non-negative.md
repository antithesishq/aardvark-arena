# elo-non-negative

## Evidence

The ELO calculation in `internal/elo.go:18-30` uses the standard formula:
```
expected = 1 / (1 + 10^((loserElo - winnerElo) / 400))
newWinner = round(winnerElo + K * (score - expected))
newLoser = round(loserElo + K * ((1-score) - (1-expected)))
```

With `K=32` and `DefaultElo=1000`, a player starting at 1000 would need to lose many consecutive games against much lower-rated opponents for their ELO to approach 0. However, the formula has no floor, so negative values are mathematically possible with extreme sequences.

## Code Paths

- `CalcElo` in `internal/elo.go` — pure function, no bounds checking
- `ReportSessionResult` in `internal/matchmaker/db.go:316-339` — calls `CalcElo` and writes the result to the DB via `updatePlayerStats`
- The DB schema has no CHECK constraint on the `elo` column

## What Goes Wrong on Violation

Negative ELO would cause incorrect matchmaking (the `MatchElo` function uses absolute ELO difference), display issues on the leaderboard, and potentially signed/unsigned conversion bugs if ELO is ever used as an unsigned value.

## Instrumentation Status

**NOT COVERED** — No assertion exists. A SUT-side `Always(newWinner >= 0 && newLoser >= 0)` in `ReportSessionResult` after `CalcElo` would catch this.
