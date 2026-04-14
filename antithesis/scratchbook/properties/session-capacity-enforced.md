# session-capacity-enforced — Game Server Respects Max Sessions

## Evidence

`SessionManager.CreateSession` (`internal/gameserver/session.go:60-137`) holds `mu` and checks `len(s.sessions) >= s.cfg.MaxSessions` before creating a new session. If at capacity, returns `ErrMaxSessions` with a `RetryAt` time based on the earliest deadline.

The test `TestServerSanity/sessions_full` (`internal/gameserver/server_test.go:84-133`) verifies a second session creation returns 503 when MaxSessions=1.

## Failure Scenario

The mutex ensures the check-and-insert is atomic. However, the session map includes sessions that may have just finished (context cancelled) but not yet run their cleanup callback. `IsFinished()` checks `ctx.Err() != nil`, but the session remains in the map until `cleanup()` runs (deferred in the goroutine). This means `len(s.sessions)` could over-count, making the server appear full when it's not — a conservatively-safe failure mode.

The more dangerous scenario: `cleanup()` deletes from the map, then a burst of `CreateSession` calls arrive. Each acquires the mutex and sees the freshly freed slot. Only one should succeed per freed slot, and the mutex serializes them correctly.

## Relevant Code Paths
- `internal/gameserver/session.go:60-137` — `CreateSession`
- `internal/gameserver/session.go:168` — `IsFinished`
- `internal/gameserver/session.go:89-101` — cleanup callback

## SUT Instrumentation
- **Missing**: `Always` assertion after successful session creation that `len(s.sessions) <= s.cfg.MaxSessions`.
