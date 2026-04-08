# third-player-rejected

## Evidence

In `protocol.handleConn` (`internal/gameserver/protocol.go:171-213`):
- If `pid` is already in `p.players`, it's a reconnection (handled above)
- If `len(p.players) == 0`, assigned P1
- If `len(p.players) == 1`, assigned P2
- `else` (len >= 2 and pid not in players): error sent, player not added

```go
} else {
    assert.Reachable("extra player connections are sometimes rejected", ...)
    conn <- PlayerMsg{Error: "too many players connected"}
}
```

Note: the error is sent but the connection channel is not closed in this branch. The caller (`sessionHandle.Join` at `session.go:230-265`) sets up read/write goroutines regardless. The write goroutine will drain `stateCh` (which has the error) and close the websocket. The read goroutine will exit when the websocket closes.

## Code Paths

- `protocol.handleConn` — `protocol.go:171-213`
- `sessionHandle.Join` — `session.go:229-265` — sets up goroutines after sending to inbox
- Evil player extra-connect chaos — `session.go:81-110` — sends random-ID connections

## Instrumentation Status

**PARTIALLY COVERED** — R10 confirms the rejection path is reached. No `Always(len(p.players) <= 2)` exists as a persistent invariant check.
