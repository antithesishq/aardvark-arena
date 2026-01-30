package matchmaker

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/google/uuid"
)

func TestMatchQueueSanity(t *testing.T) {
	t.Run("queue returns nil before match", func(t *testing.T) {
		fleet := NewFleet(nil, internal.NilToken, 5*time.Minute)
		q := NewMatchQueue(fleet)

		session, err := q.Queue(&PlayerModel{
			PlayerID: uuid.New(),
			Elo:      1000,
		}, &game.AllGames[0])
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session != nil {
			t.Fatal("expected nil session for lone player")
		}
	})

	t.Run("two similar players get matched", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL)
		fleet := NewFleet([]*url.URL{u}, internal.NilToken, 5*time.Minute)
		q := NewMatchQueue(fleet)

		p1 := &PlayerModel{PlayerID: uuid.New(), Elo: 1000}
		p2 := &PlayerModel{PlayerID: uuid.New(), Elo: 1050}

		// Queue both players for same game
		if _, err := q.Queue(p1, &game.AllGames[0]); err != nil {
			t.Fatalf("queue p1: %v", err)
		}
		if _, err := q.Queue(p2, &game.AllGames[0]); err != nil {
			t.Fatalf("queue p2: %v", err)
		}

		// Run the matcher
		q.findMatches()

		// Both players should now see a session
		s1, err := q.Queue(p1, &game.AllGames[0])
		if err != nil {
			t.Fatalf("queue p1 after match: %v", err)
		}
		s2, err := q.Queue(p2, &game.AllGames[0])
		if err != nil {
			t.Fatalf("queue p2 after match: %v", err)
		}
		if s1 == nil || s2 == nil {
			t.Fatal("expected both players to be matched")
		}
		if s1.SessionID != s2.SessionID {
			t.Error("expected both players in the same session")
		}
	})

	t.Run("elo too far apart not matched", func(t *testing.T) {
		fleet := NewFleet(nil, internal.NilToken, 5*time.Minute)
		q := NewMatchQueue(fleet)

		p1 := &PlayerModel{PlayerID: uuid.New(), Elo: 1000}
		p2 := &PlayerModel{PlayerID: uuid.New(), Elo: 1500}

		if _, err := q.Queue(p1, &game.AllGames[0]); err != nil {
			t.Fatalf("queue p1: %v", err)
		}
		if _, err := q.Queue(p2, &game.AllGames[0]); err != nil {
			t.Fatalf("queue p2: %v", err)
		}

		q.findMatches()

		s1, _ := q.Queue(p1, &game.AllGames[0])
		s2, _ := q.Queue(p2, &game.AllGames[0])
		if s1 != nil || s2 != nil {
			t.Error("expected players with distant elo to remain unmatched")
		}
	})
}
