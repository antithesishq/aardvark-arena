# game-terminates

## Evidence

Every exit path from `Protocol.RunToCompletion` (`internal/gameserver/protocol.go:115-169`) calls `p.report()`, which sends a `resultMsg`:
- `done` channel closed (external cancellation) -> reports Cancelled
- `timer.C` fires (session deadline) -> reports Cancelled  
- `turnTimer.C` fires (no move) -> reports opponent wins
- Normal game completion (`state.Status.IsTerminal()`) -> reports actual status
- Channel closed (`!ok`) -> breaks out of loop (but `defer h.cancel()` + `defer h.cleanup()` in `RunToCompletion` at session.go:179-180 handle cleanup)

The session timeout (1 minute in Antithesis config) and turn timeout (10 seconds) provide hard time bounds.

## Code Paths

- `RunToCompletion` — `protocol.go:115-169` — main game loop
- `report` — `protocol.go:96-112` — sends result on channel
- Turn timer reset — `protocol.go:246` — reset after each valid move
- Session handle — `session.go:178-226` — `defer cancel()`, `defer cleanup()`

## Potential Violation Scenario

If the turn timer is repeatedly reset by invalid moves (which don't reset it — only valid moves at line 246 do), the timer could fire even while moves are being sent. But the evil player's out-of-turn and malformed moves go through `handleMove`, which only resets the timer on successful `MakeMove`. So a stream of invalid moves will trigger the turn timer correctly.

The real concern: if `p.inbox` blocks (channel full) and no timer fires. The inbox has a buffer of 2 (session.go:84), and `handleMove` processes synchronously, so the inbox shouldn't stay full long. The session deadline provides an absolute backstop.

## Instrumentation Status

**PARTIALLY COVERED** — S11, S12, S13 assert that various terminal states are sometimes reached. But no per-session assertion guarantees every started game reaches a terminal state. The session cleanup in `RunToCompletion` is the code-level guarantee.
