# turn-order-maintained — Turn Order Never Violated

## Evidence

`CanMakeMove` (`internal/game/game.go:167-175`) checks `player != s.CurrentPlayer` and returns an error. Every game's `MakeMove` calls `CanMakeMove` as its first operation.

The Protocol's `handleMove` (`internal/gameserver/protocol.go:206-227`) calls `session.MakeMove`, which includes the turn check. If the move is from the wrong player, an error is sent back via `SendErr`.

## Failure Scenario

The Protocol processes messages sequentially from the inbox channel (single goroutine), so true concurrent move processing isn't possible. However, the inbox channel has capacity 2, meaning two moves could be buffered. If both players send moves simultaneously, the Protocol processes them in channel-receive order. The turn check in `CanMakeMove` handles this correctly — the second move would fail the turn check.

The more subtle risk is in `handleConn`: when a player reconnects, their new connection could send a move before the Protocol processes the connection message. Since both go through the same inbox channel, ordering is preserved.

## Relevant Code Paths
- `internal/game/game.go:167-175` — `CanMakeMove`
- `internal/gameserver/protocol.go:206-227` — `handleMove`
- `internal/gameserver/session.go:241-255` — Read goroutine sends to inbox

## SUT Instrumentation
- **Missing**: `Always` assertion in `MakeMove` confirming `player == state.CurrentPlayer` before mutation.
