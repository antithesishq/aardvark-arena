# expired-sessions-cancelled — Deadline Enforcement Works

## Evidence

`StartSessionMonitor` (`internal/matchmaker/db.go:167-180`) runs a ticker that calls `cancelExpiredSessions`. The query `selectExpiredSessions` finds sessions where `deadline < ? AND completed_at IS NULL`.

For each expired session, `ReportSessionResult(sid, true, uuid.Nil)` is called, then `onCancel(sid)` (which is `MatchQueue.Untrack`).

## Failure Scenario

1. **Monitor not running**: If the matchmaker context is cancelled before the monitor starts, expired sessions accumulate forever. The monitor goroutine exits when `ctx.Done()` fires.
2. **DB error on query**: The monitor logs and returns — no retry until next tick.
3. **Cancel races with completion**: If a game completes normally between the `selectExpiredSessions` query and the `ReportSessionResult` call, the cancel overwrites the normal result (see no-double-elo-update property).

## Relevant Code Paths
- `internal/matchmaker/db.go:167-200` — Monitor loop and `cancelExpiredSessions`
- `internal/matchmaker/db.go:72-75` — `selectExpiredSessions` SQL

## SUT Instrumentation
- **Missing**: `Sometimes` assertion when `cancelExpiredSessions` successfully cancels a session — confirms the monitor is working.
- **Missing**: `Reachable` assertion on the cancel path to confirm Antithesis exercises deadline expiry.
