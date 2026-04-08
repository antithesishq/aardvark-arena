---
commit: 0db32cd5760f7547845837c4bcd1ebe342a45707
updated: 2026-04-07
---

# Property Catalog

## Category: Session Lifecycle Integrity

Properties ensuring game sessions are created, played, and completed correctly.

### cancelled-session-no-winner — Cancelled Sessions Never Declare a Winner

| | |
|---|---|
| **Type** | Safety |
| **Property** | If a session is cancelled, the result must have winner == uuid.Nil. |
| **Invariant** | `Always(!cancelled \|\| winner == uuid.Nil)` — checked at three code points (matchmaker HTTP handler, matchmaker DB layer, gameserver reporter). Already instrumented as A1, A2, A4. The assertion type matches because this is a universal invariant that must hold on every evaluation. |
| **Antithesis Angle** | Fault injection during the result-reporting path: network partitions between game server and matchmaker, crash of game server mid-report, concurrent session cancellation by the session monitor while a result is in flight. |
| **Why It Matters** | A cancelled session with a winner would corrupt ELO ratings and game history. |
| **Existing Coverage** | **FULLY COVERED** — A1, A2, A4 already assert this at all relevant code points. |

### session-capacity-respected — Active Sessions Never Exceed Max

| | |
|---|---|
| **Type** | Safety |
| **Property** | The number of active sessions on a game server never exceeds `MaxSessions`. |
| **Invariant** | `Always(activeSessions <= maxSessions)` — checked in health endpoint. Already instrumented as A3. The `Always` type fits because the condition must hold at every health check. |
| **Antithesis Angle** | Rapid concurrent session creation requests arriving at the same game server while sessions are also completing. Race between `CreateSession` acquiring the lock and checking `len(s.sessions)`. |
| **Why It Matters** | Exceeding capacity could cause resource exhaustion and degraded service. |
| **Existing Coverage** | **FULLY COVERED** — A3 asserts this in the health endpoint. R7 and R8 confirm the capacity-full path is reached. |

### session-always-two-players — Every Session Has Exactly Two Players

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a non-cancelled session completes and ELO is updated, it always has exactly two players in the `player_session` table. |
| **Invariant** | `Unreachable` if player count != 2. Already instrumented as U1. The `Unreachable` type is correct because the 2-player invariant is structural — it should be impossible to reach a state where it's violated. |
| **Antithesis Angle** | Concurrent session creation with the same player (race in matchmaking), database corruption under concurrent writes, orphaned player_session rows from failed transactions. |
| **Why It Matters** | ELO calculation assumes exactly two players. A mismatch causes incorrect ratings or a crash. |
| **Existing Coverage** | **FULLY COVERED** — U1 asserts this at `internal/matchmaker/db.go:304`. |

### elo-non-negative — ELO Ratings Never Go Negative

| | |
|---|---|
| **Type** | Safety |
| **Property** | No player's ELO rating ever becomes negative after a session result is processed. |
| **Invariant** | `Always(newWinner >= 0 && newLoser >= 0)` — would be placed in `ReportSessionResult` after `CalcElo`. The `Always` type is correct because negative ELO is never acceptable. |
| **Antithesis Angle** | Long chains of losses for low-ELO players, combined with draws. The ELO formula can theoretically produce negative values with extreme rating differentials. Antithesis explores many more game sequences than unit tests. |
| **Why It Matters** | Negative ELO is semantically meaningless and could cause display/sorting bugs. |
| **Existing Coverage** | **NOT COVERED** — No assertion exists on ELO bounds. `internal/elo.go` has no assertions. |

### elo-conservation — ELO Changes Are Zero-Sum Per Match

| | |
|---|---|
| **Type** | Safety |
| **Property** | For every completed non-cancelled session, the total ELO change across both players sums to zero (within rounding tolerance). |
| **Invariant** | `Always(abs(deltaWinner + deltaLoser) <= 1)` — placed in `ReportSessionResult` after computing new ratings. `Always` is correct because ELO is defined as zero-sum. Rounding tolerance of 1 accounts for integer arithmetic. |
| **Antithesis Angle** | Exercise all game outcomes (win, loss, draw) across diverse ELO differentials. Antithesis generates varied match sequences that may trigger rounding edge cases the unit tests don't cover. |
| **Why It Matters** | If ELO is not conserved, ratings drift and the leaderboard becomes meaningless over time. |
| **Existing Coverage** | **NOT COVERED** — No assertion exists. `internal/elo_test.go` has unit tests but no conservation property. |

### result-reported-for-every-session — Every Session Eventually Gets a Result

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Every session that is created in the matchmaker DB eventually has `completed_at` set (either via normal completion or timeout cancellation). |
| **Invariant** | `Sometimes(true)` on a workload check that verifies no uncompleted sessions remain after all players finish. This is partially covered by the `eventually_queue_empty.sh` test driver, but a more targeted assertion in the workload or SUT could verify the DB directly. |
| **Antithesis Angle** | Network partitions between game server and matchmaker preventing result delivery. Game server crash before reporting. Reporter channel full. Session monitor's periodic sweep is the backstop. |
| **Why It Matters** | Orphaned sessions leave players stuck and prevent them from re-queuing. |
| **Existing Coverage** | **PARTIALLY COVERED** — `eventually_queue_empty.sh` validates end-to-end flow works post-test, but doesn't directly check for orphaned sessions in the DB. S6 asserts sessions sometimes expire. |

### no-duplicate-result-application — Session Results Are Not Applied Twice

| | |
|---|---|
| **Type** | Safety |
| **Property** | A session result (ELO update) is applied at most once. If `ReportSessionResult` is called again for the same session, ELO is not recalculated. |
| **Invariant** | `AlwaysOrUnreachable` — placed at the start of `ReportSessionResult`: if the session already has `completed_at` set, the function should bail without modifying ELO. Currently the UPDATE is unconditional. This is a potential bug, not just a missing assertion. |
| **Antithesis Angle** | Reporter retries on temporary error (re-enqueues to `resultCh`), but the first attempt may have actually succeeded. The second call applies ELO changes again. Network partitions that cause timeouts but not actual failures are the trigger. |
| **Why It Matters** | Double application of ELO changes corrupts ratings. |
| **Existing Coverage** | **NOT COVERED** — No idempotency check exists in `ReportSessionResult`. The UPDATE at `db.go:283` is unconditional. |

## Category: Matchmaking Correctness

Properties ensuring the matchmaking algorithm works correctly.

### matched-players-removed-from-queue — Matched Players Are Cleared From Queue

| | |
|---|---|
| **Type** | Safety |
| **Property** | After a match is published, neither player remains in the `queued` map. They are in `matched` or neither (if they left). |
| **Invariant** | `Always` — after `publishMatch` succeeds (hasA && hasB), both players must be removed from `queued` and present in `matched`. Already implemented in the code logic but not explicitly asserted. A SUT-side `Always` after the delete/insert operations would add verification. |
| **Antithesis Angle** | Concurrent `Unqueue` calls racing with `publishMatch`. The lock-release window between `collectMatches` and `publishMatch` is where races happen. |
| **Why It Matters** | A player left in both `queued` and `matched` could be matched to a second session. |
| **Existing Coverage** | **NOT COVERED** — The code logic handles this correctly, but there's no assertion verifying the postcondition. R2 confirms the race window is reached. |

### elo-matching-respects-bounds — Players Only Match Within ELO Bounds

| | |
|---|---|
| **Type** | Safety |
| **Property** | When two players are matched, their ELO difference does not exceed `MaxEloDiff` (200) plus the time-based relaxation. |
| **Invariant** | `Always(MatchElo(a.elo, b.elo, a.entry, b.entry))` — would be placed in `collectMatches` after a pair is selected. The `Always` type is correct because every match must satisfy ELO bounds. |
| **Antithesis Angle** | Players with extreme ELO differentials queueing simultaneously. Time manipulation by Antithesis could affect the wait-time relaxation calculation. Many concurrent matches where ELO changes between when a player is read and when they're matched. |
| **Why It Matters** | Unfair matches degrade the competitive experience and make ELO meaningless. |
| **Existing Coverage** | **NOT COVERED** — `MatchElo` is called but not asserted at the match-output boundary. |

### queue-fifo-ordering — Longest-Waiting Players Match First

| | |
|---|---|
| **Type** | Safety |
| **Property** | The matching algorithm considers players in queue-entry-time order (oldest first), preventing starvation of long-waiting players. |
| **Invariant** | `Always` — the sorted candidates slice should be ordered by entry time. Could be asserted after `sortedQueuedCandidates()`. |
| **Antithesis Angle** | Many players queueing simultaneously (same entry time), UUID tie-breaking behavior, players repeatedly queueing and unqueueing. |
| **Why It Matters** | Without FIFO ordering, players could starve while newer arrivals get matched. |
| **Existing Coverage** | **NOT COVERED** — The code sorts correctly (`match_queue.go:129-136`) but no assertion verifies this property under concurrent access. |

## Category: Game Protocol Correctness

Properties ensuring the game protocol handles moves, turns, and outcomes correctly.

### turn-alternation — Players Alternate Turns Correctly

| | |
|---|---|
| **Type** | Safety |
| **Property** | After a valid move by the current player, `CurrentPlayer` switches to the opponent (except in Battleship where hits give extra turns). |
| **Invariant** | `Always` — after each successful `MakeMove`, the state's `CurrentPlayer` is either the opponent (standard) or the same player (Battleship hit). Would need to be checked in `protocol.handleMove` after `session.MakeMove` succeeds. |
| **Antithesis Angle** | Rapid move submissions, evil players sending out-of-turn moves, race between turn timer and move arrival. |
| **Why It Matters** | Broken turn alternation means one player could make all moves, corrupting game integrity. |
| **Existing Coverage** | **NOT COVERED** — `CanMakeMove` rejects out-of-turn moves, but there's no assertion that the turn transition itself is correct. R25 confirms out-of-turn moves are attempted. |

### invalid-moves-never-change-state — Invalid Moves Don't Alter Game State

| | |
|---|---|
| **Type** | Safety |
| **Property** | If `MakeMove` returns an error, the game state is unchanged from before the call. |
| **Invariant** | `Always(stateAfterError == stateBefore)` — would be placed in `protocol.handleMove` when `MakeMove` returns an error. The game implementations currently return the pre-mutation state on error, but a deep-equality check would verify this. |
| **Antithesis Angle** | Evil players sending all forms of invalid moves (malformed JSON, out-of-bounds, occupied cells, wrong phase). Antithesis exercises many combinations of invalid input sequences. |
| **Why It Matters** | A partially-applied invalid move would corrupt the game board. |
| **Existing Coverage** | **NOT COVERED** — R11 and R12 confirm invalid moves are reached, but no assertion checks state preservation. |

### game-terminates — Every Game Eventually Reaches a Terminal State

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Every `RunToCompletion` call eventually exits with a terminal game status (P1Win, P2Win, Draw, or Cancelled). |
| **Invariant** | `Sometimes(true)` — on the result message emission in `protocol.report()`. S11-S13 already cover some terminal states. The `Sometimes` type is appropriate because this is a progress property. |
| **Antithesis Angle** | Turn timer + session deadline provide hard guarantees, but what if the protocol loop gets stuck? Evil players sending streams of invalid moves that reset the turn timer. |
| **Why It Matters** | A stuck game session blocks players and consumes server resources. |
| **Existing Coverage** | **PARTIALLY COVERED** — S11, S12, S13 assert that draws, cancellations, and wins sometimes occur. But there's no per-session assertion that every started game reaches a terminal state. |

### turn-timer-forces-completion — Turn Timer Terminates Stalled Games

| | |
|---|---|
| **Type** | Safety |
| **Property** | If no valid move is received within `turnTimeout`, the waiting player loses and the game ends. |
| **Invariant** | `Always` — when the `turnTimer` fires and players are connected, the result should be the opponent's win status. This is implemented in the code (`protocol.go:134-135`) but not asserted. |
| **Antithesis Angle** | Player disconnects right as turn timer fires. Evil player sends invalid moves that don't reset the timer. Two players connected but neither sends a valid move. |
| **Why It Matters** | Without turn timer enforcement, a stalled player blocks the game indefinitely. |
| **Existing Coverage** | **NOT COVERED** — The code implements this but there's no assertion on the specific outcome when the turn timer fires. |

## Category: Fleet and Server Management

Properties ensuring the fleet correctly manages game servers.

### fleet-recovery-after-backoff — Fleet Recovers Servers After Backoff

| | |
|---|---|
| **Type** | Liveness |
| **Property** | After a game server enters backoff (temporary error or capacity full), it eventually becomes a candidate again for session creation. |
| **Invariant** | `Sometimes(true)` — placed when a previously-backed-off server is selected as a candidate again. The `Sometimes` type fits because this is a progress/recovery property. |
| **Antithesis Angle** | All game servers simultaneously entering backoff. Time progression during Antithesis exploration. Server recovery after transient network partitions. |
| **Why It Matters** | If servers permanently stay in backoff, effective capacity drops and matchmaking stalls. |
| **Existing Coverage** | **NOT COVERED** — R4 confirms "no servers available" is reached. R5 confirms temporary failures occur. But no assertion verifies recovery. |

### fleet-returns-valid-sessions — Fleet Only Returns Successfully Created Sessions

| | |
|---|---|
| **Type** | Safety |
| **Property** | `Fleet.CreateSession` only returns a non-nil `SessionInfo` when the game server responded with HTTP 200. |
| **Invariant** | `Always` — already implemented in code logic (returns on 200, continues on 503, errors on other). U2 guards the unexpected-status path. |
| **Antithesis Angle** | Malformed HTTP responses, connection resets mid-response, race between timeout and response. |
| **Why It Matters** | Returning a session that wasn't actually created would send players to a nonexistent game. |
| **Existing Coverage** | **PARTIALLY COVERED** — U2 asserts unexpected status codes are unreachable. S8 asserts successful creation sometimes happens. But no assertion directly ties the return value to a 200 response. |

## Category: Result Reporting and Data Integrity

Properties ensuring game results are correctly reported and persisted.

### reporter-retries-temporary-errors — Reporter Retries on Temporary Failures

| | |
|---|---|
| **Type** | Liveness |
| **Property** | When the reporter encounters a temporary transport error, it re-enqueues the result for retry. |
| **Invariant** | `Reachable` — already instrumented as R13. The `Reachable` type is correct because we want to confirm this retry path is exercised. |
| **Antithesis Angle** | Network partitions between game server and matchmaker. Antithesis can inject faults to trigger temporary errors and verify the retry path. |
| **Why It Matters** | Without retries, game results could be permanently lost during transient network issues. |
| **Existing Coverage** | **FULLY COVERED** — R13 asserts this path is reachable. |

### elo-updates-match-game-outcome — ELO Changes Reflect the Actual Game Outcome

| | |
|---|---|
| **Type** | Safety |
| **Property** | After a session with a winner, the winner's ELO increases (or stays same) and the loser's ELO decreases (or stays same). After a draw, both change symmetrically. |
| **Invariant** | `Always` — placed in `ReportSessionResult` after `CalcElo`. For wins: `Always(newWinner >= winnerElo)`. For draws: `Always(abs(newWinner - newLoser) <= abs(winnerElo - loserElo) + 1)` (ELO converges). |
| **Antithesis Angle** | Extreme ELO differentials (very high vs very low), many consecutive wins/losses, rapid session completion generating many concurrent ELO updates. |
| **Why It Matters** | Incorrect ELO movement undermines the entire ranking system. |
| **Existing Coverage** | **NOT COVERED** — `internal/elo_test.go` has unit tests but no in-SUT assertion on ELO direction correctness. |

## Category: Connection Handling

Properties ensuring WebSocket connections and player sessions are managed correctly.

### reconnect-preserves-player-assignment — Reconnecting Players Keep Their Player Number

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a player reconnects to an in-progress session, they retain their original player assignment (P1 or P2). |
| **Invariant** | `Always(reconnectedPlayer == existingPlayer)` — placed in `protocol.handleConn` when replacing an existing connection. Currently the code preserves `existing.player` in the new `playerConn`. |
| **Antithesis Angle** | Player WebSocket drops and reconnects during active game. Evil player's extra-connect chaos triggers connection replacement. Multiple rapid reconnections. |
| **Why It Matters** | If a reconnecting player gets a different player number, they'd be playing the wrong side of the board. |
| **Existing Coverage** | **PARTIALLY COVERED** — R9 confirms reconnection is reached. The code preserves the player assignment, but no assertion verifies it. |

### third-player-rejected — Sessions Reject Connections Beyond Two Players

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a session already has two distinct connected players, any additional connection from a new player ID receives an error and is not added to the game. |
| **Invariant** | `Always(len(players) <= 2)` — could be added as an invariant check in `handleConn`. R10 already confirms this path is reached. |
| **Antithesis Angle** | Evil players' extra-connect chaos generates random-ID connection attempts. Multiple evil players targeting the same session simultaneously. |
| **Why It Matters** | A third player in a two-player game would corrupt the game state. |
| **Existing Coverage** | **PARTIALLY COVERED** — R10 confirms extra connections are rejected. The code logic handles it correctly, but no `Always` asserts the player count invariant. |

## Category: System Recovery

Properties ensuring the system recovers from failures.

### services-recover-to-healthy — All Services Eventually Become Healthy

| | |
|---|---|
| **Type** | Liveness |
| **Property** | After faults are stopped, all services (matchmaker + 3 game servers) eventually respond to health checks. |
| **Invariant** | Checked by `eventually_health_check.sh` test driver. `Sometimes` would also be appropriate as an SDK assertion if health checks were instrumented. |
| **Antithesis Angle** | Process crashes, network partitions, resource exhaustion during testing. Recovery after all faults are lifted. |
| **Why It Matters** | A system that doesn't recover after transient faults is not production-ready. |
| **Existing Coverage** | **FULLY COVERED** — `eventually_health_check.sh` validates this. |

### post-fault-game-completion — System Can Complete Games After Faults

| | |
|---|---|
| **Type** | Liveness |
| **Property** | After all test drivers complete and faults are stopped, two fresh players can queue, match, play a game, and complete successfully. |
| **Invariant** | Checked by `eventually_queue_empty.sh` test driver. |
| **Antithesis Angle** | Validates that accumulated state corruption from testing doesn't prevent future operations. |
| **Why It Matters** | Even if individual sessions fail during fault injection, the system must remain functional afterward. |
| **Existing Coverage** | **FULLY COVERED** — `eventually_queue_empty.sh` validates this. |

## Category: Reachability and Coverage

Properties ensuring diverse system behaviors are exercised.

### all-game-types-played — All Three Game Types Are Played

| | |
|---|---|
| **Type** | Reachability |
| **Property** | During a test run, sessions of all three game types (Tic-Tac-Toe, Connect4, Battleship) are created and played. |
| **Invariant** | `Reachable` — already instrumented as R21, R22, R23 (one per game type). |
| **Antithesis Angle** | Random game selection might skew toward one type. Antithesis-controlled RNG ensures diverse selection. |
| **Why It Matters** | All game implementations need testing, not just the most commonly selected one. |
| **Existing Coverage** | **FULLY COVERED** — R21, R22, R23 individually assert each game type is played. |

### all-game-outcomes-observed — All Terminal States Are Reached

| | |
|---|---|
| **Type** | Reachability |
| **Property** | During a test run, games end in all possible terminal states: P1Win, P2Win, Draw, and Cancelled. |
| **Invariant** | `Sometimes` — already partially instrumented as S11 (draws), S12 (cancellations), S13 (wins). Win coverage doesn't distinguish P1Win vs P2Win. |
| **Antithesis Angle** | With enough games and fault injection, all outcomes should be reached. Evil players increase cancellation rate. |
| **Why It Matters** | Each terminal state exercises different code paths (ELO calculation, no-op for draw, cleanup for cancellation). |
| **Existing Coverage** | **MOSTLY COVERED** — S11, S12, S13 cover the major categories. No separate assertion for P1Win vs P2Win. |

### evil-behavior-exercised — Evil Player Behaviors Are All Reached

| | |
|---|---|
| **Type** | Reachability |
| **Property** | During a test run, all evil player behaviors are exercised: bad moves, out-of-turn moves, malformed JSON, extra connections, queue abandonment. |
| **Invariant** | `Reachable` — already instrumented as R16, R25, R26, R27. Additionally R11 and R12 confirm invalid moves reach the server. |
| **Antithesis Angle** | Evil player rates are configured to ensure all behaviors fire. Antithesis-controlled RNG may influence which behaviors trigger. |
| **Why It Matters** | Adversarial behavior is the primary mechanism for surfacing protocol robustness issues. |
| **Existing Coverage** | **FULLY COVERED** — R16, R25, R26, R27 cover all evil behaviors. |

## Category: Session Timeout and Deadline Management

### session-deadline-enforced — Expired Sessions Are Eventually Cancelled

| | |
|---|---|
| **Type** | Liveness |
| **Property** | Any session that passes its deadline without completing is eventually marked as cancelled by the session monitor. |
| **Invariant** | `Sometimes(len(expired) > 0)` — already instrumented as S6. The `Sometimes` type is appropriate because this is a progress property that should happen at least once in a sufficiently long run. |
| **Antithesis Angle** | Sessions that stall due to player disconnection, network partitions preventing result delivery, game server crashes. The 2-second monitor interval in Antithesis config means deadlines are checked frequently. |
| **Why It Matters** | Unreaped expired sessions leak resources and prevent players from re-queuing. |
| **Existing Coverage** | **FULLY COVERED** — S6 asserts expiration sometimes occurs. |

### game-winner-has-winning-condition — Game Winners Have a Valid Winning Condition on the Board

| | |
|---|---|
| **Type** | Safety |
| **Property** | When a game ends with P1Win or P2Win, the game board contains a valid winning condition for the winner (3-in-a-row for TicTacToe, 4-in-a-row for Connect4, all opponent ships sunk for Battleship). |
| **Invariant** | `Always` — checked in the protocol's `report()` function when `status` is P1Win or P2Win. Game-specific: verify `checkWinFor(winner)` for TicTacToe, `checkWinAt(lastCol, lastRow, winner)` for Connect4, `hitCount(winner) == totalShipCells` for Battleship. |
| **Antithesis Angle** | Evil players sending corrupt moves may trigger edge cases in win detection. Antithesis explores game state space more thoroughly than unit tests. The combination of valid and invalid moves in rapid succession could expose win-detection bugs. |
| **Why It Matters** | A false win declaration would end the game prematurely and award incorrect ELO. |
| **Existing Coverage** | **NOT COVERED** — No assertion verifies that the win condition on the board matches the declared winner. The game implementations check this internally but there's no independent verification at the protocol level. |
| **Note** | Gap-fill from Coverage Balance evaluation. |

### player-eventually-matched — Queued Players Eventually Receive a Match

| | |
|---|---|
| **Type** | Liveness |
| **Property** | A player who remains in the matchmaker queue eventually receives a session assignment, assuming at least one other player is also queued and game servers are available. |
| **Invariant** | `Sometimes(true)` — in the player loop when a match assignment is received. Already partially instrumented as R17 ("players sometimes receive a new match assignment"). A stronger form would be an `eventually_` test command that verifies no players remain in the queue after all drivers complete. |
| **Antithesis Angle** | Under fault injection, game servers may be unavailable or in backoff. The ELO relaxation over time (`EloDiffRelaxRate = 50/sec`) should ensure that even mismatched players eventually pair. Network partitions between players and matchmaker could prevent polling. |
| **Why It Matters** | A stuck player represents a permanent resource leak and a broken user experience. |
| **Existing Coverage** | **PARTIALLY COVERED** — R17 confirms players sometimes get matched. `eventually_queue_empty.sh` validates post-test matching works. But no assertion checks that all players who want to match eventually do. |
| **Note** | Gap-fill from Coverage Balance evaluation. |

### no-completed-session-expires — Completed Sessions Are Not Re-Cancelled

| | |
|---|---|
| **Type** | Safety |
| **Property** | The session monitor never selects a session that already has `completed_at` set. The SQL query `WHERE completed_at IS NULL` ensures this, but concurrent completion during the monitor scan could cause a race. |
| **Invariant** | `Always` — in `cancelExpiredSessions`, after fetching expired session IDs, each call to `ReportSessionResult` should verify the session is still uncompleted. This ties into the `no-duplicate-result-application` property. |
| **Antithesis Angle** | Tight timing between a game completing normally and the session monitor scanning. The 2-second monitor interval and 1-minute session timeout create a window where both could fire near-simultaneously. |
| **Why It Matters** | Re-cancelling a completed session would overwrite the legitimate result and corrupt ELO. |
| **Existing Coverage** | **NOT COVERED** — The SQL query filters correctly, but there's no assertion guarding against TOCTOU between the SELECT and the UPDATE. This is closely related to `no-duplicate-result-application`. |
