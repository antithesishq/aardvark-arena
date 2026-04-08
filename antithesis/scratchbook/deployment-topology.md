# Deployment Topology

## Overview

4-container topology: one matchmaker, one game server, and two identical workload client containers. The two workload containers are clones — same image, same test template — placed in separate containers so Antithesis can partition them independently. This enables scenarios where one player in a game session is network-isolated while the other remains connected.

No external dependencies required — SQLite is embedded in the matchmaker binary.

## SDK Selection

- **Go SDK** (`antithesis-sdk-go`): Used in both the SUT services (matchmaker, game server) for SUT-side assertions and in the workload clients for workload assertions and `setup_complete` signaling.

## Container Topology

```text
+--------------------+
|   workload-1       | ---+
|   (test client)    |    |     +--------------------+      +--------------------+
+--------------------+    +---> |   matchmaker       | <--- |   gameserver       |
                          +---> |   (SUT)            | ---> |   (SUT)            |
+--------------------+    |     +--------------------+      +--------------------+
|   workload-2       | ---+         SQLite (embedded)        game sessions
|   (test client)    |              ELO, queue, sessions     WebSocket protocol
+--------------------+
   parallel players
   (identical clones)
```

### Container: matchmaker

| | |
|---|---|
| **Role** | Service (SUT) |
| **Image** | New multi-stage Dockerfile: Go build → minimal runtime |
| **Binary** | `cmd/matchmaker` |
| **Flags** | `-addr=:8080 -token=<shared> -gameserver=http://gameserver:8081 -db-path=/data/matchmaker.db -match-interval=100ms -session-timeout=2m -monitor-interval=1s` |
| **Ports** | 8080 (HTTP) |
| **Connects to** | gameserver:8081 (create sessions) |
| **Notes** | Use file-backed SQLite (`/data/matchmaker.db`) so state survives process restarts within a timeline. Shortened intervals for faster Antithesis exploration. |

### Container: gameserver

| | |
|---|---|
| **Role** | Service (SUT) |
| **Image** | New multi-stage Dockerfile: Go build → minimal runtime |
| **Binary** | `cmd/gameserver` |
| **Flags** | `-addr=:8081 -token=<shared> -matchmaker=http://matchmaker:8080 -turn-timeout=10s -max-sessions=6` |
| **Ports** | 8081 (HTTP + WebSocket) |
| **Connects to** | matchmaker:8080 (report results) |
| **Notes** | Shortened turn timeout for faster game completion. MaxSessions=6 keeps state space bounded and is low enough that Antithesis can realistically hit capacity by scaling up parallel player drivers, exercising the `max-sessions-reached` property and the fleet backpressure mechanism. |

### Container: workload-1

| | |
|---|---|
| **Role** | Client (test workload) |
| **Image** | New multi-stage Dockerfile: Go build → runtime with test template |
| **Binary** | Entrypoint emits `setup_complete` then sleeps. Test commands from `/opt/antithesis/test/v1/main/` drive the workload. |
| **Ports** | None |
| **Connects to** | matchmaker:8080 (queue for games), gameserver:8081 (WebSocket play) |

### Container: workload-2

| | |
|---|---|
| **Role** | Client (test workload) |
| **Image** | Same image as workload-1 (identical clone) |
| **Binary** | Entrypoint emits `setup_complete` then sleeps. Test commands from `/opt/antithesis/test/v1/main/` drive the workload. |
| **Ports** | None |
| **Connects to** | matchmaker:8080 (queue for games), gameserver:8081 (WebSocket play) |

#### Test Template: `/opt/antithesis/test/v1/main/` (shared by both workload containers)

| Command | Description |
|---------|-------------|
| `parallel_driver_player` | Launch a single well-behaved player that plays a random number of sessions (1–20) then exits (`cmd/player -matchmaker=http://matchmaker:8080 -num-sessions=$((RANDOM % 20 + 1))`) |
| `parallel_driver_evil_player` | Launch a single evil player that plays a random number of sessions (1–10) then exits (`cmd/player -evil -matchmaker=http://matchmaker:8080 -num-sessions=$((RANDOM % 10 + 1))`) |
| `eventually_check_leaderboard` | Verify leaderboard has entries and ELO values are reasonable |
| `anytime_check_health` | Check both matchmaker and gameserver health endpoints respond |

Players exit after a bounded random number of sessions rather than running forever. This is important because Antithesis re-invokes `parallel_driver_` commands after they exit, creating a continuous stream of fresh players with new IDs. This exercises the full player lifecycle (register → queue → play → exit) and ensures the matchmaker's player table grows over time, the queue sees churn, and the system handles players cleanly departing mid-run. Antithesis's Test Composer decides how many instances of each command to run per timeline, so the player count and evil/normal ratio are explored automatically.

## Justification

- **No separate DB container**: SQLite is embedded — no network dependency to simulate.
- **Single game server**: Simpler state space. The fleet logic (retry, failover) still exercises its code path when the game server is transiently unavailable due to Antithesis fault injection (e.g., network partition between matchmaker and gameserver).
- **Two workload containers (identical clones)**: Players from workload-1 and workload-2 get matched against each other via the matchmaker. Because they run in separate containers, Antithesis can partition one workload while leaving the other connected. This creates asymmetric failure scenarios where one player in an active game session loses connectivity (to the game server, to the matchmaker, or both) while their opponent remains fully connected — exercising reconnection, turn timeouts, session deadlines, and the game server's handling of half-connected sessions.
- **No UI container**: The UI is read-only and doesn't affect system correctness.

## Network Considerations

All four containers share a Docker network. Antithesis can inject:
- Network partitions between matchmaker ↔ gameserver (tests fleet failover, result reporting retries)
- Network partitions between one workload ↔ gameserver while the other stays connected (tests asymmetric player disconnection — one player in a game loses connectivity while the other keeps playing, exercising turn timeouts, reconnection, and session cleanup)
- Network partitions between one workload ↔ matchmaker (tests queue polling resilience for a subset of players while others continue to match normally)
- Full partition of one workload container (tests one-sided session abandonment — the connected player should eventually win by timeout or the session should be cancelled)
- Process restarts on matchmaker or gameserver (tests recovery)

## Assumptions

- Docker compose networking allows containers to reach each other by hostname.
- File-backed SQLite in the matchmaker container will survive process restarts (Antithesis preserves container filesystems within a timeline).
