package player

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/antithesishq/aardvark-arena/internal/matchmaker"
)

// MatchmakerClient wraps HTTP communication with the matchmaker.
type MatchmakerClient struct {
	url    *url.URL
	pid    internal.PlayerID
	client *http.Client
}

// NewMatchmakerClient creates a MatchmakerClient for the given matchmaker URL and player.
func NewMatchmakerClient(baseURL *url.URL, pid internal.PlayerID) *MatchmakerClient {
	return &MatchmakerClient{
		url:    baseURL,
		pid:    pid,
		client: internal.NewHttpClient(),
	}
}

// Queue enqueues the player for a match. Returns non-nil SessionInfo when matched.
func (c *MatchmakerClient) Queue(ctx context.Context, pref *game.Kind) (*matchmaker.SessionInfo, error) {
	body, err := internal.EncodeJSON(matchmaker.QueueRequest{Game: pref})
	if err != nil {
		return nil, err
	}
	reqURL := c.url.JoinPath("queue", c.pid.String())
	req, err := http.NewRequestWithContext(ctx, "PUT", reqURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	info, err := internal.BindJSON[matchmaker.SessionInfo](resp.Body)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
