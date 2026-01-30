package matchmaker

import (
	"database/sql"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
)

// DB wraps an SQLite database for matchmaker persistence.
type DB struct {
	db *sqlx.DB
}

// PlayerModel represents a player row in the database.
type PlayerModel struct {
	PlayerID internal.PlayerID `db:"player_id"`
	Elo      int
	Wins     int
	Losses   int
	Draws    int
}

// SessionModel represents a session row in the database.
type SessionModel struct {
	SessionID   internal.SessionID          `db:"session_id"`
	Server      string
	Game        string
	CreatedAt   time.Time                   `db:"created_at"`
	Deadline    time.Time
	CompletedAt sql.NullTime                `db:"completed_at"`
	Cancelled   sql.NullBool
	WinnerID    sql.Null[internal.PlayerID] `db:"winner_id"`
}

const selectPlayer = `
	SELECT player_id, elo, wins, losses, draws
	FROM players
	WHERE player_id = ?
`

const insertPlayer = `
	INSERT INTO players (player_id, elo, wins, losses, draws)
	VALUES (?, ?, 0, 0, 0)
	RETURNING player_id, elo, wins, losses, draws
`

const insertSession = `
	INSERT INTO sessions (session_id, server, game, created_at, deadline)
	VALUES (?, ?, ?, ?, ?)
	RETURNING
		session_id, server, game, created_at, deadline,
		completed_at, cancelled, winner_id
`

const insertPlayerSession = `
	INSERT INTO player_session (player_id, session_id)
	VALUES (?, ?)
`

// NewDB returns a disk-backed database stored at the provided path.
func NewDB(path string) (*DB, error) {
	db, err := sqlx.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	err = ensureSchema(db)
	return &DB{db: db}, err
}

func ensureSchema(conn sqlx.Execer) error {
	_, err := conn.Exec("pragma journal_mode=wal;")
	if err != nil {
		return err
	}
	_, err = conn.Exec("pragma synchronous=normal;")
	if err != nil {
		return err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			player_id BLOB NOT NULL PRIMARY KEY,
			elo INTEGER NOT NULL,
			wins INTEGER NOT NULL,
			losses INTEGER NOT NULL,
			draws INTEGER NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			session_id BLOB NOT NULL PRIMARY KEY,
			server TEXT NOT NULL,
			game TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			deadline INTEGER NOT NULL,
			completed_at INTEGER,
			cancelled BOOL,
			winner_id BLOB,

			FOREIGN KEY(winner_id) REFERENCES players(player_id)
		);
	`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS player_session (
			player_id BLOB NOT NULL,
			session_id BLOB NOT NULL,

			PRIMARY KEY (player_id, session_id),
			FOREIGN KEY(player_id) REFERENCES players(player_id),
			FOREIGN KEY(session_id) REFERENCES sessions(session_id)
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

// GetOrCreatePlayer returns the player with the given ID, creating one if needed.
func (db *DB) GetOrCreatePlayer(pid internal.PlayerID) (*PlayerModel, error) {
	tx, err := db.db.Beginx()
	if err != nil {
		return nil, err
	}
	var p PlayerModel
	err = tx.Get(&p, selectPlayer, pid)
	if err == sql.ErrNoRows {
		err = tx.Get(&p, insertPlayer, pid, internal.DefaultElo)
	}
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) CreateSession(
	sid internal.SessionID,
	p1 internal.PlayerID,
	p2 internal.PlayerID,
	server *url.URL,
	game game.Kind,
	deadline time.Time,
) (*SessionModel, error) {
	tx, err := db.db.Beginx()
	if err != nil {
		return nil, err
	}
	var s SessionModel
	err = tx.Get(&s, insertSession,
		sid, server.String(), game,
		time.Now(), deadline,
	)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(insertPlayerSession, p1, sid)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(insertPlayerSession, p2, sid)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) ReportSessionResult(sid internal.SessionID, cancelled bool, winner internal.PlayerID) error {
	_, err := db.db.Exec(`
		UPDATE sessions WHERE session_id = ?
		SET cancelled = ?, winner_id = ?
	`, sid, cancelled, winner)
	return err
}
