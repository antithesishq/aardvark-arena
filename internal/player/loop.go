// Package player implements the AI player loop.
package player

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"
	"github.com/google/uuid"
)

// DefaultSpecificGameSelectionRate determines the probability a player will select a specific game to queue for.
const DefaultSpecificGameSelectionRate = 0.20

// Config holds player configuration.
type Config struct {
	MatchmakerURL *url.URL
	PlayerID      internal.PlayerID
	PollInterval  time.Duration
	Behavior      Behavior
	// SpecificGameSelectionRate is the probability [0,1] that a player
	// queues with a specific game preference instead of queueing for any game.
	// If this value is 0, specific-game selection is disabled.
	SpecificGameSelectionRate float64

	// NumSessions is the number of games this player should play before exiting.
	// If 0 the player will play games until interrupted.
	NumSessions int

	// MoveDelay adds an artificial pause before each move, useful for demos.
	MoveDelay time.Duration
}

// Loop runs the player game loop.
type Loop struct {
	cfg    Config
	client *MatchmakerClient
	rng    *rand.Rand
}

// New creates a new Loop.
func New(cfg Config) *Loop {
	return &Loop{
		cfg:    cfg,
		client: NewMatchmakerClient(cfg.MatchmakerURL, cfg.PlayerID),
		rng:    internal.NewRand(),
	}
}

// Run repeatedly queues for matches and plays games until ctx is cancelled.
func (l *Loop) Run(ctx context.Context) error {
	var lastSID internal.SessionID
	sessions := 0

	for ctx.Err() == nil {
		sessions++
		session, err := l.waitForMatch(ctx, lastSID)
		if err != nil {
			return err
		}
		log.Printf("matched: player=%s session=%s game=%s sessions=%d/%d",
			l.cfg.PlayerID, session.SessionID, session.Game, sessions, l.cfg.NumSessions)
		lastSID = session.SessionID

		if err := l.playGame(ctx, session); err != nil {
			log.Printf("player %s: game error: %v", l.cfg.PlayerID, err)
		}

		if sessions >= l.cfg.NumSessions && l.cfg.NumSessions > 0 {
			log.Printf("reached configured number of sessions (%d); exiting", l.cfg.NumSessions)
			return nil
		}
	}
	return ctx.Err()
}

// waitForMatch polls the matchmaker queue until a new session is assigned.
func (l *Loop) waitForMatch(ctx context.Context, lastSID internal.SessionID) (*matchmaker.SessionInfo, error) {
	pref := l.chooseGamePreference()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(l.cfg.PollInterval):
		}

		info, err := l.client.Queue(ctx, pref)
		if err != nil {
			log.Printf("queue error (retrying): %v", err)
			continue
		}
		if l.cfg.Behavior.doQueueAbandon(l.rng) {
			newPID := uuid.New()
			l.client = NewMatchmakerClient(l.cfg.MatchmakerURL, newPID)
		}
		if info != nil && info.SessionID != lastSID {
			return info, nil
		}
	}
}

func (l *Loop) chooseGamePreference() *game.Kind {
	if l.rng == nil {
		l.rng = internal.NewRand()
	}
	if len(game.AllGames) == 0 || l.cfg.SpecificGameSelectionRate <= 0 || l.rng.Float64() >= l.cfg.SpecificGameSelectionRate {
		return nil
	}

	chosen := game.AllGames[l.rng.Intn(len(game.AllGames))]
	log.Printf("player %s: queueing for specific game: %s", l.cfg.PlayerID, chosen)
	return &chosen
}

// playGame connects to the game server and plays until the game ends.
func (l *Loop) playGame(ctx context.Context, info *matchmaker.SessionInfo) error {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(info.Timeout))
	defer cancel()

	session := NewSession(info.Server, info.SessionID, l.cfg.PlayerID, l.cfg.Behavior)
	go session.Run(ctx)

	var completion Completion
	var err error

	switch info.Game {
	case game.TicTacToe:
		p := NewProtocol(session.protocolRx, session.protocolTx, game.NewTicTacToeAi(), l.cfg.Behavior, l.cfg.MoveDelay)
		completion, err = p.RunToCompletion()
	case game.Connect4:
		p := NewProtocol(session.protocolRx, session.protocolTx, game.NewConnect4Ai(), l.cfg.Behavior, l.cfg.MoveDelay)
		completion, err = p.RunToCompletion()
	case game.Battleship:
		p := NewProtocol(session.protocolRx, session.protocolTx, game.NewBattleshipAi(), l.cfg.Behavior, l.cfg.MoveDelay)
		completion, err = p.RunToCompletion()
	default:
		return fmt.Errorf("unsupported game: %s", info.Game)
	}

	if err != nil {
		return err
	}
	log.Printf("player %s: game finished: status=%s interrupted=%v", l.cfg.PlayerID, completion.Status, completion.Interrupted)
	return nil
}
