# no-double-elo-update — Session Result Processed Exactly Once

## Evidence

There are three paths that can call `ReportSessionResult` for the same session:
1. **Reporter** (`internal/gameserver/reporter.go:50-81`): Game server's normal completion path. Re-enqueues on temporary HTTP errors.
2. **Session Monitor** (`internal/matchmaker/db.go:184-200`): Cancels expired sessions.
3. **Cancel Endpoint** (`internal/matchmaker/server.go:173-211`): Manual cancel via DELETE `/session/{sid}`.

The `updateSessionResult` SQL (`UPDATE sessions SET cancelled = ?, winner_id = ?, completed_at = ? WHERE session_id = ?`) has no `WHERE completed_at IS NULL` guard — it will succeed even if the session is already completed. This means a second call will overwrite the first result and apply ELO changes again.

## Failure Scenario

1. Game ends normally. Reporter submits result. ELO updated for both players.
2. Network delay causes the Reporter to see a temporary error and re-enqueue.
3. Meanwhile, session monitor finds the session expired (deadline < now, completed_at is now set but the monitor query ran before the first update committed).
4. Reporter retries and succeeds — second ELO update applied.

Or more simply: the Reporter's temporary-error retry re-enqueues the result to `resultCh`. If the channel is read twice (once on error, once on retry), the result is submitted twice.

## Relevant Code Paths
- `internal/gameserver/reporter.go:72-75` — Re-enqueue on temporary error
- `internal/matchmaker/db.go:262-323` — `ReportSessionResult` (no idempotency guard)
- `internal/matchmaker/db.go:72-82` — `selectExpiredSessions` (race window)

## SUT Instrumentation
- **Missing**: `Always` assertion tracking processed session IDs. On second call for same session, assert that it's a no-op (or panic/log the duplicate).
- **Missing**: `WHERE completed_at IS NULL` guard on `updateSessionResult` SQL to make it idempotent.
