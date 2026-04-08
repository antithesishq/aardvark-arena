# turn-timer-forces-completion

## Evidence

The turn timer is created in `NewProtocol` (`protocol.go:78`) with `turnTimeout * 2` as initial duration (grace period for connection). When `turnTimer.C` fires in `RunToCompletion` (`protocol.go:128-135`):

```go
case <-p.turnTimer.C:
    if len(p.players) == 0 {
        p.turnTimer.Reset(p.turnTimeout * 2)
        continue
    }
    p.report(p.state.CurrentPlayer.Opponent().Wins())
    break outer
```

The current player's opponent wins (the current player is the one who timed out).

The timer is reset after each valid move (`protocol.go:246`): `p.turnTimer.Reset(p.turnTimeout)`.

## Code Paths

- Timer creation: `protocol.go:78` — `time.NewTimer(turnTimeout * 2)`
- Timer check: `protocol.go:128-135` — fires if no valid move arrives in time
- Timer reset: `protocol.go:246` — after each successful MakeMove
- Timer NOT reset: on invalid moves (malformed JSON, semantic errors, out-of-turn)

## Edge Cases

1. Timer fires at exactly the same moment a move arrives: `select` in Go chooses randomly among ready cases, so either the move is processed or the timer fires. Both outcomes are valid.
2. No players connected: Timer fires but is reset (grace period for connection). This loops until players connect or the session deadline hits.
3. Evil players sending only invalid moves: Timer is not reset, so it fires after `turnTimeout` and the evil player's opponent wins.

## Instrumentation Status

**NOT COVERED** — No assertion verifies the specific outcome when the turn timer fires. An `Always` checking that the reported winner is the opponent of the current player when the turn timer triggers would close this gap.
