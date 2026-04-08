# turn-alternation

## Evidence

Turn alternation is enforced by `State.CanMakeMove` (`internal/game/game.go:167-175`):
```go
func (s State[Shared]) CanMakeMove(player Player) error {
    if s.Status != Active { return fmt.Errorf("game is over") }
    if player != s.CurrentPlayer { return fmt.Errorf("not your turn") }
    return nil
}
```

Each `MakeMove` implementation calls `CanMakeMove` first, then after a successful move, switches `CurrentPlayer`:
- TicTacToe: `state.CurrentPlayer = state.CurrentPlayer.Opponent()` (`tictactoe.go:123`)
- Connect4: `state.CurrentPlayer = state.CurrentPlayer.Opponent()` (`connect4.go:116`)
- Battleship: switches on miss (`battleship.go:209`), stays on hit (extra turn)

The protocol in `handleMove` (`protocol.go:215-248`) calls `MakeMove` and handles errors. If MakeMove succeeds, the state is updated with the new CurrentPlayer.

## Code Paths

- `CanMakeMove` — `game.go:167-175` — guard
- Each game's `MakeMove` — switches CurrentPlayer on success
- `protocol.handleMove` — `protocol.go:215-248` — calls MakeMove, broadcasts state

## What Goes Wrong on Violation

If turn alternation breaks, one player could make consecutive moves, leading to an unfair game. In TicTacToe/Connect4, this would allow a player to fill the board while the opponent waits.

## Instrumentation Status

**NOT COVERED** — The logic is correct per code inspection, but no assertion verifies post-move that CurrentPlayer switched correctly. An `Always` in `handleMove` after a successful `MakeMove` checking that `newState.CurrentPlayer != oldState.CurrentPlayer` (for non-Battleship) or that the transition follows Battleship rules would provide verification.
