package gameserver

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	t.Run("healthcheck", func(t *testing.T) {
		srv := New(Config{
			TurnTimeout: 30 * time.Second,
			MaxSessions: 10,
		})
		health := CheckHealth(t, srv, 0)
		if health.MaxSessions != 10 {
			t.Errorf("expected MaxSessions=10, got %d", health.MaxSessions)
		}
		if health.Full {
			t.Errorf("expected Full=false, got true")
		}
	})

	t.Run("create session", func(t *testing.T) {
		srv := New(Config{
			TurnTimeout: 30 * time.Second,
			MaxSessions: 10,
		})
		sid := uuid.New()
		body, err := internal.EncodeJSON(CreateSessionRequest{
			Game:    game.TicTacToe,
			Timeout: 5 * time.Minute,
		})
		if err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/session/%s", sid), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		CheckHealth(t, srv, 1)
	})

	t.Run("sessions full", func(t *testing.T) {
		sessionTimeout := 5 * time.Minute
		srv := New(Config{
			TurnTimeout: 30 * time.Second,
			MaxSessions: 1,
		})

		sid := uuid.New()
		body, err := internal.EncodeJSON(CreateSessionRequest{
			Game:    game.TicTacToe,
			Timeout: sessionTimeout,
		})
		if err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/session/%s", sid), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		// second request should fail due to MaxSessions
		sid = uuid.New()
		body, err = internal.EncodeJSON(CreateSessionRequest{
			Game:    game.TicTacToe,
			Timeout: sessionTimeout,
		})
		if err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}

		req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/session/%s", sid), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d: %s", rec.Code, rec.Body.String())
		}
		retryAfterSecs, err := strconv.Atoi(rec.Header().Get("Retry-After"))
		if err != nil {
			t.Fatal("failed to parse retry-after")
		}
		if retryAfterSecs != int(sessionTimeout.Seconds()) {
			t.Fatalf("incorrect retry-after deadline; got=%d; expected=%d", retryAfterSecs, int(sessionTimeout.Seconds()))
		}

		CheckHealth(t, srv, 1)
	})
}
