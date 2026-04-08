# session-always-two-players

## Evidence

When `ReportSessionResult` processes a non-cancelled session, it queries `player_session` for the two players (`internal/matchmaker/db.go:299`). If `len(players) != 2`, it hits the `Unreachable` assertion (U1) and returns an error.

## Code Paths

- `DB.CreateSession` (`db.go:231`) inserts exactly two `player_session` rows in a transaction
- `DB.ReportSessionResult` (`db.go:267`) queries those rows when updating ELO
- The `Unreachable` at line 304 catches any deviation

## Potential Violation Scenario

If a transaction in `CreateSession` partially commits (one player_session inserted, second fails), the session would have only one player. SQLite transactions are atomic, so this should be impossible — the `Unreachable` is a defense-in-depth check.

## Instrumentation Status

**FULLY COVERED** — U1 asserts this invariant at the critical code point.
