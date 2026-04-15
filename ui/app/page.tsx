"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  fetchStatus,
  fetchLeaderboard,
  StatusResponse,
  LeaderboardEntry,
} from "@/lib/api";
import { PlayerQueueRing } from "@/components/PlayerQueueRing";
import { Leaderboard } from "@/components/Leaderboard";

function fmtAvgWait(sessions: StatusResponse["sessions"]): string {
  if (sessions.length === 0) {
    return "—";
  }
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
    const t = setTimeout(refresh, 0);
    const id = setInterval(refresh, 3000);
    return () => {
      clearTimeout(t);
      clearInterval(id);
    };
  }, [refresh]);

  const sessions = (status?.sessions ?? [])
    .slice()
    .sort((a, b) => a.session_id.localeCompare(b.session_id));
  const queue = (status?.queue ?? [])
    .slice()
    .sort((a, b) => a.player_id.localeCompare(b.player_id));
  const board = leaderboard ?? [];
  const queuedIds = useMemo(
    () => new Set(queue.map((p) => p.player_id)),
    [queue],
  );
  const playingIds = useMemo(
    () => new Set(sessions.flatMap((s) => s.player_ids)),
    [sessions],
  );

  return (
    <div className="max-w-7xl mx-auto px-6 space-y-4">
      {error && (
        <div className="bg-red-950 border border-red-800 text-red-300 text-xs px-4 py-2 rounded">
          Cannot reach matchmaker: {error}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <PlayerQueueRing queue={queue} avgWait={fmtAvgWait(sessions)} />
        <Leaderboard
          entries={board}
          queuedIds={queuedIds}
          playingIds={playingIds}
        />
      </div>
    </div>
  );
}
