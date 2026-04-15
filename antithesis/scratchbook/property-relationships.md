# Property Relationships

## Clusters

### Cluster 1: Game Rule Correctness
**Properties**: `game-rules-enforced`, `correct-winner-detection`, `turn-order-maintained`, `board-state-valid`, `spectator-receives-valid-state`

These properties form a hierarchy:
- `turn-order-maintained` is a prerequisite for `game-rules-enforced` (if turn order is violated, the board state is likely invalid).
- `game-rules-enforced` and `board-state-valid` cover different failure classes: `game-rules-enforced` catches mutation bugs in `MakeMove`, while `board-state-valid` catches serialization or structural corruption. Neither dominates the other.
- `spectator-receives-valid-state` is a SUT-side version of `board-state-valid` placed in `BroadcastState`, catching validation gaps before state reaches any observer.
- `correct-winner-detection` depends on valid board state — winner detection checks the board.
- `evil-move-rejected` is an exercise mechanism for this cluster — it generates the invalid inputs that stress-test rule enforcement.

**Shared code paths**: `CanMakeMove`, `MakeMove`, `Protocol.handleMove`, `BroadcastState`.

Note: These properties operate on a single-goroutine protocol, so their Antithesis value is primarily catching serialization or state corruption bugs, not concurrency issues.

### Cluster 2: ELO Integrity
**Properties**: `elo-zero-sum`, `no-elo-change-on-cancel`, `no-double-elo-update`, `leaderboard-reflects-games`

These properties protect the rating system from different angles:
- `no-double-elo-update` is the strongest per-transaction check — if results are processed exactly once, ELO zero-sum follows naturally from CalcElo correctness.
- `no-elo-change-on-cancel` is independent — it can be violated even if double-update is prevented (a cancel that races with a normal completion could produce ELO changes on a session that ends up marked as cancelled).
- `leaderboard-reflects-games` is the global end-to-end check that catches cumulative drift from any cause, including per-transaction bugs that the other properties miss.
- `correct-winner-detection` feeds into this cluster — wrong winner means wrong ELO update target.

**Shared code paths**: `ReportSessionResult`, `CalcElo`, `cancelExpiredSessions`, `Leaderboard`.

### Cluster 3: Session Lifecycle
**Properties**: `session-eventually-completes`, `expired-sessions-cancelled`, `result-reported-to-matchmaker`, `session-cleanup-complete`, `session-capacity-enforced`, `protocol-not-blocked`

These form a lifecycle chain:
- `session-eventually-completes` → `result-reported-to-matchmaker` → ELO update (Cluster 2).
- `expired-sessions-cancelled` is the safety net when normal completion fails.
- `session-cleanup-complete` recovers capacity after session ends.
- `session-capacity-enforced` is the upstream guard that prevents resource exhaustion.
- `protocol-not-blocked` ensures the protocol goroutine remains responsive even when result delivery is delayed. Without this, sessions can appear hung despite timers firing.
- `orphaned-session-handled` and `ghost-player-sessions-cleaned-up` are special cases where sessions are created but players don't connect.

**Shared code paths**: `Protocol.RunToCompletion`, `Protocol.report`, `Reporter.submitResult`, `SessionManager.cleanup`, `DB.cancelExpiredSessions`.

### Cluster 4: Matchmaking
**Properties**: `no-duplicate-match`, `matchmaking-progress`, `orphaned-session-handled`, `ghost-player-sessions-cleaned-up`

- `no-duplicate-match` is a safety property protecting the queue invariant.
- `matchmaking-progress` is a liveness property requiring the entire matchmaking pipeline to work.
- `orphaned-session-handled` is a failure-mode property for the race between matching and session creation.
- `ghost-player-sessions-cleaned-up` covers the evil player's queue-abandon behavior creating phantom entries.

**Shared code paths**: `matchPlayers`, `collectMatches`, `publishMatch`, `Fleet.CreateSession`.

### Cluster 5: Connection Resilience
**Properties**: `player-reconnect-works`, `third-player-rejected`, `player-sees-terminal-state`

- `player-reconnect-works` and `third-player-rejected` both test `handleConn`.
- `player-sees-terminal-state` depends on connection health at game end.

**Shared code paths**: `Protocol.handleConn`, `sessionHandle.Join`, `sendLatest`.

### Cluster 6: System Recovery
**Properties**: `player-recovers-after-matchmaker-restart`, `fleet-failover`

- `player-recovers-after-matchmaker-restart` tests the hardest recovery scenario — volatile state loss with persistent state retained.
- `fleet-failover` tests game server unavailability and retry-backoff recovery.

**Shared code paths**: `matchmaker.New()`, `Fleet.CreateSession`, `NewDB`.

### Cluster 7: Reachability Targets
**Properties**: `all-game-types-played`, `draw-outcome-reached`, `turn-timeout-fires`, `evil-move-rejected`, `session-deadline-fires`, `max-sessions-reached`

These are exploration hints rather than correctness properties. They share no code paths with each other but guide Antithesis toward interesting regions of the state space.

## Cross-Cluster Dependencies

| Upstream | Downstream | Relationship |
|----------|-----------|--------------|
| Cluster 1 (Game Rules) | Cluster 2 (ELO) | Correct winners feed correct ELO updates |
| Cluster 3 (Session Lifecycle) | Cluster 2 (ELO) | Results must be reported for ELO to update |
| Cluster 4 (Matchmaking) | Cluster 3 (Session Lifecycle) | Matches create sessions |
| Cluster 5 (Connection) | Cluster 3 (Session Lifecycle) | Connected players needed for normal game completion |
| Cluster 6 (System Recovery) | Cluster 4 (Matchmaking) | Recovery restores matchmaking ability |
| Cluster 7 (Reachability) | All clusters | Exploration hints improve coverage of all other clusters |

## Suspected Dominance

- `no-double-elo-update` likely implies `elo-zero-sum` (if each result is processed once, and CalcElo is zero-sum, then the overall ELO changes are zero-sum).
- `leaderboard-reflects-games` is the global check that catches failures missed by per-transaction properties.
- `session-eventually-completes` partially implies `session-cleanup-complete` (completion triggers cleanup, though cleanup could still fail independently).
- `protocol-not-blocked` is a prerequisite for `session-eventually-completes` (a blocked protocol goroutine prevents timer processing).
