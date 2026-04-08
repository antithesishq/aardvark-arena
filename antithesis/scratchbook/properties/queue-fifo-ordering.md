# queue-fifo-ordering

## Evidence

`sortedQueuedCandidates` (`internal/matchmaker/match_queue.go:124-136`) sorts candidates by entry time (oldest first), with player ID as a deterministic tie-breaker:
```go
sort.Slice(candidates, func(i, j int) bool {
    if candidates[i].entry.Equal(candidates[j].entry) {
        return candidates[i].pid.String() < candidates[j].pid.String()
    }
    return candidates[i].entry.Before(candidates[j].entry)
})
```

The matching algorithm then iterates this sorted list, pairing the first compatible candidate it finds for each player.

## Code Paths

- `sortedQueuedCandidates` — `match_queue.go:124-136` — sorting
- `collectMatches` — `match_queue.go:82-120` — iterates sorted candidates

## Potential Violation Scenario

The sort is correct for the data present at the time of the call. However, `entry` is set to `time.Now()` when a player first queues (`match_queue.go:219`). If two players queue within the same nanosecond (or with truncated time resolution), the UUID tie-breaker provides determinism but not fairness.

This is a minor concern — the FIFO guarantee is about preventing starvation, and the ELO relaxation over time ensures long-waiting players eventually match even with higher ELO differences.

## Instrumentation Status

**NOT COVERED** — An assertion in `collectMatches` verifying that the candidates slice is sorted would provide verification. Low priority since the sort is straightforward.
