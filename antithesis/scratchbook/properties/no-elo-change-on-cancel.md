# no-elo-change-on-cancel — Cancelled Games Don't Affect ELO

## Evidence

`ReportSessionResult` (`internal/matchmaker/db.go:262-323`) has an explicit guard:
```go
if cancelled && winner != uuid.Nil {
    log.Panicf("BUG: received cancelled result with a winner set")
}
```

When `cancelled == true`, the ELO update block is skipped entirely (`if !cancelled { ... }`). The `updateSessionResult` SQL updates `cancelled` and `completed_at` but doesn't touch player stats directly.

The test `TestReportSessionResult/cancelled` (`internal/matchmaker/db_test.go:163-169`) verifies that cancelled sessions leave ELO unchanged.

## Failure Scenario

The race: game server reports a normal completion (winner = P1) via Reporter, while the matchmaker's session monitor simultaneously cancels the same session (deadline passed). Both call `ReportSessionResult`:
1. Reporter's call: `cancelled=false, winner=P1` → ELO updated.
2. Monitor's call: `cancelled=true, winner=Nil` → ELO not updated, but `completed_at` overwritten.

The second call overwrites the session's `completed_at` and sets `cancelled=true`, but the ELO change from the first call persists. The session now appears cancelled but has ELO changes — an inconsistency.

## Relevant Code Paths
- `internal/matchmaker/db.go:262-323` — `ReportSessionResult`
- `internal/matchmaker/db.go:184-200` — `cancelExpiredSessions`
- `internal/matchmaker/server.go:134-153` — `handleResult`
- `internal/matchmaker/server.go:173-211` — `handleCancelSession`

## SUT Instrumentation
- **Missing**: `Always` assertion that when a session's final state in the DB is `cancelled=true`, neither player's ELO, wins, losses, or draws changed from their pre-session values.
- **Missing**: Idempotency guard on `ReportSessionResult` to reject duplicate submissions.
