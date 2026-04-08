# Evaluation Lens: Implementability

## Assessment

### Fully Implementable Without Changes

Properties that can be implemented with existing assertions or simple additions:

- **`cancelled-session-no-winner`**: Already implemented (A1, A2, A4). No work needed.
- **`session-capacity-respected`**: Already implemented (A3). No work needed.
- **`session-always-two-players`**: Already implemented (U1). No work needed.
- **`reporter-retries-temporary-errors`**: Already implemented (R13). No work needed.
- **`services-recover-to-healthy`**: Already implemented (eventually_health_check.sh). No work needed.
- **`post-fault-game-completion`**: Already implemented (eventually_queue_empty.sh). No work needed.
- **`all-game-types-played`**: Already implemented (R21-R23). No work needed.
- **`evil-behavior-exercised`**: Already implemented (R16, R25-R27). No work needed.
- **`session-deadline-enforced`**: Already implemented (S6). No work needed.
- **`all-game-outcomes-observed`**: Already implemented (S11-S13). No work needed.

### Implementable With SUT-Side Additions

Properties that need new assertions added to the SUT code:

- **`elo-non-negative`**: Add `assert.Always(newWinner >= 0 && newLoser >= 0)` in `ReportSessionResult` after `CalcElo`. Simple, no code restructuring needed.
- **`elo-conservation`**: Add `assert.Always(abs(delta1 + delta2) <= 1)` in same location. Simple.
- **`elo-updates-match-game-outcome`**: Add `assert.Always(newWinner >= winnerElo)` for wins. Simple.
- **`matched-players-removed-from-queue`**: Add `assert.Always` after `publishMatch` promotion checking neither player is in `queued`. Simple.
- **`elo-matching-respects-bounds`**: Add `assert.Always(MatchElo(...))` in `publishMatch`. Simple.
- **`turn-alternation`**: Add `assert.Always` in `handleMove` after successful MakeMove. Needs game-specific logic (Battleship exception). Moderate complexity.
- **`reconnect-preserves-player-assignment`**: Add `assert.Always` in `handleConn` reconnection branch. Simple.
- **`third-player-rejected`**: Add `assert.Always(len(p.players) <= 2)` in `handleConn`. Simple.
- **`turn-timer-forces-completion`**: Add `assert.Always` checking winner is opponent of current player when timer fires. Simple.

### Requires Code Changes Beyond Assertions

- **`no-duplicate-result-application`**: Needs an idempotency check in `ReportSessionResult`: query the session's `completed_at` before the UPDATE, and bail if already completed. This is a functional change, not just an assertion. After the fix, add `AlwaysOrUnreachable` guarding the early-return path.
- **`no-completed-session-expires`**: Addressed by the same fix as above. The idempotency check in `ReportSessionResult` prevents the session monitor from overwriting completed sessions.

### Implementable From Workload Only

- **`game-terminates`**: Observable from the workload — each driver plays N sessions and exits. If a game doesn't terminate, the session timeout eventually fires, but the driver would observe a hang. No SUT-side change needed beyond what's already there.
- **`fleet-recovery-after-backoff`**: Can be observed via `Sometimes` on successful session creation after a period of failures. Needs a SUT-side assertion in `CreateSession` when a previously-backed-off server is re-selected. Moderate complexity (need to track whether the server was backed off).
- **`result-reported-for-every-session`**: Could be checked by a `finally_` or `eventually_` test command that queries the matchmaker for uncompleted sessions after drivers finish.
- **`queue-fifo-ordering`**: Would need an assertion in `collectMatches` checking sort order. Simple but low value.
- **`invalid-moves-never-change-state`**: Would need a before/after comparison in `handleMove`. The Go value semantics make this safe, so the assertion is more defensive than necessary. Moderate complexity for deep-equality check.

## Deployment Topology Compatibility

All properties are compatible with the existing topology:
- Network faults between game server and matchmaker: exercises `no-duplicate-result-application`, `fleet-recovery-after-backoff`, `reporter-retries-temporary-errors`
- Network faults between client and game server: exercises `reconnect-preserves-player-assignment`, `turn-timer-forces-completion`
- Process crashes: exercises `services-recover-to-healthy`, `post-fault-game-completion`

No properties require a different topology.

## Findings Summary

| Scope | Finding | Suggested Action |
|-------|---------|-----------------|
| `no-duplicate-result-application` | Requires code fix (idempotency check), not just assertion | Implement the fix, then add assertion |
| `invalid-moves-never-change-state` | Deep-equality comparison is complex for marginal value | Consider downgrading priority or simplifying to a hash comparison |
| `fleet-recovery-after-backoff` | Needs state tracking to distinguish "previously backed off" | Moderate implementation complexity |
