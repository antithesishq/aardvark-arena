package gameserver

import (
	"encoding/json"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
)

const (
	// WatchEventSession is sent when a session is created or its state changes.
	WatchEventSession = "session"
	// WatchEventSessionEnd is sent when a session is removed from the server.
	WatchEventSessionEnd = "session_end"
	// WatchEventHealth is sent on connect and when the session count changes.
	WatchEventHealth = "health"
)

// WatchEvent is a multiplexed event sent to server-level watchers.
type WatchEvent struct {
	Type string `json:"type"`

	// session / session_end fields
	SessionID internal.SessionID `json:"session_id,omitempty"`
	Game      game.Kind          `json:"game,omitempty"`
	Players   map[string]int     `json:"players,omitempty"`
	State     json.RawMessage    `json:"state,omitempty"`
	Deadline  *time.Time         `json:"deadline,omitempty"`

	// health fields
	ActiveSessions int `json:"active_sessions,omitempty"`
	MaxSessions    int `json:"max_sessions,omitempty"`
}
