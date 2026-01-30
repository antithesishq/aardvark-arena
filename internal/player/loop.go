package player

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
)

type Config struct {
	MatchmakerURL *url.URL
	PlayerID      internal.PlayerID
}

type Loop struct {
}

func New(cfg Config) *Loop {
	return &Loop{}
}

func (l *Loop) Run(ctx context.Context) error {
	for {
		time.Sleep(time.Second)
		log.Print("looping")
	}
}
