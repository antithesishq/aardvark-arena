# evil-move-rejected — Malicious Moves Are Handled Gracefully

## Evidence

Evil players corrupt moves in two ways (`internal/player/protocol.go:103-116`):
1. **Malformed JSON**: `{"evil":` — fails `json.Unmarshal` in `handleMove`.
2. **Invalid values**: `[999999,999999]` or `{"evil":true,"x":999999,"y":999999}` — passes unmarshal but fails game validation.

`handleMove` (`internal/gameserver/protocol.go:206-227`) catches both:
- Unmarshal error → `SendErr(pid, "invalid move: ...")`
- MakeMove error → `SendErr(pid, "invalid move: ...")`

In both cases, the state is NOT modified. The game continues.

## Relevant Code Paths
- `internal/player/protocol.go:103-116` — `corruptMove`
- `internal/gameserver/protocol.go:206-227` — `handleMove`
- `internal/player/behavior.go` — Chaos rates

## SUT Instrumentation
- **Missing**: `Reachable` assertion in `handleMove` when an error is sent back to the player.
