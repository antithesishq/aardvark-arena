# matchmaking-progress — Queued Players Eventually Get Matched

## Evidence

The matching algorithm (`collectMatches`) sorts candidates by queue entry time (oldest first), pairs by ELO compatibility and game preference. The ELO matching (`internal/elo.go:35-40`) relaxes the acceptable difference over time: `relaxedDiff = eloDiff - (EloDiffRelaxRate * waitTime)`. With `EloDiffRelaxRate = 50` per second and `MaxEloDiff = 200`, any two players will become compatible after at most `eloDiff/50` seconds.

The matcher runs on a configurable interval (`MatchInterval`). With the default 1-second interval, a match attempt occurs every second.

## Failure Scenario

Progress stalls if:
1. **Fleet exhausted**: All game servers unavailable. `CreateSession` returns `ErrNoServersAvailable`, and the match is silently dropped. Players remain in queue. This is self-healing — once a server becomes available, the next tick will match them.
2. **Odd number of players**: One player always left over. Self-healing when another player joins.
3. **Game preference mismatch**: Two players wanting different specific games won't match. The 80% "any game" rate mitigates this.
4. **Matcher goroutine stopped**: If the matchmaker's context is cancelled, the matcher stops. No safety net.

## Relevant Code Paths
- `internal/matchmaker/match_queue.go:66-79` — `matchPlayers`
- `internal/elo.go:35-40` — `MatchElo` with relaxation
- `internal/matchmaker/fleet.go:68-143` — `CreateSession` with retry logic

## SUT Instrumentation
- **Missing**: `Sometimes` assertion when a match is successfully created — confirms matchmaking is progressing.
- Workload-side: Players assert they eventually receive a SessionInfo from polling.
