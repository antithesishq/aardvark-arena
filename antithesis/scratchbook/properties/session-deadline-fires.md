# session-deadline-fires — Session Deadline Cancels Game

## Evidence

The Protocol's select loop (`internal/gameserver/protocol.go:114-125`):
```go
timer := time.NewTimer(time.Until(p.deadline))
// ...
case <-timer.C:
    p.report(game.Cancelled)
    break outer
```

With a 2-minute session timeout in the Antithesis config, this should fire for any game that takes too long.

## Relevant Code Paths
- `internal/gameserver/protocol.go:114-125` — Deadline timer handling
- `cmd/matchmaker/main.go:18` — DefaultSessionTimeout (5 minutes, overridden in Antithesis config)

## SUT Instrumentation
- **Missing**: `Reachable` assertion on the deadline cancellation path.
