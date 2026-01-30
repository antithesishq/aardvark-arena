package matchmaker

import (
	"database/sql"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
)

// DB wraps an SQLite database for matchmaker persistence.
type DB struct {
	db *sqlx.DB
}

// PlayerModel represents a player row in the database.
type PlayerModel struct {
	PlayerID internal.PlayerID
	Elo      int
	Wins     int
	Losses   int
	Draws    int
}

// SessionModel represents a session row in the database.
type SessionModel struct {
	SessionID   internal.SessionID
	Server      string
	Game        string
	CreatedAt   time.Time
	Deadline    time.Time
	CompletedAt sql.NullTime
	Cancelled   sql.NullBool
	WinnerID    sql.Null[internal.PlayerID]
}

const selectPlayer = `
	SELECT PlayerID, Elo, Wins, Losses, Draws
	FROM players
	WHERE PlayerID = ?
`

const insertPlayer = `
	INSERT INTO players (PlayerID, Elo, Wins, Losses, Draws)
	VALUES (?, ?, 0, 0, 0)
	RETURNING PlayerID, Elo, Wins, Losses, Draws
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
			PlayerID BLOB NOT NULL PRIMARY KEY,
			Elo INTEGER NOT NULL,
			Wins INTEGER NOT NULL,
			Losses INTEGER NOT NULL,
			Draws INTEGER NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			SessionID BLOB NOT NULL PRIMARY KEY,
			Server TEXT NOT NULL,
			Game TEXT NOT NULL,
			CreatedAt INTEGER NOT NULL,
			Deadline INTEGER NOT NULL,
			CompletedAt INTEGER,
			Cancelled INTEGER,
			WinnerId BLOB,

			FOREIGN KEY(WinnerId) REFERENCES players(PlayerID)
		);
	`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS players_sessions (
			PlayerID BLOB NOT NULL,
			SessionID BLOB NOT NULL,

			PRIMARY KEY (PlayerID, SessionID),
			FOREIGN KEY(PlayerID) REFERENCES players(PlayerID),
			FOREIGN KEY(SessionID) REFERENCES sessions(SessionID)
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
	var p *PlayerModel
	err = tx.Get(p, selectPlayer, pid)
	if err == sql.ErrNoRows {
		err = tx.Get(p, insertPlayer, pid, internal.DefaultElo)
	}
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return p, nil
}
