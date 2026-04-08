# Existing Antithesis SDK Assertions

**SDK Version:** `github.com/antithesishq/antithesis-sdk-go v0.6.0`

**Random Integration:** `internal/rand.go` — All `math/rand` RNG instances use `antirandom.Source()` from `antithesis-sdk-go/random`, ensuring Antithesis can control nondeterminism.

## Summary

| Type | Count |
|------|-------|
| Always | 4 |
| Sometimes | 14 |
| Reachable | 26 |
| Unreachable | 4 |
| **Total** | **48** |

## Always (Safety) Assertions

| # | File | Line | Message | Details |
|---|------|------|---------|---------|
| A1 | `internal/matchmaker/server.go` | 159 | "cancelled session results never declare a winner" | Checks `!body.Cancelled \|\| body.Winner == uuid.Nil` on the HTTP result endpoint |
| A2 | `internal/matchmaker/db.go` | 273 | "cancelled sessions never report a winner" | Same invariant checked at the DB layer before writing |
| A3 | `internal/gameserver/server.go` | 80 | "gameserver active sessions never exceed max sessions" | Checks `health.ActiveSessions <= health.MaxSessions` in health endpoint |
| A4 | `internal/gameserver/reporter.go` | 53 | "gameserver reports never include a winner for cancelled sessions" | Checks `!result.cancelled \|\| result.winner == uuid.Nil` before HTTP submit |

## Sometimes (Liveness/Coverage) Assertions

| # | File | Line | Message | Condition |
|---|------|------|---------|-----------|
| S1 | `internal/matchmaker/server.go` | 102 | "players sometimes request a specific game kind" | `body.Game != nil` |
| S2 | `internal/matchmaker/server.go` | 117 | "queue requests sometimes wait before a session is assigned" | `true` (line reached) |
| S3 | `internal/matchmaker/server.go` | 125 | "queue requests sometimes return an immediate session assignment" | `true` (line reached) |
| S4 | `internal/matchmaker/match_queue.go` | 89 | "more than two players are sometimes queued" | `len(q.queued) > 2` |
| S5 | `internal/matchmaker/match_queue.go` | 185 | "queued players are sometimes promoted to matched sessions" | `true` (line reached) |
| S6 | `internal/matchmaker/db.go` | 192 | "sessions sometimes expire before completion" | `len(expired) > 0` |
| S7 | `internal/matchmaker/db.go` | 315 | "some completed sessions end in a draw" | `draw` (winner == uuid.Nil for non-cancelled) |
| S8 | `internal/matchmaker/fleet.go` | 127 | "session creation sometimes succeeds on a gameserver" | `true` (line reached) |
| S9 | `internal/gameserver/server.go` | 122 | "gameserver sometimes accepts session creation requests" | `true` (line reached) |
| S10 | `internal/gameserver/reporter.go` | 97 | "result reporting sometimes receives non-ok responses" | `resp.StatusCode != http.StatusOK` |
| S11 | `internal/player/protocol.go` | 69 | "games sometimes end in draws" | `state.Status == game.Draw` |
| S12 | `internal/player/protocol.go` | 70 | "games sometimes end due to cancellation" | `state.Status == game.Cancelled` |
| S13 | `internal/player/protocol.go` | 71 | "games sometimes end with a winner" | `state.Status == game.P1Win \|\| state.Status == game.P2Win` |

## Reachable Assertions

| # | File | Line | Message |
|---|------|------|---------|
| R1 | `internal/matchmaker/match_queue.go` | 112 | "matchmaker found a pair of compatible players" |
| R2 | `internal/matchmaker/match_queue.go` | 192 | "match publishing sometimes races with player unqueueing" |
| R3 | `internal/matchmaker/match_queue.go` | 206 | "queue polling sometimes returns an existing match" |
| R4 | `internal/matchmaker/fleet.go` | 80 | "fleet sometimes has no currently available gameserver candidates" |
| R5 | `internal/matchmaker/fleet.go` | 114 | "fleet sometimes encounters temporary transport failures" |
| R6 | `internal/matchmaker/fleet.go` | 140 | "gameservers sometimes reject session creation due to capacity" |
| R7 | `internal/gameserver/server.go` | 110 | "gameserver sometimes reaches max session capacity" |
| R8 | `internal/gameserver/session.go` | 67 | "session creation sometimes fails because the server is full" |
| R9 | `internal/gameserver/protocol.go` | 173 | "players sometimes reconnect to an in-progress session" |
| R10 | `internal/gameserver/protocol.go` | 193 | "extra player connections are sometimes rejected" |
| R11 | `internal/gameserver/protocol.go` | 228 | "sessions sometimes receive invalid move payloads" |
| R12 | `internal/gameserver/protocol.go` | 238 | "sessions sometimes receive invalid semantic moves" |
| R13 | `internal/gameserver/reporter.go` | 80 | "result reporting sometimes retries after temporary transport errors" |
| R14 | `internal/gameserver/reporter.go` | 88 | "result reporting sometimes fails due to non-temporary errors" |
| R15 | `internal/player/loop.go` | 97 | "players sometimes see transient queue request errors" |
| R16 | `internal/player/loop.go` | 107 | "evil players sometimes submit throwaway queue requests that are never polled again" |
| R17 | `internal/player/loop.go` | 110 | "players sometimes receive a new match assignment" |
| R18 | `internal/player/loop.go` | 117 | "queue polling sometimes repeats the currently assigned session" |
| R19 | `internal/player/loop.go` | 130 | "players sometimes queue for any available game" |
| R20 | `internal/player/loop.go` | 138 | "players sometimes queue for a specific game preference" |
| R21 | `internal/player/loop.go` | 159 | "players sometimes play tic-tac-toe sessions" |
| R22 | `internal/player/loop.go` | 163 | "players sometimes play connect4 sessions" |
| R23 | `internal/player/loop.go` | 167 | "players sometimes play battleship sessions" |
| R24 | `internal/player/protocol.go` | 57 | "players sometimes receive protocol-level error messages" |
| R25 | `internal/player/protocol.go` | 79 | "evil players sometimes send nuisance out-of-turn moves" |
| R26 | `internal/player/protocol.go` | 113 | "evil players sometimes submit intentionally bad moves" |
| R27 | `internal/player/session.go` | 94 | "evil players sometimes attempt random-id background joins" |

## Unreachable Assertions

| # | File | Line | Message |
|---|------|------|---------|
| U1 | `internal/matchmaker/db.go` | 304 | "every active session should map to exactly two players" |
| U2 | `internal/matchmaker/fleet.go` | 157 | "gameserver should only return 200 or 503 for session creation" |
| U3 | `internal/gameserver/session.go` | 220 | "session manager should only run supported game kinds" |
| U4 | `internal/gameserver/protocol.go` | 218 | "moves should only arrive from connected players" |

## Existing Test Drivers

| File | Type | Description |
|------|------|-------------|
| `antithesis/test/parallel_driver_player.sh` | parallel_driver | Normal AI player, 20 sessions |
| `antithesis/test/parallel_driver_evil_player.sh` | parallel_driver | Evil AI player with chaos (35% bad moves, 20% out-of-turn, 55% malformed, 18% extra-connect, 16% queue-abandon), 20 sessions |
| `antithesis/test/eventually_health_check.sh` | eventually | Verifies all services are healthy after testing |
| `antithesis/test/eventually_queue_empty.sh` | eventually | Runs two fresh players to verify end-to-end flow still works |

## Coverage Observations

**Well-covered areas:**
- Cancelled-session-never-has-winner invariant (triple-checked: A1, A2, A4)
- Session capacity limits (A3, R7, R8, R6)
- Match lifecycle (R1-R3, S2-S5)
- Evil player behavior reachability (R16, R25, R26, R27)
- Game type diversity (R21-R23)
- Game outcome diversity (S7, S11-S13)

**Potential gaps (no existing assertions):**
- ELO calculation correctness (no assertions in `elo.go`)
- Player-to-session assignment integrity beyond the 2-player count check
- Turn alternation correctness in game protocols
- Battleship-specific invariants (ship placement validity, hit tracking correctness)
- Session timeout/deadline enforcement correctness
- WebSocket connection lifecycle invariants
- Database transaction isolation/correctness beyond result reporting
- Queue ordering fairness (oldest-first guarantee)
- Fleet retry/backoff correctness
