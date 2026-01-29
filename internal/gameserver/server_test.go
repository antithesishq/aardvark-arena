package gameserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/google/uuid"
)

func CheckHealth(t *testing.T, srv *Server, active int) HealthResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health check failed with status %d", rec.Code)
	}

	health, err := internal.BindJSON[HealthResponse](rec.Body)
	if err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if health.ActiveSessions != active {
		t.Errorf("expected ActiveSessions=%d, got %d", active, health.ActiveSessions)
	}

	return health
}

func TestServerSanity(t *testing.T) {
	srv := New(Config{
		TurnTimeout: 30 * time.Second,
		MaxSessions: 10,
	})

	t.Run("healthcheck", func(t *testing.T) {
		health := CheckHealth(t, srv, 0)
		if health.MaxSessions != 10 {
			t.Errorf("expected MaxSessions=10, got %d", health.MaxSessions)
		}
		if health.Full {
			t.Errorf("expected Full=false, got true")
		}
	})

	t.Run("create session", func(t *testing.T) {
		sid := uuid.New()
		body, err := internal.EncodeJSON(CreateSessionRequest{
			Game:     game.TicTacToe,
			Deadline: time.Now().Add(5 * time.Minute),
		})
		if err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/session/%s", sid), body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		CheckHealth(t, srv, 1)
	})
}
