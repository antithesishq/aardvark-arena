# elo-zero-sum — ELO Changes Are Zero-Sum

## Evidence

`CalcElo` (`internal/elo.go:18-29`) computes new ratings using the standard ELO formula. The test (`internal/elo_test.go:63-66`) explicitly verifies `(newWinner - winner) + (newLoser - loser) == 0` for all test cases.

`ReportSessionResult` (`internal/matchmaker/db.go:262-323`) calls `CalcElo` and applies the results in a single transaction. For non-cancelled games, it finds the two players, computes new ELOs, and updates both players in the same transaction.

## Failure Scenario

The zero-sum property could be violated if:
1. The transaction commits after updating one player but not the other (SQLite transactions should prevent this).
2. Two concurrent calls to `ReportSessionResult` for the same session both read the same initial ELO values (SQLite serializes writes, but the read-modify-write pattern could still have issues if the transaction isolation isn't strict enough).
3. The `CalcElo` function itself has a rounding error that breaks zero-sum (the test covers this for specific inputs, but not exhaustively).

The most Antithesis-relevant scenario is #2: concurrent result delivery for the same session from different paths (normal completion vs. cancellation by session monitor).

## Relevant Code Paths
- `internal/elo.go:18-29` — `CalcElo`
- `internal/matchmaker/db.go:262-323` — `ReportSessionResult`
- `internal/matchmaker/db.go:50-57` — `updatePlayerStats` SQL

## SUT Instrumentation
- **Missing**: `Always` assertion in `ReportSessionResult` after computing both new ELOs, verifying the delta sums to zero before writing to DB.
