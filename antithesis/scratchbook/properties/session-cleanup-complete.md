# session-cleanup-complete — Finished Sessions Are Removed

## Evidence

Session cleanup is handled by the `cleanup` callback (`internal/gameserver/session.go:89-101`), which is set when the session is created and called via `defer h.cleanup()` in `RunToCompletion` (`session.go:175`).

The cleanup:
1. Acquires `mu` and deletes the session from the sessions map.
2. Acquires `watchMu` and deletes lastEvents, sends session-end watch event.
3. Broadcasts updated health.

## Failure Scenario

The cleanup runs deferred, so it executes even if `RunToCompletion` panics (as long as the goroutine doesn't crash the process). The concern is timing:
1. Between session completion and cleanup, `JoinSession` could still find the session in the map. It checks `IsFinished()` and returns an error, which is correct.
2. Between cleanup's `delete(s.sessions, sid)` and the health broadcast, the session count is briefly inconsistent with what watchers see.
3. If `broadcastHealth` panics (unlikely), the watch state would be stale.

## Relevant Code Paths
- `internal/gameserver/session.go:89-101` — cleanup callback
- `internal/gameserver/session.go:173-176` — deferred cancel and cleanup
- `internal/gameserver/session.go:140-154` — JoinSession checks IsFinished

## SUT Instrumentation
- **Missing**: `Always` assertion after `delete(s.sessions, sid)` confirming the session ID is no longer in the map.
