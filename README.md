# Aardvark Arena

A turn-based game simulation that pitches AI players against each other in simple 2D games. Built as a reference project for using [Antithesis](https://antithesis.com).

Aardvark Arena runs three types of games -- **Tic-Tac-Toe**, **Connect4**, and **Battleship** -- across a distributed system of services:

- **Matchmaker** -- tracks player ELO ratings and matches queued players with available game sessions.
- **Game Servers** -- each runs up to 50 concurrent sessions, accepting any of the configured game types.
- **Players** -- AI bots that queue for matches, play games over WebSocket, and repeat.
- **UI** -- a Next.js dashboard for spectating live games, viewing the leaderboard, and monitoring system status.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) (for the UI)
- [Docker](https://docs.docker.com/get-docker/)
- [Hivemind](https://github.com/DarthSim/hivemind) (recommended for local development)

## Quickstart

Clone the repo, install hivemind, and start everything with a single command.

### Run with hivemind

The repo includes a `Procfile` that starts all services — matchmaker, two game servers, a 100-player swarm, and the UI — in a single command:

```bash
hivemind
```

Open http://localhost:3001 to see the dashboard. Games should start appearing within a few seconds.

### Run manually

If you prefer to run each service in its own terminal:

```bash
# Terminal 1: start the matchmaker
go run ./cmd/matchmaker -addr=:8080 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -gameserver=http://localhost:8081

# Terminal 2: start a game server
go run ./cmd/gameserver -addr=:8081 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -matchmaker=http://localhost:8080

# Terminal 3: start the UI
cd ui && npm install && npm run dev

# Terminal 4: launch players
go run ./cmd/swarm -n 7 -move-delay 500ms
```

Open http://localhost:3001 to see the dashboard. You can also run `go run ./cmd/player` to launch a single player if you prefer.

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
ui/                 # Next.js spectator dashboard
```
