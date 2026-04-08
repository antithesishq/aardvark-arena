# Evaluation Lens: Antithesis Fit

## Findings

### Property-Specific

**`queue-fifo-ordering` — Low Antithesis Value**
The FIFO ordering property is a deterministic sort correctness check. The `sortedQueuedCandidates` function uses `sort.Slice` with a well-defined comparator. This is better verified by a unit test with crafted inputs than by Antithesis fault injection. The Antithesis run doesn't add value because the sort behavior doesn't change under faults.

**Recommendation**: Downgrade to low priority. Keep in catalog for completeness but don't invest workload effort.

**`invalid-moves-never-change-state` — Moderate Antithesis Value**
While state preservation on error is a code correctness property (not timing-dependent), Antithesis adds value by exercising many diverse invalid move sequences that unit tests wouldn't construct. The evil player workload generates varied corrupt payloads. However, the core invariant (Go value semantics for struct copies) is a language guarantee, not a concurrency property.

**Recommendation**: Keep at medium priority. The value is in input diversity, not timing exploration.

**`no-duplicate-result-application` — High Antithesis Value (Underestimated)**
This property is at the intersection of timing and concurrency: the reporter retries on temporary errors, but temporary errors (TCP timeouts) are exactly what Antithesis injects via network faults. The retry-after-success scenario requires specific network timing that deterministic tests can't easily reproduce.

**Recommendation**: Elevate to P0. This is an Antithesis sweet spot — timing-sensitive, hard to reproduce deterministically, and has real data corruption consequences.

**`elo-conservation` — Low Antithesis Value**
ELO conservation is a mathematical property of the `CalcElo` function. It holds or doesn't hold based on the formula and rounding, independent of timing or faults. Better verified by property-based unit testing with random inputs.

**Recommendation**: Keep for completeness but note this is primarily a unit test property. Antithesis adds marginal value through input diversity but not through fault injection.

**`elo-non-negative` — Moderate Antithesis Value**
While the core formula is deterministic, Antithesis explores many more game sequences than unit tests, potentially reaching extreme ELO values through long chains of one-sided outcomes. The value is in exercising diverse game histories, not fault injection per se.

**Recommendation**: Keep at medium priority.

### Properties Well-Suited to Antithesis

The following properties are strongly in Antithesis's sweet spot:

- **`no-duplicate-result-application`**: Network fault timing
- **`matched-players-removed-from-queue`**: Concurrent queue modification race
- **`fleet-recovery-after-backoff`**: Time-based recovery after faults
- **`cancelled-session-no-winner`**: Concurrent cancellation and result reporting
- **`turn-timer-forces-completion`**: Timer/message race conditions
- **`reconnect-preserves-player-assignment`**: Connection drop/reconnect timing
- **`result-reported-for-every-session`**: Network partition recovery

## Passes

- All Reachability properties are good Antithesis fit (exploring code paths under fault injection is core Antithesis functionality).
- All System Recovery properties are good Antithesis fit.
- Session lifecycle properties leverage Antithesis's strength in timing exploration.

## Uncertainties

- Whether `elo-updates-match-game-outcome` would be more productively tested by exhaustive property-based unit testing or by Antithesis. Antithesis adds game-sequence diversity but the ELO formula itself is deterministic.
