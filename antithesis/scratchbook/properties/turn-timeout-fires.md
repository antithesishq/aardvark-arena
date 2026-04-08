# turn-timeout-fires — Turn Timer Triggers Game End

## Evidence

The Protocol's select loop (`internal/gameserver/protocol.go:127-133`) handles the turn timer:

```go
case <-p.turnTimer.C:
    if len(p.players) == 0 {
        p.turnTimer.Reset(p.turnTimeout * 2)
        continue
    }
    p.report(p.state.CurrentPlayer.Opponent().Wins())
    break outer
```

If no players are connected, the timer resets (connection grace period). Otherwise, the opponent of the current player wins.

## Relevant Code Paths
- `internal/gameserver/protocol.go:76-78` — Initial turn timer (2x timeout as grace)
- `internal/gameserver/protocol.go:127-133` — Turn timeout handling
- `internal/gameserver/protocol.go:225` — Timer reset on valid move

## SUT Instrumentation
- **Missing**: `Reachable` assertion on the turn timeout path (when `len(p.players) > 0`).
