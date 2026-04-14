# SUT Analysis — Aardvark Arena

## System Overview

Aardvark Arena is a distributed turn-based game platform where AI players compete in TicTacToe, Connect4, and Battleship. It consists of four components communicating over HTTP and WebSocket.

### Product Context

The system is a reference project for demonstrating Antithesis. Users observe AI bots playing games through a spectator dashboard. User-visible failures include: games hanging, incorrect winners, leaderboard corruption, spectator state divergence, and game servers becoming unavailable.

## Architecture and Data Flow

### Components

| Component | Language | Transport | Role |
|-----------|----------|-----------|------|
| Matchmaker | Go | HTTP | Queue players, track ELO, create sessions via Fleet, persist to SQLite |
| Game Server | Go | HTTP + WebSocket | Host game sessions, run game protocols, report results |
| Player | Go | HTTP + WebSocket client | Poll matchmaker queue, connect to game server, play via AI |
| Swarm | Go | — | Launch N players concurrently |
| UI | Next.js | WebSocket client | Spectate games, view leaderboard (read-only) |

### Request Path: Full Game Lifecycle

1. **Queue**: Player → `PUT /queue/{pid}` → Matchmaker. Matchmaker stores candidate in in-memory `MatchQueue.queued` map.
2. **Match**: Background ticker → `matchPlayers()` → `collectMatches()` (lock → sort candidates → pair by ELO/game preference → unlock) → `Fleet.CreateSession()` (HTTP PUT to game server) → `publishMatch()` (lock → verify players still queued → write session to SQLite → move from `queued` to `matched`).
3. **Poll**: Player → `PUT /queue/{pid}` again → Matchmaker returns `SessionInfo` with game server URL + session ID.
4. **Connect**: Player → WebSocket `/session/{sid}/{pid}` → Game Server. WebSocket upgraded, player joined to session via inbox channel. First connection becomes P1, second becomes P2.
5. **Play**: Protocol goroutine runs select loop over inbox (moves/connections), turn timer, session deadline timer. Validates moves via `Session.MakeMove()`. Broadcasts state to all players and spectators.
6. **Complete**: Game reaches terminal state → Protocol calls `report()` → sends `resultMsg` on channel → Reporter goroutine → `PUT /results/{sid}` to Matchmaker.
7. **ELO Update**: Matchmaker receives result → SQLite transaction updates both players' ELO, wins/losses/draws.
8. **Cleanup**: Session removed from SessionManager map, players untracked from MatchQueue, ready to re-queue.

### Service Boundaries

- Matchmaker ↔ Game Server: HTTP with bearer token auth. Matchmaker creates sessions (PUT), game server reports results (PUT).
- Player ↔ Matchmaker: HTTP (queue/poll). No auth.
- Player ↔ Game Server: WebSocket (join and play). No auth on player connections.
- UI ↔ Game Server: WebSocket (spectate). Read-only.

## State Management and Persistence

### Matchmaker (SQLite)
- **Tables**: `players` (player_id, elo, wins, losses, draws), `sessions` (session_id, server, game, created_at, deadline, completed_at, cancelled, winner_id), `player_session` (junction table).
- **Mode**: WAL journal mode, `synchronous=normal`. In-memory mode forces `MaxOpenConns=1`.
- **Transactions**: `GetOrCreatePlayer`, `CreateSession`, `ReportSessionResult` all use explicit transactions with `defer tx.Rollback()`.

### Matchmaker (In-Memory)
- `MatchQueue.queued`: `map[PlayerID]*candidate` — players waiting for a match.
- `MatchQueue.matched`: `map[PlayerID]*SessionInfo` — players matched but not yet done playing.
- Protected by `sync.Mutex`.

### Game Server (In-Memory)
- `SessionManager.sessions`: `map[SessionID]sessionHandle` — active sessions.
- `SessionManager.lastEvents`: `map[SessionID]*WatchEvent` — last watch event per session (for late-joining spectators).
- `SessionManager.watchers`: `map[chan WatchEvent]struct{}` — server-level spectator channels.
- Session state lives in the Protocol goroutine's local variables (not shared).

### Player (In-Memory)
- AI state per game type (e.g., BattleshipAi tracks setup phase).
- Session manages WebSocket connection lifecycle with reconnect logic.

## Concurrency Model

### Matchmaker
- **MatchQueue.mu** (sync.Mutex): Protects `queued` and `matched` maps. Held during queue/unqueue/poll operations and during `publishMatch`. Released between `collectMatches` and `publishMatch` (Fleet.CreateSession is a network call done without the lock).
- **Background goroutines**: `StartMatcher` (ticker-driven matchmaking), `StartSessionMonitor` (ticker-driven expired session cleanup).

### Game Server
- **SessionManager.mu** (sync.Mutex): Protects `sessions` map. Held during create/join/cancel/list.
- **SessionManager.watchMu** (sync.Mutex): Protects `watchers` and `lastEvents`. Lock ordering: mu before watchMu.
- **Per-session**: Each session runs a single goroutine (`RunToCompletion`) with a select loop. All player interactions go through the `inbox` channel (capacity 2). Player connections spawn read and write goroutines that bridge WebSocket ↔ channels.
- **Reporter**: Single goroutine consuming `resultCh` channel, submitting results to matchmaker.

### Player
- **Main loop** in `Loop.Run()`: Sequential queue-then-play.
- **Session.Run()**: Goroutine managing WebSocket lifecycle with reconnect.
- **Session.bridge()**: Two goroutines — one reading from WebSocket to `protocolRx`, one writing from `protocolTx` to WebSocket.
- **Evil mode**: Additional goroutine (`runExtraConnectChaos`) periodically attempts random WebSocket connections.

### Shared Mutable State Concerns
- `MatchQueue.queued`/`matched` maps: Protected by mutex, but the window between `collectMatches()` and `publishMatch()` allows state to change (players could unqueue).
- `Fleet.servers[].retryAt`: Written from `CreateSession` and `matchPlayers`, read during candidate gathering. No explicit synchronization — relies on single-goroutine access from the matcher ticker.
- `Protocol` state: Accessed only from the single protocol goroutine — safe by design.
- `sendLatest` channel pattern: Non-blocking send with drain-and-retry can still lose messages if channel fills between drain and retry.

## Safety Guarantees (Claimed or Implied)

1. **Game rules enforced**: `CanMakeMove` checks turn order and game-over status. `MakeMove` validates move legality (bounds, cell occupancy, column fullness, ship placement). Invalid moves return errors without mutating state.
2. **ELO zero-sum**: `CalcElo` computes symmetric updates. The ELO test explicitly checks `(newWinner - winner) + (newLoser - loser) == 0`.
3. **No ELO change on cancel**: `ReportSessionResult` skips ELO update when `cancelled == true`. Panic if `cancelled && winner != uuid.Nil`.
4. **Session capacity**: `CreateSession` returns `ErrMaxSessions` if `len(sessions) >= MaxSessions`.
5. **Token authentication**: `TokenAuth` middleware on matchmaker's `/results/{sid}` and game server's `PUT /session/{sid}`.
6. **Session idempotency**: `CreateSession` is idempotent for the same session ID — if already exists and not finished, returns success. Different game kind returns error.

## Liveness Guarantees (Claimed or Implied)

1. **Turn timeout**: If a player doesn't move within `turnTimeout`, the opponent wins.
2. **Session deadline**: Games are cancelled after `deadline` passes.
3. **Expired session cleanup**: Background monitor cancels sessions past their deadline.
4. **Player reconnection**: Players can reconnect to an in-progress session (existing connection is replaced).
5. **Fleet failover**: If a game server returns 503 or has a network error, the fleet tries the next server.

## Bug History and Density

No issue tracker or bug history available. The codebase has a single initial commit. Focus on areas with complex state transitions and concurrency as likely bug-hiding spots.

## Existing Test Strategy

### Unit Tests
- `internal/elo_test.go`: ELO calculation correctness and zero-sum property.
- `internal/game/harness_test.go`: Game harness drives TicTacToe, Battleship, Connect4 to completion with AI. Verifies terminal state is valid.
- `internal/game/battleship_test.go`: AI targeting behavior (adjacent-to-hit, random fallback).
- `internal/matchmaker/db_test.go`: DB CRUD, expired session cancellation, result reporting with ELO update.
- `internal/matchmaker/fleet_test.go`: Fleet session creation, 503 handling, all-unavailable case.
- `internal/matchmaker/match_queue_test.go`: Queue behavior, ELO-based matching.
- `internal/gameserver/server_test.go`: Health check, session creation, max sessions enforcement.

### Integration Tests
- `internal/player/integration_test.go`: Two players play 10 sessions each against a real matchmaker and game server (in-process httptest servers). Tests the full lifecycle.

### What Tests Don't Cover
- **Concurrent access patterns**: No tests exercise multiple goroutines hitting the same endpoints simultaneously.
- **Network failures**: No fault injection — all tests use reliable in-process HTTP.
- **Partial failure**: No tests for game server going down mid-session, matchmaker unavailable during result reporting, etc.
- **Reconnection**: No tests for player WebSocket disconnect/reconnect during a game.
- **Evil player behavior**: The evil mode is implemented but never tested.
- **Multiple game servers**: Integration test uses a single game server.
- **Session timeout and deadline enforcement under load**: Only tested in isolation, not under concurrent traffic.

## Failure and Degradation Modes

### Matchmaker
- **Fleet: all servers unavailable**: Returns `ErrNoServersAvailable`, match is silently dropped. Players remain in queue for next cycle.
- **Fleet: temporary network error**: Sets `retryAt` for the server (1 minute grace period). Server excluded from candidates until `retryAt` passes.
- **DB errors**: Most paths log and return 500. `cancelExpiredSessions` logs and continues to next session.
- **Panic paths**: `publishMatch` panics on DB error. `matchPlayers` panics on non-`ErrNoServersAvailable` fleet errors.

### Game Server
- **Result submission failure**: Reporter re-enqueues on temporary error. Non-temporary errors are logged and dropped.
- **Session cancel**: Cancels context, cleanup runs deferred. If session already finished, returns error.
- **WebSocket close**: Player's read goroutine exits, session continues. Player can reconnect.

### Player
- **Queue error**: Logs and retries on next poll interval.
- **Dial error (temporary)**: Retries with `reconnectInterval` (1 second).
- **WebSocket error (non-close)**: Logs, reconnects.
- **AI failure**: Returns error, player loop logs and continues to next session.

### Unhandled Partial Failures
- Game server down during active sessions: Matchmaker has sessions in DB but game server has lost all state. Sessions will eventually expire via deadline monitor, but players experience hangs.
- Matchmaker down during result reporting: Game server reporter retries temporary errors but drops permanent errors. Result could be lost → session stays "active" in matchmaker DB until deadline expires it.
- Concurrent cancel from matchmaker and game-initiated completion: Both paths call `Untrack(sid)`, which is safe (idempotent delete from matched map). Both could call `ReportSessionResult`, leading to double-update of ELO.

## External Dependencies

- **SQLite** (via `go-sqlite3`): Only external storage. Single-writer model with WAL.
- **`coder/websocket`**: WebSocket library for game server and player connections.
- **`google/uuid`**: UUID generation for session/player IDs.
- No external services, message brokers, or cloud dependencies.

## Unproven Assumptions

1. **Single matcher goroutine**: The matcher ticker assumes only one goroutine calls `matchPlayers` at a time. If `matchInterval` is very short and `matchPlayers` takes longer than the interval, two iterations could overlap. (Go's `time.Ticker` will drop ticks if the receiver is slow, so this is likely safe.)
2. **Timer.Reset safety**: `Protocol.handleMove` calls `p.turnTimer.Reset(p.turnTimeout)`. Per Go docs, Reset should only be called on stopped timers or after draining the channel. The protocol's select loop should drain the channel, but there's a theoretical race.
3. **Channel capacity sufficiency**: The inbox channel has capacity 2, which could back-pressure if both players connect simultaneously. The `sendLatest` pattern assumes capacity-1 channels and can silently drop messages.
4. **Result delivery**: No persistent queue for game results. If the reporter goroutine crashes or the game server restarts, undelivered results are lost.
5. **Session-DB consistency**: The matchmaker's in-memory `matched` map and SQLite sessions table can drift if the process crashes between writing one and the other.
6. **Fleet's rand is not thread-safe**: `Fleet.rng` is `*rand.Rand` (not safe for concurrent use), but is only accessed from the matcher goroutine. If fleet methods were called from multiple goroutines, this would be a data race.

## Assumptions

- The UI is read-only and not part of the SUT for Antithesis testing.
- SQLite in-memory mode is acceptable for Antithesis (the system defaults to it).
- The "evil" player behavior is part of the workload, not the SUT.

## Open Questions

- None — system is fully contained in this repo with no external dependencies to investigate.
