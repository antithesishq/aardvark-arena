// Package main implements the matchmaker binary.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"

	"github.com/antithesishq/antithesis-sdk-go/assert"
)

var DefaultSessionTimeout = 5 * time.Minute
var DefaultMatchInterval = time.Second
var DefaultSessionMonitorInterval = 5 * time.Second

func main() {
	log.SetOutput(os.Stdout)

	addr := flag.String("addr", ":8080", "server listen address")
	sessionTimeout := flag.Duration("session-timeout", DefaultSessionTimeout, "duration after which unfinished sessions are cancelled")
	matchInterval := flag.Duration("match-interval", DefaultMatchInterval, "interval between checking the match queue")
	sessionMonitorInterval := flag.Duration("monitor-interval", DefaultSessionMonitorInterval, "interval between checking for expired sessions")
	var token internal.Token
	flag.Var(&token, "token", "token for authenticating gameserver requests")
	databasePath := flag.String("db-path", ":memory:", "path to the SQLite database")
	flag.Parse()

	log.Println("starting matchmaker...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := matchmaker.Config{
		SessionTimeout:         *sessionTimeout,
		MatchInterval:          *matchInterval,
		SessionMonitorInterval: *sessionMonitorInterval,
		Token:                  token,
		DatabasePath:           *databasePath,
	}
	srv, err := matchmaker.New(ctx, cfg)
	if err != nil {
		log.Panicf("failed to create server: %v", err)
	}

	httpServer := &http.Server{Addr: *addr, Handler: srv}
	go func() {
		<-ctx.Done()
		log.Println("shutting down matchmaker...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	assert.Reachable("matchmaker startup path executed", nil)

	log.Printf("listening on %s", *addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Panicf("server error: %v", err)
	}
	log.Println("matchmaker stopped")
}
