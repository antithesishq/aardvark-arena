# spectator-state-consistency — Spectators See Valid Game States

## Evidence

Spectators receive state through two paths:
1. **On join** (`internal/gameserver/protocol.go:140-146`): The current state is marshaled and sent immediately.
2. **On broadcast** (`internal/gameserver/protocol.go:229-242`): After every valid move, `BroadcastState` serializes the state and sends to all players and spectators via `sendLatest`.

The `sendLatest` function (`protocol.go:254-267`) is non-blocking. If the spectator's channel (capacity 1) is full, it drains the old message and sends the new one. This means spectators may miss intermediate states but always receive the latest.

## Failure Scenario

The concern is not missing states (that's by design) but receiving *invalid* states. Since state is serialized from the Protocol's goroutine-local variable immediately after mutation, and the Protocol is single-threaded, the serialized state should always be consistent.

However, if `json.Marshal` is called on a partially-constructed state (a bug in MakeMove that panics mid-mutation, recovered by something), the serialized state could be inconsistent. Go's deferred recovery from panics in the protocol goroutine would need to be investigated.

## Relevant Code Paths
- `internal/gameserver/protocol.go:140-146` — Spectator join state delivery
- `internal/gameserver/protocol.go:229-242` — `BroadcastState`
- `internal/gameserver/protocol.go:254-267` — `sendLatest`

## SUT Instrumentation
- **Missing**: `Always` assertion (workload-side) that each state received by a spectator client passes structural validation for the game type.
