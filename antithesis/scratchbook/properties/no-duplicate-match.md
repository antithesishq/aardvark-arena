# no-duplicate-match — Players Matched to Exactly One Session

## Evidence

The matchmaking process has a race window between `collectMatches()` and `publishMatch()`:

1. `collectMatches()` (`internal/matchmaker/match_queue.go:81-116`) holds `mu`, pairs players, and returns matches. Lock released.
2. For each match, `fleet.CreateSession()` is called (HTTP network call, no lock held).
3. `publishMatch()` (`internal/matchmaker/match_queue.go:154-183`) re-acquires `mu` and checks whether both players are still in `queued`. If either left, the match is abandoned (but the game server session was already created).

The `publishMatch` check (`_, hasA := q.queued[a.pid]`) prevents double-matching because a matched player is moved from `queued` to `matched`. Once in `matched`, subsequent `collectMatches` calls won't see them.

However, there's no protection against a player being paired by two overlapping `matchPlayers()` invocations. The ticker-driven design should prevent this (only one goroutine calls `matchPlayers`), but if the ticker fires while a previous invocation is still running (during the fleet HTTP call), the same player could be collected twice.

Go's `time.Ticker` drops ticks when the receiver is busy, so this overlap shouldn't happen. But Antithesis can explore scheduler interleavings that might reveal unexpected behavior.

## Relevant Code Paths
- `internal/matchmaker/match_queue.go:66-79` — `matchPlayers`
- `internal/matchmaker/match_queue.go:81-116` — `collectMatches`
- `internal/matchmaker/match_queue.go:154-183` — `publishMatch`
- `internal/matchmaker/match_queue.go:44-57` — `StartMatcher` ticker loop

## SUT Instrumentation
- **Missing**: `Always` assertion in `publishMatch` that neither player is already in `matched` map.
