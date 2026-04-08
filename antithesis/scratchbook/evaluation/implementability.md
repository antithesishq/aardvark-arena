# Implementability Evaluation — Aardvark Arena Property Catalog

## Evaluation Lens

For each property: Can the invariant be observed from the workload, or does it require internal state? Does the deployment topology support the needed failure scenarios? Can the workload construct the needed operation sequences? Are there resource/timing constraints?

---

## Property-Specific Findings

### F1: board-state-valid — Workload-Side Validation Requires Game-Specific Deserializers

| | |
|---|---|
| **Properties** | board-state-valid, spectator-state-consistency |
| **Scope** | Property-specific |
| **Concern** | Both properties propose workload-side assertions that validate board state received over WebSocket. This requires the workload to deserialize the game state JSON into typed structures and then run game-specific structural checks (TTT piece counts, Connect4 gravity, Battleship bounds). The player's existing `protocol.go` already deserializes into `game.State[S]`, but the generic type parameter `S` is resolved at compile time per game type — the workload assertion code would need to handle all three game types dynamically from a single assertion site, or be placed in each per-game protocol branch in `loop.go:133-145`. |
| **Evidence** | `internal/player/protocol.go:59-60` deserializes state with a compile-time-known `S`. The workload would need to know which game type it is playing to select the right validator. The `SessionInfo.Game` field (`internal/matchmaker/fleet.go:63`) is available at match time. |
| **Suggested action** | Implement board validation as a generic function parameterized on `S` that is called inside each game-specific branch of `playGame` (loop.go:133-145), or implement it SUT-side in `BroadcastState` where the typed state is directly available. The SUT-side approach is simpler and avoids duplicating validation logic in the workload. |

### F2: correct-winner-detection — Battleship Winner Verification Requires Private State

| | |
|---|---|
| **Properties** | correct-winner-detection |
| **Scope** | Property-specific |
| **Concern** | For TicTacToe and Connect4, the winning configuration is visible in the shared board state (3-in-a-row, 4-in-a-row). For Battleship, verifying the winner requires knowing ship positions, which are stored in `BattleshipSession.shipCells` — a private field in the server-side session, never serialized to players. The shared state only contains the `Attacks` map (hit/miss results). A workload-side assertion cannot verify Battleship winners because it cannot see ship positions. A SUT-side assertion can access `shipCells` since it runs in the same process as the `BattleshipSession`. |
| **Evidence** | `internal/game/battleship.go:128-129`: `shipCells PlayerMap[[]Position]` is a field on `BattleshipSession`, not on `BattleshipSharedState`. The shared state (`battleship.go:97-101`) only has `Attacks`. |
| **Suggested action** | Place the winner-verification assertion SUT-side in `Protocol.report()`. For Battleship, the assertion would check `hitCount(winner) == totalShipCells`. For TTT/Connect4, it checks the board for the winning pattern. This is consistent with the catalog's proposal but the Battleship limitation should be noted as a hard constraint on workload-side approaches. |

### F3: no-double-elo-update — Assertion Requires Persistent Tracking State

| | |
|---|---|
| **Properties** | no-double-elo-update |
| **Scope** | Property-specific |
| **Concern** | The invariant "each session's ELO update is applied exactly once" requires tracking which session IDs have been processed. The proposed approach places an `Always` assertion in `ReportSessionResult` checking against a set of processed IDs. However, this tracking state must survive across calls. In the current code, `ReportSessionResult` is a method on `DB` — the tracking set would need to be a field on `DB` or a package-level variable. This is straightforward to implement but represents mutable assertion infrastructure that must be correctly synchronized (the DB methods can be called concurrently from HTTP handlers). |
| **Evidence** | `internal/matchmaker/db.go:262-323`: `ReportSessionResult` is called from `handleResult` (HTTP handler), `cancelExpiredSessions` (background goroutine), and `handleCancelSession` (HTTP handler). All three paths can execute concurrently. The tracking set needs a mutex or must be embedded in the SQLite transaction (e.g., `WHERE completed_at IS NULL` guard). |
| **Suggested action** | The simplest implementation is adding `WHERE completed_at IS NULL` to `updateSessionResult` SQL and checking `rows_affected == 0` to detect duplicates. The assertion fires on the duplicate detection. This avoids managing a separate in-memory tracking set. Alternatively, a `sync.Map` on `DB` tracks processed session IDs with an `Always` assertion on the second occurrence. |

### F4: player-reconnect-works — Workload Must Actually Disconnect and Reconnect

| | |
|---|---|
| **Properties** | player-reconnect-works |
| **Scope** | Property-specific |
| **Concern** | The catalog says this is a workload-side assertion checked after reconnection. But the current player workload does not intentionally disconnect and reconnect — reconnection only happens on network errors. Antithesis can inject network partitions between workload and gameserver, which would trigger reconnection in `Session.Run()` (session.go:61-77). However, the workload assertion needs to detect that a reconnection occurred (not just a fresh connection) and then validate the received state. The player's `Session` struct doesn't currently track reconnection count or expose reconnection events. |
| **Evidence** | `internal/player/session.go:55-77`: The `Run` loop retries `dial` + `bridge` on non-close errors. There is no callback or flag indicating a reconnection happened. The assertion would need to be placed in the `Run` loop after the second `dial` succeeds. |
| **Suggested action** | Add a reconnection counter to `Session`. After a successful reconnection (second+ iteration of the Run loop), assert that the first message received on `protocolRx` is a valid state message. Alternatively, rely on Antithesis network fault injection to trigger reconnections and place a SUT-side `Reachable` assertion on the reconnection path in `handleConn` (when `existing, ok := p.players[pid]; ok`). |

### F5: orphaned-session-handled — Triggering Requires Precise Timing

| | |
|---|---|
| **Properties** | orphaned-session-handled |
| **Scope** | Property-specific |
| **Concern** | This race requires a player to unqueue during the window between `collectMatches` (lock released) and `publishMatch` (lock re-acquired), which spans a Fleet HTTP round-trip. The evil player's `QueueAbandonRate` (0.05) creates throwaway queue entries but does not unqueue existing players at the right moment — it queues a *new* random player and never polls, which would not trigger this race. The race requires the matched player's *original* queue entry to disappear. This could happen if the player's context is cancelled (swarm shutdown) or if the player's poll loop somehow removes them. Since the workload uses `Unqueue` (DELETE /queue/{pid}), not just stopping polling, unqueue during the race window is possible but requires Antithesis to cancel a player context at exactly the right moment. |
| **Evidence** | `internal/matchmaker/match_queue.go:154-183`: `publishMatch` checks `_, hasA := q.queued[a.pid]`. Player A must be deleted from `queued` between `collectMatches` return and `publishMatch` lock acquisition. `internal/player/behavior.go:49`: `doQueueAbandon` queues a *new* random pid, does not unqueue the current player. |
| **Suggested action** | This is still implementable — Antithesis can pause player processes or inject network partitions that cause player context cancellation, and the matcher's unlock-network-call-relock window is a natural race point. However, note that the workload does not have an explicit mechanism to trigger this race. The `Sometimes` assertion is appropriate because it may not fire in every run. Consider whether the evil player should also have a "mid-match unqueue" behavior to increase the probability. |

### F6: fleet-failover — Single Game Server Limits Failover Testing

| | |
|---|---|
| **Properties** | fleet-failover |
| **Scope** | Property-specific |
| **Concern** | The deployment topology has only one game server. Fleet failover logic (try next server after 503 or network error) is only meaningful with multiple servers. With one server, if it returns 503 or is partitioned, the fleet returns `ErrNoServersAvailable` — it never exercises the "skip this server, try the next one" path. The `retryAt` backoff logic is exercised (server enters and exits retry state), but the core failover behavior (routing to an alternate server) is not tested. |
| **Evidence** | Deployment topology: "Single game server: Simpler state space." `internal/matchmaker/fleet.go:96-139`: The loop over `candidates` only exercises one iteration with a single server. |
| **Suggested action** | The `AlwaysOrUnreachable` assertion is still valid — it checks that a *successful* CreateSession used a non-backoff server, and with one server this will be trivially true when the server is available. The assertion passes but does not test the interesting failover path. To actually test multi-server failover, the topology would need a second game server container. If that is out of scope, document that this property is partially tested (retry-backoff only, not cross-server failover). |

### F7: session-eventually-completes — Liveness Depends on Test Duration

| | |
|---|---|
| **Properties** | session-eventually-completes, matchmaking-progress, result-reported-to-matchmaker, player-sees-terminal-state, expired-sessions-cancelled |
| **Scope** | Catalog-wide (all liveness properties) |
| **Concern** | All `Sometimes` assertions require the event to actually occur during the test run. With a 2-minute session timeout and Antithesis runs that are typically 10-30 minutes, most sessions should complete. However, if Antithesis heavily partitions the network early, sessions could be created but never played, only expiring via deadline. The `Sometimes(session completed)` assertion for `session-eventually-completes` would still fire (via cancellation), but `Sometimes(terminal state received)` for `player-sees-terminal-state` might not fire if players are always partitioned from the game server. Similarly, `result-reported-to-matchmaker` requires the gameserver-to-matchmaker link to be functional for at least some results. |
| **Evidence** | Session timeout is 2 minutes (deployment topology). Turn timeout is 10 seconds. With 7 well-behaved players, sessions should complete quickly (TTT in ~5 moves, Connect4 in ~20 moves, Battleship in ~50 moves). The `Sometimes` semantics mean the assertion only needs to fire once across the entire run. |
| **Suggested action** | This is acceptable for Antithesis testing. The `Sometimes` assertion semantics handle this correctly — if the event never occurs, the assertion reports as "not reached" rather than failing, which is informative. Ensure the test commands launch players early (the `first_start_swarm` template does this) so sessions begin before Antithesis ramps up fault injection. No changes needed. |

### F8: elo-zero-sum — Rounding Could Violate Zero-Sum

| | |
|---|---|
| **Properties** | elo-zero-sum |
| **Scope** | Property-specific |
| **Concern** | `CalcElo` uses `math.Round` independently on both new ratings. The formula `newWinner = Round(winner + K*(score - expected))` and `newLoser = Round(loser + K*((1-score) - (1-expected)))` simplifies to `newLoser = Round(loser - K*(score - expected))`. Since `Round(a) + Round(-a) == 0` when `a` is not exactly at `.5`, this is generally safe, but `Round(x.5) = Round(x+1) - 1` cases could theoretically break zero-sum by +/-1. The existing test checks specific cases but not exhaustively. |
| **Evidence** | `internal/elo.go:26-27`: Two independent `math.Round` calls. `internal/elo_test.go:63-66`: Zero-sum checked for test cases. The formula simplification shows `delta_winner + delta_loser = Round(K*(s-e)) + Round(-K*(s-e))` which could differ from 0 by 1 due to rounding of exactly-half values. |
| **Suggested action** | The SUT-side assertion `(newPlayer - oldPlayer) + (newOpponent - oldOpponent) == 0` placed in `ReportSessionResult` will catch any rounding violation at runtime under real ELO distributions. This is straightforward to implement. If a violation is found, the fix is to compute one delta and negate it rather than computing two independent rounded values. The assertion is fully implementable. |

### F9: third-player-rejected — Error Channel Not Closed Creates Goroutine Leak

| | |
|---|---|
| **Properties** | third-player-rejected |
| **Scope** | Property-specific |
| **Concern** | The catalog says the connection receives an error and is not added to the players map. The implementation (`protocol.go:188`) sends the error on the channel but never closes it. The write goroutine spawned in `Join` (`session.go:231-239`) ranges over `stateCh` and will block after delivering the error, creating a goroutine leak per rejected connection. The assertion is implementable (check that `len(p.players)` didn't increase), but the evidence notes a real bug (goroutine leak) that the property would expose. The assertion itself is straightforward. |
| **Evidence** | `internal/gameserver/protocol.go:186-189`: Third player case sends error but does not close the channel. `internal/gameserver/session.go:231-239`: Write goroutine ranges over `stateCh`, will block forever after error delivery unless the websocket write fails. |
| **Suggested action** | The assertion is implementable as specified. Additionally, the evidence suggests closing the channel after sending the error in the "too many players" case would fix the goroutine leak. This is a potential real bug the property testing could expose. |

### F10: no-elo-change-on-cancel — Cross-Request State Comparison Needed

| | |
|---|---|
| **Properties** | no-elo-change-on-cancel |
| **Scope** | Property-specific |
| **Concern** | The catalog's invariant says "when a session is cancelled, neither player's ELO changes." Verifying this requires knowing the players' ELO *before* the session started and comparing it to their ELO *after* cancellation. The SUT-side assertion in `ReportSessionResult` only sees `cancelled == true` and skips the ELO update — it cannot detect the race where a non-cancelled result was already processed for the same session (see no-double-elo-update). The simpler version (assert the code path skips the update when `cancelled == true`) is trivially implementable. The stronger version (assert DB state didn't change) requires snapshotting player ELOs before and after, which is complex for an inline assertion. |
| **Evidence** | `internal/matchmaker/db.go:277-283`: The `if !cancelled` guard correctly skips ELO updates for cancelled sessions. But the race scenario in the evidence file shows that if a non-cancelled result was processed first, the ELO *already changed* before the cancel call arrives. The cancel assertion would pass (it didn't change ELO) while the overall property is violated (session appears cancelled but ELO was changed by the earlier non-cancelled call). |
| **Suggested action** | The inline assertion in `ReportSessionResult` when `cancelled == true` is necessary but insufficient. The stronger property requires an end-of-session check: query the DB for the session's final state and both players' stats, verify consistency. This is better implemented as a workload test command (`eventually_check_leaderboard` could be extended) or as a post-hoc assertion that checks all cancelled sessions in the DB have no associated ELO changes. The no-double-elo-update property's idempotency fix would also resolve this. |

### F11: max-sessions-reached — Depends on Workload Pressure vs Session Duration

| | |
|---|---|
| **Properties** | max-sessions-reached |
| **Scope** | Property-specific |
| **Concern** | The `Reachable` assertion fires when `ErrMaxSessions` is returned. With `MaxSessions=50` and 7 well-behaved players (plus 1 evil), at most 4 concurrent sessions can exist (8 players / 2 per session). The game server will never reach 50 sessions. The topology config would need `MaxSessions` to be lowered (e.g., to 4 or 5) for this assertion to fire. |
| **Evidence** | Deployment topology: `-max-sessions=50`. Workload: 7 well-behaved + 1 evil = 8 players = max 4 concurrent sessions. 4 < 50, so capacity is never reached. |
| **Suggested action** | Lower `MaxSessions` in the Antithesis deployment config to a value close to the concurrent session count (e.g., 4-6). Alternatively, increase the player count significantly. The current topology makes this reachability assertion unreachable. |

### F12: draw-outcome-reached — AI Strategy Affects Reachability

| | |
|---|---|
| **Properties** | draw-outcome-reached |
| **Scope** | Property-specific |
| **Concern** | Draws are only possible in TicTacToe and Connect4. The TTT AI uses a simple strategy (build lines, block opponent) that frequently produces draws. Connect4 draws are much rarer due to the 7x6 board. Both AIs have randomized elements that should eventually produce draws in TTT. However, the evil player's chaos rate (30%) means ~30% of evil moves are corrupted, which are rejected — the game continues with the evil player's turn. This doesn't prevent draws but reduces the AI's effectiveness, potentially changing game outcomes. |
| **Evidence** | `internal/game/tictactoe.go:128-167`: TTT AI blocks and builds, likely produces draws. `internal/game/connect4.go:121-170`: Connect4 AI uses shuffled column preference, draws are rare on 7x6 board. |
| **Suggested action** | The `Reachable` assertion is appropriate. TTT draws are common enough that this should fire within a few games. No changes needed. |

---

## Catalog-Wide Findings

### CW1: No Antithesis SDK in Codebase — All Assertions Must Be Added

| | |
|---|---|
| **Scope** | Catalog-wide |
| **Concern** | The existing-assertions analysis confirms zero Antithesis SDK usage. Every property in the catalog requires new instrumentation. The Go SDK (`antithesis-sdk-go`) must be imported into both SUT services (matchmaker, gameserver) and the workload (player/swarm). This is a prerequisite for all 22 properties. |
| **Evidence** | `antithesis/scratchbook/existing-assertions.md`: "No Antithesis SDK assertions were found in the codebase." |
| **Suggested action** | Start implementation by adding the SDK dependency to `go.mod` and creating a shared assertion helper package. Prioritize SUT-side assertions (they cover more properties) over workload-side assertions. |

### CW2: SUT-Side vs Workload-Side Assertion Placement is Generally Sound

| | |
|---|---|
| **Scope** | Catalog-wide |
| **Concern** | The catalog correctly identifies which assertions need SUT-side placement (game rules, ELO, capacity, session lifecycle) vs workload-side (board validation from player perspective, terminal state observation, matchmaking progress). The single-goroutine protocol design means most game state assertions can be placed SUT-side without synchronization concerns. |
| **Suggested action** | No changes needed. The placement decisions are well-reasoned. |

### CW3: Existing log.Panicf/log.Fatal Calls Can Be Augmented, Not Replaced

| | |
|---|---|
| **Scope** | Catalog-wide |
| **Concern** | The 6 existing `log.Panicf`/`log.Fatal` calls act as implicit assertions but kill the process. In Antithesis, process death is a valid signal but it prevents further exploration. The catalog should note that these should be augmented with `Always(false, ...)` assertions *before* the panic, so Antithesis records the violation even if the process restarts. |
| **Evidence** | `antithesis/scratchbook/existing-assertions.md`: Lists 6 panic/fatal locations. |
| **Suggested action** | For each existing panic/fatal, add an `Always(false, "descriptive message")` call immediately before the panic. This ensures Antithesis captures the event in its property database regardless of process restart behavior. |

---

## Passes

### P1: Game Rule Properties Are Cleanly Implementable SUT-Side
The three game-rule properties (`game-rules-enforced`, `turn-order-maintained`, `correct-winner-detection` for TTT/Connect4) can be asserted in the Protocol's single goroutine with no synchronization overhead. The state is local, mutations are sequential, and the assertion sites are clear (`MakeMove` return, `CanMakeMove` check, `report()` method).

### P2: Session Capacity Enforcement Is Straightforward
`session-capacity-enforced` places an `Always` assertion inside a mutex-protected section (`CreateSession`). The check `len(s.sessions) <= MaxSessions` is trivially implementable after the session is added to the map. The mutex guarantees atomic check-and-insert.

### P3: ELO Zero-Sum SUT-Side Assertion Is Clean
The `elo-zero-sum` assertion can be placed immediately after `CalcElo` returns both new values and before the DB writes. The delta check `(newPlayer - oldPlayer) + (newOpponent - oldOpponent) == 0` requires no external state. Fully implementable.

### P4: Session Cleanup Assertion Is Trivial
`session-cleanup-complete` checks `_, ok := s.sessions[sid]; !ok` after `delete(s.sessions, sid)`. The mutex is held. Trivially implementable.

### P5: Reachability Assertions Are All Straightforward
All 6 reachability properties (`all-game-types-played`, `draw-outcome-reached`, `turn-timeout-fires`, `evil-move-rejected`, `session-deadline-fires`, `max-sessions-reached` — with caveat from F11) are single-site `Reachable` calls at well-defined code locations. No state tracking or cross-component coordination needed.

### P6: Liveness Assertions Use Correct `Sometimes` Semantics
The liveness properties (`session-eventually-completes`, `expired-sessions-cancelled`, `result-reported-to-matchmaker`, `matchmaking-progress`, `player-sees-terminal-state`) correctly use `Sometimes` rather than `Always`. This handles the reality that not every session will complete via the desired path in every run.

### P7: No-Duplicate-Match Is Implementable in publishMatch
The `no-duplicate-match` property checks that players being moved to `matched` are not already there. The assertion site is inside the `mu`-protected `publishMatch` function. Adding `Always(q.matched[a.pid] == nil && q.matched[b.pid] == nil, ...)` before the map writes is clean and correct.

### P8: Evil Player Workload Covers Needed Chaos Patterns
The evil player implementation provides malformed JSON, out-of-bounds moves, out-of-turn moves, and extra connection attempts. These directly exercise the `evil-move-rejected`, `third-player-rejected`, `turn-order-maintained`, and `game-rules-enforced` properties. The workload design supports the needed operation sequences.

### P9: Network Topology Supports Key Fault Scenarios
The three-container topology (matchmaker, gameserver, workload) allows Antithesis to partition any pair, which supports: fleet retry/backoff (matchmaker-gameserver partition), player reconnection (workload-gameserver partition), result delivery failure (gameserver-matchmaker partition), and queue polling resilience (workload-matchmaker partition).

---

## Uncertainties

### U1: Timer Behavior Under Antithesis Time Manipulation
Several properties depend on Go timers (`time.Timer`, `time.Ticker`): turn timeout, session deadline, session monitor, match interval. Antithesis can manipulate virtual time or pause processes, which could affect timer behavior in unexpected ways. It is unclear whether Antithesis's Go instrumentation intercepts `time.NewTimer`/`time.NewTicker` or whether timers fire based on wall-clock time. If timers use wall-clock time and Antithesis pauses processes, timers could fire immediately when the process resumes, creating bursts of timer events.

**Properties affected**: `turn-timeout-fires`, `session-deadline-fires`, `expired-sessions-cancelled`, `session-eventually-completes`.

### U2: SQLite Behavior Under Process Restart
The deployment uses file-backed SQLite (`/data/matchmaker.db`). If Antithesis restarts the matchmaker process, the DB state persists but in-memory state (`MatchQueue.queued`, `MatchQueue.matched`) is lost. Sessions in the DB that were active before the restart will have no corresponding in-memory `matched` entries. The session monitor will eventually cancel them. It is unclear whether this state divergence between DB and in-memory structures could cause assertion false positives (e.g., `no-duplicate-match` checking `matched` map which was just reset to empty).

**Properties affected**: `no-duplicate-match`, `no-double-elo-update`, `no-elo-change-on-cancel`.

### U3: Workload Spectator Client Not Yet Implemented
`spectator-state-consistency` and `board-state-valid` (workload-side variant) require a spectator client in the workload that connects to game sessions via WebSocket and validates received states. The current workload only has player clients and no spectator logic. Implementing a spectator client is additional workload code that needs to be written. The catalog mentions this as workload-side but does not acknowledge the implementation gap.

**Properties affected**: `spectator-state-consistency`, `board-state-valid` (workload-side variant).

### U4: Reporter Re-Enqueue Creates Unbounded Retry Loop
The Reporter re-enqueues results on temporary error (`reporter.go:72-75`) back to the same channel it reads from. If the matchmaker is unreachable for an extended period, the same result bounces between the channel and `submitResult` indefinitely. Each iteration consumes a channel slot and an HTTP attempt. It is unclear whether this could fill the `resultCh` (capacity `MaxSessions = 50`) and block protocol goroutines from sending their results, which would stall games. This affects the `result-reported-to-matchmaker` property's liveness guarantee.

**Properties affected**: `result-reported-to-matchmaker`, `session-eventually-completes`.

### U5: CalcElo Draw Path Updates Both Players From Winner's Perspective
In `ReportSessionResult` (`db.go:296-319`), the loop `for i, player := range players` breaks after the first player that matches the winner or when `draw == true`. For draws, it enters the block on the first player (`draw || player.PlayerID == winner` — draw is true), computes ELO from that player's perspective, and updates both players. The second iteration never executes due to `break`. This is correct if `CalcElo` is symmetric for draws, but it means the first player in the `players` slice is always treated as the "winner" argument to `CalcElo` for draws. The query order depends on the DB join, which may not be deterministic. Whether this subtlety affects the `elo-zero-sum` property depends on whether `CalcElo(a, b, true)` produces the same net deltas regardless of argument order.

**Properties affected**: `elo-zero-sum`.
