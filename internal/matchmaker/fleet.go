package matchmaker

import (
	"net/url"
	"time"
)

// FailureTimeout is the grace period we give game servers to recover after we
// see a network error
var FailureTimeout = time.Minute

// Fleet monitors a collection of GameServers and handles session creation
type Fleet struct {
	GameServers []*gameserver
}

// Keep track of a single game server
type gameserver struct {
	url url.URL
	// retryAt is set when we either see a network error, or the gameserver is
	// full. When this time passes, we will resume attempting to create Sessions
	// on this gameserver.
	retryAt *time.Time
}

func NewFleet(urls []*url.URL) *Fleet {
	var gs []*gameserver
	for _, url := range urls {
		gs = append(gs, &gameserver{url: *url})
	}
	return &Fleet{GameServers: gs}
}
