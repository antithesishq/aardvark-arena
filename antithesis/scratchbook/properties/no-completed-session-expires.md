# no-completed-session-expires

## Evidence

The session monitor's SQL query (`internal/matchmaker/db.go:73-76`):
```sql
SELECT session_id FROM sessions WHERE deadline < ? AND completed_at IS NULL
```

The `completed_at IS NULL` filter prevents re-selecting completed sessions. However, there's a TOCTOU window:

1. Monitor runs query, gets session X (completed_at IS NULL, deadline past)
2. Game server reports result for session X, `ReportSessionResult` sets completed_at
3. Monitor calls `ReportSessionResult(cancelled=true)` for session X

Since `ReportSessionResult` does an unconditional UPDATE (no check for existing completion), the second call overwrites the legitimate result with a cancellation. This also re-runs the ELO calculation with `cancelled=true`, which skips the ELO update — but the first call already applied ELO changes, so the player stats are now inconsistent (they got ELO for a win, but the session shows as cancelled).

## Code Paths

- `cancelExpiredSessions` — `db.go:185-206` — iterates expired sessions
- `ReportSessionResult` — `db.go:267-342` — unconditional UPDATE
- Race window: between the SELECT at line 187 and the UPDATE inside `ReportSessionResult`

## Relationship to no-duplicate-result-application

This property is a specific instance of the more general `no-duplicate-result-application` property. Both stem from the same root cause: `ReportSessionResult` lacks an idempotency check.

## Instrumentation Status

**NOT COVERED** — The SQL filter is correct for the read, but no assertion guards against the TOCTOU between read and write. Closely related to `no-duplicate-result-application`.
