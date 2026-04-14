# third-player-rejected — Excess Connections Are Rejected

## Evidence

`handleConn` (`internal/gameserver/protocol.go:170-204`) handles player connections:

```go
if existing, ok := p.players[pid]; ok {
    // reconnection — replace existing
} else if len(p.players) == 0 {
    // first player → P1
} else if len(p.players) == 1 {
    // second player → P2
} else {
    conn <- PlayerMsg{Error: "too many players connected"}
}
```

A third unique player ID gets an error message on their channel. The channel is not added to `p.players`, and the connection's write goroutine (spawned in `Join`) will deliver the error and then the channel will be garbage collected.

The evil player mode includes `runExtraConnectChaos` (`internal/player/session.go:80-105`) which periodically connects with random player IDs. These probe connections are closed quickly after connecting.

## Failure Scenario

The error is sent on the `conn` channel but the channel is never closed by the Protocol. The write goroutine in `Join` (`session.go:231-239`) ranges over `stateCh`, so it would block forever waiting for more messages. However, the goroutine also writes to the websocket — if the websocket is closed by the client (as evil players do immediately), the write goroutine exits on the websocket error.

The dangling goroutine would only be a problem if the evil player keeps the websocket open indefinitely. In practice, evil players close quickly (`chaos join probe`).

## Relevant Code Paths
- `internal/gameserver/protocol.go:170-204` — `handleConn` third-player rejection
- `internal/gameserver/session.go:220-256` — `Join` spawns goroutines for the connection
- `internal/player/session.go:80-105` — `runExtraConnectChaos`

## SUT Instrumentation
- **Missing**: `Always` assertion that when `len(p.players) >= 2` and a new unknown pid arrives, the pid is NOT added to `p.players`.
- **Missing**: `Reachable` assertion on the "too many players" path to confirm evil players exercise it.
