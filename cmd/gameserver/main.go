// Package main implements the gameserver binary.
package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"
)

func main() {
	log.SetOutput(os.Stdout)

	addr := flag.String("addr", ":8081", "server listen address")
	turnTimeout := flag.Duration("turn-timeout", 30*time.Second, "max duration for a player to submit a move")
	maxSessions := flag.Int("max-sessions", 100, "maximum concurrent sessions")
	var token internal.Token
	flag.Var(&token, "token", "token for authenticating with matchmaker")
	var matchmakerURL *url.URL
	flag.Func("matchmaker-url", "matchmaker base URL", internal.URLParser(matchmakerURL))
	flag.Parse()

	log.Println("starting gameserver...")

	cfg := gameserver.Config{
		TurnTimeout:   *turnTimeout,
		MaxSessions:   *maxSessions,
		MatchmakerURL: matchmakerURL,
		Token:         token,
	}
	srv := gameserver.New(cfg)
	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Panicf("server error: %v", err)
	}
}
