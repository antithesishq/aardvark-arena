package matchmaker

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
)

func TestFleetSanity(t *testing.T) {
	t.Run("create session on healthy server", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL)
		fleet := NewFleet([]*url.URL{u}, internal.NilToken, 5*time.Minute)

		info, err := fleet.CreateSession(game.TicTacToe)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Game != game.TicTacToe {
			t.Errorf("expected game %s, got %s", game.TicTacToe, info.Game)
		}
		if info.SessionID.String() == "" {
			t.Error("expected non-empty session ID")
		}
	})

	t.Run("skip unavailable server", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			if calls == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL)
		// Two entries pointing at the same test server so the fleet has a
		// second candidate after the first returns 503.
		fleet := NewFleet([]*url.URL{u, u}, internal.NilToken, 5*time.Minute)

		info, err := fleet.CreateSession(game.TicTacToe)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info == nil {
			t.Fatal("expected session info, got nil")
		}
	})

	t.Run("all servers unavailable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL)
		fleet := NewFleet([]*url.URL{u}, internal.NilToken, 5*time.Minute)

		_, err := fleet.CreateSession(game.TicTacToe)
		if err != ErrNoServersAvailable {
			t.Fatalf("expected ErrNoServersAvailable, got %v", err)
		}
	})

	t.Run("no servers configured", func(t *testing.T) {
		fleet := NewFleet(nil, internal.NilToken, 5*time.Minute)

		_, err := fleet.CreateSession(game.TicTacToe)
		if err != ErrNoServersAvailable {
			t.Fatalf("expected ErrNoServersAvailable, got %v", err)
		}
	})
}
