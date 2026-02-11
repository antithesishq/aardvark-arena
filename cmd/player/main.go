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
	var evil = flag.Bool("evil", false, "enable probabilistic malicious behavior")
	var evilChaosRate = flag.Float64("evil-chaos-rate", 0.30, "probability [0,1] of sending a bad move instead of a valid move")
	var evilOutOfTurnRate = flag.Float64("evil-out-of-turn-rate", 0.10, "probability [0,1] of sending a nuisance move out of turn")
	var evilMalformedRate = flag.Float64("evil-malformed-rate", 0.50, "probability [0,1] that a bad move is malformed JSON")
	var evilExtraConnectRate = flag.Float64("evil-extra-connect-rate", 0.08, "probability [0,1] of random-id background websocket join attempts")
	var evilQueueAbandonRate = flag.Float64("evil-queue-abandon-rate", 0.05, "probability [0,1] of queueing a random player id and never polling it again")
	flag.Parse()

	log.Println("starting player...")

	if matchmakerURL.Host == "" {
		parsed, _ := url.Parse(DefaultMatchmakerURL)
		matchmakerURL = *parsed
	}

	log.Printf("player id: %s", playerID)

	cfg := player.Config{
		MatchmakerURL:             &matchmakerURL,
		PlayerID:                  playerID,
		NumSessions:               *numSessions,
		PollInterval:              *pollInterval,
		SpecificGameSelectionRate: player.DefaultSpecificGameSelectionRate,
		Behavior: player.Behavior{
			Evil:             *evil,
			ChaosRate:        *evilChaosRate,
			OutOfTurnRate:    *evilOutOfTurnRate,
			MalformedRate:    *evilMalformedRate,
			ExtraConnectRate: *evilExtraConnectRate,
			QueueAbandonRate: *evilQueueAbandonRate,
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	player := player.New(cfg)
	if err := player.Run(ctx); err != nil {
		log.Fatalf("player error: %v", err)
	}
}
