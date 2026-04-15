# Wildcard Evaluation — Aardvark Arena Property Catalog

## Purpose

This evaluation covers what the Antithesis Fit, Coverage Balance, and Implementability lenses are structurally unlikely to catch: framing errors, implicit assumptions in the catalog, emergent interactions between properties, semantic gaps hiding behind syntactic completeness, and self-consistency issues in the analysis artifacts themselves.

---

## Findings

### F1. ReportSessionResult has a real double-update bug — no property catches the *consequence*

| | |
|---|---|
| **Properties** | `no-double-elo-update`, `no-elo-change-on-cancel`, `elo-zero-sum` |
| **Scope** | Cross-property |
| **Concern** | The three ELO properties independently identify the `ReportSessionResult` double-call race, but no property captures the *composite* failure mode: a session that is first completed (ELO updated for a winner) and then overwritten by a cancellation (completed_at and cancelled flags reset) ends up with the DB showing a cancelled session **and** ELO changes already committed. The `no-elo-change-on-cancel` property as written checks that "when a session is cancelled, neither player's ELO changes" — but it would need to compare pre-session ELO snapshots against post-session ELO, which requires cross-request state. The property evidence (`no-elo-change-on-cancel.md`) explicitly describes this scenario but the invariant definition only covers the `ReportSessionResult` codepath when `cancelled==true`, not the aftermath of a completion-then-cancel overwrite. |
| **Evidence** | `updateSessionResult` SQL (`db.go:77-81`) has no `WHERE completed_at IS NULL` guard. `no-elo-change-on-cancel.md` failure scenario describes exactly this race. The invariant says "In `ReportSessionResult` when `cancelled == true`, assert no player stats are modified" — but the ELO damage was already done by the *first* (non-cancelled) call. |
| **Suggested action** | Either (a) add an idempotency guard to `ReportSessionResult` so the second call is a no-op, or (b) add a *separate* property ("session-result-consistent") that periodically asserts that for every cancelled session in the DB, neither player's ELO differs from what it would be without that session. This would be a database-level assertion run from the workload's `eventually_check_leaderboard` command. The `no-double-elo-update` property should be the primary fix — if it holds, the other two follow — but the catalog should acknowledge that `no-elo-change-on-cancel` as currently specified cannot detect the overwrite scenario. |

### F2. Battleship turn-on-hit rule makes Battleship qualitatively different — no property or reachability target addresses this

| | |
|---|---|
| **Properties** | `game-rules-enforced`, `turn-order-maintained`, `all-game-types-played` |
| **Scope** | Catalog-wide gap |
| **Concern** | Battleship's `handleAttack` (`battleship.go:194-219`) does NOT switch turns on a Hit — the attacking player goes again. This means `turn-order-maintained` (which asserts `state.CurrentPlayer == player` before each move) behaves fundamentally differently for Battleship: a player can make many consecutive moves. The catalog treats all three games as equivalent for the turn-order property, but the "hit means extra turn" mechanic is a distinct code path that is never explicitly called out. More importantly, the Battleship setup phase has both players submit setup moves in alternation (P1 then P2), and after setup, the first attacker is the player after the last setup player. None of the game rule properties distinguish setup-phase validation from attack-phase validation, despite them being completely different code paths with different move types. |
| **Evidence** | `battleship.go:208-209`: on Miss, switches turn; on Hit, does not. `game-rules-enforced.md` mentions "Battleship ships don't overlap" but not the hit-continues rule. `turn-order-maintained` invariant is stated generically. |
| **Suggested action** | Add a Battleship-specific reachability assertion for "hit-continues-turn" — a `Reachable` that fires when a player makes two consecutive attack moves. Without this, Antithesis might never explore the consecutive-attack path deeply enough to find bugs in the state machine around hit streaks. |

### F3. Evil player's `doQueueAbandon` creates ghost players that can starve matchmaking — not addressed by any property

| | |
|---|---|
| **Properties** | `matchmaking-progress` |
| **Scope** | Catalog-wide gap |
| **Concern** | When `doQueueAbandon` fires (`loop.go:99-102`), the evil player queues a random new UUID and never polls for it. This player sits in `MatchQueue.queued` forever. If it gets paired by `collectMatches`, the fleet creates a real game server session, then `publishMatch` succeeds (the ghost is still in `queued`), moves it to `matched`, and a real opponent now has a SessionInfo pointing to a game where one player will never connect. The opponent waits until the turn timer fires. This is by design as a stress test, but no property verifies that the system handles it correctly: (a) the ghost player in `matched` is never cleaned up until the session expires and `Untrack` runs, and (b) the opponent experiences a degraded game (waiting for turn timeout). The `matchmaking-progress` property only asserts that *queued* players eventually get matched — it does not assert that matched players eventually play a real game. |
| **Evidence** | `loop.go:99-102` creates a `NewMatchmakerClient` with a random UUID, then the *original* player continues its loop. The ghost UUID remains queued. `match_queue.go` has no TTL or cleanup for stale queue entries. |
| **Suggested action** | Add a property or reachability target for "ghost player matched" — confirming the system handles the case where a matched player never connects. This overlaps with `orphaned-session-handled` but from the matchmaking side: the matchmaker's `matched` map retains the ghost until session expiry. Consider a liveness property: "every player in `matched` is eventually moved out of `matched`." |

### F4. The `sendLatest` pattern has a documented message-loss window — `player-sees-terminal-state` may be structurally unfalsifiable

| | |
|---|---|
| **Properties** | `player-sees-terminal-state` |
| **Scope** | Property-specific |
| **Concern** | `sendLatest` (`protocol.go:254-267`) has a three-step pattern: try send, drain stale, retry send. Between the drain and the retry, another `BroadcastState` call (from a different code path within the same goroutine — not actually concurrent, so this specific race doesn't exist in the current single-goroutine protocol). However, the SUT analysis notes that the `sendLatest` pattern "can still lose messages if channel fills between drain and retry." Since the protocol is single-threaded, this is actually safe within the protocol goroutine. BUT: the *write goroutine* in `session.go:231-239` reads from the channel and writes to the WebSocket. If the WebSocket write is slow, the channel stays full, and `sendLatest`'s drain-then-retry can lose the intermediate state (which is by design). The terminal state is the last thing sent before the channel is closed, so it should always be delivered. The property is typed as `Sometimes` rather than `Always`, acknowledging that network failure can prevent delivery. This is reasonable but makes it a weak property — it can pass even if terminal state delivery is broken for most games, as long as it works once. |
| **Evidence** | `protocol.go:95-111`: `report()` sets status, calls `BroadcastState()`, then the loop breaks and channels are closed. The terminal state is the last `sendLatest` call before close. `player-sees-terminal-state.md` failure scenario describes WebSocket write failure preventing delivery. |
| **Suggested action** | Consider strengthening to "AlwaysOrUnreachable: if a player is connected when the game ends (WebSocket open), they receive the terminal state." The current `Sometimes` formulation does not distinguish between "network made delivery impossible" and "bug prevented delivery despite good network." At minimum, add a companion reachability assertion confirming the terminal-state delivery path is reached frequently. |

### F5. The property catalog has no property for data persistence across matchmaker restarts

| | |
|---|---|
| **Properties** | (none) |
| **Scope** | Catalog-wide gap |
| **Concern** | The deployment topology uses file-backed SQLite (`/data/matchmaker.db`) specifically so "state survives process restarts within a timeline." Antithesis can restart the matchmaker process. When the matchmaker restarts, the in-memory `MatchQueue` (both `queued` and `matched` maps) is lost, but the SQLite `sessions` and `players` tables persist. This creates a split-brain: (a) sessions that were active before the restart are in the DB but not in the in-memory `matched` map, so players polling `PUT /queue/{pid}` won't get their SessionInfo back, (b) the session monitor will eventually cancel these DB-orphaned sessions. No property asserts that after a matchmaker restart, the system recovers to a consistent state. This is squarely in Antithesis territory — restart-induced inconsistency between persisted and in-memory state. |
| **Evidence** | `match_queue.go:33-39`: `queued` and `matched` are plain Go maps, not persisted. `server.go:48-63`: `New()` creates fresh `MatchQueue`. `deployment-topology.md`: "Use file-backed SQLite ... so state survives process restarts." |
| **Suggested action** | Add a liveness property: "after a matchmaker restart, all sessions previously in the DB that are not yet completed are eventually either completed normally or cancelled." This property verifies the session monitor's ability to clean up DB-orphaned sessions. Also consider a safety property: "after a matchmaker restart, no player experiences a permanent hang" — i.e., players re-queuing after restart eventually get matched. |

### F6. Reporter result channel re-enqueue creates an unbounded retry loop with no backoff

| | |
|---|---|
| **Properties** | `result-reported-to-matchmaker` |
| **Scope** | Property-specific / catalog-wide |
| **Concern** | `reporter.go:72-74` re-enqueues the result on `resultCh` on temporary error, with no delay, backoff, or retry count. If the matchmaker is partitioned from the game server (which Antithesis will do), every result submission fails, gets re-enqueued, gets retried immediately, fails again — a tight loop. This consumes CPU and floods the matchmaker (once reachable) with simultaneous retries. The `result-reported-to-matchmaker` property is typed as `Sometimes` (results eventually get through), but does not address the tight-loop degradation or the possibility that the `resultCh` channel (capacity `MaxSessions`) fills up and blocks `Protocol.report()`, which blocks the protocol goroutine, which prevents games from ending. This is a cascading failure: network partition -> reporter spin -> channel full -> protocol blocked -> games hang -> turn timers can't fire (select loop stuck on channel send). |
| **Evidence** | `reporter.go:72-74`: `r.resultCh <- result` re-enqueue with no delay. `protocol.go:106`: `p.result <- resultMsg{...}` is a blocking send. `result-reported-to-matchmaker.md` failure scenario #3 mentions the re-enqueue loop. `gameserver/server.go:39`: `resultCh` capacity is `cfg.MaxSessions`. |
| **Suggested action** | Add a safety property: "the protocol goroutine's `report()` call completes within a bounded time" or at least a reachability assertion for the resultCh-full condition. More practically, this is a bug that should be fixed (add backoff or retry limit), but the property catalog should at least have a property that would detect the cascading failure: "session-eventually-completes should fire even when matchmaker is partitioned from game server." The current `session-eventually-completes` might actually catch this (the session deadline timer is in the same select as the inbox), but only if the protocol goroutine isn't blocked on the `p.result <- resultMsg` send. If the send blocks, the select loop never runs, so the deadline timer never fires. This is a real liveness bug. |

### F7. Battleship has no draw outcome — `draw-outcome-reached` evidence is misleading

| | |
|---|---|
| **Properties** | `draw-outcome-reached` |
| **Scope** | Property-specific |
| **Concern** | The catalog says "At least one game ends in a draw (possible in TicTacToe and Connect4)." This is correct — Battleship cannot draw (one player must eventually sink all 17 opponent cells). However, the catalog's parenthetical is the only acknowledgment. The `draw-outcome-reached` reachability assertion is placed in `Protocol.report()` which handles all game types. If it fires for a Battleship game reporting Draw status, that would indicate a bug. The catalog should explicitly note that Draw is unreachable for Battleship and consider making the reachability assertion game-type-specific. |
| **Evidence** | `battleship.go:213-216`: win condition is `hitCount == totalShipCells`. No draw check. No `isFull` equivalent. The game can only end by one player sinking all ships or by timeout/cancellation (which is Cancelled, not Draw). |
| **Suggested action** | Clarify in the catalog that `draw-outcome-reached` is only expected to fire for TTT and Connect4. Consider adding a companion safety property: "Battleship games never end in Draw status." This would catch bugs in the Battleship win-detection logic. |

### F8. Property relationships document claims `game-rules-enforced` implies `board-state-valid`, but the assertion locations differ in a meaningful way

| | |
|---|---|
| **Properties** | `game-rules-enforced`, `board-state-valid` |
| **Scope** | Cross-property / framing |
| **Concern** | `property-relationships.md` claims `game-rules-enforced` dominates `board-state-valid`. But `game-rules-enforced` is instrumented SUT-side in `MakeMove`, while `board-state-valid` is specified as workload-side (deserialize state from WebSocket and validate). These check different things: `game-rules-enforced` verifies moves are validated before mutation, while `board-state-valid` verifies the *serialized and transmitted* state is structurally valid. A bug in `json.Marshal`/`json.Unmarshal` for game state, or a bug in the `sendLatest` pattern that corrupts messages, would violate `board-state-valid` without violating `game-rules-enforced`. The dominance claim is wrong — they are complementary, not hierarchical. |
| **Evidence** | `game-rules-enforced` invariant: "Assertion placed SUT-side in each game's MakeMove implementation." `board-state-valid` invariant: "Workload-side assertion -- deserialize state from WebSocket message and check." These are different assertion points covering different failure modes. |
| **Suggested action** | Correct the dominance claim in `property-relationships.md`. Both properties should be implemented. `board-state-valid` catches serialization/transmission bugs that `game-rules-enforced` cannot. |

### F9. The workload topology creates a deterministic-randomness problem for Antithesis

| | |
|---|---|
| **Properties** | `all-game-types-played`, `draw-outcome-reached` |
| **Scope** | Catalog-wide |
| **Concern** | `internal/rand.go` seeds new RNGs with `rand.Int63()`, which draws from the global `math/rand` source. Under Antithesis, the global source is deterministic (seeded by the platform). All player AIs, game selection, and evil behavior share this deterministic seed chain. This is actually *good* for Antithesis (reproducibility). However, the AI strategies for TicTacToe and Connect4 are not purely random — they have heuristic preferences (TTT: build lines / block opponent; Connect4: center-column preference). These heuristics may make draws extremely rare or certain game types systematically faster, biasing which code paths Antithesis explores. The reachability assertions (`all-game-types-played`, `draw-outcome-reached`) are meant to counteract this, but they are hints, not guarantees. If the AI heuristics produce a near-zero draw rate, the `draw-outcome-reached` assertion will guide Antithesis to explore draw paths, but success depends on whether Antithesis can find a seed/interleaving that produces a draw. |
| **Evidence** | `tictactoe.go:140-167`: AI prefers building lines and blocking. `connect4.go:138-170`: AI has shuffled column preference, checks for win/block first. `rand.go:8-10`: `NewRand` uses global source. |
| **Suggested action** | This is an observation, not a defect. The reachability assertions are the correct mitigation. However, consider whether the evil player's chaos rate should be tuned to occasionally make "draw-friendly" moves (e.g., for TTT, the evil player playing center then corners). The current evil behavior corrupts moves randomly, which may not systematically produce draws. |

### F10. The `third-player-rejected` property has a goroutine leak that no property addresses

| | |
|---|---|
| **Properties** | `third-player-rejected` |
| **Scope** | Property-specific |
| **Concern** | When `handleConn` rejects a third player (`protocol.go:188-189`), it sends an error on the `conn` channel but never closes it. The write goroutine spawned by `Join` (`session.go:231-239`) does `for state := range stateCh` — since `stateCh` is never closed by the protocol (only legitimate player channels are closed in `RunToCompletion`), this goroutine will block on the range forever. It only exits if the WebSocket write fails (client disconnects). The `third-player-rejected.md` evidence acknowledges this: "The dangling goroutine would only be a problem if the evil player keeps the websocket open indefinitely." Under Antithesis, if an evil player's chaos probe connects and then Antithesis pauses/delays its close, the goroutine leaks accumulate. No property checks for goroutine leaks. |
| **Evidence** | `protocol.go:188-189`: sends error, doesn't close channel. `session.go:231-239`: write goroutine ranges over channel that is never closed for rejected players. `protocol.go:162-165`: cleanup only closes channels in `p.players`. |
| **Suggested action** | This is arguably a bug rather than a property gap, but either way: close the channel after sending the error in `handleConn`, or add `close(conn)` after the error send. For the property catalog specifically, no action needed — the leak is bounded by session lifetime (the goroutine holds a reference to the channel but the WebSocket will eventually fail when the session ends). |

### F11. No property covers the panic paths in `publishMatch` and `matchPlayers`

| | |
|---|---|
| **Properties** | (none) |
| **Scope** | Catalog-wide gap |
| **Concern** | `matchPlayers` (`match_queue.go:73-75`) panics on non-`ErrNoServersAvailable` fleet errors. `publishMatch` (`match_queue.go:164-172`) panics on DB errors. These panics crash the matchmaker process. Under Antithesis, a matchmaker process crash triggers a restart. The catalog has no property that detects or addresses matchmaker crashes. Crash-and-restart is a normal Antithesis fault mode, but the catalog should have at least a reachability assertion confirming these panic paths are exercised, since they represent unhandled error conditions that bring down the process. |
| **Evidence** | `match_queue.go:75`: `log.Panicf("fleet error: %v", err)`. `match_queue.go:172`: `log.Panicf("db error: %v", err)`. SUT analysis section "Failure and Degradation Modes" lists these panics. |
| **Suggested action** | Add `Reachable` assertions just before these panic calls to confirm Antithesis explores them. Consider whether the panics should be converted to error returns (a design decision, not a property decision). More importantly, the crash-restart scenario feeds back into F5 — matchmaker restart recovery should be a tested property. |

### F12. Spectator state consistency is tested but spectator *connection lifecycle* is not

| | |
|---|---|
| **Properties** | `spectator-state-consistency` |
| **Scope** | Catalog-wide gap |
| **Concern** | The catalog has `spectator-state-consistency` (states received are valid) but no property for spectator connection robustness. The `WatchSession` code (`session.go:354-368`) can fail if the session ends between the mutex check and the inbox send. The server-level `/watch` endpoint (`server.go:186-206`) uses `RegisterWatcher`/`UnregisterWatcher` with its own channel pattern. If the spectator's WebSocket write fails, the write loop exits and `UnregisterWatcher` closes the channel — but what about the messages already buffered? No property checks that spectator channels are properly cleaned up, or that the `watchers` map doesn't grow unboundedly. Under Antithesis, if many spectator connections are opened and dropped (which the workload doesn't currently do, since the UI is excluded), this could be a resource leak. |
| **Evidence** | `session.go:354-368`: `WatchSession` sends on inbox which could block. `session.go:296-315`: `RegisterWatcher` adds to watchers map. `session.go:318-323`: `UnregisterWatcher` removes and closes. Workload has no spectator client. |
| **Suggested action** | Low priority — the workload doesn't include a spectator client, so spectator code paths won't be exercised by Antithesis at all. If spectator testing is desired, add a spectator to the workload first, then add spectator lifecycle properties. Flag this as a known coverage gap. |

---

## Passes

### P1. Property type assignments are correct

Every `Safety` property describes something that must hold at every state. Every `Liveness` property describes something that must eventually happen. Every `Reachability` property describes a code path that should be explored. No property has a mismatched type.

### P2. The cluster analysis in property-relationships.md is accurate

Clusters 1-6 correctly group related properties. The cross-cluster dependency table is correct and matches the actual data flow in the codebase (matchmaking creates sessions, sessions produce results, results update ELO).

### P3. SUT-side vs workload-side placement decisions are reasonable

Properties that check internal invariants (turn order, capacity limits, ELO calculations) are correctly placed SUT-side. Properties that check observable behavior (board state received by players, terminal state delivery) are correctly placed workload-side. The split avoids the common trap of putting everything SUT-side (which can mask observer-visible bugs).

### P4. Reachability targets cover the key code branches

The six reachability properties (`all-game-types-played`, `draw-outcome-reached`, `turn-timeout-fires`, `evil-move-rejected`, `session-deadline-fires`, `max-sessions-reached`) cover the most important conditional branches in the system. Each targets a code path that is exercised only under specific conditions.

### P5. The evil player workload is well-designed for property stress-testing

The evil player mode exercises all the right negative paths: malformed moves, out-of-turn moves, extra connections, queue abandonment. Each maps directly to at least one safety property. The `parallel_driver_evil_player` test command runs it alongside the well-behaved swarm.

### P6. Single-goroutine protocol design simplifies many properties

The catalog correctly identifies that the Protocol goroutine's single-threaded design makes most game-state properties straightforward to assert. The evidence files consistently note this, avoiding false complexity in the analysis.

---

## Uncertainties

### U1. Whether Antithesis can meaningfully exercise the `sendLatest` message-loss path

The `sendLatest` pattern's drop-and-retry only matters when the channel consumer (WebSocket write goroutine) is slower than the producer (Protocol goroutine). Under Antithesis, process pausing can create this condition, but it's unclear whether Antithesis's scheduler granularity is fine enough to hit the exact window between drain and retry in `sendLatest`. This affects `spectator-state-consistency` and `player-sees-terminal-state`.

### U2. Whether SQLite WAL mode behaves correctly under Antithesis process kills

The deployment uses WAL mode with `synchronous=normal`. If Antithesis kills the matchmaker process (not just pauses it), WAL recovery depends on the filesystem. The deployment doc says "Antithesis preserves container filesystems within a timeline," but whether SQLite's WAL recovery works correctly in the Antithesis environment is an open question. If WAL recovery fails, the database could be corrupted, which would surface as mysterious DB errors in the matchmaker — a failure mode no property addresses.

### U3. Whether the evil player's queue-abandon rate is tuned correctly for Antithesis

With `QueueAbandonRate = 0.05` and a 1-second poll interval, the evil player creates roughly one ghost queue entry every 20 seconds. With a single evil player in the workload, this produces a modest ghost population. Whether this rate is sufficient to trigger the ghost-player matchmaking scenario (F3) during a typical Antithesis test run depends on the test duration and matchmaking speed.

### U4. Whether `Timer.Reset` in the protocol is actually safe

The SUT analysis flags `Timer.Reset` safety as an unproven assumption. The Protocol calls `p.turnTimer.Reset(p.turnTimeout)` in `handleMove` (`protocol.go:225`) without first draining the timer channel. Per Go documentation, this is safe only if the timer has already been stopped or drained. In the Protocol's select loop, the timer channel is only read in the `<-p.turnTimer.C` case, which drains it. But if `handleMove` is called between the timer firing and the select reading the timer channel, `Reset` could be called on a fired-but-undrained timer, potentially causing the next select to see a spurious timer event. No property directly tests this race.
