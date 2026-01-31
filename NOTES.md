### 1. Player: Full loop implementation — `internal/player/loop.go`

The player loop is a stub that just sleeps and logs. Per DESIGN.md, the player needs to:

1. Queue with matchmaker — `PUT /queue/{pid}` with optional game preference
2. Poll until matched — keep calling `PUT /queue/{pid}` until a 200 with `SessionInfo` is returned (vs 202 for still-queued) (treat a session info object that matches our last session as a 202 and continue polling)
3. Connect to gameserver — open WebSocket to `/session/{sid}/{pid}`
4. Play the game — receive `STATE` messages, send `MOVE` messages when it's the player's turn, using the appropriate `GameAi` implementation
5. Disconnect when game status is terminal
6. Loop back to step 1

The `Loop` struct also doesn't store its `Config` currently.
