package matchmaker

import (
	"net/url"
	"testing"
	"time"

	"github.com/antithesishq/aardvark-arena/internal"
	"github.com/antithesishq/aardvark-arena/internal/game"
	"github.com/google/uuid"
)

func TestDBSanity(t *testing.T) {
	t.Run("create and retrieve player", func(t *testing.T) {
		db := MustDB(t)
		pid := uuid.New()

		p, err := db.GetOrCreatePlayer(pid)
		if err != nil {
			t.Fatalf("GetOrCreatePlayer: %v", err)
		}
		if p.PlayerID != pid {
			t.Errorf("expected player ID %s, got %s", pid, p.PlayerID)
		}
		if p.Elo != internal.DefaultElo {
			t.Errorf("expected default elo %d, got %d", internal.DefaultElo, p.Elo)
		}
		if p.Wins != 0 || p.Losses != 0 || p.Draws != 0 {
			t.Errorf("expected 0/0/0 record, got %d/%d/%d", p.Wins, p.Losses, p.Draws)
		}
	})

	t.Run("create session", func(t *testing.T) {
		db := MustDB(t)
		p1 := uuid.New()
		p2 := uuid.New()

		_, err := db.GetOrCreatePlayer(p1)
		if err != nil {
			t.Fatalf("create player 1: %v", err)
		}
		_, err = db.GetOrCreatePlayer(p2)
		if err != nil {
			t.Fatalf("create player 2: %v", err)
		}

		sid := uuid.New()
		server, _ := url.Parse("http://localhost:8080")
		deadline := time.Now().Add(5 * time.Minute)

		s, err := db.CreateSession(sid, p1, p2, server, game.TicTacToe, deadline)
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if s.SessionID != sid {
			t.Errorf("expected session ID %s, got %s", sid, s.SessionID)
		}
		if s.Server != server.String() {
			t.Errorf("expected server %s, got %s", server, s.Server)
		}
		if s.Game != string(game.TicTacToe) {
			t.Errorf("expected game %s, got %s", game.TicTacToe, s.Game)
		}
	})
}

func TestCancelExpiredSessions(t *testing.T) {
	db := MustDB(t)
	p1 := uuid.New()
	p2 := uuid.New()
	for _, pid := range []internal.PlayerID{p1, p2} {
		if _, err := db.GetOrCreatePlayer(pid); err != nil {
			t.Fatalf("create player: %v", err)
		}
	}

	server, _ := url.Parse("http://localhost:8080")

	// Create a session with an already-expired deadline.
	expired := uuid.New()
	if _, err := db.CreateSession(expired, p1, p2, server, game.TicTacToe, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("CreateSession (expired): %v", err)
	}

	// Create a session with a future deadline that should not be cancelled.
	active := uuid.New()
	p3, p4 := uuid.New(), uuid.New()
	for _, pid := range []internal.PlayerID{p3, p4} {
		if _, err := db.GetOrCreatePlayer(pid); err != nil {
			t.Fatalf("create player: %v", err)
		}
	}
	if _, err := db.CreateSession(active, p3, p4, server, game.TicTacToe, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("CreateSession (active): %v", err)
	}

	// Run the session monitor's inner function.
	var cancelled []internal.SessionID
	db.cancelExpiredSessions(func(sid internal.SessionID) {
		cancelled = append(cancelled, sid)
	})

	// Only the expired session should have been cancelled.
	if len(cancelled) != 1 || cancelled[0] != expired {
		t.Fatalf("expected [%s] cancelled, got %v", expired, cancelled)
	}

	// Verify players in the expired session have unchanged stats and ELO.
	gotP1, _ := db.GetOrCreatePlayer(p1)
	gotP2, _ := db.GetOrCreatePlayer(p2)
	if gotP1.Elo != internal.DefaultElo || gotP2.Elo != internal.DefaultElo {
		t.Errorf("expected default elo, got p1=%d p2=%d", gotP1.Elo, gotP2.Elo)
	}
	if gotP1.Wins != 0 || gotP1.Losses != 0 || gotP1.Draws != 0 {
		t.Errorf("p1: expected 0/0/0, got %d/%d/%d", gotP1.Wins, gotP1.Losses, gotP1.Draws)
	}
	if gotP2.Wins != 0 || gotP2.Losses != 0 || gotP2.Draws != 0 {
		t.Errorf("p2: expected 0/0/0, got %d/%d/%d", gotP2.Wins, gotP2.Losses, gotP2.Draws)
	}

	// Running again should be a no-op since the expired session is now completed.
	cancelled = nil
	db.cancelExpiredSessions(func(sid internal.SessionID) {
		cancelled = append(cancelled, sid)
	})
	if len(cancelled) != 0 {
		t.Errorf("expected no sessions cancelled on second run, got %v", cancelled)
	}
}

func TestReportSessionResult(t *testing.T) {
	var (
		p1 = uuid.New()
		p2 = uuid.New()
	)

	tests := []struct {
		name      string
		cancelled bool
		winner    internal.PlayerID
		wantP1    PlayerModel
		wantP2    PlayerModel
	}{
		{
			name:   "p1 wins",
			winner: p1,
			wantP1: PlayerModel{Wins: 1, Losses: 0, Draws: 0},
			wantP2: PlayerModel{Wins: 0, Losses: 1, Draws: 0},
		},
		{
			name:   "p2 wins",
			winner: p2,
			wantP1: PlayerModel{Wins: 0, Losses: 1, Draws: 0},
			wantP2: PlayerModel{Wins: 1, Losses: 0, Draws: 0},
		},
		{
			name:   "draw",
			winner: uuid.Nil,
			wantP1: PlayerModel{Wins: 0, Losses: 0, Draws: 1},
			wantP2: PlayerModel{Wins: 0, Losses: 0, Draws: 1},
		},
		{
			name:      "cancelled",
			cancelled: true,
			winner:    uuid.Nil,
			wantP1:    PlayerModel{Wins: 0, Losses: 0, Draws: 0},
			wantP2:    PlayerModel{Wins: 0, Losses: 0, Draws: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := MustDB(t)

			for _, pid := range []internal.PlayerID{p1, p2} {
				if _, err := db.GetOrCreatePlayer(pid); err != nil {
					t.Fatalf("create player: %v", err)
				}
			}
			sid := uuid.New()
			server, _ := url.Parse("http://localhost:8080")
			if _, err := db.CreateSession(sid, p1, p2, server, game.TicTacToe, time.Now().Add(5*time.Minute)); err != nil {
				t.Fatalf("CreateSession: %v", err)
			}

			if err := db.ReportSessionResult(sid, tt.cancelled, tt.winner); err != nil {
				t.Fatalf("ReportSessionResult: %v", err)
			}

			gotP1, err := db.GetOrCreatePlayer(p1)
			if err != nil {
				t.Fatalf("get player 1: %v", err)
			}
			gotP2, err := db.GetOrCreatePlayer(p2)
			if err != nil {
				t.Fatalf("get player 2: %v", err)
			}

			if gotP1.Wins != tt.wantP1.Wins || gotP1.Losses != tt.wantP1.Losses || gotP1.Draws != tt.wantP1.Draws {
				t.Errorf("p1 record: got %d/%d/%d, want %d/%d/%d",
					gotP1.Wins, gotP1.Losses, gotP1.Draws,
					tt.wantP1.Wins, tt.wantP1.Losses, tt.wantP1.Draws)
			}
			if gotP2.Wins != tt.wantP2.Wins || gotP2.Losses != tt.wantP2.Losses || gotP2.Draws != tt.wantP2.Draws {
				t.Errorf("p2 record: got %d/%d/%d, want %d/%d/%d",
					gotP2.Wins, gotP2.Losses, gotP2.Draws,
					tt.wantP2.Wins, tt.wantP2.Losses, tt.wantP2.Draws)
			}

			if tt.cancelled {
				if gotP1.Elo != internal.DefaultElo || gotP2.Elo != internal.DefaultElo {
					t.Errorf("cancelled: expected default elo, got p1=%d p2=%d", gotP1.Elo, gotP2.Elo)
				}
			} else if tt.winner == uuid.Nil {
				if gotP1.Elo != gotP2.Elo {
					t.Errorf("draw: expected equal elos, got p1=%d p2=%d", gotP1.Elo, gotP2.Elo)
				}
			} else {
				winnerElo := gotP1.Elo
				loserElo := gotP2.Elo
				if tt.winner == p2 {
					winnerElo, loserElo = loserElo, winnerElo
				}
				if winnerElo <= loserElo {
					t.Errorf("expected winner elo > loser elo, got winner=%d loser=%d", winnerElo, loserElo)
				}
			}
		})
	}
}
