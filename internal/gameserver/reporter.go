package gameserver

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/antithesishq/aardvark-arena/internal"
)

// Reporter sends results back to the matchmaker
type Reporter struct {
	resultCh      chan resultMsg
	token         internal.Token
	matchmakerURL *url.URL
	client        *http.Client
}

func NewReporter(resultCh chan resultMsg, token internal.Token, matchmaker *url.URL) *Reporter {
	return &Reporter{
		resultCh:      resultCh,
		token:         token,
		matchmakerURL: matchmaker,
		client:        internal.NewHttpClient(),
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
	_, err = r.client.Do(req)
	if urlerr, ok := err.(*url.Error); ok {
		if urlerr.Temporary() {
			r.resultCh <- result
			return
		}
	}
	if err != nil {
		log.Panicf("failed to submit result for session %s: %v", result.sid, err)
	}
}
