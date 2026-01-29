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
var DefaultGameServer = "http://localhost:8081"

func main() {
	log.SetOutput(os.Stdout)

	var gameServers internal.URLList
	addr := flag.String("addr", ":8080", "server listen address")
	sessionTimeout := flag.Duration("session-timeout", DefaultSessionTimeout, "duration after which unfinished sessions are cancelled")
	flag.Var(&gameServers, "gameserver", "gameserver URL (can be repeated)")
	var token internal.Token
	flag.Var(&token, "key", "token for authenticating gameserver requests")
	flag.Parse()

	log.Println("starting matchmaker...")

	if len(gameServers) == 0 {
		defaultURL, _ := url.Parse(DefaultGameServer)
		gameServers = append(gameServers, defaultURL)
	}

	cfg := matchmaker.Config{
		SessionTimeout: *sessionTimeout,
		GameServers:    gameServers,
		Token:          token,
	}
	srv := matchmaker.New(cfg)
	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
