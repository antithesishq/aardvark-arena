package gameserver

import (
	"bytes"
	"context"
	"fmt"
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
	token         internal.Token
	matchmakerURL *url.URL
	client        *http.Client
}

// NewReporter creates a Reporter that sends results to the given matchmaker URL.
func NewReporter(resultCh chan resultMsg, token internal.Token, matchmaker *url.URL) *Reporter {
	return &Reporter{
		resultCh:      resultCh,
		token:         token,
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
	if !r.token.IsNil() {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.token.String()))
	}
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
		assert.Reachable(
			"result reporting sometimes fails due to non-temporary errors",
			map[string]any{"sid": result.sid.String()},
		)
		log.Printf("failed to submit result for session %s: %v", result.sid, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	assert.Sometimes(
		resp.StatusCode != http.StatusOK,
		"result reporting sometimes receives non-ok responses",
		map[string]any{
			"sid":    result.sid.String(),
			"status": resp.StatusCode,
		},
	)
}
