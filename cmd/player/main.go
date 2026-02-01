// Package main implements the player binary.
package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/player"
	"github.com/google/uuid"
)

var DefaultMatchmakerURL = "http://localhost:8080"
var DefaultPollInterval = time.Second

func main() {
	log.SetOutput(os.Stdout)

	var matchmakerURL url.URL
	flag.Func("matchmaker", "matchmaker base URL (default http://localhost:8080)", internal.URLParser(&matchmakerURL))
	playerID := uuid.New()
	flag.Func("pid", "player UUID (generated if empty)", internal.UUIDParser(&playerID))
	var numSessions = flag.Int("num-sessions", 0, "number of sessions to play before exiting (0 to play indefinitely)")
	var pollInterval = flag.Duration("poll-interval", DefaultPollInterval, "duration between polling the matchmaker queue for a session")
	flag.Parse()

	log.Println("starting player...")

	if matchmakerURL.Host == "" {
		parsed, _ := url.Parse(DefaultMatchmakerURL)
		matchmakerURL = *parsed
	}

	log.Printf("player id: %s", playerID)

	cfg := player.Config{
		MatchmakerURL: &matchmakerURL,
		PlayerID:      playerID,
		NumSessions:   *numSessions,
		PollInterval:  *pollInterval,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	player := player.New(cfg)
	if err := player.Run(ctx); err != nil {
		log.Fatalf("player error: %v", err)
	}
}
