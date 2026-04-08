# matched-players-removed-from-queue

## Evidence

In `publishMatch` (`internal/matchmaker/match_queue.go:158-196`), after confirming both players are still in the queue (`hasA && hasB`), the code:
1. Creates a DB session (`db.CreateSession`)
2. Deletes both players from `queued` (`delete(q.queued, a.pid)`, `delete(q.queued, b.pid)`)
3. Adds both to `matched` (`q.matched[a.pid] = session`, `q.matched[b.pid] = session`)

This all happens under `q.mu` lock. The concern is the window between `collectMatches` (which reads `queued` under lock, then releases) and `publishMatch` (which re-acquires the lock). During this window, a player could `Unqueue`.

## Code Paths

- `collectMatches` — `match_queue.go:82-120` — reads queue under lock, returns matches
- `matchPlayers` — `match_queue.go:67-79` — iterates matches, calls fleet, then publishMatch
- `publishMatch` — `match_queue.go:158-196` — re-acquires lock, checks both still queued
- `Unqueue` — `match_queue.go:229-233` — deletes from queued under lock

## Race Window

1. `collectMatches` reads [A, B] as a match, releases lock
2. Player A calls `Unqueue`, removes from `queued` under lock
3. `publishMatch` acquires lock, sees `hasA=false`
4. Neither player is promoted to `matched`; the game server session was already created but will be orphaned

The code handles this correctly (orphans the session), but no assertion verifies the postcondition: "a player is never simultaneously in both `queued` and `matched`."

## Instrumentation Status

**NOT COVERED** — R2 confirms the race window is reached. An `Always` assertion after the promotion (line 188) checking `_, inQueue := q.queued[a.pid]; !inQueue` for both players would close this gap.
