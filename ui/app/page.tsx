"use client";

import { useCallback, useEffect, useState } from "react";
import {
  fetchStatus,
  fetchLeaderboard,
  StatusResponse,
  LeaderboardEntry,
} from "@/lib/api";
import { StatCard } from "@/components/StatCard";
import { PlayerQueue } from "@/components/PlayerQueue";
import { Leaderboard } from "@/components/Leaderboard";
import { ActiveSessions } from "@/components/ActiveSessions";

function fmtAvgWait(sessions: StatusResponse["sessions"]): string {
  if (sessions.length === 0) return "—";
  const now = Date.now();
  const avg =
    sessions.reduce(
      (sum, s) => sum + (now - new Date(s.created_at).getTime()),
      0,
    ) /
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

  const refresh = useCallback(async () => {
    try {
      const [s, l] = await Promise.all([fetchStatus(), fetchLeaderboard()]);
      setStatus(s);
      setLeaderboard(l);
      setError(null);
    } catch (e) {
      setError(String(e));
    }
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 3000);
    return () => clearInterval(id);
  }, [refresh]);

  const sessions = (status?.sessions ?? [])
    .slice()
    .sort((a, b) => a.session_id.localeCompare(b.session_id));
  const queue = (status?.queue ?? [])
    .slice()
    .sort((a, b) => a.player_id.localeCompare(b.player_id));
  const board = leaderboard ?? [];

  return (
    <div className="max-w-7xl mx-auto px-6 space-y-4">
      {error && (
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
          sub={queue.length > 0 ? `players waiting` : "empty"}
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

      <ActiveSessions sessions={sessions} onRefresh={refresh} />
    </div>
  );
}
