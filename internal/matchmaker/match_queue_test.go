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
	"hegel.dev/go/hegel"
)

func MustDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite db: %v", err)
	}
	return db
}

func TestMatchQueueSanity(t *testing.T) {
	t.Run("queue returns nil before match", func(t *testing.T) {
		fleet := NewFleet(nil, internal.NilToken, 5*time.Minute)
		q := NewMatchQueue(fleet, MustDB(t))

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
		q := NewMatchQueue(fleet, MustDB(t))

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
		q.matchPlayers()

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
		} else if s1.SessionID != s2.SessionID {
			t.Error("expected both players in the same session")
		}
	})

	t.Run("elo too far apart not matched", func(t *testing.T) {
		fleet := NewFleet(nil, internal.NilToken, 5*time.Minute)
		q := NewMatchQueue(fleet, MustDB(t))

		p1 := &PlayerModel{PlayerID: uuid.New(), Elo: 1000}
		p2 := &PlayerModel{PlayerID: uuid.New(), Elo: 1500}

		if _, err := q.Queue(p1, &game.AllGames[0]); err != nil {
			t.Fatalf("queue p1: %v", err)
		}
		if _, err := q.Queue(p2, &game.AllGames[0]); err != nil {
			t.Fatalf("queue p2: %v", err)
		}

		q.matchPlayers()

		s1, _ := q.Queue(p1, &game.AllGames[0])
		s2, _ := q.Queue(p2, &game.AllGames[0])
		if s1 != nil || s2 != nil {
			t.Error("expected players with distant elo to remain unmatched")
		}
	})
}

func TestSelectMatchGameProperties(t *testing.T) {
	allKinds := game.AllGames[:]

	t.Run("same preference always matches", hegel.Case(func(ht *hegel.T) {
		k := hegel.Draw(ht, hegel.SampledFrom(allKinds))
		a := &k
		b := &k
		chosen, ok := selectMatchGame(a, b)
		if !ok {
			ht.Fatalf("same preference %v should match", k)
		}
		if chosen != k {
			ht.Fatalf("expected %v, got %v", k, chosen)
		}
	}))

	t.Run("different preferences do not match", hegel.Case(func(ht *hegel.T) {
		i := hegel.Draw(ht, hegel.Integers[int](0, len(allKinds)-1))
		j := hegel.Draw(ht, hegel.Integers[int](0, len(allKinds)-1))
		ht.Assume(i != j)
		a := &allKinds[i]
		b := &allKinds[j]
		_, ok := selectMatchGame(a, b)
		if ok {
			ht.Fatalf("different preferences %v and %v should not match", *a, *b)
		}
	}))

	t.Run("one preference is used", hegel.Case(func(ht *hegel.T) {
		k := hegel.Draw(ht, hegel.SampledFrom(allKinds))
		pref := &k

		// a has preference, b is nil
		chosen, ok := selectMatchGame(pref, nil)
		if !ok {
			ht.Fatalf("one preference should match")
		}
		if chosen != k {
			ht.Fatalf("expected preference %v, got %v", k, chosen)
		}

		// b has preference, a is nil
		chosen, ok = selectMatchGame(nil, pref)
		if !ok {
			ht.Fatalf("one preference should match")
		}
		if chosen != k {
			ht.Fatalf("expected preference %v, got %v", k, chosen)
		}
	}))

	t.Run("no preference returns a valid game", hegel.Case(func(ht *hegel.T) {
		chosen, ok := selectMatchGame(nil, nil)
		if !ok {
			ht.Fatal("nil/nil should always match")
		}
		valid := false
		for _, k := range game.AllGames {
			if chosen == k {
				valid = true
				break
			}
		}
		if !valid {
			ht.Fatalf("chosen game %v is not a valid kind", chosen)
		}
	}))
}
