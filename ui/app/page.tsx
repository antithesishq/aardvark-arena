"use client";

import { useEffect, useState } from "react";
import { fetchStatus, fetchLeaderboard, StatusResponse, LeaderboardEntry } from "@/lib/api";
import { StatCard } from "@/components/StatCard";
import { PlayerQueue } from "@/components/PlayerQueue";
import { Leaderboard } from "@/components/Leaderboard";
import { ActiveSessions } from "@/components/ActiveSessions";

function fmtAvgWait(sessions: StatusResponse["sessions"]): string {
  if (sessions.length === 0) return "—";
  const now = Date.now();
  const avg =
    sessions.reduce((sum, s) => sum + (now - new Date(s.created_at).getTime()), 0) /
    sessions.length /
    1000;
  const m = Math.floor(avg / 60);
  const s = Math.floor(avg % 60);
  return `${m}:${String(s).padStart(2, "0")}`;
}

export default function MatchmakerPage() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
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
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, []);

  const sessions = status?.sessions ?? [];
  const queue = status?.queue ?? [];

  return (
    <div className="max-w-7xl mx-auto space-y-6">
      {error && (
        <div className="bg-red-950 border border-red-800 text-red-300 text-xs px-4 py-2 rounded">
          Cannot reach matchmaker: {error}
        </div>
      )}

      {/* Stat cards */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
        <StatCard
          label="Active Sessions"
          value={sessions.length}
          sub={sessions.length === 1 ? "1 server" : `across ${new Set(sessions.map((s) => s.server)).size} servers`}
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
        <StatCard
          label="Leaderboard"
          value={leaderboard.length}
          sub="tracked players"
        />
        <StatCard
          label="Top ELO"
          value={leaderboard[0]?.Elo ?? "—"}
          sub={leaderboard[0] ? leaderboard[0].PlayerID.slice(0, 8) : "no data"}
        />
      </div>

      {/* Queue + Leaderboard */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <PlayerQueue queue={queue} />
        <Leaderboard entries={leaderboard} />
      </div>

      {/* Active sessions table */}
      <ActiveSessions sessions={sessions} />
    </div>
  );
}
