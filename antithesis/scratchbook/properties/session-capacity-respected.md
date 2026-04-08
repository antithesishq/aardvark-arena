# session-capacity-respected

## Evidence

The game server enforces a maximum session count via `SessionManager.CreateSession` (`internal/gameserver/session.go:66`): if `len(s.sessions) >= s.cfg.MaxSessions`, it returns `ErrMaxSessions`. The health endpoint (`internal/gameserver/server.go:80`) asserts `Always(health.ActiveSessions <= health.MaxSessions)`.

## Code Paths

- `SessionManager.CreateSession` checks capacity under `mu` lock
- `server.handleHealth` reports current active count and asserts the invariant
- The fleet (`internal/matchmaker/fleet.go`) routes away from full servers based on 503 responses

## Potential Violation Scenario

A race where `CreateSession` is called concurrently on the same game server (matchmaker sends two create requests simultaneously). The mutex protects against this, but the health check assertion provides an independent verification.

## Instrumentation Status

**FULLY COVERED** — A3 asserts this at every health check. R7 and R8 confirm the capacity-full code path is reached.
