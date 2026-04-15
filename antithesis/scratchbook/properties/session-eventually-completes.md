# session-eventually-completes — Every Session Reaches Terminal State

## Evidence

Sessions have two safety nets ensuring completion:
1. **Session deadline timer** (`internal/gameserver/protocol.go:115-116`): `time.NewTimer(time.Until(p.deadline))` fires and cancels the game.
2. **Turn timeout timer** (`internal/gameserver/protocol.go:127-133`): If no valid move arrives within `turnTimeout`, the opponent wins.

Additionally, the matchmaker's session monitor (`internal/matchmaker/db.go:167-180`) cancels sessions past their DB deadline.

## Failure Scenario

A session could fail to complete if:
1. The protocol goroutine panics — the deferred `cleanup()` would still run via `defer h.cancel()` and `defer h.cleanup()`, but `report()` wouldn't be called, so the matchmaker wouldn't know the result.
2. Both timers are somehow drained without the select loop processing them (theoretically possible with Go's timer behavior, but unlikely).
3. The `done` channel closes (context cancelled) but the cleanup path has a bug.

The matchmaker's session monitor is the ultimate safety net — even if the game server fails to report, the session will be cancelled after the deadline.

## Relevant Code Paths
- `internal/gameserver/protocol.go:114-168` — `RunToCompletion` select loop
- `internal/gameserver/session.go:173-217` — `sessionHandle.RunToCompletion` (deferred cleanup)
- `internal/matchmaker/db.go:167-200` — Session monitor

## SUT Instrumentation
- **Missing**: `Sometimes` assertion in `RunToCompletion` when a terminal state is reached — confirms sessions do complete.
- **Missing**: Workload-side assertion tracking that each joined session eventually delivers a terminal state.
