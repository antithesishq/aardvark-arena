"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { fetchStatus, fetchLeaderboard, StatusResponse, LeaderboardEntry } from "@/lib/api";
import { StatCard } from "@/components/StatCard";
import { PlayerQueue } from "@/components/PlayerQueue";
import { Leaderboard } from "@/components/Leaderboard";
import { ActiveSessions } from "@/components/ActiveSessions";

function makeDemoStatus(): StatusResponse {
  return {
    queue: [
      { player_id: "kazimir-7-uuid-0001", elo: 1842, wait_seconds: 23 },
      { player_id: "nyx-alpha-uuid-0002", elo: 2105, wait_seconds: 67 },
      { player_id: "drax-iii-uuid-0003", elo: 1677, wait_seconds: 45 },
      { player_id: "zerograd-uuid-0004", elo: 1923, wait_seconds: 151 },
    ],
    sessions: [
      { session_id: "a3f1b2c3-0000-0000-0000-000000000001", server: "http://gs-01:8081", game: "tictactoe",   player_ids: ["zephyr-9-uuid-0005", "havik-uuid-00006"], created_at: new Date(Date.now() - 192000).toISOString(), deadline: new Date(Date.now() + 60000).toISOString() },
      { session_id: "b7c2d3e4-0000-0000-0000-000000000002", server: "http://gs-01:8081", game: "connect4",   player_ids: ["mindforge-uuid-007", "kazimir-7-uuid-0001"], created_at: new Date(Date.now() - 464000).toISOString(), deadline: new Date(Date.now() + 60000).toISOString() },
      { session_id: "c0d5e6f7-0000-0000-0000-000000000003", server: "http://gs-01:8081", game: "battleship", player_ids: ["nyx-alpha-uuid-0002", "vanta-uuid-00008"], created_at: new Date(Date.now() - 725000).toISOString(), deadline: new Date(Date.now() + 60000).toISOString() },
      { session_id: "e2a9f0b1-0000-0000-0000-000000000004", server: "http://gs-02:8081", game: "connect4",   player_ids: ["drax-iii-uuid-0003", "riot-2-uuid-00009"], created_at: new Date(Date.now() - 1727000).toISOString(), deadline: new Date(Date.now() + 60000).toISOString() },
      { session_id: "f9b3a2c1-0000-0000-0000-000000000005", server: "http://gs-02:8081", game: "tictactoe",  player_ids: ["zerograd-uuid-0004", "helix-5-uuid-0010"], created_at: new Date(Date.now() - 320000).toISOString(), deadline: new Date(Date.now() + 60000).toISOString() },
    ],
  };
}

const DEMO_LEADERBOARD: LeaderboardEntry[] = [
  { PlayerID: "nyx-alpha-uuid-0002", Elo: 2105, Wins: 142, Losses: 38, Draws: 4 },
  { PlayerID: "mindforge-uuid-007",  Elo: 2044, Wins: 119, Losses: 41, Draws: 7 },
  { PlayerID: "zephyr-9-uuid-0005",  Elo: 1971, Wins: 98,  Losses: 55, Draws: 3 },
  { PlayerID: "zerograd-uuid-0004",  Elo: 1923, Wins: 87,  Losses: 60, Draws: 5 },
  { PlayerID: "kazimir-7-uuid-0001", Elo: 1842, Wins: 76,  Losses: 71, Draws: 2 },
  { PlayerID: "havik-uuid-00006",    Elo: 1789, Wins: 65,  Losses: 80, Draws: 6 },
];

function fmtAvgWait(sessions: StatusResponse["sessions"]): string {
  if (sessions.length === 0) return "—";
  const now = Date.now();
  const avg =
    sessions.reduce((sum, s) => sum + (now - new Date(s.created_at).getTime()), 0) /
    sessions.length / 1000;
  const m = Math.floor(avg / 60);
  const s = Math.floor(avg % 60);
  return `${m}:${String(s).padStart(2, "0")}`;
}

export default function MatchmakerPage() {
  const params = useSearchParams();
  const demo = params.get("demo") === "1";

  const [demoStatus] = useState<StatusResponse>(makeDemoStatus);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (demo) return;
    let cancelled = false;

    async function poll() {
      try {
        const [s, l] = await Promise.all([fetchStatus(), fetchLeaderboard()]);
        if (cancelled) return;
        setStatus(s);
        setLeaderboard(l);
        setError(null);
      } catch (e) {
        if (!cancelled) setError(String(e));
      }
    }

    poll();
    const id = setInterval(poll, 3000);
    return () => { cancelled = true; clearInterval(id); };
  }, [demo]);

  const sessions = demo ? demoStatus.sessions : (status?.sessions ?? []);
  const queue    = demo ? demoStatus.queue    : (status?.queue ?? []);
  const board    = demo ? DEMO_LEADERBOARD     : (leaderboard ?? []);

  return (
    <div className="max-w-7xl mx-auto px-6 space-y-4">
      {error && !demo && (
        <div className="bg-red-950 border border-red-800 text-red-300 text-xs px-4 py-2 rounded">
          Cannot reach matchmaker: {error}
        </div>
      )}

      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
        <StatCard
          label="Total Players"
          value={sessions.length * 2 + queue.length}
          sub="playing or queued"
        />
        <StatCard
          label="Active Sessions"
          value={sessions.length}
          sub={`across ${new Set(sessions.map((s) => s.server)).size} servers`}
        />
        <StatCard
          label="Queue Depth"
          value={queue.length}
          sub={queue.length > 0 ? `+${queue.length} waiting` : "empty"}
        />
        <StatCard
          label="Avg Wait Time"
          value={fmtAvgWait(sessions)}
          sub="target < 0:45"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <PlayerQueue queue={queue} />
        <Leaderboard entries={board} />
      </div>

      <ActiveSessions sessions={sessions} />
    </div>
  );
}
