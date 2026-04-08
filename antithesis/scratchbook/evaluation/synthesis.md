# Evaluation Synthesis

## Refinements (Applied)

### R1: Elevate `no-duplicate-result-application` to highest priority
**Source:** Antithesis Fit, Coverage Balance
**Finding:** This property sits at the intersection of timing, network faults, and data integrity — Antithesis's sweet spot. It also has cross-cutting impact on all ELO properties. The Coverage Balance lens flagged its priority as too low.
**Action:** Updated priority framing in catalog. This property also requires a code fix (idempotency check), not just an assertion.

### R2: Downgrade `queue-fifo-ordering` to low priority
**Source:** Antithesis Fit
**Finding:** FIFO ordering is a deterministic sort correctness check, better verified by unit tests. Antithesis fault injection doesn't add value.
**Action:** Noted in catalog. Keep for completeness but deprioritize in implementation.

### R3: Downgrade `elo-conservation` to low priority for Antithesis
**Source:** Antithesis Fit
**Finding:** Mathematical property of the formula, not timing/concurrency-sensitive. Better verified by property-based unit testing.
**Action:** Noted in catalog. Still worth adding as a SUT-side assertion for defense-in-depth.

### R4: Note `invalid-moves-never-change-state` implementation complexity
**Source:** Implementability
**Finding:** Deep-equality comparison adds complexity for marginal value. The Go value semantics guarantee this at the language level.
**Action:** Keep in catalog at medium priority but note that implementation may not justify the effort.

## Gaps (Filled)

### G1: Game-specific invariants
**Source:** Coverage Balance
**Finding:** Zero properties for game logic correctness. TicTacToe, Connect4, and Battleship have distinct rules that aren't verified.
**Action:** Added two gap-fill properties to the catalog:
- `game-winner-has-winning-condition` — when a game ends with a winner, the game board shows a valid winning condition
- `player-eventually-matched` — queued players eventually receive a match assignment

### G2: Queued-player liveness
**Source:** Coverage Balance
**Finding:** No property verifies that queued players eventually get matched.
**Action:** Added `player-eventually-matched` property.

### G3: Orphaned session cleanup
**Source:** Wildcard
**Finding:** Sessions created on game servers but abandoned by matchmaking are not verified to be cleaned up.
**Action:** Documented in wildcard findings. This is addressed by existing `session-deadline-enforced` property — orphaned sessions will timeout. Added note to that property's evidence file.

### G4: `sendLatest` player state drops
**Source:** Wildcard
**Finding:** The `sendLatest` function can drop player state updates under contention, but this is acceptable for the system's design (players poll and the latest state is sufficient).
**Action:** Documented as a design observation, not a property gap. The protocol is designed for "latest state wins" semantics.

## Biases

### B1: Safety-heavy catalog
**Observation:** 10 safety properties vs 5 liveness properties. The catalog is oriented toward "bad things never happen" rather than "good things eventually happen."
**Assessment:** This is appropriate for this system. The primary risks are data corruption (ELO, session results) and state inconsistency, which are safety concerns. The liveness properties (`game-terminates`, `result-reported-for-every-session`, `fleet-recovery-after-backoff`, `services-recover-to-healthy`, `post-fault-game-completion`) adequately cover the progress guarantees. No action needed.

### B2: Matchmaker-centric analysis
**Observation:** The matchmaker has the most properties (8) because it holds the most state (SQLite DB, queue, matched map). The game server has 7 properties. The game logic implementations have very few.
**Assessment:** This reflects the actual risk distribution — the matchmaker is the single point of failure and the source of persistent state. Adding game-specific properties (G1 above) partially addresses this. Presented to user for awareness.

## Summary

| Category | Count | Action |
|----------|-------|--------|
| Refinements | 4 | Applied to catalog |
| Gaps | 4 | 2 filled with new properties, 2 documented as design observations |
| Biases | 2 | Assessed as appropriate, presented for awareness |

## Final Property Count

After gap-filling: **27 properties** (25 original + 2 gap-fill)
- Safety: 14 (Always, AlwaysOrUnreachable, Unreachable)
- Liveness: 6 (Sometimes)
- Reachability: 3 (Reachable)
- Already fully covered by existing assertions: 10 (no new instrumentation needed)
- Needing new SUT-side assertions: 13
- Needing code changes beyond assertions: 2 (`no-duplicate-result-application`, `no-completed-session-expires`)
