package gameserver

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
)

// Reporter sends results back to the matchmaker
type Reporter struct {
	resultCh      chan resultMsg
	token         internal.Token
	matchmakerURL *url.URL
	client        *http.Client
}

func NewReporter(resultCh chan resultMsg, token internal.Token, matchmaker *url.URL) *Reporter {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	return &Reporter{
		resultCh:      resultCh,
		token:         token,
		matchmakerURL: matchmaker,
		client:        client,
	}
}

func (r *Reporter) StartReporter() {
	go func() {
		for result := range r.resultCh {
			r.submitResult(result)
		}
	}()
}

func (r *Reporter) submitResult(result resultMsg) {
	reqURL := r.matchmakerURL.JoinPath("results", result.sid.String())
	body, err := internal.EncodeJSON(struct{ Status game.Status }{Status: result.status})
	if err != nil {
		log.Fatalf("failed to encode json: %v", err)
	}
	req, err := http.NewRequest("PUT", reqURL.String(), bytes.NewReader(body))
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if !r.token.IsNil() {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.token.String()))
	}
	_, err = r.client.Do(req)
	if urlerr, ok := err.(*url.Error); ok {
		if urlerr.Temporary() {
			r.resultCh <- result
			return
		}
	}
	if err != nil {
		log.Fatalf("failed to submit result for session %s: %v", result.sid, err)
	}
}
