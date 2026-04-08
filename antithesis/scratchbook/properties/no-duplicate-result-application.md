# no-duplicate-result-application

## Evidence

This is potentially the most significant gap in the existing assertions. The reporter (`internal/gameserver/reporter.go:78-86`) retries on temporary errors by re-enqueuing: `r.resultCh <- result`. However, a "temporary" error (e.g., TCP timeout) does not guarantee the request failed — the matchmaker may have received and processed it.

When the retry arrives, `ReportSessionResult` (`internal/matchmaker/db.go:267`) does an unconditional UPDATE:
```sql
UPDATE sessions SET cancelled = ?, winner_id = ?, completed_at = ? WHERE session_id = ?
```

This overwrites any previous result. If the session was already completed, the UPDATE succeeds silently. Worse, the ELO calculation runs again:
- `selectSessionPlayers` gets the two players
- `CalcElo` computes new ratings
- `updatePlayerStats` applies the delta

This means ELO changes are applied twice for the same game.

## Code Paths

- `Reporter.submitResult` — `reporter.go:78-86` — retry path
- `DB.ReportSessionResult` — `db.go:267-342` — no check for existing completion
- `updatePlayerStats` SQL — `db.go:52-58` — incremental updates (wins + 1, losses + 1)

## What Goes Wrong on Violation

- ELO is updated twice: winner gains double points, loser loses double points
- Win/loss/draw counters are incremented twice
- `completed_at` timestamp is overwritten (minor)

## Fix Required

Before processing in `ReportSessionResult`, check if `completed_at IS NOT NULL` and bail early. This is both a missing assertion and a potential bug.

## Instrumentation Status

**NOT COVERED** — No idempotency check or assertion exists. This needs both a code fix (early return if already completed) and an `Always` or `AlwaysOrUnreachable` assertion.
