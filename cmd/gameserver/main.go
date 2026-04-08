// Package main implements the gameserver binary.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"

	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func main() {
	log.SetOutput(os.Stdout)

	addr := flag.String("addr", ":8081", "server listen address")
	turnTimeout := flag.Duration("turn-timeout", 30*time.Second, "max duration for a player to submit a move")
	maxSessions := flag.Int("max-sessions", 100, "maximum concurrent sessions")
	var token internal.Token
	flag.Var(&token, "token", "token for authenticating with matchmaker")
	var matchmakerURL url.URL
	flag.Func("matchmaker", "matchmaker base URL", internal.URLParser(&matchmakerURL))
	flag.Parse()

	log.Println("starting gameserver...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := gameserver.Config{
		TurnTimeout:   *turnTimeout,
		MaxSessions:   *maxSessions,
		MatchmakerURL: &matchmakerURL,
		Token:         token,
	}
	srv := gameserver.New(ctx, cfg)

	httpServer := &http.Server{Addr: *addr, Handler: srv}
	go func() {
		<-ctx.Done()
		log.Println("shutting down gameserver...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	assert.Reachable("gameserver startup path executed", nil)

	log.Printf("listening on %s", *addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Panicf("server error: %v", err)
	}
	log.Println("gameserver stopped")
}
