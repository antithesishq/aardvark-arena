# leaderboard-reflects-games — Global ELO Conservation

## Evidence

`CalcElo` (`internal/elo.go:18-29`) is zero-sum by construction — the test at `elo_test.go:63-66` verifies this. Every non-cancelled `ReportSessionResult` call applies `CalcElo` to two players in a transaction. Cancelled sessions skip ELO updates entirely.

Therefore, starting from `DefaultElo = 1000` for every player, the global sum should always equal `count(players) * 1000`. Any deviation indicates:
- Double ELO update (same session processed twice)
- Missing ELO update (result lost)
- Rounding error in CalcElo (theoretically possible with independent `math.Round` calls)

The leaderboard endpoint (`GET /leaderboard`, `internal/matchmaker/db.go:347-352`) returns the top 20 players ordered by ELO. For a full global check, a query like `SELECT SUM(elo), COUNT(*) FROM players` would be needed.

## Failure Scenario

The `no-double-elo-update` evidence documents that `updateSessionResult` has no idempotency guard. If a session's result is submitted twice, one player gains ELO twice and the other loses ELO twice. Per-transaction zero-sum holds for each call, but the cumulative effect is non-zero-sum for that session: the net delta is `2 * (winner_gain - loser_loss)`, which is non-zero unless the gains are perfectly symmetric (they're not due to the ELO formula's asymmetry for unequal ratings).

## Relevant Code Paths
- `internal/elo.go:18-29` — `CalcElo`
- `internal/matchmaker/db.go:262-323` — `ReportSessionResult`
- `internal/matchmaker/db.go:347-352` — `Leaderboard` query

## SUT Instrumentation
- **Missing**: `Always` assertion in a workload test command that queries `SELECT SUM(elo), COUNT(*) FROM players` and verifies `sum == count * 1000`.
- Alternative: Add a `/debug/elo-sum` endpoint to the matchmaker for workload assertion access.
