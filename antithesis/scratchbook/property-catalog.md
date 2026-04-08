---
commit: 273ec77315ab2fc55bda352ca557b57b8c9908c5
updated: 2026-04-08
---

# Property Catalog — Aardvark Arena

## Priority Summary

Properties ranked by Antithesis value — how much benefit fault injection and interleaving exploration provide beyond what deterministic tests can cover.

| Priority | Properties |
|----------|-----------|
| **High** | `no-double-elo-update`, `elo-zero-sum`, `no-elo-change-on-cancel`, `leaderboard-reflects-games`, `protocol-not-blocked`, `player-recovers-after-matchmaker-restart`, `no-duplicate-match`, `player-reconnect-works`, `session-eventually-completes`, `result-reported-to-matchmaker`, `matchmaking-progress` |
| **Medium** | `session-capacity-enforced`, `expired-sessions-cancelled`, `session-cleanup-complete`, `orphaned-session-handled`, `third-player-rejected`, `player-sees-terminal-state`, `fleet-failover`, `ghost-player-sessions-cleaned-up`, `all-game-types-played`, `turn-timeout-fires`, `evil-move-rejected`, `session-deadline-fires`, `max-sessions-reached` |
| **Low** | `game-rules-enforced`, `correct-winner-detection`, `turn-order-maintained`, `board-state-valid`, `spectator-receives-valid-state`, `draw-outcome-reached` |

**High**: Timing-sensitive races, concurrent access to shared state, network failure recovery, known code deficiencies. These are the properties where Antithesis provides the most value.

**Medium**: Properties exercising important code paths that benefit from fault injection but have simpler concurrency models or are partially covered by existing tests.

**Low**: Single-goroutine properties kept as defense-in-depth. Their primary Antithesis value is catching unexpected state corruption, not concurrency bugs.

## Game Rule Integrity

Properties ensuring game logic is correct under all conditions. Note: these properties operate on a single-goroutine protocol, so their primary Antithesis value is catching serialization or state corruption bugs rather than concurrency issues. Keep as lightweight defense-in-depth assertions.

### game-rules-enforced — Game Rules Never Violated

| | |
|---|---|
| **Type** | Safety |
| **Property** | A game session never accepts an invalid move that mutates board state. |
| **Invariant** | `Always`: After every `MakeMove` call that returns `nil` error, the resulting board state is legal for that game type (e.g., TTT has at most 1 more X than O, Connect4 pieces obey gravity, Battleship ships don't overlap). Assertion placed SUT-side in each game's `MakeMove` implementation. |
| **Antithesis Angle** | Evil players send malformed, out-of-bounds, and out-of-turn moves under various timing conditions. Antithesis explores interleavings where malicious and valid moves arrive nearly simultaneously. |
| **Why It Matters** | Invalid board states break win detection and spectator rendering. The evil player mode exercises these paths. |

### correct-winner-detection — Winner Matches Board State

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a game transitions to P1Win or P2Win, the board state actually contains a winning configuration for that player. |
| **Invariant** | `Always`: At each terminal state, verify the declared winner has a winning position (3-in-a-row for TTT, 4-in-a-row for Connect4, all opponent ships sunk for Battleship). SUT-side assertion in Protocol.report(). Note: Battleship winner verification requires SUT-side assertion because private `shipCells` state is not visible to workload clients. |
| **Antithesis Angle** | Fault injection during the move-validate-broadcast sequence could expose timing where status is set before the board is fully updated, or vice versa. |
| **Why It Matters** | A wrong winner invalidates the entire ELO system downstream. |

### turn-order-maintained — Turn Order Never Violated

| | |
|---|---|
| **Type** | Safety |
| **Property** | A move is only applied when it comes from the current player. |
| **Invariant** | `Always`: Before applying a move, `state.CurrentPlayer == player`. Assertion in `CanMakeMove`. |
| **Antithesis Angle** | Evil players send out-of-turn moves. Antithesis explores interleavings where two moves from different players arrive close together on the inbox channel. |
| **Why It Matters** | Turn order violation would produce illegal board states. |

### board-state-valid — Board Configuration Always Legal

| | |
|---|---|
| **Type** | Safety |
| **Property** | The board never enters an impossible configuration (e.g., TTT with more than one extra piece for either player, Connect4 with floating pieces). |
| **Invariant** | `Always`: After every state broadcast, validate the board against game-specific structural invariants. Workload-side assertion — deserialize state from WebSocket message and check. |
| **Antithesis Angle** | State corruption bugs typically manifest under specific timing — Antithesis systematically explores move arrival ordering and concurrent reconnections. |
| **Why It Matters** | Structural board corruption is a class of bug that causes silent downstream failures. |

## ELO and Scoring Integrity

Properties ensuring the rating system behaves correctly.

### elo-zero-sum — ELO Changes Are Zero-Sum

| | |
|---|---|
| **Type** | Safety |
| **Property** | After a non-cancelled game result is processed, the sum of ELO changes across both players is zero. |
| **Invariant** | `Always`: In `ReportSessionResult`, after computing new ELOs, assert `(newPlayer - oldPlayer) + (newOpponent - oldOpponent) == 0`. SUT-side assertion. |
| **Antithesis Angle** | Concurrent result submissions, retries from the Reporter, and session expiry racing with game completion could cause double-updates or partial writes. |
| **Why It Matters** | ELO inflation/deflation corrupts the leaderboard. The CalcElo function is correct in isolation (tested), but the DB transaction path under concurrency hasn't been tested. |

### no-elo-change-on-cancel — Cancelled Games Don't Affect ELO

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a session is cancelled, neither player's ELO, wins, losses, or draws change. |
| **Invariant** | `Always`: In `ReportSessionResult` when `cancelled == true`, assert no player stats are modified. SUT-side assertion. |
| **Antithesis Angle** | Cancellation can race with normal game completion. The session monitor cancels expired sessions, but the game server may simultaneously report a normal result. Antithesis explores this overlap. |
| **Why It Matters** | The existing `log.Panicf("BUG: received cancelled result with a winner set")` guard suggests the developers know this is a risk area. |

### no-double-elo-update — Session Result Processed Exactly Once

| | |
|---|---|
| **Type** | Safety |
| **Property** | Each session's ELO update is applied exactly once — not zero times, not twice. |
| **Invariant** | `Always`: Track session IDs that have been processed. On each `ReportSessionResult` call, assert the session hasn't been processed before. SUT-side assertion in the DB layer. |
| **Antithesis Angle** | The Reporter re-enqueues on temporary error. Matchmaker's cancel-session path also calls `ReportSessionResult`. Session monitor also calls it. All three paths could deliver the same session result. |
| **Why It Matters** | Double ELO updates corrupt ratings. The current code has no idempotency guard on ReportSessionResult — the `completed_at` check is implicit (UPDATE succeeds even if already completed). Note: This property will likely fail immediately due to the missing idempotency guard. This is intentional — validates that Antithesis can find this class of bug. |

## Session Lifecycle

Properties ensuring sessions are created, run, and cleaned up correctly.

### session-capacity-enforced — Game Server Respects Max Sessions

| | |
|---|---|
| **Type** | Safety |
| **Property** | The number of active sessions on a game server never exceeds `MaxSessions`. |
| **Invariant** | `Always`: After every session creation, `len(sessions) <= MaxSessions`. SUT-side assertion in `SessionManager.CreateSession`. |
| **Antithesis Angle** | Concurrent session creation requests could race past the capacity check. The mutex should prevent this, but Antithesis explores whether there's a TOCTOU window. |
| **Why It Matters** | Exceeding capacity could cause resource exhaustion and degrade all active games. |

### session-eventually-completes — Every Session Reaches Terminal State

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Every created session eventually reaches a terminal state (win, loss, draw, or cancel). |
| **Invariant** | `Sometimes(session completed)`: After each session creation, eventually observe a terminal status. Workload-side assertion — players track whether each session they joined reached a terminal state. |
| **Antithesis Angle** | Player disconnection, game server restarts, network partitions between player and game server — all could leave sessions hanging. The deadline timer and session monitor are the safety nets. |
| **Why It Matters** | Stuck sessions consume capacity and leave players unable to re-queue. |

### expired-sessions-cancelled — Deadline Enforcement Works

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Sessions that exceed their deadline are eventually cancelled by the session monitor. |
| **Invariant** | `Sometimes(expired session cancelled)`: The session monitor's `cancelExpiredSessions` function finds and cancels an expired session. SUT-side assertion — emit when an expired session is successfully cancelled. |
| **Antithesis Angle** | The session monitor runs on a ticker. If the matchmaker is restarted or the ticker goroutine is delayed, expired sessions could accumulate. Antithesis can pause the monitor goroutine and then release it. |
| **Why It Matters** | Without deadline enforcement, stuck sessions would permanently consume capacity. |

### result-reported-to-matchmaker — Game Results Reach the Matchmaker

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Every session that completes on the game server has its result eventually delivered to the matchmaker. |
| **Invariant** | `Sometimes(result delivered)`: The Reporter successfully submits a result (HTTP 200 from matchmaker). SUT-side assertion in Reporter.submitResult. |
| **Antithesis Angle** | Network partition between game server and matchmaker during result submission. The Reporter retries temporary errors but drops permanent ones. Antithesis explores whether retries eventually succeed or results are permanently lost. |
| **Why It Matters** | Lost results mean matchmaker sessions stay "active" forever (until deadline expires them as cancelled), corrupting player stats. |

### session-cleanup-complete — Finished Sessions Are Removed

| | |
|---|---|
| **Type** | Safety |
| **Property** | After a session finishes, it is removed from the SessionManager's active sessions map. |
| **Invariant** | `Always`: After the cleanup callback runs, the session ID is no longer in the sessions map. SUT-side assertion in the cleanup function. |
| **Antithesis Angle** | Concurrent operations (new session creation, JoinSession) could interfere with cleanup. Antithesis explores whether the mutex correctly serializes these. |
| **Why It Matters** | Leaked sessions reduce effective capacity and could serve stale data to spectators. |

## Matchmaking

Properties ensuring the queue and matching system works correctly.

### no-duplicate-match — Players Matched to Exactly One Session

| | |
|---|---|
| **Type** | Safety |
| **Property** | A player never appears in two concurrent active sessions. |
| **Invariant** | `Always`: When moving a player from `queued` to `matched`, assert they're not already in `matched`. SUT-side assertion in `publishMatch`. |
| **Antithesis Angle** | The window between `collectMatches` (releases lock) and `publishMatch` (re-acquires) is a race opportunity. Two concurrent `matchPlayers` invocations could both pair the same player. While the ticker design should prevent this, Antithesis explores whether timer coalescing or system pauses create the overlap. |
| **Why It Matters** | A double-matched player would be expected in two game sessions simultaneously, causing one session to hang waiting for them. |

### matchmaking-progress — Queued Players Eventually Get Matched

| | |
|---|---|
| **Type** | Liveness |
| **Property** | When two or more compatible players are in the queue, they are eventually matched. |
| **Invariant** | `Sometimes(match created)`: After queueing, a player eventually receives a SessionInfo. Workload-side assertion — players track how long they wait. |
| **Antithesis Angle** | ELO difference relaxation over time means even mismatched players should eventually pair. But if the matcher goroutine is delayed (Antithesis can pause it), or fleet errors prevent session creation, queued players could wait indefinitely. |
| **Why It Matters** | Players stuck in queue forever is the most visible user-facing failure. |

### orphaned-session-handled — Race-Created Sessions Don't Leak

| | |
|---|---|
| **Type** | Liveness |
| **Property** | If `collectMatches` pairs players who then unqueue before `publishMatch`, the orphaned game server session is eventually cleaned up by the deadline timeout. |
| **Invariant** | `Sometimes(orphaned session expired)`: A session that was created on the game server but never had both players connect is eventually cancelled. SUT-side `Reachable` in `cancelExpiredSessions` when cancelling a session with no result. |
| **Antithesis Angle** | This race is difficult to trigger deterministically — it requires a player to unqueue during the network round-trip to the game server. Antithesis can systematically explore this timing. |
| **Why It Matters** | Orphaned sessions consume game server capacity until they expire. |

## Player Connection

Properties ensuring WebSocket connections and reconnections work correctly.

### player-reconnect-works — Reconnected Players Resume Game

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a player disconnects and reconnects to an active session, they receive the current game state and can continue playing. |
| **Invariant** | `Always`: After a reconnection (replacing an existing player connection), the new connection receives a state message with the current board. Workload-side assertion — reconnecting player checks it receives valid state. |
| **Antithesis Angle** | Reconnection races with move processing, other player's connection, or session completion. Antithesis explores whether the old connection's write goroutine interferes with the new one. |
| **Why It Matters** | WebSocket connections are inherently fragile. If reconnection fails, the game hangs until turn timeout. |

### third-player-rejected — Excess Connections Are Rejected

| | |
|---|---|
| **Type** | Safety |
| **Property** | A session with two connected players rejects additional connection attempts from unknown player IDs with an error message. |
| **Invariant** | `Always`: When `handleConn` is called with a pid not in `p.players` and `len(p.players) == 2`, the connection receives an error and is not added to the players map. SUT-side assertion. |
| **Antithesis Angle** | Evil players' `runExtraConnectChaos` sends random-ID connections. Under heavy load with many concurrent sessions, these probes could arrive at unexpected times. |
| **Why It Matters** | A third player corrupting a session would break the 2-player game model. |

## State Consistency

Properties ensuring all observers see consistent state.

### ~~spectator-state-consistency~~ — SUPERSEDED by `spectator-receives-valid-state`

This property has been superseded by `spectator-receives-valid-state` (in System Health and Recovery section), which places the validation SUT-side in `BroadcastState` instead of requiring a workload spectator client. The SUT-side approach validates state before it reaches any observer (players and spectators alike), providing strictly stronger coverage without workload dependencies.

### player-sees-terminal-state — Players Observe Game Completion

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a game reaches a terminal state, connected players receive the terminal state message before the WebSocket is closed. |
| **Invariant** | `AlwaysOrUnreachable`: When a player is connected at game end, they receive a terminal state message. Workload-side assertion — players check that they received a terminal state when the game completes normally. |
| **Antithesis Angle** | The `sendLatest` pattern could drop the terminal state if the channel is full. Network disruption right as the game ends could prevent delivery. Antithesis explores these timing windows. |
| **Why It Matters** | Players that never see the terminal state can't properly complete their game loop. |

## Fleet and Availability

Properties ensuring the system handles game server availability correctly.

### fleet-failover — Fleet Uses Available Servers

| | |
|---|---|
| **Type** | Safety |
| **Property** | When creating a session, the fleet skips servers that returned 503 or had network errors within the retry window, and uses available servers. |
| **Invariant** | `AlwaysOrUnreachable`: When Fleet.CreateSession succeeds, the chosen server was a valid candidate (not in retry backoff). SUT-side assertion. |
| **Antithesis Angle** | Antithesis can partition the game server, causing the fleet to enter retry-backoff state, then heal the partition. The fleet should resume using the server after `retryAt` passes. |
| **Why It Matters** | Incorrect failover logic could route sessions to unavailable servers, causing them to fail immediately. |

## Reachability

Properties guiding Antithesis to explore interesting code paths.

### all-game-types-played — All Three Game Types Are Exercised

| | |
|---|---|
| **Type** | Reachability |
| **Property** | TicTacToe, Connect4, and Battleship games are all played during a test run. |
| **Invariant** | `Reachable`: One assertion per game type, placed where a session is created for that game kind. SUT-side in `SessionManager.CreateSession`. |
| **Antithesis Angle** | Ensures the workload's random game selection actually covers all game types. Without this, Antithesis might get stuck in a local optimum testing only one game type. |
| **Why It Matters** | Each game has different code paths (especially Battleship with its setup phase). Testing all three is necessary for coverage. |

### draw-outcome-reached — Draw Outcomes Are Reachable

| | |
|---|---|
| **Type** | Reachability |
| **Property** | At least one TicTacToe or Connect4 game ends in a draw. Battleship cannot draw by game rules (game ends only when all opponent ships are sunk). |
| **Invariant** | `Reachable`: Assertion fires when `state.Status == Draw` at game completion. SUT-side in Protocol.report(). |
| **Antithesis Angle** | Draws require specific board configurations. The AIs' strategies may rarely produce draws. This assertion ensures Antithesis explores paths leading to draws. |
| **Why It Matters** | Draw handling is a distinct code path (no winner, both players update ELO symmetrically). |

### turn-timeout-fires — Turn Timer Triggers Game End

| | |
|---|---|
| **Type** | Reachability |
| **Property** | The turn timeout fires at least once, ending a game because a player was too slow. |
| **Invariant** | `Reachable`: Assertion in the `<-p.turnTimer.C` case of the protocol's select loop. SUT-side. |
| **Antithesis Angle** | Antithesis can pause player processes, causing turn timeouts to fire. This exercises the timeout→opponent-wins path. |
| **Why It Matters** | The turn timeout is a critical liveness mechanism. If it never fires in testing, the timeout handling code is untested. |

### evil-move-rejected — Malicious Moves Are Handled Gracefully

| | |
|---|---|
| **Type** | Reachability |
| **Property** | An evil player's malformed or invalid move is rejected with an error, and the game continues normally. |
| **Invariant** | `Reachable`: Assertion fires when `handleMove` returns an error to the player. SUT-side in Protocol.handleMove(). |
| **Antithesis Angle** | Evil players send malformed JSON, out-of-bounds moves, and out-of-turn moves. Antithesis explores whether these are always handled gracefully or whether specific timing causes them to corrupt state. |
| **Why It Matters** | Error handling for invalid input is a common source of bugs, especially under concurrent access. |

### session-deadline-fires — Session Deadline Cancels Game

| | |
|---|---|
| **Type** | Reachability |
| **Property** | The session deadline timer fires at least once, cancelling a game. |
| **Invariant** | `Reachable`: Assertion in the `<-timer.C` case of the protocol's select loop. SUT-side. |
| **Antithesis Angle** | With shortened timeouts in the Antithesis deployment, deadline cancellation is more likely. Antithesis can also delay player connections to trigger deadlines. |
| **Why It Matters** | The deadline timer is the last-resort cleanup mechanism. If it never fires, that code path is untested. |

### max-sessions-reached — Game Server Hits Capacity Limit

| | |
|---|---|
| **Type** | Reachability |
| **Property** | The game server reaches its maximum session count and returns 503. |
| **Invariant** | `Reachable`: Assertion fires when `ErrMaxSessions` is returned. SUT-side in `SessionManager.CreateSession`. |
| **Antithesis Angle** | With a bounded MaxSessions and enough concurrent players, the game server should eventually fill up. Antithesis can slow down game completion to increase the chance of hitting capacity. |
| **Why It Matters** | The 503 + Retry-After response path is the backpressure mechanism for the entire system. |

## System Health and Recovery

Properties ensuring the system recovers from failures and maintains global consistency.

### leaderboard-reflects-games — Global ELO Conservation

| | |
|---|---|
| **Type** | Safety |
| **Property** | The sum of all player ELO ratings equals `count(players) * DefaultElo` (1000). Any deviation indicates ELO inflation or deflation from double-updates, lost results, or other corruption. |
| **Invariant** | `Always`: Periodically query the leaderboard and verify `sum(elo) == count(players) * 1000`. Workload-side assertion in the `eventually_check_leaderboard` test command. |
| **Antithesis Angle** | Per-transaction zero-sum (elo-zero-sum) doesn't catch cumulative drift from double-updates across different transactions. This global check catches drift regardless of cause. |
| **Why It Matters** | The strongest end-to-end ELO integrity check. Subsumes per-transaction checks for detecting real corruption. |

### player-recovers-after-matchmaker-restart — Matchmaker Restart Recovery

| | |
|---|---|
| **Type** | Liveness |
| **Property** | After the matchmaker process restarts, players that were mid-queue or mid-match can eventually re-queue and play new games. |
| **Invariant** | `Sometimes(post-restart game completed)`: After a matchmaker restart, at least one game completes successfully. Workload-side assertion — players track whether they complete a game after observing a queue error (indicating restart). |
| **Antithesis Angle** | Matchmaker restart loses in-memory `queued`/`matched` maps while SQLite persists session records. Players in `matched` at crash time have active DB sessions but no in-memory tracking. Antithesis can kill the matchmaker process at various points in the lifecycle. |
| **Why It Matters** | Process restarts are the most common failure mode. If the system can't recover, the entire platform is unavailable. |

### protocol-not-blocked — Protocol Goroutine Remains Responsive

| | |
|---|---|
| **Type** | Safety |
| **Property** | The protocol goroutine's select loop continues to process timer events even when result delivery is delayed. |
| **Invariant** | `Always`: The turn timeout and session deadline timers fire within a bounded time of their scheduled expiry. SUT-side assertion — track time between timer fire and processing. |
| **Antithesis Angle** | When `resultCh` fills (matchmaker unreachable while games complete), `Protocol.report()` blocks on `p.result <- resultMsg{...}`, stalling the entire select loop. Turn timers and deadline timers fire but aren't processed. Antithesis can partition the matchmaker to trigger this. |
| **Why It Matters** | A blocked protocol goroutine makes games appear hung — no moves processed, no timeouts enforced, no cleanup. This is a real liveness bug. |

### ghost-player-sessions-cleaned-up — Abandoned Queue Entries Don't Leak

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Sessions created for ghost queue entries (from evil player's `QueueAbandonRate`) are eventually cancelled via the deadline timeout. |
| **Invariant** | `Sometimes(ghost session expired)`: A session where one player never connects is eventually cancelled. SUT-side assertion in `cancelExpiredSessions` or the protocol's deadline path when `len(p.players) < 2`. |
| **Antithesis Angle** | Ghost queue entries are matched to real players, creating sessions where one player connects normally but the ghost never does. The session hangs until the turn timer gives the connected player a win, or the session deadline cancels it. |
| **Why It Matters** | Ghost entries consume matchmaking resources and create frustrating experiences for real players who get matched against no-shows. |

### spectator-receives-valid-state — Broadcast State Is Structurally Valid

| | |
|---|---|
| **Type** | Safety |
| **Property** | Every state broadcast via `BroadcastState` is structurally valid for its game type. |
| **Invariant** | `Always`: In `BroadcastState`, validate the serialized state against game-specific structural invariants before sending. SUT-side assertion — avoids dependency on a spectator workload client. |
| **Antithesis Angle** | Replaces the workload-side `spectator-state-consistency` with a SUT-side check that doesn't require a spectator client in the workload. Catches serialization bugs, partial state updates, or any mutation that bypasses validation. |
| **Why It Matters** | All observers (players and spectators) receive state through this path. A single invalid broadcast corrupts every connected client's view. |
