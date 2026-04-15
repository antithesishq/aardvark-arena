package player

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
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

// startGameServer creates a gameserver httptest.Server.
func startGameServer(t *testing.T, mmURL *url.URL) *httptest.Server {
	t.Helper()
	gs := gameserver.New(context.Background(), gameserver.Config{
		ID:            uuid.New(),
		TurnTimeout:   30 * time.Second,
		MaxSessions:   100,
		MatchmakerURL: mmURL,
	})
	srv := httptest.NewServer(gs)
	return srv
}

// registerGameServer registers a game server with the matchmaker.
func registerGameServer(t *testing.T, mmURL *url.URL, gsURL *url.URL) {
	t.Helper()
	body, err := internal.EncodeJSON(matchmaker.RegisterRequest{
		ID:  uuid.New().String(),
		URL: gsURL.String(),
	})
	if err != nil {
		t.Fatalf("encode register request: %v", err)
	}
	resp, err := http.Post(mmURL.JoinPath("servers", "register").String(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register game server: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register game server: got %s", resp.Status)
	}
}

// startMatchmaker creates a matchmaker and gameserver, connects them together,
// and returns the matchmaker url.
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

	mm, err := matchmaker.New(context.Background(), matchmaker.Config{
		SessionTimeout:         5 * time.Minute,
		MatchInterval:          10 * time.Millisecond,
		SessionMonitorInterval: time.Minute,
		DatabasePath:           ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}
	mmHandler = mm

	// Register the game server with the matchmaker.
	registerGameServer(t, mmURL, gsURL)

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
			MatchmakerURL:             mmURL,
			PlayerID:                  uuid.New(),
			NumSessions:               10,
			PollInterval:              time.Millisecond * 10,
			SpecificGameSelectionRate: 0,
		})
		err := p1.Run(ctx)
		if err != nil {
			t.Errorf("player 1 run error: %v", err)
		}
	})

	wg.Go(func() {
		p2 := New(Config{
			MatchmakerURL:             mmURL,
			PlayerID:                  uuid.New(),
			NumSessions:               10,
			PollInterval:              time.Millisecond * 10,
			SpecificGameSelectionRate: 0,
		})
		err := p2.Run(ctx)
		if err != nil {
			t.Errorf("player 2 run error: %v", err)
		}
	})

	wg.Wait()
}
