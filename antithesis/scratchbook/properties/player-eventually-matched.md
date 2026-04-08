# player-eventually-matched

## Evidence

The matchmaking cycle:
1. Player PUTs `/queue/{pid}` — enters queue (`match_queue.go:200-226`)
2. Matcher goroutine runs `matchPlayers` every 500ms (Antithesis config)
3. `collectMatches` sorts candidates by wait time, pairs by ELO compatibility
4. `MatchElo` relaxes ELO bounds over time (`EloDiffRelaxRate = 50/sec`)

The ELO relaxation ensures that even extreme ELO mismatches eventually become compatible. With `MaxEloDiff=200` and `EloDiffRelaxRate=50/sec`, a 1000-ELO difference would be relaxed after `(1000-200)/50 = 16 seconds` of waiting.

## Code Paths

- `MatchQueue.Queue` — `match_queue.go:200-226` — adds to queue
- `MatchQueue.collectMatches` — `match_queue.go:82-120` — pairs candidates
- `MatchElo` — `elo.go:35-40` — relaxation calculation

## Failure Scenarios

1. **All game servers in backoff**: `matchPlayers` skips pairs when `CreateSession` returns `ErrNoServersAvailable`. Players stay in queue. Resolved when servers recover.
2. **Odd number of players**: One player has no pair. Resolved when another player joins.
3. **ELO too extreme**: Relaxation handles this within seconds.
4. **Evil player queue abandonment**: Orphaned queue entries from R16 behavior. These entries will never be polled, so they'll stay in the queue indefinitely, consuming a slot. They do get paired eventually (since they're valid candidates), creating a session that times out.

## Instrumentation Status

**PARTIALLY COVERED** — R17 confirms players sometimes receive matches. `eventually_queue_empty.sh` validates post-test matching. A `Sometimes` or `eventually_` check verifying that all active players complete at least one game would strengthen this.
