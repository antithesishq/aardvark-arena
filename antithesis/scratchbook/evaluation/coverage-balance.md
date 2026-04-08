# Evaluation Lens: Coverage Balance

## Assessment Against SUT Analysis

### Well-Covered Areas

1. **Session lifecycle** (6 properties): `cancelled-session-no-winner`, `session-capacity-respected`, `session-always-two-players`, `result-reported-for-every-session`, `session-deadline-enforced`, `no-completed-session-expires` — comprehensive coverage of the session creation-to-completion path.

2. **ELO system** (4 properties): `elo-non-negative`, `elo-conservation`, `elo-updates-match-game-outcome`, `elo-matching-respects-bounds` — good coverage of the rating system.

3. **Game protocol** (4 properties): `turn-alternation`, `invalid-moves-never-change-state`, `game-terminates`, `turn-timer-forces-completion` — solid coverage of in-game correctness.

4. **System recovery** (4 properties): `services-recover-to-healthy`, `post-fault-game-completion`, `fleet-recovery-after-backoff`, `session-deadline-enforced` — recovery paths are well-represented.

### Gaps

**Gap 1: No properties for specific game logic correctness**
The SUT analysis identifies three game implementations (TicTacToe, Connect4, Battleship) with distinct rules. The catalog has no properties verifying game-specific invariants:
- TicTacToe: no assertion that a winning condition (3 in a row) is correctly detected
- Connect4: no assertion that gravity works (pieces fall to lowest row)
- Battleship: no assertion that ships are placed without overlap, or that hit/miss tracking is accurate

These are partially covered by unit tests (`game/battleship_test.go`, `game/harness_test.go`), but Antithesis could exercise them under more diverse conditions.

**Gap 2: Orphaned session cleanup**
The SUT analysis identifies orphaned sessions (created on game server but players never join) as a known failure mode. No property explicitly verifies that these are cleaned up. The session deadline provides a backstop, but no assertion checks that orphaned sessions don't accumulate indefinitely.

**Gap 3: Queue starvation/liveness**
While `result-reported-for-every-session` covers session completion, no property verifies that queued players eventually get matched. A player could remain in the queue indefinitely if:
- Their ELO is too extreme (though relaxation over time should prevent this)
- All game servers are in backoff permanently
- The matcher goroutine crashes

### Type Balance

| Type | Count | Assessment |
|------|-------|------------|
| Safety (Always) | 10 | Good — covers core invariants |
| Liveness (Sometimes) | 5 | Adequate — covers recovery and progress |
| Reachability | 3 | Good — evil behaviors and game types |

The balance is reasonable. Safety properties dominate because the system's primary risks are data corruption and state inconsistency. Liveness properties cover recovery. Reachability properties ensure the workload exercises diverse paths.

### Component Balance

| Component | Properties | Assessment |
|-----------|-----------|------------|
| Matchmaker | 8 | Well-covered (queue, matching, DB, results) |
| Game Server | 7 | Well-covered (sessions, protocol, connection) |
| Fleet | 3 | Adequate (recovery, creation, capacity) |
| Player/Workload | 3 (reachability) | Adequate for driving coverage |
| Game Logic | 0 | **Gap** — no game-specific invariants |

### Missing from Catalog That SUT Analysis Flagged

1. **SQLite foreign key non-enforcement** — identified as an assumption but no property tests whether FK violations actually occur
2. **Reporter channel blocking** — SUT analysis flagged `resultCh` blocking under high load, but no property exercises this
3. **`sendLatest` message dropping** — SUT analysis flagged the drop-and-retry pattern for watch channels, but no property verifies spectator correctness

## Findings Summary

| Scope | Finding | Suggested Action |
|-------|---------|-----------------|
| Catalog-wide | No game-specific invariants | Add 2-3 game logic properties |
| Catalog-wide | No queued-player liveness property | Add a property for player-eventually-matched |
| `no-duplicate-result-application` | Priority too low for its cross-cutting impact | Elevate to highest priority |
| Component balance | Game logic has zero properties | Gap-fill with game-specific invariants |
