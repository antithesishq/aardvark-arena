# Coverage Balance Evaluation

## Methodology

This evaluation maps every high-risk area identified in the SUT analysis against the property catalog, then inspects low-risk areas for over-investment. It also checks whether the deployment topology creates blind spots the catalog should address but does not. Property types (safety, liveness, reachability) are evaluated for balance within each risk area.

---

## Risk Area Coverage Matrix

### 1. Game Rule Integrity (Medium Risk)

**SUT risk**: Move validation across three game types, especially Battleship's complex setup phase. Single-goroutine Protocol design makes concurrent mutation unlikely; the main risk is validation logic bugs.

**Catalog properties**: `game-rules-enforced`, `correct-winner-detection`, `turn-order-maintained`, `board-state-valid` (4 properties).

**Assessment**: Well-covered. Both SUT-side (validation) and workload-side (deserialized state checking) approaches are represented. The four properties cover the input validation path, the state mutation path, win detection, and structural invariants.

**Property type balance**: 4 safety, 0 liveness, 0 reachability. Adequate -- game rules are inherently safety properties. The `evil-move-rejected` reachability property in the Reachability section ensures the validation paths are actually exercised.

**Verdict**: **PASS** -- proportional investment for a medium-risk area.

---

### 2. ELO and Scoring Integrity (High Risk)

**SUT risk**: The `ReportSessionResult` function has no idempotency guard. Three distinct call paths (Reporter, session monitor, cancel endpoint) can all call it for the same session. The race between normal completion and deadline-based cancellation is explicitly called out in the SUT analysis ("Concurrent cancel from matchmaker and game-initiated completion... leading to double-update of ELO"). This is the highest-risk concurrency bug in the system.

**Catalog properties**: `elo-zero-sum`, `no-elo-change-on-cancel`, `no-double-elo-update` (3 properties).

**Assessment**: Good coverage of the three core ELO correctness dimensions. The `no-double-elo-update` property directly targets the idempotency gap. The evidence files correctly identify that the `updateSessionResult` SQL has no `WHERE completed_at IS NULL` guard.

**Gap**: No property checks **global ELO conservation** across the entire system over time. The existing `elo-zero-sum` checks a single transaction's deltas, but does not detect drift from double-updates across transactions. A property like "sum of all player ELOs == N * DefaultElo" (checked periodically via the leaderboard endpoint) would catch cumulative corruption that per-transaction checks miss.

**Property type balance**: 3 safety, 0 liveness, 0 reachability. The `draw-outcome-reached` reachability property indirectly exercises the draw ELO path, which is good. However, there is no reachability assertion confirming that the Reporter retry path is actually exercised, or that the cancel-races-completion scenario is reached.

**Verdict**: **GAP** -- missing a global/aggregate ELO conservation property.

---

### 3. Session Lifecycle (High Risk)

**SUT risk**: Sessions must be created, run, completed, reported, and cleaned up. Failures at any stage leak capacity or lose results. The SUT analysis identifies multiple partial-failure scenarios: game server down during active sessions, matchmaker down during result reporting, concurrent cancel and completion.

**Catalog properties**: `session-capacity-enforced`, `session-eventually-completes`, `expired-sessions-cancelled`, `result-reported-to-matchmaker`, `session-cleanup-complete` (5 properties).

**Assessment**: Strong coverage of the session lifecycle stages. The five properties form a logical chain: capacity gating, liveness/completion, deadline enforcement, result delivery, and resource cleanup.

**Gap**: No property verifying **session-DB consistency** -- i.e., that the matchmaker's in-memory `matched` map and the SQLite `sessions` table agree. The SUT analysis explicitly identifies this as unproven assumption #5: "The matchmaker's in-memory `matched` map and SQLite sessions table can drift if the process crashes between writing one and the other." Since the deployment uses file-backed SQLite (survives process restarts) but the in-memory `matched` map does not survive restarts, a matchmaker restart creates immediate inconsistency. Players in `matched` are lost from memory but their sessions remain in the DB. These players cannot re-queue until the session expires in the DB and `Untrack` is called by the session monitor, but since they are no longer in `matched`, the monitor's `Untrack` call is a no-op. The players are effectively stuck.

**Property type balance**: 2 safety, 3 liveness, 0 reachability. The `session-deadline-fires` and `max-sessions-reached` reachability properties in the Reachability section cover key paths. This is well balanced.

**Verdict**: **GAP** -- missing a property for in-memory/DB state consistency after matchmaker restart.

---

### 4. Matchmaking (High Risk)

**SUT risk**: The `collectMatches` to `publishMatch` race window is the primary concurrency concern. The lock is released between collecting matches and publishing them, allowing state to change. The SUT analysis also identifies the `Fleet.rng` thread-safety assumption and the `Fleet.servers[].retryAt` synchronization gap.

**Catalog properties**: `no-duplicate-match`, `matchmaking-progress`, `orphaned-session-handled` (3 properties).

**Assessment**: The three properties cover the main matchmaking risks well. `no-duplicate-match` targets the race window, `matchmaking-progress` is the key liveness property, and `orphaned-session-handled` covers the consequence of the race (leaked game server sessions).

**Gap**: No property verifying that **a player in the `matched` map always corresponds to a valid, active session in the DB**. If the `publishMatch` DB write succeeds but the in-memory state update fails (e.g., panic between `db.CreateSession` and the map updates), the player would not be in `matched` but would have a session in the DB. This is related to the session-DB consistency gap above.

**Property type balance**: 1 safety, 2 liveness, 0 reachability. Reasonable -- the matchmaking risks are about both correctness and progress.

**Verdict**: **PASS** with one caveat (the in-memory/DB consistency gap spans both matchmaking and session lifecycle).

---

### 5. Player Connection and Reconnection (High Risk)

**SUT risk**: WebSocket connections are fragile. The SUT analysis highlights reconnection as untested (no existing tests exercise it), and the evidence file identifies the old-write-goroutine race when a player reconnects. Evil player chaos connections add another dimension.

**Catalog properties**: `player-reconnect-works`, `third-player-rejected` (2 properties).

**Assessment**: The two properties cover the reconnection and excess-connection scenarios. However, this area is under-invested relative to its risk level.

**Gap 1**: No property for **connection ordering** -- specifically, the case where P2 connects before P1. The SUT analysis describes "First connection becomes P1, second becomes P2" (section 4 of the request path). If both players connect simultaneously, the assignment depends on inbox channel ordering. The catalog does not verify that player-role assignment is stable or correct.

**Gap 2**: No property for **WebSocket close behavior** during active gameplay. The SUT analysis notes "WebSocket close: Player's read goroutine exits, session continues." There is no property checking that a mid-game disconnect by one player leads to the expected outcome (opponent wins via turn timeout or player reconnects). The `session-eventually-completes` property is related but does not specifically tie disconnection to the turn-timeout path.

**Gap 3**: The `player-reconnect-works` evidence file identifies a subtle bug: "close(existing.conn) triggers the old write goroutine to exit... The old goroutine will finish its write and then close the websocket with StatusNormalClosure. This could cause the old websocket to send a close frame to the player." No property specifically checks that the player correctly handles this server-initiated close on the old connection and does not confuse it with a session-ended signal.

**Property type balance**: 1 safety, 0 liveness, 0 reachability. This is lopsided. There should be a reachability property confirming that reconnection actually happens during testing (the evidence file even notes this as missing). The `player-reconnect-works` property is safety but has no companion `Sometimes` assertion confirming reconnections are exercised.

**Verdict**: **GAP** -- under-invested for a high-risk area. Missing reconnection reachability, disconnect-during-game behavior, and connection ordering.

---

### 6. State Consistency / Spectator (Low-Medium Risk)

**SUT risk**: Spectators are read-only. The `sendLatest` pattern intentionally drops intermediate states. The risk is receiving invalid states, not missing states.

**Catalog properties**: `spectator-state-consistency`, `player-sees-terminal-state` (2 properties).

**Assessment**: Appropriate investment. The `spectator-state-consistency` property checks structural validity of received states. The `player-sees-terminal-state` ensures the final state is delivered.

**Note**: The deployment topology does not include a spectator workload container or a spectator test command. The `spectator-state-consistency` property says it is workload-side ("a spectator client validates each received state"), but no spectator client exists in the test template commands. This property cannot be implemented as described without adding a spectator to the workload.

**Verdict**: **GAP** -- the `spectator-state-consistency` property requires a spectator workload component that is not present in the deployment topology.

---

### 7. Fleet and Availability (Medium Risk)

**SUT risk**: Fleet failover logic, 503 handling, retry-after parsing, all-servers-unavailable case. The deployment uses a single game server, so fleet failover is exercised only when Antithesis partitions or restarts the game server.

**Catalog properties**: `fleet-failover` (1 property).

**Assessment**: Minimal but adequate given the single-game-server deployment. The property covers the core safety invariant (don't route to unavailable servers). The `max-sessions-reached` reachability property ensures the 503 path is exercised.

**Gap**: No property for the **all-servers-unavailable** scenario. When all servers are in retry backoff, `ErrNoServersAvailable` is returned and the match is silently dropped. The SUT analysis identifies this as a failure mode, and the evidence for `matchmaking-progress` notes it as a stall condition. However, no property specifically checks that this state is transient (i.e., that servers eventually leave retry backoff and matches resume). The `matchmaking-progress` liveness property partially covers this, but only indirectly.

**Verdict**: **PASS** -- proportional to the single-server deployment. The indirect coverage from `matchmaking-progress` is sufficient.

---

### 8. Panic Paths (Medium-High Risk)

**SUT risk**: The codebase has 7 `log.Panicf` calls in production code: in `publishMatch` (DB error), `matchPlayers` (fleet error), `Protocol.marshal` (marshal error), and `Reporter` (encode/request creation errors). A panic in the matchmaker or game server process would crash the process. With the file-backed SQLite, the matchmaker can restart with DB state intact but in-memory state lost.

**Catalog properties**: None directly targeting panic behavior.

**Assessment**: No property monitors whether panics occur or whether the system recovers from them. Antithesis can trigger panics by injecting faults that cause DB errors, fleet errors, or marshal failures. When these occur, the process crashes and restarts. The catalog has no property that detects this happened or verifies recovery.

**Gap**: A safety property like "matchmaker and game server processes remain healthy" (heartbeat-style, or tracking the health endpoints) would detect crash-restart cycles. The `anytime_check_health` test command in the deployment partially covers this, but it is not represented as a property in the catalog.

**Verdict**: **GAP** -- no property covers process crash/restart behavior.

---

### 9. Token Authentication (Low Risk)

**SUT risk**: Bearer token auth protects matchmaker's `/results/{sid}` and game server's `PUT /session/{sid}`. If the token is nil, auth is skipped. Player connections have no auth.

**Catalog properties**: None.

**Assessment**: Appropriate -- auth is a low-risk area for an Antithesis test. The token is a shared secret configured at startup. There is no dynamic auth, no token rotation, and no user-facing auth. Antithesis fault injection does not naturally exercise auth bypass. No property needed.

**Verdict**: **PASS** -- correctly omitted.

---

### 10. Leaderboard Correctness (Low-Medium Risk)

**SUT risk**: The leaderboard is a read-only SQL query (`SELECT ... ORDER BY elo DESC`). Its correctness depends entirely on the ELO update path being correct.

**Catalog properties**: None directly. The `eventually_check_leaderboard` test command verifies leaderboard entries exist and ELO values are reasonable.

**Assessment**: The leaderboard is covered transitively by the ELO properties and the test command. However, the test command is not reflected in the property catalog. If the leaderboard check fails, it produces a test failure but not a named Antithesis property violation.

**Gap**: The `eventually_check_leaderboard` test command should correspond to a catalog property so that leaderboard correctness is tracked as a named assertion.

**Verdict**: **MINOR GAP** -- the test command exists but is not reflected in the catalog as a property.

---

## Deployment Topology Blind Spots

### Single Game Server

The deployment uses one game server. This means:
- Fleet **multi-server selection** logic (shuffling candidates, trying the next server after 503) is never exercised with actual multiple servers. It is only exercised as failover-to-no-servers when the single server is unavailable.
- The catalog's `fleet-failover` property is `AlwaysOrUnreachable`, which correctly accounts for this -- if the failover path is never reached, the assertion passes vacuously.
- No property checks that session creation is **load-balanced** across multiple servers, but this is appropriate given the deployment.

### No Spectator Workload

As noted in finding #6, the deployment has no spectator client. The `spectator-state-consistency` property is workload-side and requires a spectator to function. Either the property needs to be reformulated as SUT-side (assertions in `BroadcastState`), or a spectator must be added to the workload.

### Matchmaker Restart Recovery

The deployment uses file-backed SQLite, meaning the matchmaker can survive process restarts with DB state intact. But in-memory state (`queued` and `matched` maps) is lost. No property verifies that the system recovers correctly after a matchmaker restart -- specifically, that players who were in `matched` can eventually re-queue and play again.

---

## Property Type Distribution

| Type | Count | Percentage |
|------|-------|------------|
| Safety | 13 | 50% |
| Liveness | 7 | 27% |
| Reachability | 6 | 23% |

This distribution is healthy. Antithesis is strongest at finding safety violations through systematic exploration, and the catalog correctly weights safety properties. The liveness properties cover the key progress guarantees (matchmaking, session completion, result delivery, deadline enforcement). The reachability properties ensure critical code paths are exercised.

---

## Findings

### Finding 1: No Global ELO Conservation Property

- **Properties affected**: `elo-zero-sum`, `no-elo-change-on-cancel`, `no-double-elo-update`
- **Concern**: Per-transaction ELO checks cannot detect cumulative drift from double-updates across different transactions. If session result A is processed twice (once normally, once via retry), each individual call satisfies `elo-zero-sum`, but the aggregate effect is non-zero-sum. A global conservation invariant ("sum of all ELOs == N * 1500") would catch this.
- **Scope**: Catalog-wide
- **Evidence**: `no-double-elo-update` evidence file states: "the `updateSessionResult` SQL has no `WHERE completed_at IS NULL` guard -- it will succeed even if the session is already completed." This confirms double-processing is possible. `elo-zero-sum` only checks within a single `ReportSessionResult` call.
- **Suggested action**: Add a safety property `elo-global-conservation` that periodically queries the leaderboard and verifies `sum(elo) == count(players) * DefaultElo`. This can be implemented as a workload-side property in the `eventually_check_leaderboard` test command.

### Finding 2: No Property for In-Memory/DB State Consistency After Restart

- **Properties affected**: `session-eventually-completes`, `matchmaking-progress`
- **Concern**: When the matchmaker process restarts, the in-memory `queued` and `matched` maps are lost, but the SQLite `sessions` table persists. Players who were in the `matched` map are now in limbo -- they have active sessions in the DB but no in-memory tracking. These players cannot re-queue (their session is still "active" in DB), and the `Untrack` call from the session monitor when the session eventually expires operates on the `matched` map which no longer contains them. The consequence: players might be unable to rejoin the game flow after a matchmaker restart.
- **Scope**: Catalog-wide
- **Evidence**: SUT analysis unproven assumption #5: "The matchmaker's in-memory `matched` map and SQLite sessions table can drift if the process crashes between writing one and the other." Deployment topology: "Use file-backed SQLite... so state survives process restarts within a timeline."
- **Suggested action**: Add a liveness property `player-recovers-after-restart` that verifies players who were active before a matchmaker restart eventually resume playing. Alternatively, add a safety property verifying that the matchmaker reconciles in-memory state with DB state on startup.

### Finding 3: Spectator Property Requires Missing Workload Component

- **Property affected**: `spectator-state-consistency`
- **Concern**: The property is defined as workload-side ("a spectator client validates each received state"), but the deployment topology has no spectator workload. The test template commands are: `first_start_swarm` (players), `parallel_driver_evil_player` (evil player), `eventually_check_leaderboard`, and `anytime_check_health`. None of these are spectator clients.
- **Scope**: Property-specific
- **Evidence**: Deployment topology test template section lists four commands, none of which involve spectating. Property catalog states the invariant uses "workload-side assertion -- a spectator client validates each received state."
- **Suggested action**: Either (a) add a spectator test command to the workload that connects to the game server's watch endpoint and validates received states, or (b) reformulate the property as SUT-side by placing assertions in `BroadcastState` and `emitWatchEvent` within the game server. Option (b) is simpler and still provides coverage since the serialization is the same for players and spectators.

### Finding 4: Player Reconnection Under-Invested

- **Properties affected**: `player-reconnect-works`
- **Concern**: Reconnection is identified as untested in the SUT analysis ("No tests for player WebSocket disconnect/reconnect during a game"), yet the catalog has only one safety property with no companion reachability assertion. The evidence file for `player-reconnect-works` explicitly calls out a missing `Reachable` assertion on the reconnection path. Without a reachability property, there is no guarantee Antithesis exercises the reconnection code, meaning the safety property could pass vacuously.
- **Scope**: Property-specific
- **Evidence**: `player-reconnect-works` evidence file: "Missing: `Reachable` assertion on the reconnection path (existing player replaced)." SUT analysis "What Tests Don't Cover" section: "Reconnection: No tests for player WebSocket disconnect/reconnect during a game."
- **Suggested action**: Add a reachability property `reconnection-exercised` confirming that the `handleConn` reconnection branch (where an existing player entry is replaced) is reached during testing.

### Finding 5: No Property for Process Health / Crash Detection

- **Properties affected**: None (gap)
- **Concern**: The codebase contains 7 `log.Panicf` calls in production paths. A panic crashes the process. With Antithesis fault injection (DB errors, network errors), these panics are reachable. No property monitors whether the matchmaker and game server processes remain healthy throughout the test. The `anytime_check_health` test command exists in the deployment but is not a catalog property.
- **Scope**: Catalog-wide
- **Evidence**: Panic calls at: `match_queue.go:75` (fleet error), `match_queue.go:173` (DB error), `protocol.go:273` (marshal error), `reporter.go:61` (JSON encode error), `reporter.go:65` (request creation error). SUT analysis: "publishMatch panics on DB error. matchPlayers panics on non-ErrNoServersAvailable fleet errors."
- **Suggested action**: Add a liveness property `services-remain-healthy` that periodically checks the `/health` endpoints of both matchmaker and game server. Alternatively, convert the `anytime_check_health` test command into a named catalog property so crash-restart cycles are tracked.

### Finding 6: Leaderboard Test Command Not Reflected in Catalog

- **Properties affected**: None (gap)
- **Concern**: The `eventually_check_leaderboard` test command validates leaderboard state but has no corresponding property in the catalog. If this check fails, it is a test failure but not a tracked Antithesis assertion. This means leaderboard corruption would not appear in the Antithesis property dashboard.
- **Scope**: Catalog-wide
- **Evidence**: Deployment topology lists `eventually_check_leaderboard` as a test command. No catalog property mentions the leaderboard.
- **Suggested action**: Add a safety property `leaderboard-valid` that checks the leaderboard has entries, ELO values are within bounds, and (ideally) global ELO conservation holds. This subsumes the suggested action from Finding 1 and provides a home for the `eventually_check_leaderboard` logic.

---

## Passes

### Game Rule Properties Are Well-Balanced

The four game-rule properties (`game-rules-enforced`, `correct-winner-detection`, `turn-order-maintained`, `board-state-valid`) cover the input validation, state mutation, outcome detection, and structural invariant dimensions of game correctness. They are appropriately proportioned for a medium-risk area -- thorough but not over-invested.

### ELO Properties Target the Right Risks

The three ELO properties directly address the three most likely ELO corruption scenarios: non-zero-sum updates, ELO changes on cancellation, and double-processing. The evidence files correctly identify the missing idempotency guard as the root cause.

### Session Lifecycle Coverage Is Comprehensive

The five session lifecycle properties form a complete chain from creation through cleanup. The combination of safety properties (capacity, cleanup) and liveness properties (completion, deadline enforcement, result delivery) ensures both correctness and progress are monitored.

### Reachability Properties Target High-Value Code Paths

The six reachability properties (`all-game-types-played`, `draw-outcome-reached`, `turn-timeout-fires`, `evil-move-rejected`, `session-deadline-fires`, `max-sessions-reached`) all target code paths that are difficult to exercise deterministically and important for coverage. None are trivially reachable (e.g., "health endpoint called"), which means they genuinely guide Antithesis exploration.

### Orphaned Session Property Is Well-Designed

The `orphaned-session-handled` property targets a subtle race condition (the `collectMatches`/`publishMatch` window) that would be extremely difficult to reproduce with conventional testing. The property is correctly typed as liveness (eventual cleanup) and correctly instrumented as a `Reachable` + `Sometimes` combination.

### Property Type Distribution Is Healthy

The 50/27/23 split across safety/liveness/reachability reflects appropriate priorities: safety properties for invariants, liveness for progress guarantees, and reachability for coverage guidance.

---

## Uncertainties

### Uncertainty 1: Timer.Reset Safety

The SUT analysis identifies a theoretical race in `Protocol.handleMove` calling `p.turnTimer.Reset(p.turnTimeout)` (unproven assumption #2). No property directly targets this. It is unclear whether Antithesis can trigger the race (it requires the timer to fire between the select case match and the Reset call). The `turn-timeout-fires` reachability property exercises the timer path but does not specifically check for the reset race.

### Uncertainty 2: sendLatest Message Loss

The `sendLatest` pattern (`protocol.go:254-267`) can lose messages if the channel fills between the drain and retry. The `player-sees-terminal-state` property is typed as `Sometimes` (not `Always`), which implicitly acknowledges that some terminal states may be lost. It is unclear whether this is an intentional design tradeoff or a bug that should have a stronger property.

### Uncertainty 3: Evil Player Coverage Sufficiency

The evil player workload (`parallel_driver_evil_player`) runs a single evil player. It is unclear whether one evil player is sufficient to exercise all evil behavior paths (malformed moves, out-of-turn moves, extra connection chaos, queue abandonment) with meaningful frequency. The `evil-move-rejected` reachability property will detect if evil moves are never reached, but it cannot detect if only a subset of evil behaviors are exercised.

### Uncertainty 4: Matchmaker Panic Recovery

The `publishMatch` function calls `log.Panicf` on DB errors, and `matchPlayers` calls `log.Panicf` on non-`ErrNoServersAvailable` fleet errors. If Antithesis triggers a DB error during `publishMatch`, the matchmaker crashes. It is uncertain whether the system can recover: the match was partially committed (game server session created, but DB write failed and in-memory update didn't happen). After restart, the game server has an orphaned session and the matchmaker has no record of it. The `orphaned-session-handled` property should cover this, but only if the game server's deadline timer fires and the session expires.

### Uncertainty 5: Fleet.rng Thread Safety

The SUT analysis identifies `Fleet.rng` as a `*rand.Rand` that is not safe for concurrent use (unproven assumption #6). It is accessed only from the matcher goroutine, which should be safe. No property targets this. If Antithesis somehow creates a scenario where fleet methods are called from multiple goroutines (e.g., if the matcher ticker overlaps), this would be a data race. Go's race detector might catch it, but no Antithesis property would.
