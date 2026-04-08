# cancelled-session-no-winner

## Evidence

This is the most heavily guarded invariant in the codebase, asserted at three distinct points:

1. **Matchmaker HTTP handler** (`internal/matchmaker/server.go:159`): `assert.Always(!body.Cancelled || body.Winner == uuid.Nil, "cancelled session results never declare a winner")` — checks the incoming HTTP request body.
2. **Matchmaker DB layer** (`internal/matchmaker/db.go:273`): Same assertion before writing to the database. Also has a `log.Panicf` at line 289 as a hard safety net.
3. **Game Server reporter** (`internal/gameserver/reporter.go:53`): `assert.Always(!result.cancelled || result.winner == uuid.Nil, "gameserver reports never include a winner for cancelled sessions")` — checks before sending the HTTP request.

## Code Paths

The result flows: `protocol.report()` -> `resultMsg` channel -> `Reporter.submitResult()` -> HTTP PUT `/results/{sid}` -> `server.handleResult()` -> `DB.ReportSessionResult()`.

The invariant is checked at the reporter (sender), the HTTP handler (receiver), and the DB writer (persister). This defense-in-depth approach means a violation at any layer is caught.

## What Goes Wrong on Violation

A cancelled session with a winner would cause `ReportSessionResult` to enter the ELO update branch for a session that should have no outcome. This would give one player an unearned win and the other an unearned loss.

## Instrumentation Status

**FULLY COVERED** — Three `Always` assertions (A1, A2, A4) cover this invariant comprehensively. No additional instrumentation needed.
