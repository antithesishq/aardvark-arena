# SUT Analysis: Aardvark Arena

## System Overview

Aardvark Arena is a distributed game tournament system with three server-side components (matchmaker, game servers, AI players) communicating over HTTP and WebSocket. It supports three game types (Tic-Tac-Toe, Connect4, Battleship) with ELO-based matchmaking, session management, and adversarial fault injection via "evil" players.

## Architecture and Data Flow

### Components

1. **Matchmaker** (`cmd/matchmaker`, `internal/matchmaker/`)
   - Central coordinator: queues players, matches them by ELO proximity, assigns sessions to game servers
   - Persists all state in SQLite (`:memory:` by default, can be file-backed)
   - REST API: `PUT /queue/{pid}`, `DELETE /queue/{pid}`, `PUT /results/{sid}`, `GET /status`, `GET /leaderboard`, `GET /servers`, `DELETE /session/{sid}`
   - Background goroutines: `StartMatcher` (periodic match attempts), `StartSessionMonitor` (periodic expired session cleanup)

2. **Game Server** (`cmd/gameserver`, `internal/gameserver/`)
   - Runs game sessions: accepts create-session requests from matchmaker, manages WebSocket connections for players
   - Multiple instances (3 in Antithesis config) behind the matchmaker's `Fleet`
   - REST API: `PUT /session/{sid}` (create), `GET/WS /session/{sid}/{pid}` (player join), `GET /sessions`, `GET /session/{sid}/watch` (spectate), `DELETE /session/{sid}` (cancel), `GET /watch` (server-level events)
   - Reports results back to matchmaker via `PUT /results/{sid}` with retry on temporary errors

3. **Player** (`cmd/player`, `internal/player/`)
   - AI client: polls matchmaker queue, connects to assigned game server via WebSocket, plays game moves
   - Supports "evil" mode with configurable chaos rates for malformed moves, out-of-turn moves, extra connections, queue abandonment
   - Reconnects transparently on WebSocket failures

4. **Swarm** (`cmd/swarm`)
   - Spawns N concurrent player goroutines; thin wrapper over `player.Loop`

### Request Paths

**Match lifecycle (happy path):**
1. Player PUTs `/queue/{pid}` on matchmaker -> returns 202 (queued) or 200 (already matched)
2. Matchmaker's `StartMatcher` goroutine periodically calls `collectMatches` -> pairs players by ELO, creates sessions on game servers via Fleet
3. Fleet PUTs `/session/{sid}` on a game server -> server creates `sessionHandle`, starts `RunToCompletion` goroutine
4. Player polls queue again -> gets `SessionInfo` with server URL
5. Player dials WebSocket to `/session/{sid}/{pid}` on game server
6. Both players connected -> game protocol exchanges state/moves via WebSocket
7. Game ends -> `protocol.report()` sends `resultMsg` on channel
8. Reporter PUTs `/results/{sid}` on matchmaker -> ELO updated, session marked completed
9. Players poll queue again for next match

**Session timeout path:**
- Matchmaker's `StartSessionMonitor` finds sessions past deadline -> calls `ReportSessionResult(cancelled=true)` -> calls `queue.Untrack(sid)`

**Turn timeout path:**
- Game server's protocol has a `turnTimer`; if it fires, the current player's opponent wins

## State Management and Persistence

### SQLite (Matchmaker)
- Tables: `players` (player_id, elo, wins, losses, draws), `sessions` (session_id, server, game, created_at, deadline, completed_at, cancelled, winner_id), `player_session` (player_id, session_id)
- WAL mode, synchronous=normal
- In-memory by default (`:memory:` with `MaxOpenConns(1)`)
- Transactions used for `GetOrCreatePlayer`, `CreateSession`, `ReportSessionResult`
- **No explicit foreign key enforcement** — SQLite foreign keys are off by default unless `PRAGMA foreign_keys=ON` is set (not present in `ensureSchema`)

### In-Memory State (Matchmaker)
- `MatchQueue.queued`: `map[PlayerID]*candidate` — players waiting for a match
- `MatchQueue.matched`: `map[PlayerID]*SessionInfo` — players assigned to sessions
- Protected by `MatchQueue.mu` (sync.Mutex)

### In-Memory State (Game Server)
- `SessionManager.sessions`: `map[SessionID]sessionHandle` — active game sessions
- Each `sessionHandle` has its own context for cancellation and an `inbox` channel for messages
- `Protocol` struct holds game state (`game.State[Shared]`) in the goroutine that runs `RunToCompletion`
- Two lock groups: `SessionManager.mu` (sessions map) and `SessionManager.watchMu` (watchers/events), with documented lock ordering: `mu` before `watchMu`

### State Consistency Concerns
- Matchmaker queue and DB can diverge: a session is created in DB during `publishMatch` but if a player left the queue between `collectMatches` and `publishMatch`, the session is created on the game server but the players are not assigned. The orphaned session eventually times out.
- Game server's reporter retries on temporary errors by re-enqueuing the result (`r.resultCh <- result`). Non-temporary errors are logged and dropped — the matchmaker session monitor eventually times it out.
- Result reporting has no idempotency token — if the reporter retries and the first attempt actually succeeded, the matchmaker's `ReportSessionResult` will overwrite the existing result (UPDATE is unconditional).

## Concurrency Model

### Goroutine Structure
- **Matchmaker:** Main HTTP server goroutine + `StartMatcher` goroutine + `StartSessionMonitor` goroutine
- **Game Server:** Main HTTP server goroutine + `StartReporter` goroutine + per-session `RunToCompletion` goroutine + per-player read/write goroutines (2 per connected player)
- **Player:** Main loop goroutine + `Session.Run` goroutine + bridge read/write goroutines + optional `runExtraConnectChaos` goroutine

### Synchronization
- Matchmaker: `MatchQueue.mu` protects `queued` and `matched` maps. Fleet has no synchronization — it's only called from the matcher goroutine (single caller).
- Game Server: `SessionManager.mu` protects the sessions map. `SessionManager.watchMu` protects watcher channels. Lock ordering is documented. Per-session communication is via channels (`inbox`, `result`), avoiding shared state within a session.
- Player: Communication between session and protocol is via channels (`protocolRx`, `protocolTx`).

### Concurrency Risks
1. **Fleet.rng is not goroutine-safe**: `Fleet.rng` is a `*rand.Rand` used in `CreateSession` to shuffle candidates. `math/rand.Rand` is not safe for concurrent use. Currently safe because `matchPlayers` (the only caller of `CreateSession`) runs in a single goroutine, but this is an implicit assumption.
2. **Reporter retry re-enqueue**: `r.resultCh <- result` in `submitResult` can block if the channel is full (buffered at `cfg.MaxSessions`). Under high load with many temporary failures, this could stall the reporter goroutine.
3. **`publishMatch` race window**: Between `collectMatches` (which reads the queue under lock) and `publishMatch` (which re-acquires the lock), a player can unqueue. The code handles this correctly (orphans the session) but it creates game server sessions that will never have players.
4. **`sendLatest` channel pattern**: The `sendLatest` function drains a stale value and retries. Under high contention, the drain-and-retry can itself fail (default branch), silently dropping a message. This is acceptable for display (watch/spectator) but would be problematic for game-critical messages.

## Safety Guarantees (Claimed or Implied)

1. **Cancelled sessions never have a winner** — Asserted at three points (A1, A2, A4). This is the most heavily guarded invariant.
2. **Active sessions never exceed max capacity** — Asserted in health endpoint (A3).
3. **Every active session maps to exactly two players** — Asserted via Unreachable at DB layer (U1).
4. **Only supported game kinds are run** — Asserted via Unreachable in session manager (U3).
5. **Moves only arrive from connected players** — Asserted via Unreachable in protocol (U4).
6. **Game servers only return 200 or 503 for session creation** — Asserted via Unreachable in fleet (U2).

## Liveness Guarantees (Claimed or Implied)

1. **Players eventually get matched** — Implied by the periodic matcher and ELO relaxation over time.
2. **Sessions eventually complete or are cancelled** — The session monitor cancels expired sessions; the turn timer forces completion.
3. **Game results are eventually reported to matchmaker** — Reporter retries on temporary errors; session monitor provides a backstop.
4. **Services recover to healthy state** — Tested by `eventually_health_check.sh`.
5. **Post-test system remains functional** — Tested by `eventually_queue_empty.sh`.

## Failure and Degradation Modes

### Network Failures
- **Fleet -> Game Server**: Temporary errors trigger per-server retry backoff (`FailureTimeout = 1 minute`). If all servers are in backoff, `ErrNoServersAvailable` is returned and the match is skipped.
- **Reporter -> Matchmaker**: Temporary errors cause re-enqueue to `resultCh`. Non-temporary errors are logged and dropped.
- **Player -> Matchmaker**: Queue poll errors are logged and retried on next poll interval.
- **Player -> Game Server**: WebSocket connection failures trigger reconnection after `reconnectInterval` (1s).

### Partial Failures
- **One game server down**: Fleet skips it for `FailureTimeout` (1 minute), routes to others.
- **Matchmaker down**: Game servers can't report results (reporter retries). Players can't queue.
- **Player disconnects mid-game**: Turn timer eventually fires, opponent wins.

### Timeout Hierarchy
- Session timeout: 1 minute (in Antithesis config)
- Turn timeout: 10 seconds (in Antithesis config)
- Session monitor interval: 2 seconds (in Antithesis config)
- Match interval: 500ms (in Antithesis config)
- HTTP dial timeout: 5 seconds
- WebSocket reconnect interval: 1 second

## Existing Test Strategy

### Unit Tests
- `internal/elo_test.go` — ELO calculation
- `internal/game/battleship_test.go`, `internal/game/harness_test.go` — Game logic
- `internal/matchmaker/db_test.go`, `internal/matchmaker/match_queue_test.go`, `internal/matchmaker/fleet_test.go` — Matchmaker components
- `internal/gameserver/server_test.go` — Game server health and session creation

### Integration Tests
- `internal/player/integration_test.go` — End-to-end test: starts matchmaker + gameserver, runs 2 players for 10 sessions each

### Antithesis Testing
- Full Antithesis integration with 48 SDK assertions (see `existing-assertions.md`)
- Two workload drivers (normal + evil players) and two eventually checks
- Instrumented Go compilation via `antithesis-go-instrumentor`
- Antithesis-controlled RNG via `antirandom.Source()`

### Gaps Antithesis Can Fill
- Timing-sensitive races (queue-match-publish window, reporter retry storms, turn timer edge cases)
- Multi-game-server failure combinations
- Concurrent evil player behavior at scale
- Database consistency under concurrent session creation/completion
- WebSocket reconnection races during state transitions

## Attack Surfaces for Antithesis

1. **Match-publish race**: `collectMatches` -> (player unqueues) -> `publishMatch` creates orphaned sessions
2. **Reporter retry semantics**: Re-enqueue on temporary error; no idempotency; potential duplicate result reports
3. **Turn timer edge cases**: Timer fires at exact moment a move arrives; `sendLatest` drops state during high-frequency broadcasts
4. **Session capacity boundary**: Rapid session creation/completion near `MaxSessions` limit
5. **Fleet server rotation**: All servers in backoff simultaneously; recovery timing after backoff expires
6. **Evil player chaos**: Malformed JSON, out-of-turn moves, extra connections, queue abandonment — all at high rates simultaneously
7. **ELO calculation under concurrent draws/wins**: Multiple sessions completing simultaneously for the same player (possible if a player somehow gets into two sessions)
8. **SQLite under concurrent access**: Even with `MaxOpenConns(1)` for in-memory, the WAL mode for file-backed DBs could have interleaving issues

## Assumptions

- SQLite foreign keys are not enforced (no `PRAGMA foreign_keys=ON`)
- The matchmaker is a singleton — no distributed state concerns
- Game servers are stateless between sessions (no persistent storage)
- Player IDs are globally unique UUIDs (collision is astronomically unlikely)
- Time-based operations (deadlines, timeouts) use wall clock time, not monotonic (relevant under Antithesis time manipulation)

## Open Questions

- Can a player end up in two simultaneous sessions? The queue clears them from `queued` when matched, but if two matchers ran concurrently (they don't — single goroutine), this could happen. With the current design this is prevented by the single-goroutine matcher.
- What happens if `ReportSessionResult` is called twice for the same session? The UPDATE is unconditional, so it would overwrite. ELO would be recalculated incorrectly.
- How does the system behave when the SQLite DB is file-backed (not `:memory:`)? The Antithesis config uses the default (`:memory:`), so disk I/O failures aren't exercised.
