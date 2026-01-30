// Package main implements the matchmaker binary.
package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"
)

var DefaultSessionTimeout = 5 * time.Minute
var DefaultMatchInterval = time.Second
var DefaultGameServer = "http://localhost:8081"

func main() {
	log.SetOutput(os.Stdout)

	var gameServers internal.URLList
	addr := flag.String("addr", ":8080", "server listen address")
	sessionTimeout := flag.Duration("session-timeout", DefaultSessionTimeout, "duration after which unfinished sessions are cancelled")
	matchInterval := flag.Duration("match-interval", DefaultMatchInterval, "interval between checking the match queue")
	flag.Var(&gameServers, "gameserver", "gameserver URL (can be repeated)")
	var token internal.Token
	flag.Var(&token, "key", "token for authenticating gameserver requests")
	databasePath := flag.String("db-path", ":memory:", "path to the SQLite database")
	flag.Parse()

	log.Println("starting matchmaker...")

	if len(gameServers) == 0 {
		defaultURL, _ := url.Parse(DefaultGameServer)
		gameServers = append(gameServers, defaultURL)
	}

	cfg := matchmaker.Config{
		SessionTimeout: *sessionTimeout,
		MatchInterval:  *matchInterval,
		GameServers:    gameServers,
		Token:          token,
		DatabasePath:   *databasePath,
	}
	srv, err := matchmaker.New(cfg)
	if err != nil {
		log.Panicf("failed to create server: %v", err)
	}
	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Panicf("server error: %v", err)
	}
}
