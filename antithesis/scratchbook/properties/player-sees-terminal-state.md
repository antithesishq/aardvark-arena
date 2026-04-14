# player-sees-terminal-state — Players Observe Game Completion

## Evidence

When a game ends, `Protocol.report()` (`internal/gameserver/protocol.go:95-111`) sets `p.state.Status` to the terminal status and calls `BroadcastState()`. Then the select loop breaks, and `RunToCompletion` closes all player channels (`protocol.go:162-165`).

The sequence is: set terminal status → broadcast → break → close channels. The broadcast uses `sendLatest`, which may drop the message if the channel is full.

Player-side, the Protocol (`internal/player/protocol.go:51-82`) reads from `protocolRx`. When it sees `state.Status.IsTerminal()`, it returns the completion.

## Failure Scenario

If the player's `protocolRx` channel is full when the terminal state is broadcast, `sendLatest` will drain the old message and send the terminal state. So the terminal state should always be the last message in the channel. But if the channel is then closed before the player reads the terminal state, the player reads the terminal state from the closed channel (Go guarantees buffered values are read before the zero value).

The more realistic risk: the player's WebSocket write fails (network issue) before the terminal state is written. The write goroutine exits, the channel drains on close, and the player never sees the terminal state. The player's protocol loop would exit via `for msg := range p.rx` when `protocolRx` is closed.

## Relevant Code Paths
- `internal/gameserver/protocol.go:95-111` — `report()` sets status and broadcasts
- `internal/gameserver/protocol.go:162-165` — Close player channels
- `internal/player/protocol.go:51-82` — Player protocol reads terminal state

## SUT Instrumentation
- **Missing**: `Sometimes` assertion (workload-side) that a player receives a terminal state before the channel closes.
