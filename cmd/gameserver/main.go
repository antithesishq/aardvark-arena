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
	"strings"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/gameserver"

	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/google/uuid"
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
	var selfURL url.URL
	flag.Func("self-url", "this server's externally reachable URL (used to identify itself to the matchmaker)", internal.URLParser(&selfURL))
	flag.Parse()

	id := uuid.New()

	// Derive self-url from addr when not explicitly provided.
	if selfURL.Host == "" {
		host := *addr
		if strings.HasPrefix(host, ":") {
			host = "localhost" + host
		}
		derived, _ := url.Parse("http://" + host)
		if derived != nil {
			selfURL = *derived
		}
	}

	log.Printf("starting gameserver %s...", internal.ShortID(id))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := gameserver.Config{
		ID:            id,
		TurnTimeout:   *turnTimeout,
		MaxSessions:   *maxSessions,
		MatchmakerURL: &matchmakerURL,
		SelfURL:       &selfURL,
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
