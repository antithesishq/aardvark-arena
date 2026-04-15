# protocol-not-blocked — Protocol Goroutine Remains Responsive

## Evidence

The protocol goroutine's select loop (`internal/gameserver/protocol.go:114-168`) handles four channels: `done`, `timer.C`, `turnTimer.C`, and `inbox`. The `report()` function (`protocol.go:106-110`) sends synchronously on `p.result`:

```go
p.result <- resultMsg{...}
```

`resultCh` has capacity `MaxSessions` (`session.go:41`). The Reporter (`reporter.go:34-48`) is the sole consumer. When the matchmaker is unreachable, `submitResult` re-enqueues the result (`reporter.go:72-75`), consuming and producing one message. But if many sessions complete while the matchmaker is down, the channel fills.

Once `resultCh` is full, `Protocol.report()` blocks. Since `report()` is called from within the select loop, the entire loop stalls. Turn timeouts and session deadlines fire but aren't processed. The game appears hung.

## Failure Scenario

1. Network partition isolates matchmaker from game server.
2. Multiple sessions reach turn timeout simultaneously.
3. Each calls `report()` which blocks on full `resultCh`.
4. Remaining active sessions can't process moves, timeouts, or deadlines.
5. The system appears completely frozen until the partition heals and the Reporter drains the channel.

## Relevant Code Paths
- `internal/gameserver/protocol.go:106-110` — `report()` blocks on `p.result`
- `internal/gameserver/reporter.go:72-75` — Re-enqueue on temporary error
- `internal/gameserver/session.go:39-41` — `resultCh` capacity = `MaxSessions`

## SUT Instrumentation
- **Missing**: `Always` assertion tracking time between timer expiry and processing in the select loop. If the delta exceeds a threshold (e.g., 2x the expected processing time), the assertion fires.
- Alternative approach: Make `report()` non-blocking with a `select { case p.result <- msg: default: log.Printf("result channel full") }`.
