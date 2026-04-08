# ghost-player-sessions-cleaned-up — Abandoned Queue Entries Don't Leak

## Evidence

Evil players' `doQueueAbandon` (`internal/player/behavior.go:48-50`) has a 5% chance of creating a new MatchmakerClient with a random UUID, replacing the existing one. The old player ID is abandoned in the queue.

In `Loop.waitForMatch` (`internal/player/loop.go:84-107`):
```go
if l.cfg.Behavior.doQueueAbandon(l.rng) {
    newPID := uuid.New()
    l.client = NewMatchmakerClient(l.cfg.MatchmakerURL, newPID)
}
```

The abandoned player ID stays in `MatchQueue.queued`. When the matcher pairs it with a real player:
1. A session is created on the game server.
2. `publishMatch` moves both players to `matched`.
3. The real player polls, gets the SessionInfo, connects to the game server.
4. The ghost player never connects.
5. The session's turn timer fires — if no players are connected, it resets (grace period). If one player is connected, the connected player eventually wins by timeout.

## Failure Scenario

The turn timeout fires and the connected player wins. This is correct behavior — the ghost player loses. But:
- The ghost player ID gets a loss recorded in the DB (player was created in `GetOrCreatePlayer` during queueing).
- The real player wastes time waiting for the timeout.
- Session capacity is consumed while waiting.

## Relevant Code Paths
- `internal/player/behavior.go:48-50` — `doQueueAbandon`
- `internal/player/loop.go:98-102` — Queue abandon execution
- `internal/gameserver/protocol.go:127-132` — Turn timer with grace period for no players

## SUT Instrumentation
- **Missing**: `Sometimes` assertion when a session completes with only one connected player — confirms the ghost-matching scenario is exercised.
- **Missing**: `Reachable` assertion on the turn timeout path when `len(p.players) == 1`.
