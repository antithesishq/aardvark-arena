package gameserver

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/google/uuid"
)

// Reporter sends results back to the matchmaker.
type Reporter struct {
	resultCh      chan resultMsg
	matchmakerURL *url.URL
	client        *http.Client
}

// NewReporter creates a Reporter that sends results to the given matchmaker URL.
func NewReporter(resultCh chan resultMsg, matchmaker *url.URL) *Reporter {
	return &Reporter{
		resultCh:      resultCh,
		matchmakerURL: matchmaker,
		client:        internal.NewHTTPClient(),
	}
}

// StartReporter begins processing results in the background.
// It stops when the context is cancelled.
func (r *Reporter) StartReporter(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-r.resultCh:
				if !ok {
					return
				}
				r.submitResult(result)
			}
		}
	}()
}

func (r *Reporter) submitResult(result resultMsg) {
	assert.Always(
		!result.cancelled || result.winner == uuid.Nil,
		"gameserver reports never include a winner for cancelled sessions",
		map[string]any{"sid": result.sid.String()},
	)
	reqURL := r.matchmakerURL.JoinPath("results", result.sid.String())
	type resultReq struct {
		Cancelled bool
		Winner    internal.PlayerID
	}
	body, err := internal.EncodeJSON(resultReq{
		Cancelled: result.cancelled,
		Winner:    result.winner,
	})
	if err != nil {
		log.Panicf("failed to encode json: %v", err)
	}
	req, err := http.NewRequest("PUT", reqURL.String(), bytes.NewReader(body))
	if err != nil {
		log.Panicf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if internal.HTTPIsTemporary(err) {
		assert.Reachable(
			"result reporting sometimes retries after temporary transport errors",
			map[string]any{"sid": result.sid.String()},
		)
		r.resultCh <- result
		return
	}
	if err != nil {
		log.Printf("failed to submit result for session %s: %v", result.sid, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
}
