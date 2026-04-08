# Antithesis Fit Evaluation

Evaluation of whether each property in the catalog is in Antithesis's sweet spot (timing-sensitive, concurrency-sensitive, partial failure, combinatorial state exploration) versus better served by unit or integration tests.

## Rating Scale

- **Strong Fit**: The property fundamentally requires timing/concurrency/fault exploration that Antithesis excels at. Cannot be fully verified by deterministic tests.
- **Moderate Fit**: The property benefits from Antithesis exploration but has significant overlap with what unit/integration tests already cover or could cover.
- **Weak Fit**: The property can be (or already is) fully verified by unit tests with fixed inputs. Including it in Antithesis is low-value or misleading.

---

## Property-by-Property Assessment

### game-rules-enforced — Weak Fit

**Rating**: Weak Fit

The core claim is that `MakeMove` never accepts an invalid move. The Protocol goroutine processes moves sequentially from a single inbox channel — there is no concurrent mutation of game state. Each game's `MakeMove` is a pure function: `(State, Player, Move) -> (State, error)`. The validation logic (bounds checking, cell occupancy, column fullness, ship overlap) is deterministic and depends only on the current state and the move.

The "Antithesis Angle" claims evil players sending moves "under various timing conditions" adds value, but the timing of move arrival does not change the validation logic. Whether a move arrives 1ms or 1s after the previous one, the same `CanMakeMove` + `MakeMove` path runs with the same state. The inbox channel serializes everything.

This is classic unit test territory. The existing `harness_test.go` already drives games to completion. A property-based test (e.g., with Hegel) generating random move sequences would be a more efficient way to find validation bugs than burning Antithesis time. The evil player workload is useful for coverage, but the property itself does not require Antithesis's scheduling exploration.

**One caveat**: Battleship's setup phase has complex validation (5 ships, overlap, orientation). A combinatorial fuzzer would be more efficient than Antithesis for finding edge cases here, but the evil player generating random placements under Antithesis does provide some value as a side effect.

### correct-winner-detection — Moderate Fit

**Rating**: Moderate Fit

Win detection itself is deterministic (check board state). However, the property evidence identifies a real timing concern: `playerToID` iterates `p.players` to map game.Player to UUID. If a reconnection replaces a player entry between the win detection and the `playerToID` call, the wrong UUID could be reported. Since `handleConn` and `handleMove` both run in the same Protocol goroutine's select loop, this race cannot actually happen — the map is only mutated by `handleConn`, which runs in the same goroutine.

The property has moderate Antithesis fit because: (a) it serves as a cross-check on the ELO cluster — if winner detection is wrong, downstream ELO updates are wrong, and (b) fault injection during the move-validate-broadcast sequence could expose bugs where status is set before the board is fully updated. But the single-goroutine design largely eliminates the timing concerns.

### turn-order-maintained — Weak Fit

**Rating**: Weak Fit

`CanMakeMove` is a simple comparison: `player != s.CurrentPlayer`. The Protocol processes messages sequentially. Two moves arriving simultaneously are buffered in the capacity-2 inbox channel and processed one at a time. The second out-of-turn move deterministically fails the turn check.

The evidence file acknowledges this: "The Protocol processes messages sequentially from the inbox channel (single goroutine), so true concurrent move processing isn't possible." This is a unit-testable property. The existing `harness_test.go` exercises the turn alternation path.

### board-state-valid — Weak Fit

**Rating**: Weak Fit

Same reasoning as `game-rules-enforced`. Board state is mutated only by `MakeMove` in the single Protocol goroutine. The evidence explicitly states: "State is only mutated in the Protocol's single goroutine, so concurrent mutation isn't a concern." The evidence then identifies the only realistic failure as "a bug in MakeMove that violates a structural invariant despite passing validation" — this is a functional correctness bug, not a concurrency bug. Unit tests and property-based tests are the right tool.

The workload-side assertion (spectator/player validates received state) is useful as a defense-in-depth sanity check during Antithesis runs but does not need Antithesis to find bugs.

### elo-zero-sum — Strong Fit

**Rating**: Strong Fit

`CalcElo` is zero-sum by construction (unit tested). But the zero-sum property at the DB transaction level is where Antithesis shines. The evidence identifies three paths that call `ReportSessionResult` for the same session: the Reporter, the session monitor, and the cancel endpoint. If two concurrent calls both read the same initial ELO values, both compute deltas, and both write updates, the zero-sum property is violated at the system level even though each individual call is zero-sum.

SQLite serializes writes, but the read-modify-write pattern within `ReportSessionResult` (read players, compute new ELO, write updates) happens inside a transaction. Whether SQLite's transaction isolation prevents the double-read depends on timing of concurrent transactions — exactly what Antithesis explores.

The existing unit test verifies CalcElo arithmetic. No test exercises concurrent `ReportSessionResult` calls. This is a gap that only Antithesis (or a concurrency-focused integration test) can fill.

### no-elo-change-on-cancel — Strong Fit

**Rating**: Strong Fit

The evidence identifies a specific, timing-dependent race: a game completes normally (Reporter submits result, ELO updated) while the session monitor simultaneously cancels the same session (deadline passed). The second call overwrites `completed_at` and sets `cancelled=true`, but the ELO change from the first call persists. The session is now marked cancelled but has ELO changes — an inconsistent state.

This race requires specific timing between the Reporter's HTTP call and the session monitor's ticker — exactly the kind of interleaving Antithesis systematically explores. No existing test covers this. The `log.Panicf("BUG: ...")` guard only catches the case where a cancel arrives with a winner, not the case where a normal result arrives first and a cancel arrives second.

### no-double-elo-update — Strong Fit

**Rating**: Strong Fit

The evidence is compelling: `updateSessionResult` SQL has no `WHERE completed_at IS NULL` guard. Three paths can deliver the same session result. The Reporter re-enqueues on temporary error (creating a retry loop). The session monitor runs on a separate ticker. There is no idempotency guard.

This is Antithesis's bread and butter: exploring the timing of concurrent result submissions to find double-update bugs. The evidence explicitly notes "the current code has no idempotency guard on ReportSessionResult." A unit test cannot reproduce the race between the Reporter's retry, the session monitor's cancel, and the normal completion path without artificial synchronization.

### session-capacity-enforced — Moderate Fit

**Rating**: Moderate Fit

The mutex makes the check-and-insert atomic. The existing unit test verifies the 503 path. The evidence identifies a conservatively-safe failure mode (over-counting due to finished-but-not-yet-cleaned sessions) and confirms the mutex prevents the dangerous case.

Antithesis could explore whether a burst of concurrent `CreateSession` calls after cleanup races past the capacity check, but the mutex should prevent this. The property is worth including as an `Always` assertion because it's cheap and catches mutex bugs, but the probability of finding a real bug here is lower than in the ELO cluster.

### session-eventually-completes — Strong Fit

**Rating**: Strong Fit

This liveness property depends on the entire chain: turn timeout, session deadline, session monitor, and player connectivity all working correctly under adverse conditions. Player disconnection, game server restarts, network partitions — these are all partial failure scenarios that Antithesis explores by design.

The evidence identifies a key risk: if the Protocol goroutine panics, deferred cleanup runs but `report()` is never called. The matchmaker's session monitor is the safety net, but only if it's running and the DB query works. Testing this requires injecting failures at specific points in the session lifecycle — not something unit tests can do.

### expired-sessions-cancelled — Strong Fit

**Rating**: Strong Fit

The session monitor running on a ticker, potentially delayed by Antithesis pausing the goroutine, racing with normal game completion — these are timing-sensitive scenarios. The evidence identifies that the monitor logs and returns on DB errors (no retry until next tick), and that cancel races with completion (feeding into the double-ELO-update bug).

This property is also the safety net for `session-eventually-completes`. Antithesis can test whether the safety net actually works by creating conditions where normal completion fails (partition the game server, pause player processes).

### result-reported-to-matchmaker — Strong Fit

**Rating**: Strong Fit

Network partition between game server and matchmaker during result submission is the canonical Antithesis scenario. The Reporter retries temporary errors but drops permanent ones. The evidence identifies that if the matchmaker is permanently unreachable, results are dropped after one attempt with no backoff retry.

The channel capacity concern (resultCh filling up, causing Protocol.report() to block) is another concurrency issue that requires realistic load to trigger. The re-enqueue loop on persistent temporary errors is a liveness hazard that only manifests under sustained network issues.

### session-cleanup-complete — Moderate Fit

**Rating**: Moderate Fit

The cleanup runs deferred, so it executes even on panic. The concern about `JoinSession` finding a finished-but-not-yet-cleaned session is handled by the `IsFinished()` check. The timing window between `delete(s.sessions, sid)` and the health broadcast is a minor inconsistency that's hard to observe.

Including this as an `Always` assertion is low-cost and provides defense-in-depth, but the single-goroutine-per-session design and deferred cleanup make bugs here unlikely.

### no-duplicate-match — Strong Fit

**Rating**: Strong Fit

The race window between `collectMatches` (releases lock) and `publishMatch` (re-acquires) is a classic TOCTOU pattern. The evidence identifies that Go's `time.Ticker` drops ticks to prevent overlapping invocations, but Antithesis's scheduler control could potentially create scenarios where the single-goroutine assumption is violated or where the lock release window allows state changes.

More importantly, the `publishMatch` check (`_, hasA := q.queued[a.pid]`) is the guard against double-matching. If this check has a bug, two sessions would be created for the same player. Antithesis can explore the window where a player unqueues between collect and publish, and verify the orphaned session is handled correctly.

### matchmaking-progress — Strong Fit

**Rating**: Strong Fit

This liveness property requires the entire matchmaking pipeline to function: the matcher goroutine running, ELO relaxation working over time, fleet servers being available, and session creation succeeding. Fleet exhaustion (all servers unavailable) silently drops matches. Antithesis can partition game servers, creating fleet exhaustion, and verify that matchmaking resumes when servers recover.

The ELO relaxation-over-time mechanism is inherently time-dependent. Antithesis's time manipulation can test whether relaxation works correctly under time jumps or pauses.

### orphaned-session-handled — Strong Fit

**Rating**: Strong Fit

The catalog correctly identifies this as "difficult to trigger deterministically — it requires a player to unqueue during the network round-trip to the game server." This is the textbook case for Antithesis: a race condition that requires specific timing during a network call to manifest. The orphaned session consumes game server capacity until deadline expiry.

The property also exercises the interaction between the game server's deadline timer and the matchmaker's session monitor — two independent cleanup mechanisms that must cooperate correctly.

### player-reconnect-works — Strong Fit

**Rating**: Strong Fit

Reconnection is inherently timing-sensitive: the old connection's write goroutine may be mid-write when `close(existing.conn)` is called. The evidence identifies that closing the channel (not the WebSocket) terminates the old write goroutine, but the old goroutine might be in the middle of `wsjson.Write`. The old WebSocket then sends a close frame, which the player's read goroutine interprets as server-initiated close.

This interleaving of old-connection-teardown with new-connection-setup, concurrent with move processing from the other player, is exactly what Antithesis explores. No existing test covers reconnection at all (noted in the SUT analysis: "No tests for player WebSocket disconnect/reconnect during a game").

### third-player-rejected — Moderate Fit

**Rating**: Moderate Fit

The rejection logic is straightforward: `len(p.players) >= 2` and unknown pid results in an error message. This is easily unit-testable. The Antithesis angle (evil players' `runExtraConnectChaos` under heavy load) adds some value in testing that the rejection doesn't interfere with legitimate connections, but the core property is deterministic.

The dangling goroutine concern (write goroutine blocked on a never-closed channel when the evil player keeps the WebSocket open) is a resource leak that Antithesis could detect through long-running tests, giving this moderate fit.

### spectator-state-consistency — Moderate Fit

**Rating**: Moderate Fit

State is serialized from the Protocol's goroutine-local variable in a single-threaded context. The evidence confirms "no concurrent access issue." The risk of receiving invalid state is low because `json.Marshal` operates on a consistent snapshot.

The `sendLatest` drop-and-retry pattern means spectators may miss states but should only see valid ones. This is more of a design verification than a concurrency concern. Including it as a workload-side assertion is reasonable but not high-value for Antithesis specifically.

### player-sees-terminal-state — Strong Fit

**Rating**: Strong Fit

The `sendLatest` pattern could drop the terminal state if the channel is full. The evidence identifies the real risk: "the player's WebSocket write fails (network issue) before the terminal state is written." The write goroutine exits, the channel drains on close, and the player never sees the terminal state.

This is a partial failure scenario: network disruption at the exact moment of game completion. Antithesis can inject this timing by partitioning the network between the game server and player at the right moment.

### fleet-failover — Strong Fit

**Rating**: Strong Fit

Fleet failover is inherently about partial failure: a game server returns 503 or has a network error, and the fleet must correctly skip it and try another. The retry-backoff timing (`retryAt`) is time-dependent. Antithesis can partition a game server, verify the fleet enters retry-backoff, then heal the partition and verify the fleet resumes using the server.

The evidence also identifies that `Fleet.rng` is `*rand.Rand` (not thread-safe), which is safe only because fleet methods are called from a single goroutine. If this assumption is ever violated, Antithesis would detect the data race.

### all-game-types-played — Strong Fit (as Reachability)

**Rating**: Strong Fit as a Reachability assertion

This is not a correctness property but a coverage guide. It's essential for Antithesis effectiveness — without it, Antithesis might get stuck exploring only one game type. Reachability assertions are how you tell Antithesis to maximize coverage. This is exactly what they're designed for.

### draw-outcome-reached — Strong Fit (as Reachability)

**Rating**: Strong Fit as a Reachability assertion

Draws require specific board configurations that the AI may rarely produce. This reachability assertion guides Antithesis toward exploring draw paths, which are distinct code paths (symmetric ELO update, no winner mapping). Essential for coverage.

### turn-timeout-fires — Strong Fit (as Reachability)

**Rating**: Strong Fit as a Reachability assertion

Turn timeouts require player pauses. Antithesis can pause player processes to trigger this. The timeout-to-opponent-wins path is a critical liveness mechanism that's untestable without either mocking time or actually pausing the player. Antithesis can do the latter naturally.

### evil-move-rejected — Moderate Fit (as Reachability)

**Rating**: Moderate Fit as a Reachability assertion

Confirms the evil player workload is generating invalid moves and they're being rejected. Useful for workload validation (are we actually exercising error handling?). The rejection itself is deterministic, but knowing the workload is hitting this path is important.

### session-deadline-fires — Strong Fit (as Reachability)

**Rating**: Strong Fit as a Reachability assertion

The session deadline is the last-resort cleanup mechanism. Confirming it fires at least once ensures Antithesis explores the deadline-cancellation path, which feeds into `expired-sessions-cancelled` and the ELO race conditions.

### max-sessions-reached — Strong Fit (as Reachability)

**Rating**: Strong Fit as a Reachability assertion

Hitting the capacity limit triggers the 503 + Retry-After backpressure mechanism. This is the fleet failover trigger. Confirming this path is reached ensures Antithesis explores the full backpressure cascade: capacity hit -> 503 -> fleet retry-backoff -> eventual recovery.

---

## Catalog-Wide Observations

### Observation 1: Game Rule Cluster is Over-Represented for Antithesis

Four properties (`game-rules-enforced`, `turn-order-maintained`, `board-state-valid`, and partially `correct-winner-detection`) test the same single-goroutine code path where concurrency is not a factor. The Protocol design (single goroutine with channel-based inbox) deliberately eliminates concurrency in game state mutation. These properties are testing functional correctness of pure validation logic.

**Impact**: These properties will consume Antithesis assertion budget and report output without finding concurrency/timing bugs. They will likely show 100% pass rates from the start, providing no signal.

**Recommendation**: Keep `game-rules-enforced` and `board-state-valid` as lightweight `Always` assertions (they're cheap to check and provide a safety net), but recognize they're defense-in-depth rather than Antithesis-sweet-spot properties. Consider investing more in property-based testing (Hegel) for the game rule validation logic, which can exhaustively explore the combinatorial move space more efficiently than Antithesis.

### Observation 2: ELO Cluster is the Highest-Value Antithesis Target

The three ELO properties (`elo-zero-sum`, `no-elo-change-on-cancel`, `no-double-elo-update`) target a real concurrency gap: multiple paths calling `ReportSessionResult` without idempotency guards, racing with each other through SQLite transactions. No existing test covers this. The SUT analysis explicitly notes: "No tests exercise multiple goroutines hitting the same endpoints simultaneously."

This is where Antithesis is most likely to find real bugs. The missing `WHERE completed_at IS NULL` guard on `updateSessionResult` is a documented gap that Antithesis could turn into a concrete bug report.

### Observation 3: Reachability Properties Are Well-Chosen

All six reachability properties (`all-game-types-played`, `draw-outcome-reached`, `turn-timeout-fires`, `evil-move-rejected`, `session-deadline-fires`, `max-sessions-reached`) guide Antithesis toward interesting regions of the state space. They cover: game type diversity, outcome diversity (draw vs. win), timeout mechanisms, error handling, and resource limits. This is a good spread.

### Observation 4: Missing Property — Result Channel Backpressure

The `resultCh` channel (capacity `MaxSessions`) could fill if many sessions complete while the matchmaker is unreachable. When full, `Protocol.report()` blocks on `p.result <- resultMsg{...}`, stalling the Protocol goroutine. This means the game's select loop stops processing — no more moves, no turn timeout, no deadline check. The game appears hung even though the deadline timer is firing.

This is a concurrency/backpressure bug that Antithesis could find by partitioning the matchmaker while games are completing. No property currently covers "Protocol goroutine remains responsive even when result delivery is blocked."

### Observation 5: Missing Property — sendLatest Message Loss

The `sendLatest` function has a three-step non-blocking send pattern. Between the drain (`<-ch`) and the retry send (`ch <- msg`), another goroutine could send a different message, causing the retry to fail and the original message to be silently dropped. With spectator channels of capacity 1, this could mean a spectator misses the terminal state entirely (not just intermediate states).

This is a concurrency bug in a helper function that affects both `spectator-state-consistency` and `player-sees-terminal-state`, but neither property specifically targets the `sendLatest` race.

---

## Summary Table

| Property | Fit Rating | Primary Justification |
|----------|-----------|----------------------|
| game-rules-enforced | Weak | Single-goroutine, deterministic validation. Unit/PBT territory. |
| correct-winner-detection | Moderate | Single-goroutine eliminates race, but cross-check with ELO adds value. |
| turn-order-maintained | Weak | Simple comparison in single goroutine. Already unit-tested. |
| board-state-valid | Weak | Same as game-rules-enforced. No concurrency in state mutation. |
| elo-zero-sum | Strong | Concurrent ReportSessionResult paths, DB transaction races. |
| no-elo-change-on-cancel | Strong | Cancel-vs-completion race, timing-dependent state inconsistency. |
| no-double-elo-update | Strong | No idempotency guard, three concurrent delivery paths. |
| session-capacity-enforced | Moderate | Mutex should prevent race, but cheap to assert as defense-in-depth. |
| session-eventually-completes | Strong | Depends on full chain: timeouts, monitors, connectivity under failures. |
| expired-sessions-cancelled | Strong | Ticker-driven monitor racing with completion, DB error handling. |
| result-reported-to-matchmaker | Strong | Network partition, retry loops, channel backpressure. |
| session-cleanup-complete | Moderate | Deferred cleanup is robust; timing windows are narrow. |
| no-duplicate-match | Strong | TOCTOU between collect and publish, lock release during network call. |
| matchmaking-progress | Strong | Full pipeline liveness under fleet exhaustion and time manipulation. |
| orphaned-session-handled | Strong | Race during network round-trip; hard to trigger deterministically. |
| player-reconnect-works | Strong | Old/new connection teardown/setup interleaving. Zero existing test coverage. |
| third-player-rejected | Moderate | Deterministic rejection; goroutine leak is secondary concern. |
| spectator-state-consistency | Moderate | Single-goroutine serialization; sendLatest is the real risk. |
| player-sees-terminal-state | Strong | Network failure at game completion; sendLatest drop risk. |
| fleet-failover | Strong | Partial failure, retry-backoff timing, partition/recovery cycle. |
| all-game-types-played | Strong | Essential reachability guide for Antithesis coverage. |
| draw-outcome-reached | Strong | Reachability for rare-but-distinct code path. |
| turn-timeout-fires | Strong | Reachability requiring process pausing; natural Antithesis capability. |
| evil-move-rejected | Moderate | Workload validation; rejection itself is deterministic. |
| session-deadline-fires | Strong | Reachability for last-resort cleanup mechanism. |
| max-sessions-reached | Strong | Reachability for backpressure cascade. |
