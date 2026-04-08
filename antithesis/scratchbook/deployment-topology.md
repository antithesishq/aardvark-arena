# Deployment Topology

## Overview

The existing Antithesis deployment topology is already well-designed. This document describes it and notes potential adjustments.

```text
                  +-----------+     +-----------+
                  | client-1  |     | client-2  |
                  | (player)  |     | (player)  |
                  +-----+-----+     +-----+-----+
                        |                 |
                        v                 v
                  +---------------------------+
                  |       matchmaker           |
                  |  (queue + ELO + SQLite)    |
                  +--+-------+-------+--------+
                     |       |       |
                     v       v       v
               +--------+ +--------+ +--------+
               | gs-1   | | gs-2   | | gs-3   |
               | (game) | | (game) | | (game) |
               +--------+ +--------+ +--------+

                  +---------------------------+
                  |     health-checker         |
                  | (emits setup_complete)     |
                  +---------------------------+
```

## Container Inventory

| Container | Image | Role | Process | Connections | Replicas |
|-----------|-------|------|---------|-------------|----------|
| `matchmaker` | `aardvark-arena-service` | Service | `matchmaker -addr=:8080 -token=... -gameserver=http://gameserver-{1,2,3}:8081 -match-interval=500ms -monitor-interval=2s -session-timeout=1m` | HTTP from clients; HTTP to/from game servers | 1 |
| `gameserver-1` | `aardvark-arena-service` | Service | `gameserver -addr=:8081 -token=... -matchmaker=http://matchmaker:8080 -turn-timeout=10s -max-sessions=50` | WebSocket from clients; HTTP from matchmaker; HTTP to matchmaker (results) | 1 |
| `gameserver-2` | `aardvark-arena-service` | Service | (same as gs-1) | (same) | 1 |
| `gameserver-3` | `aardvark-arena-service` | Service | (same as gs-1) | (same) | 1 |
| `client-1` | `aardvark-arena-player` | Client | `sleep infinity` (test commands run by Test Composer) | HTTP to matchmaker; WebSocket to game servers | 1 |
| `client-2` | `aardvark-arena-player` | Client | `sleep infinity` (test commands run by Test Composer) | (same) | 1 |
| `health-checker` | `aardvark-arena-health-checker` | Client | `health-checker.sh` (waits for services, emits `setup_complete`) | HTTP to matchmaker + all game servers | 1 |

## Design Rationale

### Why 3 Game Servers
- Exercises fleet load balancing and server selection
- Enables partial failure testing (1 or 2 servers down while others remain)
- Exercises capacity-based routing when servers are full
- Creates interesting retry/backoff patterns

### Why 2 Client Containers
- Test Composer runs multiple `parallel_driver_*` scripts across both containers
- Creates realistic concurrent matchmaking pressure (many players from different clients)
- Normal + evil players can run in parallel on separate containers

### Why a Separate Health Checker
- Emits `setup_complete` signal only after all services are confirmed healthy
- Does not interfere with test workloads
- Clean separation of readiness signaling from test execution

## Dependencies

No external dependencies (databases, message brokers, etc.) are needed — the matchmaker uses an embedded SQLite database in memory. This is a significant simplification of the topology.

## Docker Images

Three images built from a single multi-stage `antithesis/Dockerfile`:
1. **`aardvark-arena-service`** (target: `service`) — Contains `matchmaker` and `gameserver` binaries + Antithesis instrumentation symbols
2. **`aardvark-arena-player`** (target: `player`) — Contains `player` binary + test drivers at `/opt/antithesis/test/v1/aardvark-arena/` + symbols
3. **`aardvark-arena-health-checker`** (target: `health-checker`) — Contains `health-checker.sh`

All Go binaries are compiled from Antithesis-instrumented source using `antithesis-go-instrumentor`.

## Test Template

Location: `/opt/antithesis/test/v1/aardvark-arena/` (on client images)

| Command | Type | Description |
|---------|------|-------------|
| `parallel_driver_player.sh` | parallel_driver | Normal AI player, 20 sessions |
| `parallel_driver_evil_player.sh` | parallel_driver | Evil AI player with chaos rates, 20 sessions |
| `eventually_health_check.sh` | eventually | Verifies all services are healthy post-test |
| `eventually_queue_empty.sh` | eventually | Runs 2 fresh players to verify end-to-end flow works |

## Network Topology

All containers share a single Docker network. Communication paths:
- Clients -> Matchmaker: HTTP (queue, poll)
- Clients -> Game Servers: WebSocket (game play)
- Matchmaker -> Game Servers: HTTP (create session)
- Game Servers -> Matchmaker: HTTP (report results)
- Health Checker -> All services: HTTP (health checks)

Antithesis can inject network faults between any pair, enabling partition and latency testing.

## Configuration Values (Antithesis-Specific)

The Antithesis configuration tightens timeouts compared to defaults:
- Session timeout: 1m (default 5m)
- Match interval: 500ms (default 1s)
- Session monitor interval: 2s (default 5s)
- Turn timeout: 10s (default 30s)
- Max sessions per game server: 50 (default 100)

These tighter values accelerate state transitions, making it more likely that Antithesis finds timing-sensitive bugs within a reasonable timeline budget.

## Potential Improvements

1. **Add a third client container**: Would increase player concurrency and matchmaking pressure, potentially surfacing more race conditions. Trade-off: larger state space.
2. **Add a `singleton_driver` or `first_` command**: Could perform a baseline validation before parallel drivers start. Currently the `eventually_*` checks handle post-test validation.
3. **Vary evil player rates across clients**: Different chaos profiles per client could exercise more combinations. Currently both clients have access to the same evil player script.
