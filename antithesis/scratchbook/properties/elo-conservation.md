# elo-conservation

## Evidence

ELO is defined as a zero-sum rating system: any rating points gained by one player should be lost by the other. The formula in `internal/elo.go:18-30` computes both new ratings from the same `expected` value.

For wins: `newWinner = round(old + K*(1-expected))`, `newLoser = round(old + K*(0 - (1-expected)))` = `round(old - K*(1-expected))`.
For draws: `newWinner = round(old + K*(0.5-expected))`, `newLoser = round(old + K*(0.5 - (1-expected)))`.

In exact arithmetic, `deltaWinner + deltaLoser = K*((score-expected) + ((1-score)-(1-expected))) = K*(score-expected+1-score-1+expected) = 0`.

However, each value is independently rounded to int, so `deltaWinner + deltaLoser` could be -1, 0, or +1 due to rounding.

## Code Paths

- `CalcElo` returns `(newWinner, newLoser)` — `elo.go:28-29`
- `ReportSessionResult` calls CalcElo once per completed session — `db.go:319`

## What Goes Wrong on Violation

Systematic non-conservation (always rounding up for both, or always rounding down) would inflate or deflate total ELO in the system over many games, causing leaderboard drift.

## Instrumentation Status

**NOT COVERED** — An `Always(abs((newWinner-winnerElo) + (newLoser-loserElo)) <= 1)` in `ReportSessionResult` would verify this.
