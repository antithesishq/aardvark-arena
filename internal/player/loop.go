// Package player implements the AI player loop.
package player

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"
)

const (
	pollInterval     = time.Second
	discoveryTimeout = 500 * time.Millisecond
)

// Config holds player configuration.
type Config struct {
	MatchmakerURL *url.URL
	PlayerID      internal.PlayerID
}

// Loop runs the player game loop.
type Loop struct {
	cfg    Config
	client *MatchmakerClient
}

// New creates a new Loop.
func New(cfg Config) *Loop {
	return &Loop{
		cfg:    cfg,
		client: NewMatchmakerClient(cfg.MatchmakerURL, cfg.PlayerID),
	}
}

// Run repeatedly queues for matches and plays games until ctx is cancelled.
func (l *Loop) Run(ctx context.Context) error {
	var lastSID internal.SessionID

	for ctx.Err() == nil {
		session, err := l.waitForMatch(ctx, lastSID)
		if err != nil {
			return err
		}
		log.Printf("matched: session=%s game=%s", session.SessionID, session.Game)
		lastSID = session.SessionID

		if err := l.playGame(ctx, session); err != nil {
			log.Printf("game error: %v", err)
		}
		log.Printf("game complete: session=%s", session.SessionID)
	}
	return ctx.Err()
}

// waitForMatch polls the matchmaker queue until a new session is assigned.
func (l *Loop) waitForMatch(ctx context.Context, lastSID internal.SessionID) (*matchmaker.SessionInfo, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		info, err := l.client.Queue(ctx, nil)
		if internal.HttpIsTemporary(err) {
			log.Printf("queue error (retrying): %v", err)
			continue
		}
		if err != nil {
			return nil, err
		}
		if info != nil && info.SessionID != lastSID {
			return info, nil
		}
	}
}

// playGame connects to the game server and plays until the game ends.
func (l *Loop) playGame(_ context.Context, _ *matchmaker.SessionInfo) error {
	return fmt.Errorf("not implemented")
}
