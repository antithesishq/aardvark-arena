# player-reconnect-works — Reconnected Players Resume Game

## Evidence

`handleConn` in the Protocol (`internal/gameserver/protocol.go:170-204`) handles reconnection:

```go
if existing, ok := p.players[pid]; ok {
    close(existing.conn)
    p.players[pid] = playerConn{player: existing.player, conn: conn}
}
```

When a player reconnects, the old channel is closed (which terminates the old write goroutine), and the new channel replaces it. The player retains their original player role (P1/P2). After replacing the connection, `SendState(pid)` sends the current state to the new connection.

The player-side Session (`internal/player/session.go:55-77`) handles reconnection in `Run()`. If `bridge()` returns with a non-close error, the loop retries by calling `dial()` again.

## Failure Scenario

1. **Old write goroutine race**: `close(existing.conn)` triggers the old write goroutine to exit (it iterates `for state := range stateCh`). But the old goroutine might be in the middle of a `wsjson.Write`. The websocket connection itself isn't closed here — only the channel is closed. The old goroutine will finish its write and then exit the range loop, then close the websocket with `StatusNormalClosure`. This could cause the old websocket to send a close frame to the player, which the player's read goroutine interprets as a server-initiated close.

2. **State delivery gap**: Between the old connection failing and the new connection receiving state, moves may have been made. The `SendState` call sends the *current* state, so no moves are lost. But the player may miss intermediate states (no big deal for the AI).

3. **Inbox channel backpressure**: The inbox channel has capacity 2. If a reconnection message and a move from the other player arrive simultaneously, both fit in the channel. But if a spectator join, a reconnection, and a move all arrive at once, the inbox could block.

## Relevant Code Paths
- `internal/gameserver/protocol.go:170-204` — `handleConn`
- `internal/gameserver/session.go:220-256` — `Join` (spawns read/write goroutines)
- `internal/player/session.go:55-77` — `Run` reconnection loop
- `internal/player/session.go:108-118` — `dial` with retry

## SUT Instrumentation
- **Missing**: `Always` assertion after reconnection that the new connection receives a valid state message.
- **Missing**: `Reachable` assertion on the reconnection path (existing player replaced).
