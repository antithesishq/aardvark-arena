package gameserver

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/google/uuid"
)

func CheckHealth(t *testing.T, srv *Server, active int, wantActive ...bool) HealthResponse {
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

	if len(wantActive) > 0 && health.Active != wantActive[0] {
		t.Errorf("expected Active=%v, got %v", wantActive[0], health.Active)
	}

	return health
}

func TestServerSanity(t *testing.T) {

	t.Run("healthcheck", func(t *testing.T) {
		srv := New(context.Background(), Config{
			ID:          uuid.New(),
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
		srv := New(context.Background(), Config{
			ID:          uuid.New(),
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
		srv := New(context.Background(), Config{
			ID:          uuid.New(),
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
		if retryAfterSecs != int(math.Ceil(sessionTimeout.Seconds())) {
			t.Fatalf("incorrect retry-after deadline; got=%d; expected=%d", retryAfterSecs, int(sessionTimeout.Seconds()))
		}

		CheckHealth(t, srv, 1)
	})

	t.Run("drain rejects new sessions", func(t *testing.T) {
		srv := New(context.Background(), Config{
			ID:          uuid.New(),
			TurnTimeout: 30 * time.Second,
			MaxSessions: 10,
		})
		CheckHealth(t, srv, 0, true)

		// Drain the server
		req := httptest.NewRequest(http.MethodPost, "/drain", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("drain: expected 200, got %d", rec.Code)
		}

		CheckHealth(t, srv, 0, false)

		// Attempt to create a session — should get 503
		sid := uuid.New()
		body, err := internal.EncodeJSON(CreateSessionRequest{
			Game:    game.TicTacToe,
			Timeout: 5 * time.Minute,
		})
		if err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/session/%s", sid), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 while drained, got %d: %s", rec.Code, rec.Body.String())
		}
		retryAfterSecs, err := strconv.Atoi(rec.Header().Get("Retry-After"))
		if err != nil {
			t.Fatal("failed to parse Retry-After header")
		}
		if retryAfterSecs != 300 {
			t.Fatalf("expected Retry-After=300, got %d", retryAfterSecs)
		}
	})

	t.Run("activate resumes accepting sessions", func(t *testing.T) {
		srv := New(context.Background(), Config{
			ID:          uuid.New(),
			TurnTimeout: 30 * time.Second,
			MaxSessions: 10,
		})

		// Drain then activate
		req := httptest.NewRequest(http.MethodPost, "/drain", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		req = httptest.NewRequest(http.MethodPost, "/activate", nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("activate: expected 200, got %d", rec.Code)
		}

		CheckHealth(t, srv, 0, true)

		// Session creation should succeed
		sid := uuid.New()
		body, err := internal.EncodeJSON(CreateSessionRequest{
			Game:    game.TicTacToe,
			Timeout: 5 * time.Minute,
		})
		if err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/session/%s", sid), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 after activate, got %d: %s", rec.Code, rec.Body.String())
		}

		CheckHealth(t, srv, 1, true)
	})

	t.Run("existing sessions survive drain", func(t *testing.T) {
		srv := New(context.Background(), Config{
			ID:          uuid.New(),
			TurnTimeout: 30 * time.Second,
			MaxSessions: 10,
		})

		// Create a session first
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
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		CheckHealth(t, srv, 1, true)

		// Now drain
		req = httptest.NewRequest(http.MethodPost, "/drain", nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		// Existing session still counted, server not active
		health := CheckHealth(t, srv, 1, false)
		if health.ActiveSessions != 1 {
			t.Errorf("expected session to survive drain, got ActiveSessions=%d", health.ActiveSessions)
		}
	})

	t.Run("cancel all sessions", func(t *testing.T) {
		// Provide a dummy matchmaker so the reporter doesn't panic.
		mm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer mm.Close()
		mmURL, _ := url.Parse(mm.URL)
		srv := New(context.Background(), Config{
			ID:            uuid.New(),
			TurnTimeout:   30 * time.Second,
			MaxSessions:   10,
			MatchmakerURL: mmURL,
		})

		// Create two sessions
		for range 2 {
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
				t.Fatalf("expected 200, got %d", rec.Code)
			}
		}

		CheckHealth(t, srv, 2)

		// Cancel all sessions
		req := httptest.NewRequest(http.MethodDelete, "/sessions", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("cancel all: expected 200, got %d", rec.Code)
		}

		// Sessions are cancelled asynchronously — wait briefly for cleanup
		time.Sleep(50 * time.Millisecond)
		CheckHealth(t, srv, 0)
	})
}
