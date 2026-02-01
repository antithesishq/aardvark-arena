package player

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal/gameserver"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"
	"github.com/google/uuid"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// startGameServer creates a gameserver httptest.Server
func startGameServer(t *testing.T, mmURL *url.URL) *httptest.Server {
	t.Helper()
	gs := gameserver.New(gameserver.Config{
		TurnTimeout:   30 * time.Second,
		MaxSessions:   100,
		MatchmakerURL: mmURL,
	})
	srv := httptest.NewServer(gs)
	return srv
}

// startMatchmaker creates a matchmaker and gameserver, connects them together,
// and returns the matchmaker url
func startMatchmaker(t *testing.T) *url.URL {
	t.Helper()
	var mmHandler http.Handler
	mmHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mmHandler.ServeHTTP(w, r)
	}))
	mmURL := must(url.Parse(mmHTTP.URL))

	gs := startGameServer(t, mmURL)
	gsURL := must(url.Parse(gs.URL))

	t.Cleanup(func() {
		gs.Close()
		mmHTTP.Close()
	})

	mm, err := matchmaker.New(matchmaker.Config{
		SessionTimeout:         5 * time.Minute,
		MatchInterval:          10 * time.Millisecond,
		SessionMonitorInterval: time.Minute,
		GameServers:            []*url.URL{gsURL},
		DatabasePath:           ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}
	mmHandler = mm
	return mmURL
}

func TestIntegration(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lshortfile)

	mmURL := startMatchmaker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var wg sync.WaitGroup

	wg.Go(func() {
		p1 := New(Config{
			MatchmakerURL: mmURL,
			PlayerID:      uuid.New(),
			NumSessions:   10,
			PollInterval:  time.Millisecond * 10,
		})
		err := p1.Run(ctx)
		if err != nil {
			t.Errorf("player 1 run error: %v", err)
		}
	})

	wg.Go(func() {
		p2 := New(Config{
			MatchmakerURL: mmURL,
			PlayerID:      uuid.New(),
			NumSessions:   10,
			PollInterval:  time.Millisecond * 10,
		})
		err := p2.Run(ctx)
		if err != nil {
			t.Errorf("player 2 run error: %v", err)
		}
	})

	wg.Wait()
}
