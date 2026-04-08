# Aardvark Arena

A turn-based game simulation that pitches AI players against each other in simple 2D games. Built as a reference project for testing with [Antithesis](https://antithesis.com) and [Bombadil](https://github.com/antithesishq/bombadil).

Aardvark Arena runs three types of games -- **Tic-Tac-Toe**, **Connect4**, and **Battleship** -- across a distributed system of services:

- **Matchmaker** -- tracks player ELO ratings and matches queued players with available game sessions.
- **Game Servers** -- each runs up to 50 concurrent sessions, accepting any of the configured game types.
- **Players** -- AI bots that queue for matches, play games over WebSocket, and repeat.
- **UI** -- a Next.js dashboard for spectating live games, viewing the leaderboard, and monitoring system status.


## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) (for the UI)
- [Docker](https://docs.docker.com/get-docker/) (for Antithesis testing)


## Quickstart

Clone the repo, start the services, and watch AI bots battle it out in Tic-Tac-Toe, Connect4, and Battleship.

```bash
# Terminal 1: start the matchmaker
go run ./cmd/matchmaker -addr=:8080 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -gameserver=http://localhost:8081

# Terminal 2: start a game server
go run ./cmd/gameserver -addr=:8081 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -matchmaker=http://localhost:8080

# Terminal 3: start the UI
cd ui && npm install && npm run dev
```

Open http://localhost:3000 to see the dashboard. Nothing is happening yet because there are no players. Use the `swarm` command to spin up a batch of AI players:

```bash
# Terminal 4: launch players
go run ./cmd/swarm -n 7 -move-delay 500ms
```

Games should start appearing in the UI within a few seconds. You can also run `go run ./cmd/player` to launch a single player if you prefer.


## Testing with Antithesis

The `antithesis/` directory contains everything needed to run Aardvark Arena on the Antithesis platform.

**Instrumentation** -- the `Dockerfile` builds the Go services with the [Antithesis Go SDK](https://antithesis.com/docs/using_antithesis/sdk/go_sdk.html), enabling fault injection and coverage-guided exploration across the full distributed system.

**Workloads** -- two player types drive traffic during a test run: a normal player that queues, plays, and re-queues, and an evil player that queues but never joins. Driver scripts in `antithesis/test/` run these in parallel to exercise the system under realistic and adversarial conditions.

**Bombadil** -- the UI is tested with [Bombadil](https://github.com/antithesishq/bombadil), which explores frontend paths automatically. Properties defined in `ui/bombadil-spec.ts` assert things like nav state matching the current URL and active session counts staying consistent.

## Project structure

```
cmd/
  matchmaker/       # Matchmaker service entrypoint
  gameserver/       # Game server service entrypoint
  player/           # AI player entrypoint
  swarm/            # Multi-player launcher for local testing
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
