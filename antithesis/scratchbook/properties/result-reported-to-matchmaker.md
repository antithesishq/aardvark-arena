# result-reported-to-matchmaker — Game Results Reach the Matchmaker

## Evidence

The Reporter (`internal/gameserver/reporter.go:34-48`) consumes from `resultCh` and calls `submitResult`. On temporary HTTP errors, the result is re-enqueued to `resultCh` (`reporter.go:72-75`). On permanent errors, the result is logged and dropped (`reporter.go:76-78`).

## Failure Scenario

1. **Permanent network failure**: If the matchmaker is permanently unreachable (not just temporarily), results are dropped after one attempt. No retry with backoff.
2. **Channel capacity**: `resultCh` has capacity `MaxSessions`. If many sessions complete simultaneously and the matchmaker is slow, the channel could fill. The Reporter is the only consumer, and `submitResult` is synchronous — it blocks on HTTP. If the channel fills, `Protocol.report()` could block on `p.result <- resultMsg{...}`, stalling the game protocol.
3. **Re-enqueue loop**: Temporary errors cause re-enqueue. If the error persists, the same result bounces between `submitResult` and the channel indefinitely, consuming capacity.

## Relevant Code Paths
- `internal/gameserver/reporter.go:34-81` — Reporter loop and submitResult
- `internal/gameserver/protocol.go:106-110` — `report()` sends to resultCh
- `internal/http.go:25-30` — `HTTPIsTemporary` classification

## SUT Instrumentation
- **Missing**: `Sometimes` assertion when `submitResult` gets HTTP 200 — confirms results are being delivered.
- **Missing**: `Reachable` assertion on the retry path to confirm Antithesis exercises delivery failures.
