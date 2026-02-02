// Package player implements the AI player loop.
package player

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"
)

// Config holds player configuration.
type Config struct {
	MatchmakerURL *url.URL
	PlayerID      internal.PlayerID
	PollInterval  time.Duration

	// NumSessions is the number of games this player should play before exiting.
	// If 0 the player will play games until interrupted.
	NumSessions int
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
	sessions := 0

	for ctx.Err() == nil {
		session, err := l.waitForMatch(ctx, lastSID)
		if err != nil {
			return err
		}
		log.Printf("matched: player=%s session=%s game=%s", l.cfg.PlayerID, session.SessionID, session.Game)
		lastSID = session.SessionID

		if err := l.playGame(ctx, session); err != nil {
			log.Printf("game error: %v", err)
		}
		sessions++
		log.Printf("player %s: game complete: session=%s; sessions=%d/%d", l.cfg.PlayerID, session.SessionID, sessions, l.cfg.NumSessions)

		if sessions >= l.cfg.NumSessions && l.cfg.NumSessions > 0 {
			log.Printf("reached configured number of sessions (%d); exiting", l.cfg.NumSessions)
			return nil
		}
	}
	return ctx.Err()
}

// waitForMatch polls the matchmaker queue until a new session is assigned.
func (l *Loop) waitForMatch(ctx context.Context, lastSID internal.SessionID) (*matchmaker.SessionInfo, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(l.cfg.PollInterval):
		}

		info, err := l.client.Queue(ctx, nil)
		if internal.HTTPIsTemporary(err) {
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
func (l *Loop) playGame(ctx context.Context, info *matchmaker.SessionInfo) error {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(info.Timeout))
	defer cancel()

	session := NewSession(info.Server, info.SessionID, l.cfg.PlayerID)
	go session.Run(ctx)

	var completion Completion
	var err error

	switch info.Game {
	case game.TicTacToe:
		p := NewProtocol(session.protocolRx, session.protocolTx, game.NewTicTacToeAi())
		completion, err = p.RunToCompletion()
	case game.Connect4:
		p := NewProtocol(session.protocolRx, session.protocolTx, game.NewConnect4Ai())
		completion, err = p.RunToCompletion()
	case game.Battleship:
		p := NewProtocol(session.protocolRx, session.protocolTx, game.NewBattleshipAi())
		completion, err = p.RunToCompletion()
	default:
		return fmt.Errorf("unsupported game: %s", info.Game)
	}

	if err != nil {
		return err
	}
	log.Printf("player %s: game finished: status=%s interrupted=%v", l.cfg.PlayerID, completion.Status.String(), completion.Interrupted)
	return nil
}
