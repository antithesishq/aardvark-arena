# aardvark-arena

A turn-based game simulation which pitches AI players against each other in simple 2d games. Games such as:

- Tic-Tac-Toe
- Battleship
- Connect4

The simulation is composed of three entities:

- The matchmaker: a service which tracks player ELO and matches players waiting in a queue with available game sessions.
- The gameserver: a service which runs a fixed number of "sessions", each of which may play any of the configured games.
- The player: An AI representing a player. Queues up for matches with the matchmaker before battling with another AI in a gameserver.

# Shared concepts

This section documents shared concepts and types between all of the components in this project.

## System types

```go
type PlayerId = UUID
type SessionId = UUID
type ApiKey = UUID
```

## Game types

```go
enum Player {
  P1,
  P2,
}

enum GameStatus {
	/// active state
	Active,

	/// terminal states
	P1Win,
	P2Win,
	Draw,
	Cancelled
}

type GameState[Shared any] struct {
  CurrentPlayer Player
  Status GameStatus
  Shared Shared
}

type GameSession[Move any, Shared any] interface {
  MakeMove(Player, Move) -> (GameState[Shared], error)
}

type GameAi[Move any, Shared any] interface {
  GetMove(Shared) -> (Move, error)
}

enum GameKind {
	Battleship = "battleship",
	TicTacToe = "tictactoe",
	Connect4 = "connect4"
}
```

# Player

The player binary performs the following logic:

1. get or generate a PlayerId
2. queue with the matchmaker
3. poll the queue until matched
4. connect to game server, play game
5. goto 2

# Matchmaker

The matchmaker is a long running API which acts as the coordination service betwen Players and Gameservers. Players may queue up for games, poll for a match, and then play a game on a Gameserver.

## Matchmaker Configuration

```go
type Config struct {
	// Sessions will be cancelled if they don't complete within this duration.
	SessionTimeout Duration

	// Available game servers to route to
	GameServers []URL
}
```

## Matchmaker Schema

The matchmaker uses SQLite for data storage.

```
table players:
  player_id: PlayerId
  elo: int
  wins: int
  losses: int
  draws: int

table players_sessions:
  player_id: PlayerId -> players.player_id
  session_id: SessionId -> sessions.session_id

table sessions:
  session_id: SessionId
  gameserver_url: String
  game: GameKind
  created_at: Timestamp
  deadline: Timestamp
  completed_at: Timestamp | null
  cancelled: bool
  winner_id: PlayerId | null -> players.player_id
```

## Matchmaker API

```
GET /health
  basic healthcheck

PUT /queue/{PlayerId}
  Body: {
    Game: GameKind | null
  }
  returns: {
    QueueStatus: queued | matched
    GameserverUrl: string <- if matched
    SessionId: SessionId <- if matched
    Game: GameKind <- if matched
  }
Ensures that the provided Player ID is in the queue.
If the player has been matched to a session, returns session details.
If the player does not choose a game, they will queue for any game.

DELETE /queue/{PlayerId}
Removes player from queue if they are in the queue.

POST /results/{SessionId}
  Requires API key auth.
  Body: {
    Cancelled: bool
    Winner: PlayerId | null
  }
  Called by the gameserver when a session is complete.
  If cancelled is true, then the session was cancelled.
  If cancelled is false and winner is null, then the session was a draw.
  Clears the players from the queue, requiring them to requeue.
```

## Matchmaker Background Tasks

### Match Task

A background task scans all players in the queue and attempts to match them together based on their relative ELO. Matched players are moved from the queue into a matched holding area.

### Session Timeout Task

If a session has not completed in the SessionTimeout, the session is cancelled and the players are removed from the queue.

### Gameserver Health Check

The Matchmaker checks the health of all Gameservers every second. If a Gameserver is down or is out of capacity, the Matchmaker will avoid sending new sessions its way.

# Gameserver

The Gameserver's only job is to run game sessions. Each session is the backend instance of a supported game. Sessions are created by the matchmaker, played by players, and then finally closed after notifying the matchmaker.

Each Gameserver runs up to a configurable number of sessions concurrently. Additional create session requests will be rejected with a `503 Service Unavailable`. The `Retry-After` header should be set to the time when the oldest session will timeout.

## Gameserver Configuration

```go
type Config struct {
	// A player will automatically lose if they don't submit their move within this duration.
	TurnTimeout Duration

	// The maximum number of concurrent sessions supported by this GameServer
	MaxSessions int
}
```

## Gameserver HTTP API

```
GET /health
  Response: { ActiveSessions: int, MaxSessions: int }
  Always returns a 200 response if the GameServer is up. But also reports the number of active vs max sessions so the Matchmaker can route correctly.

PUT /session/{SessionId}
  Body: {
    Game: GameType
    Deadline: Time
  }
  Create a new session if a slot is available. Returns `503 Service Unavailable` if overloaded. Returns OK if session already exists (idempotent).

CONNECT /session/{SessionId}/{PlayerId}
  Fails if no session exists.
  Open a WebSocket connection to the Session for a Player.
  Replaces an existing connection for the same player, terminating the other connection.
```

## Gameserver Websocket Protocol

Players connect to Sessions via Websocket. Players may freely reconnect.

```
Players must send HELLO immediately after connecting:

  Player -> Session: HELLO
  Session -> Player: STATE { GameState[Shared] }

Players should submit moves whenever GameState.CurrentPlayer points at them. The server will respond with an Error _or_ broadcast the new game state to all players.

  Player -> Session: MOVE { Move }
  Session -> Player: ERROR { InvalidMove }
  Session => Players: STATE { GameState[Shared] }

When the Session is complete, and the Gameserver has submitted the result to the Matchmaker, all connections will receive a QUIT message along with the final state. Game states will stick around in memory for a short amount of time before being GCed.

  Session => Players: QUIT { GameState[Shared] }
```

Players must disconnect once the Game status is terminal. Players may then rejoin the matchmaker queue.

# Other modules

This section documents standalone modules supporting one or more system components.

## ELO and Matching

```go
DEFAULT_ELO = 1000
// How much ELO changes per game
ELO_K_FACTOR = 32
// Max ELO difference for matching (relaxes over time)
MAX_ELO_DIFF = 200
// ELO diff increases by this much per second waiting
ELO_DIFF_RELAX_RATE = 50

// given the winner and loser elo, returns their updated elo
func CalcElo(winner: int, loser: int, draw: bool) (int, int) {
    expected = 1 / (1 + 10 ** ((loser_elo - winner_elo) / 400))
    score = 0.5 if draw else 1.0
    new_winner = round(winner_elo + ELO_K_FACTOR * (score - expected))
    new_loser = round(loser_elo + ELO_K_FACTOR * ((1 - score) - (1 - expected)))
    return new_winner, new_loser
}
```

Players in the queue are matched based on their ELO difference. Every second a player is in the queue, the max diff increases by the relax rate.
