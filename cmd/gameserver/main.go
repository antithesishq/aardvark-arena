// Package main implements the gameserver binary.
package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/antithesishq/aardvark-arena/internal/gameserver"
)

func main() {
	log.SetOutput(os.Stdout)

	addr := flag.String("addr", ":8080", "server listen address")
	turnTimeout := flag.Duration("turn-timeout", 30*time.Second, "max duration for a player to submit a move")
	maxSessions := flag.Int("max-sessions", 100, "maximum concurrent sessions")
	flag.Parse()

	log.Println("starting gameserver...")

	cfg := gameserver.Config{
		TurnTimeout: *turnTimeout,
		MaxSessions: *maxSessions,
	}
	srv := gameserver.New(cfg)
	if err := srv.ListenAndServe(*addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
