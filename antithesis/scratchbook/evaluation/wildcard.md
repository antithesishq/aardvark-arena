# Evaluation Lens: Wildcard

## Observations

### 1. The `sendLatest` Pattern Is Under-Analyzed

The `sendLatest` function (`protocol.go:275-288`) is used throughout the game protocol for both player-facing messages and spectator channels:

```go
func sendLatest[T any](ch chan T, msg T) {
    select {
    case ch <- msg:
    default:
        select { case <-ch: default: }
        select { case ch <- msg: default: }
    }
}
```

This is a non-blocking send that drops stale messages. It's used for:
1. Player state broadcasts (`BroadcastState`, `protocol.go:257`)
2. Spectator updates (`protocol.go:259`)
3. Server-level watch events (`session.go:336`)

For spectators and watch events, message loss is acceptable (UI will catch up). But for **player messages**, losing a game state update means a player doesn't see the result of their opponent's move. They'd play based on stale state, and their next move would likely be invalid (wrong turn or occupied cell).

The channel buffer is 1 (playerConn channels are created with buffer 1 at `session.go:231`). Under normal conditions, the write goroutine drains the channel fast enough. But under Antithesis-injected slowdowns, the write goroutine could be delayed, causing `sendLatest` to drop a state update.

**This is a property gap.** No assertion checks that players eventually receive the latest game state. A `Sometimes` on the player protocol side checking that the received state matches what the server sent would surface this.

### 2. The ELO Update Loop Has a Subtle Bug Pattern

In `ReportSessionResult` (`db.go:316-339`), the ELO update loop is:
```go
for i, player := range players {
    if draw || player.PlayerID == winner {
        opponent := players[(i+1)%2]
        newPlayer, newOpponent := internal.CalcElo(player.Elo, opponent.Elo, draw)
        _, err = tx.Exec(updatePlayerStats, newPlayer, !draw, false, draw, player.PlayerID)
        _, err = tx.Exec(updatePlayerStats, newOpponent, false, !draw, draw, opponent.PlayerID)
        break
    }
}
```

For draws, `winner == uuid.Nil`, so `player.PlayerID == winner` is always false. The loop relies on `draw || ...` to enter on the first player. This means for draws, the first player in the `players` slice is always treated as the "winner" in `CalcElo`. Since `CalcElo` with `draw=true` uses `score=0.5`, and the formula is symmetric around the expected value, the result is the same regardless of which player is "first" — but this is a subtle assumption that would break if `CalcElo` had different rounding behavior for the two arguments.

**Not a bug, but fragile.** A comment or assertion documenting this symmetry assumption would prevent future regressions.

### 3. Session Orphaning Is Intentional But Untested

When `publishMatch` discovers that one or both players left the queue (`match_queue.go:162-196`), it silently drops the match. But `matchPlayers` (`match_queue.go:71-79`) already called `fleet.CreateSession`, which created a session on the game server. This session now has no players and will run until its deadline, consuming a session slot.

The code handles this with a comment: "the erroneous game session will eventually timeout." But no property verifies:
1. That the orphaned session is actually cleaned up (the session monitor should cancel it)
2. That the game server's session count returns to normal after the cleanup
3. That orphaned sessions don't accumulate to the point of filling all game servers

Under Antithesis fault injection, the race window between `collectMatches` and `publishMatch` is widened, making orphaned sessions more likely. This is confirmed by R2 ("match publishing sometimes races with player unqueueing").

**This is a property gap.** A `Sometimes` or `Always` verifying that orphaned sessions are eventually cleaned up would be valuable.

### 4. The Token Authentication Has a Nil Bypass

`TokenAuth` in `internal/api.go:49-71` skips authentication entirely if the token is nil:
```go
if token.IsNil() {
    next(w, r)
    return
}
```

In the Antithesis config, tokens are set (`AUTH_TOKEN: d4f5a1b2-...`), so this bypass is not active. But if the token configuration were somehow cleared (e.g., environment variable not set), the `/results/{sid}` endpoint would be unauthenticated, allowing any client to report fake game results.

This is not an Antithesis property per se, but it's worth noting as a design observation. The evil player workload doesn't test unauthenticated access.

### 5. Missing Graceful Shutdown Properties

The SUT analysis notes that both matchmaker and gameserver have graceful shutdown handlers (5-second timeout). But no property tests whether in-flight requests complete during shutdown, or whether sessions are cleanly cancelled. Under Antithesis process restart, the shutdown path is exercised, but the outcomes aren't verified.

## Summary of Gaps Found

1. `sendLatest` can drop player state updates under contention — no property covers this
2. Session orphaning is intentional but unverified — no property ensures orphaned sessions are cleaned up
3. Graceful shutdown correctness is unverified
