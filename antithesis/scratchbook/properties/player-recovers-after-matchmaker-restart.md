# player-recovers-after-matchmaker-restart — Matchmaker Restart Recovery

## Evidence

The matchmaker's state is split between SQLite (persistent) and in-memory maps (volatile):
- **SQLite**: players, sessions, player_session tables survive restarts.
- **In-memory**: `MatchQueue.queued` (pending players), `MatchQueue.matched` (matched players awaiting game completion).

On restart:
1. SQLite is reopened, schema ensured. All existing data preserved.
2. `MatchQueue` is initialized with empty maps.
3. `StartMatcher` and `StartSessionMonitor` begin fresh.

Players that were in `matched` at crash time have active sessions in SQLite but no in-memory tracking. They will:
- Continue polling `PUT /queue/{pid}` — this returns 202 ("queued") because `matched` is empty. They re-enter the queue and get matched to a new session while their old session is still active on the game server.
- The old session continues on the game server until it completes or times out. The game server reports the result to the matchmaker, which updates ELO normally.

The risk: a player could be in two sessions simultaneously after a restart (old session on game server + new match). The `no-duplicate-match` property checks in-memory state only.

## Failure Scenario

1. Matchmaker crash while 4 players are in active sessions (2 sessions).
2. Matchmaker restarts. In-memory state is gone.
3. Players poll and re-queue. Two new sessions are created.
4. Old sessions complete on game server. Results reported. ELO updated.
5. New sessions also complete. Results also reported. ELO updated.
6. No double-session tracking — the system doesn't know about the overlap.

## Relevant Code Paths
- `internal/matchmaker/server.go:48-64` — `New()` initializes fresh state
- `internal/matchmaker/match_queue.go:33-40` — `NewMatchQueue` with empty maps
- `internal/matchmaker/db.go:97-109` — `NewDB` opens/creates schema

## SUT Instrumentation
- **Missing**: `Sometimes` assertion in the workload that a game completes after a queue error (indicating restart recovery).
- **Missing**: `Reachable` assertion on the matchmaker startup path to confirm Antithesis exercises restarts.
