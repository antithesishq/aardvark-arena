# Property Relationships

## Cluster: Session Result Integrity

Properties related to correct session outcomes and their reporting.

| Property | Role in Cluster |
|----------|----------------|
| `cancelled-session-no-winner` | Core invariant: cancelled sessions have no winner |
| `no-duplicate-result-application` | Guard: results are applied at most once |
| `no-completed-session-expires` | Specific case: session monitor doesn't overwrite completed sessions |
| `result-reported-for-every-session` | Liveness: every session eventually gets a result |
| `reporter-retries-temporary-errors` | Mechanism: retry ensures delivery under transient failures |

**Connections:**
- `no-duplicate-result-application` dominates `no-completed-session-expires` — fixing the former (adding an idempotency check to `ReportSessionResult`) automatically fixes the latter.
- `reporter-retries-temporary-errors` is the mechanism that makes `no-duplicate-result-application` important — without retries, duplicates wouldn't occur.
- `cancelled-session-no-winner` could be violated if `no-duplicate-result-application` fails: a session completes with a winner, then gets overwritten as cancelled, breaking the inverse invariant.

## Cluster: ELO Correctness

Properties related to the rating system.

| Property | Role in Cluster |
|----------|----------------|
| `elo-non-negative` | Bound: ratings never go below zero |
| `elo-conservation` | Invariant: rating changes are zero-sum per match |
| `elo-updates-match-game-outcome` | Invariant: winners gain, losers lose |
| `elo-matching-respects-bounds` | Input constraint: matched players are within ELO range |

**Connections:**
- `elo-updates-match-game-outcome` implies `elo-non-negative` in practice (if winners always gain, ratings trend upward), but not formally (a player could still go negative through draws with lower-rated players).
- `elo-conservation` is independent of `elo-updates-match-game-outcome` — conservation holds even if the direction is wrong (though that would be a different bug).
- `no-duplicate-result-application` (from the Result Integrity cluster) directly affects all ELO properties — double application corrupts all of them.

## Cluster: Matchmaking Lifecycle

Properties related to the queue-match-assign cycle.

| Property | Role in Cluster |
|----------|----------------|
| `matched-players-removed-from-queue` | Invariant: matched players leave the queue |
| `queue-fifo-ordering` | Fairness: oldest players match first |
| `elo-matching-respects-bounds` | Constraint: matched players are compatible |

**Connections:**
- `matched-players-removed-from-queue` prevents double-matching, which would violate `session-always-two-players` (from the Game Protocol cluster).
- `queue-fifo-ordering` affects `elo-matching-respects-bounds` — if ordering is wrong, the algorithm might skip compatible pairs in favor of incompatible ones.

## Cluster: Game Protocol

Properties related to in-game correctness.

| Property | Role in Cluster |
|----------|----------------|
| `turn-alternation` | Invariant: correct turn order |
| `invalid-moves-never-change-state` | Invariant: errors don't corrupt state |
| `game-terminates` | Liveness: games end |
| `turn-timer-forces-completion` | Mechanism: stalled games are resolved |
| `session-always-two-players` | Precondition: games have exactly two players |
| `third-player-rejected` | Guard: extra connections don't corrupt game |

**Connections:**
- `turn-timer-forces-completion` is one mechanism that ensures `game-terminates`.
- `invalid-moves-never-change-state` + `turn-alternation` together ensure game state integrity — invalid moves don't corrupt state, and valid moves advance turns correctly.
- `session-always-two-players` and `third-player-rejected` protect the precondition for `turn-alternation` (which assumes exactly two players).

## Cluster: Connection Management

Properties related to WebSocket and player connections.

| Property | Role in Cluster |
|----------|----------------|
| `reconnect-preserves-player-assignment` | Invariant: reconnection doesn't change player role |
| `third-player-rejected` | Guard: extra connections rejected |

**Connections:**
- Both properties protect game protocol integrity by ensuring the player-to-role mapping is stable.
- `reconnect-preserves-player-assignment` handles the "known player returns" case; `third-player-rejected` handles the "unknown player arrives" case.

## Cluster: System Recovery

Properties related to post-fault behavior.

| Property | Role in Cluster |
|----------|----------------|
| `services-recover-to-healthy` | Liveness: services recover |
| `post-fault-game-completion` | Liveness: system functions after faults |
| `fleet-recovery-after-backoff` | Mechanism: game servers re-enter candidate pool |
| `session-deadline-enforced` | Mechanism: expired sessions are cleaned up |

**Connections:**
- `fleet-recovery-after-backoff` is a precondition for `post-fault-game-completion` — if game servers stay in backoff, new games can't be created.
- `session-deadline-enforced` ensures stale state is cleaned up, supporting `services-recover-to-healthy`.

## Cluster: Game Logic (Gap-Fill)

Properties verifying game-specific invariants.

| Property | Role in Cluster |
|----------|----------------|
| `game-winner-has-winning-condition` | Invariant: declared winners have valid board conditions |
| `all-game-types-played` | Coverage: all game implementations are exercised |
| `all-game-outcomes-observed` | Coverage: all terminal states are reached |

**Connections:**
- `game-winner-has-winning-condition` provides independent verification of what `turn-alternation` and `invalid-moves-never-change-state` protect at the move level.
- `all-game-types-played` is a precondition for `game-winner-has-winning-condition` having full coverage — if a game type is never played, its win condition is never checked.

## Cluster: Player Lifecycle (Gap-Fill)

| Property | Role in Cluster |
|----------|----------------|
| `player-eventually-matched` | Liveness: players get matched |
| `matched-players-removed-from-queue` | Invariant: matched players leave queue |
| `elo-matching-respects-bounds` | Constraint: matches are fair |

**Connections:**
- `player-eventually-matched` depends on `fleet-recovery-after-backoff` (from System Recovery) — if no servers are available, matching stalls.
- `matched-players-removed-from-queue` is a mechanism that enables `player-eventually-matched` — if matched players don't leave the queue, they block slots.

## Cluster: Server Capacity and Routing

| Property | Role in Cluster |
|----------|----------------|
| `session-capacity-respected` | Invariant: game servers don't exceed max sessions |
| `fleet-returns-valid-sessions` | Invariant: fleet only returns successfully created sessions |

**Connections:**
- `session-capacity-respected` is enforced by the game server; `fleet-returns-valid-sessions` is enforced by the fleet client. Together they ensure sessions are correctly allocated.
- Both feed into `game-terminates` — if sessions are incorrectly allocated, games may not start correctly.

## Standalone Properties

| Property | Reason |
|----------|--------|
| `evil-behavior-exercised` | Pure reachability coverage — ensures the adversarial workload exercises all evil behavior paths. Not tied to a specific correctness cluster. |

## Cross-Cluster Dependencies

- **Result Integrity -> ELO Correctness**: `no-duplicate-result-application` is a prerequisite for all ELO properties.
- **Matchmaking Lifecycle -> Game Protocol**: `matched-players-removed-from-queue` prevents scenarios that would violate `session-always-two-players`.
- **Connection Management -> Game Protocol**: `reconnect-preserves-player-assignment` and `third-player-rejected` protect the assumptions of `turn-alternation`.
- **System Recovery -> Matchmaking Lifecycle**: `fleet-recovery-after-backoff` ensures the matchmaking system can create new sessions.
- **Server Capacity -> System Recovery**: `session-capacity-respected` prevents resource exhaustion that could impair recovery.
