# result-reported-for-every-session

## Evidence

The session lifecycle has multiple paths to completion:
1. Normal game completion: `protocol.report()` -> reporter -> matchmaker
2. Turn timeout: `turnTimer.C` fires -> `protocol.report(Cancelled)` -> reporter -> matchmaker
3. Session deadline: `timer.C` fires -> `protocol.report(Cancelled)` -> reporter -> matchmaker
4. Session monitor: `cancelExpiredSessions` finds sessions past deadline -> `ReportSessionResult(cancelled)` directly on DB

The reporter retries on temporary errors (R13). Non-temporary errors are logged and dropped (R14), but the session monitor provides a backstop by cancelling expired sessions.

## Code Paths

- `protocol.RunToCompletion` — every exit path calls `p.report()` which sends on `resultCh`
- `Reporter.submitResult` — sends HTTP PUT, retries on temp error, drops on non-temp error
- `DB.cancelExpiredSessions` — periodic sweep, cancels via `ReportSessionResult`

## Gap Analysis

The backstop (session monitor) depends on the session timeout. If a session's deadline is in the past and the monitor runs, it will cancel it. But if the reporter drops a result and the session was already past its deadline (normal completion after deadline), the monitor won't find it because the game server already reported cancellation.

The real risk is: game server completes a game -> reporter drops the result -> session monitor already ran and didn't find it (not expired yet at that time) -> session stays in "active" state in the DB until the next monitor cycle.

## Instrumentation Status

**PARTIALLY COVERED** — S6 confirms sessions sometimes expire. `eventually_queue_empty.sh` validates end-to-end flow. But no direct assertion checks that *every* session in the DB eventually gets a `completed_at` value.
