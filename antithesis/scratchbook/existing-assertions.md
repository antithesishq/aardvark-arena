# Existing Antithesis SDK Assertions

No Antithesis SDK assertions were found in the codebase.

The codebase does not import any Antithesis SDK packages. There are no calls to `assert_always`, `assert_sometimes`, `assert_reachable`, `assert_unreachable`, or their Go equivalents (`Always`, `Sometimes`, `Reachable`, `Unreachable`).

The `go.mod` module path is `github.com/antithesishq/aardvark-arena` — this is the project's own module path, not an import of the Antithesis SDK.

## Existing Runtime Checks (Non-Antithesis)

The codebase has several `log.Fatal` / `log.Panicf` calls that act as runtime assertions:

| Location | Condition | Message |
|----------|-----------|---------|
| `internal/matchmaker/db.go:279` | `cancelled && winner != uuid.Nil` | `BUG: received cancelled result with a winner set` |
| `internal/gameserver/protocol.go:148` | `msg.conn != nil && msg.move != nil` | `BUG: both move and conn are set` |
| `internal/gameserver/protocol.go:209` | Move from player not in `p.players` map | `BUG: move from disconnected player` |
| `internal/gameserver/session.go:215` | Unknown game kind | `unsupported game kind` |
| `internal/matchmaker/match_queue.go:75` | Non-ErrNoServersAvailable fleet error | `fleet error: %v` |
| `internal/matchmaker/match_queue.go:169` | DB error in publishMatch | `db error: %v` |

These could potentially be converted to Antithesis assertions when instrumenting the SUT.
