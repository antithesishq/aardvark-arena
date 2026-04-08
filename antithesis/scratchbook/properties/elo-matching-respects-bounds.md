# elo-matching-respects-bounds

## Evidence

The `MatchElo` function (`internal/elo.go:35-40`) computes whether two players' ELO difference is within bounds:
```go
eloDiff := math.Abs(float64(a - b))
waitTime := math.Max(time.Since(entryA).Seconds(), time.Since(entryB).Seconds())
relaxedDiff := eloDiff - (EloDiffRelaxRate * math.Max(0, waitTime))
return relaxedDiff <= MaxEloDiff
```

The matching algorithm in `collectMatches` (`internal/matchmaker/match_queue.go:108`) calls this and skips pairs that don't match. However, there's no assertion at the output boundary verifying that every published match satisfied the ELO constraint.

## Code Paths

- `collectMatches` — `match_queue.go:82-120` — reads ELO from candidate struct
- `MatchElo` — `elo.go:35-40` — pure function, uses current time for wait calculation
- `candidate.elo` — set from `player.Elo` when queuing (`match_queue.go:218-219`)

## Potential Violation Scenario

A player's ELO changes between when they queued and when they're matched. The `candidate.elo` is updated on each `Queue` call (`match_queue.go:215: existing.elo = player.Elo`), reading the current DB value. If a player completes a game that dramatically changes their ELO between match cycles, the cached `candidate.elo` may be stale for one cycle.

However, since `Queue` is called every poll interval and updates the cached ELO, the window is small. The real question is whether `collectMatches` uses stale ELO values from the *start* of its scan if another goroutine updates ELO mid-scan. Since `collectMatches` holds `mu` for the entire scan, and ELO updates happen via `ReportSessionResult` (which doesn't touch the queue), this is safe. But an assertion would provide defense-in-depth.

## Instrumentation Status

**NOT COVERED** — An `Always` in `publishMatch` (after confirming both players are still queued) verifying `MatchElo(a.elo, b.elo, a.entry, b.entry)` would close this gap.
