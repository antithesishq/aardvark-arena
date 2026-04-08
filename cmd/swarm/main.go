// Package main implements the swarm binary which runs multiple players concurrently.
package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
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
	var numPlayers = flag.Int("n", 7, "number of players to run")
	var numSessions = flag.Int("num-sessions", 0, "number of sessions per player before exiting (0 = indefinite)")
	var pollInterval = flag.Duration("poll-interval", DefaultPollInterval, "duration between polling the matchmaker queue")
	var moveDelay = flag.Duration("move-delay", 0, "artificial delay before each move (e.g. 500ms, 1s)")
	var evil = flag.Bool("evil", false, "enable probabilistic malicious behavior")
	var evilChaosRate = flag.Float64("evil-chaos-rate", 0.30, "probability [0,1] of sending a bad move instead of a valid move")
	var evilOutOfTurnRate = flag.Float64("evil-out-of-turn-rate", 0.10, "probability [0,1] of sending a nuisance move out of turn")
	var evilMalformedRate = flag.Float64("evil-malformed-rate", 0.50, "probability [0,1] that a bad move is malformed JSON")
	var evilExtraConnectRate = flag.Float64("evil-extra-connect-rate", 0.08, "probability [0,1] of random-id background websocket join attempts")
	var evilQueueAbandonRate = flag.Float64("evil-queue-abandon-rate", 0.05, "probability [0,1] of queueing a random player id and never polling it again")
	flag.Parse()

	if matchmakerURL.Host == "" {
		parsed, _ := url.Parse(DefaultMatchmakerURL)
		matchmakerURL = *parsed
	}

	log.Printf("starting swarm with %d player(s)...", *numPlayers)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var wg sync.WaitGroup
	for i := 0; i < *numPlayers; i++ {
		wg.Add(1)
		cfg := player.Config{
			MatchmakerURL:            &matchmakerURL,
			PlayerID:                 uuid.New(),
			NumSessions:              *numSessions,
			PollInterval:             *pollInterval,
			MoveDelay:                *moveDelay,
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
		log.Printf("  player %d: %s", i+1, cfg.PlayerID)
		go func() {
			defer wg.Done()
			p := player.New(cfg)
			if err := p.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("player %s error: %v", cfg.PlayerID, err)
			}
		}()
	}

	wg.Wait()
	log.Println("swarm stopped")
}
