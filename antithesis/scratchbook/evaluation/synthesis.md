# Evaluation Synthesis

## Categorized Findings

### Refinements (apply directly)

**R1: Downgrade game rule cluster priority** (Antithesis Fit F1)
Properties: `game-rules-enforced`, `turn-order-maintained`, `board-state-valid`, `correct-winner-detection`
Action: Keep as lightweight `Always` assertions for defense-in-depth. Note in catalog that these are low Antithesis-value due to single-goroutine protocol design. The real value is catching serialization or state corruption bugs, not concurrency issues.

**R2: Fix `max-sessions-reached` feasibility** (Implementability F11)
Property: `max-sessions-reached`
Action: Update deployment topology to set `MaxSessions=6` so the 7-player swarm can actually hit capacity.

**R3: Strengthen `player-sees-terminal-state`** (Wildcard F4)
Property: `player-sees-terminal-state`
Action: Change from `Sometimes` to `AlwaysOrUnreachable` — when a player is connected at game end, they must receive the terminal state.

**R4: Fix dominance claim** (Wildcard F8)
Action: Correct property-relationships.md to remove the claim that `game-rules-enforced` dominates `board-state-valid`. They cover different failure classes (mutation bugs vs serialization bugs).

**R5: Clarify `draw-outcome-reached` scope** (Wildcard F7)
Property: `draw-outcome-reached`
Action: Clarify in catalog that this applies to TicTacToe and Connect4 only. Add note that Battleship draws are impossible by game rules.

**R6: Note `correct-winner-detection` Battleship limitation** (Implementability F2)
Property: `correct-winner-detection`
Action: Note in catalog that Battleship winner verification requires SUT-side assertion (private `shipCells` state not visible to workload).

**R7: Note `no-double-elo-update` is a known code deficiency** (Antithesis Fit F4)
Property: `no-double-elo-update`
Action: Add note that this property will likely fail immediately due to missing idempotency guard. This is intentional — validates that Antithesis can find this class of bug.

### Gaps (fill via targeted discovery)

**G1: No global ELO conservation property** (Coverage Balance F1, Wildcard F1)
Multiple lenses flagged that per-transaction zero-sum doesn't catch cumulative drift from double-updates.
Action: Add `elo-global-conservation` property.

**G2: No matchmaker restart recovery property** (Coverage Balance F2, Wildcard F5)
In-memory state lost on restart while SQLite persists. Players stuck in limbo.
Action: Add `player-recovers-after-matchmaker-restart` liveness property.

**G3: No spectator workload component** (Coverage Balance F3, Implementability U3, Wildcard F12)
`spectator-state-consistency` requires a spectator client that doesn't exist in the workload.
Action: Add spectator test command to deployment topology. Lower priority since spectating is read-only.

**G4: No result channel backpressure property** (Antithesis Fit F2, Wildcard F6)
Protocol goroutine can deadlock if `resultCh` fills.
Action: Add `protocol-not-blocked` safety property.

**G5: Ghost queue entries from evil players** (Wildcard F3)
Evil player `QueueAbandonRate` creates phantom queue entries that get matched to real sessions.
Action: Add `ghost-player-sessions-cleaned-up` liveness property.

**G6: No health/crash recovery property** (Coverage Balance F5, Wildcard F11)
Panic paths crash the matchmaker. No property verifies restart.
Action: Add `services-remain-healthy` liveness property.

**G7: Leaderboard validation not in catalog** (Coverage Balance F6)
Test command exists but no property.
Action: Add `leaderboard-reflects-games` safety property (subsumes G1).

### Biases (escalate to user)

**B1: Single game server topology limits fleet failover testing** (Implementability F6)
The deployment topology has one game server. The `fleet-failover` property exists but the "skip server, try next" path is never exercised — only retry-backoff entry/exit. Adding a second game server container would increase state space but enable fuller fleet testing.

Question for user: Should we add a second game server container to the deployment topology? This trades Antithesis exploration efficiency for fleet failover coverage.

## Actions Taken

### Refinements Applied

All refinements (R1-R7) have been applied to the property catalog, evidence files, and property relationships.

### Gaps Filled

Added 5 new properties to the catalog:

1. **`leaderboard-reflects-games`** — Safety: Sum of all player ELOs equals `count(players) * DefaultElo`. Subsumes G1 and G7.
2. **`player-recovers-after-matchmaker-restart`** — Liveness: After matchmaker restart, players can re-queue and get matched. (G2, G6)
3. **`protocol-not-blocked`** — Safety: The protocol goroutine's select loop processes timer events within bounded time. (G4)
4. **`ghost-player-sessions-cleaned-up`** — Liveness: Sessions created for ghost queue entries are eventually cancelled. (G5)
5. **`spectator-receives-valid-state`** — Moved from workload-side to SUT-side assertion to remove dependency on spectator workload client. (G3)

### Biases Escalated

B1 escalated above. Recommendation: keep single game server for initial Antithesis runs. The fleet retry-backoff path is still exercised when Antithesis injects network faults between matchmaker and game server. Add a second game server in a follow-up if fleet bugs are a concern.
