package matchmaker

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
)

type candidate struct {
	pid   internal.PlayerID
	elo   int
	entry time.Time
	// if nil, player is ok with any game
	game *game.Kind
}

// MatchQueue pairs waiting players into game sessions.
type MatchQueue struct {
	mu    sync.Mutex
	fleet *Fleet
	db    *DB
	// map from player id to ELO
	queued  map[internal.PlayerID]*candidate
	matched map[internal.PlayerID]*SessionInfo
}

// NewMatchQueue creates a MatchQueue backed by the given Fleet.
func NewMatchQueue(fleet *Fleet, db *DB) *MatchQueue {
	return &MatchQueue{
		fleet:   fleet,
		db:      db,
		queued:  make(map[internal.PlayerID]*candidate),
		matched: make(map[internal.PlayerID]*SessionInfo),
	}
}

// StartMatcher starts the matching process in a separate goroutine.
// It stops when the context is cancelled.
func (q *MatchQueue) StartMatcher(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				q.matchPlayers()
			}
		}
	}()
}

type match struct {
	a    *candidate
	b    *candidate
	game game.Kind
}

// matchPlayers matches as many players as possible currently in the queue.
func (q *MatchQueue) matchPlayers() {
	matches := q.collectMatches()

	// create a new game session for each match
	for _, match := range matches {
		session, err := q.fleet.CreateSession(match.game)
		if err == ErrNoServersAvailable {
			continue
		} else if err != nil {
			log.Panicf("fleet error: %v", err)
		}
		q.publishMatch(session, match.a, match.b)
	}
}

func (q *MatchQueue) collectMatches() []match {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queued) < 2 {
		return nil
	}

	// match players
	var matches []match
	for _, a := range q.queued {
		for _, b := range q.queued {
			if a == b {
				continue
			}
			matchesGame := a.game == nil || b.game == nil || a.game == b.game
			if matchesGame && internal.MatchElo(a.elo, b.elo, a.entry, b.entry) {
				chosenGame := a.game
				if chosenGame == nil {
					chosenGame = b.game
				}
				if chosenGame == nil {
					chosenGame = &game.AllGames[rand.Intn(len(game.AllGames))]
				}
				matches = append(matches, match{a: a, b: b, game: *chosenGame})
				break
			}
		}
	}
	return matches
}

func (q *MatchQueue) publishMatch(session *SessionInfo, a, b *candidate) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// it's possible that a or b left the queue while we started the session, if
	// this happens leave the other candidate in the queue, they will be matched
	// in the next cycle, and the erroneous game session will eventually timeout
	_, hasA := q.queued[a.pid]
	_, hasB := q.queued[b.pid]
	if hasA && hasB {
		_, err := q.db.CreateSession(
			session.SessionID,
			a.pid,
			b.pid,
			&session.Server,
			session.Game,
			time.Now().Add(session.Timeout),
		)
		if err != nil {
			log.Panicf("db error: %v", err)
		}

		log.Printf("players matched to session %s: %s %s", session.SessionID, a.pid, b.pid)
		delete(q.queued, a.pid)
		delete(q.queued, b.pid)
		q.matched[a.pid] = session
		q.matched[b.pid] = session
	}
}

// Queue ensures the player is in the match queue. Returns a non-nil SessionInfo
// if the player is matched.
func (q *MatchQueue) Queue(player *PlayerModel, game *game.Kind) (*SessionInfo, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if the player is already matched
	if session, ok := q.matched[player.PlayerID]; ok {
		return session, nil
	}

	// Player is not yet matched, make sure they are in the queue
	q.queued[player.PlayerID] = &candidate{
		pid:   player.PlayerID,
		elo:   player.Elo,
		entry: time.Now(),
		game:  game,
	}
	return nil, nil
}

// Unqueue idempotently removes a player from the queue.
func (q *MatchQueue) Unqueue(pid internal.PlayerID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.queued, pid)
}

// Untrack removes a session and associated players, allowing them to requeue
// for another match.
func (q *MatchQueue) Untrack(sid internal.SessionID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for pid, session := range q.matched {
		if session.SessionID == sid {
			delete(q.matched, pid)
		}
	}
}
