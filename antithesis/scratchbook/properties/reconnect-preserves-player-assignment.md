# reconnect-preserves-player-assignment

## Evidence

In `protocol.handleConn` (`internal/gameserver/protocol.go:171-213`), when a player reconnects:
```go
if existing, ok := p.players[pid]; ok {
    assert.Reachable("players sometimes reconnect to an in-progress session", ...)
    close(existing.conn)
    p.players[pid] = playerConn{player: existing.player, conn: conn}
}
```

The new `playerConn` uses `existing.player` (the original P1/P2 assignment) with the new connection channel. This preserves the player number.

## Code Paths

- First connection: `protocol.go:180-185` — assigned P1
- Second connection: `protocol.go:186-191` — assigned P2
- Reconnection: `protocol.go:172-179` — reuses existing player number
- Extra connection: `protocol.go:192-198` — rejected with error

## What Goes Wrong on Violation

If a reconnecting player got a different player number, they'd receive game state for the wrong perspective and their moves would be applied to the wrong side. In Battleship, they'd see the wrong ship placements.

## Instrumentation Status

**PARTIALLY COVERED** — R9 confirms reconnection is reached. The code preserves the assignment, but an `Always(p.players[pid].player == existing.player)` after the replacement would explicitly verify it.
