# Aardvark Arena

A turn-based game simulation that pitches AI players against each other in simple 2D games. Built as a reference project for testing with [Antithesis](https://antithesis.com) and [Bombadil](https://antithesis.com/docs/bombadil/).

## Overview

Aardvark Arena runs three types of games -- **Tic-Tac-Toe**, **Connect4**, and **Battleship** -- across a distributed system of services:

- **Matchmaker** -- tracks player ELO ratings and matches queued players with available game sessions.
- **Game Servers** -- each runs up to 50 concurrent sessions, accepting any of the configured game types.
- **Players** -- AI bots that queue for matches, play games over WebSocket, and repeat.
- **UI** -- a Next.js dashboard for spectating live games, viewing the leaderboard, and monitoring system status.

## Running locally

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) (for the UI)
- [Docker](https://docs.docker.com/get-docker/) (for Antithesis testing)

### Start the services

```bash
# Terminal 1 -- matchmaker
go run ./cmd/matchmaker -addr=:8080 -token=secret \
  -gameserver=http://localhost:8081

# Terminal 2 -- game server
go run ./cmd/gameserver -addr=:8081 -token=secret \
  -matchmaker=http://localhost:8080

# Terminal 3 -- a player
go run ./cmd/player -matchmaker=http://localhost:8080

# Terminal 4 -- UI
cd ui && npm install && npm run dev
```

The UI will be available at `http://localhost:3000`.

## Running with Docker Compose

The `antithesis/` directory contains a full multi-service deployment:

```bash
# Build the images
docker build --platform linux/amd64 -f antithesis/Dockerfile --target service        -t aardvark-arena-service:latest .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target player         -t aardvark-arena-player:latest .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target health-checker -t aardvark-arena-health-checker:latest .

# Start everything
docker compose -f antithesis/docker-compose.yaml up
```

This starts a matchmaker, three game servers, two player containers, and a health checker.

## Testing with Antithesis

This project is set up as a reference for Antithesis testing. Here's how it's structured:

### Go instrumentation

The `antithesis/Dockerfile` instruments the Go source with the [Antithesis Go SDK](https://antithesis.com/docs/using_antithesis/sdk/go_sdk.html) before building. This enables fault injection and coverage-guided exploration.

### Health checker

`antithesis/health-checker.sh` waits for all services to be healthy, then emits the `setup_complete` signal so Antithesis knows when to start testing.

### Test workloads

Test driver scripts live in `antithesis/test/`:

- `parallel_driver_player.sh` -- drives normal player behavior
- `parallel_driver_evil_player.sh` -- drives adversarial player behavior
- `eventually_health_check.sh` -- validates service health properties
- `eventually_queue_empty.sh` -- validates queue draining properties

### Bombadil (UI testing)

The UI includes a [Bombadil](https://antithesis.com/docs/bombadil/) spec in `ui/bombadil-spec.ts` that defines properties for the frontend:

- **`navTabMatchesUrl`** -- the highlighted nav tab must always match the current URL path.
- **`connectingAlwaysResolves`** -- "Connecting..." placeholders must always eventually disappear.
- **`sessionCountMatchesTable`** -- the "Active Sessions" stat card must always equal the sessions table row count.

### Submitting a test run

Using [snouty](https://antithesis.com/docs/getting_started/snouty.html):

```bash
./antithesis/submit.sh --registry <your-registry> --duration 15
```

Or via the GitHub Actions workflow on pull requests (requires configuring repository secrets -- see `.github/workflows/antithesis.yml`).

## Project structure

```
cmd/
  matchmaker/       # Matchmaker service entrypoint
  gameserver/       # Game server service entrypoint
  player/           # AI player entrypoint
internal/
  matchmaker/       # Matchmaker logic, ELO, DB, HTTP handlers
  gameserver/       # Game server logic, session management, WebSocket protocol
  player/           # Player client logic
  games/            # Game implementations (tictactoe, connect4, battleship)
antithesis/
  Dockerfile        # Multi-stage build with Antithesis instrumentation
  docker-compose.yaml
  health-checker.sh
  submit.sh         # Submit a test run to Antithesis
  test/             # Test workload scripts
ui/                 # Next.js spectator dashboard + Bombadil spec
```

## Design

See [DESIGN.md](DESIGN.md) for the full system design, API contracts, and WebSocket protocol.
