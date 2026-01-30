package matchmaker

import (
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

type MatchQueue struct {
	mu    sync.Mutex
	fleet *Fleet
	// map from player id to ELO
	queued  map[internal.PlayerID]*candidate
	matched map[internal.PlayerID]*SessionInfo
}

func NewMatchQueue(fleet *Fleet) *MatchQueue {
	return &MatchQueue{
		fleet:   fleet,
		queued:  make(map[internal.PlayerID]*candidate),
		matched: make(map[internal.PlayerID]*SessionInfo),
	}
}

// StartMatcher starts the matching process in a separate goroutine.
func (q *MatchQueue) StartMatcher(interval time.Duration) {
	go func() {
		for range time.Tick(interval) {
			q.findMatches()
		}
	}()
}

// findMatches matches as many players as possible currently in the queue
func (q *MatchQueue) findMatches() {
	q.mu.Lock()

	if len(q.queued) < 2 {
		return
	}

	type match struct {
		a    *candidate
		b    *candidate
		game game.Kind
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

	// unlock here so we don't block the main thread while setting up the sessions
	q.mu.Unlock()

	// create a new game session for each match
	for _, match := range matches {
		session, err := q.fleet.CreateSession(match.game)
		if err == NoServersAvailable {
			continue
		} else if err != nil {
			log.Fatalf("fleet error: %v", err)
		}
		q.publishMatch(session, match.a, match.b)
	}
}

func (q *MatchQueue) publishMatch(session *SessionInfo, a, b *candidate) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// it's possible that a or b left the queue while we were building matches, if
	// this happens leave the other candidate in the queue, they will be matched
	// in the next cycle
	_, hasA := q.queued[a.pid]
	_, hasB := q.queued[b.pid]
	if hasA && hasB {
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
