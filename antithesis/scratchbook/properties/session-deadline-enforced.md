# session-deadline-enforced

## Evidence

Two mechanisms enforce session deadlines:

1. **Game server side**: `RunToCompletion` (`protocol.go:116-117`) creates a timer for the deadline. When it fires, the session is cancelled.
2. **Matchmaker side**: `StartSessionMonitor` (`db.go:166-181`) periodically queries for expired sessions (`SELECT session_id FROM sessions WHERE deadline < ? AND completed_at IS NULL`) and cancels them.

The Antithesis config sets `session-timeout=1m` and `monitor-interval=2s`, so the monitor checks frequently.

S6 asserts `sessions sometimes expire before completion`.

## Code Paths

- `protocol.RunToCompletion` — `protocol.go:116-117, 125-127` — session-level deadline
- `DB.cancelExpiredSessions` — `db.go:185-206` — matchmaker-level sweep
- `ReportSessionResult(cancelled=true)` — `db.go:267-342` — marks session as cancelled

## Instrumentation Status

**FULLY COVERED** — S6 asserts this liveness property. The session monitor provides the backstop mechanism.
