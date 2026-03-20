"use client";

import { useEffect, useState } from "react";
import { SessionSummary, fetchSessions, fetchHealth, HealthResponse } from "@/lib/api";
import { GameCard } from "./GameCard";
import { OnlineBadge, DegradedBadge } from "./badges";

interface Props {
  serverUrl: string;
  label: string;
}

export function GameServerSection({ serverUrl, label }: Props) {
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [startTimes, setStartTimes] = useState<Record<string, number>>({});

  useEffect(() => {
    let cancelled = false;

    async function poll() {
      try {
        const [h, s] = await Promise.all([fetchHealth(serverUrl), fetchSessions(serverUrl)]);
        if (cancelled) return;
        setHealth(h);
        setSessions(s ?? []);
        setStartTimes((prev) => {
          const next = { ...prev };
          for (const sess of s ?? []) {
            if (!next[sess.session_id]) next[sess.session_id] = Date.now();
          }
          return next;
        });
      } catch {
        // server unreachable — keep last state
      }
    }

    poll();
    const id = setInterval(poll, 3000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [serverUrl]);

  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const isDegraded = health?.Full ?? false;
  const sessionCount = sessions.length;

  return (
    <div className="mb-8">
      {/* Section header */}
      <div className="flex items-center gap-3 mb-3 border-b border-zinc-800 pb-2">
        <span className="text-sm font-bold text-zinc-300 tracking-widest" style={{ fontFamily: "var(--font-geist-mono)" }}>{label}</span>
        {isDegraded ? <DegradedBadge /> : <OnlineBadge />}
        <span className="ml-auto text-xs text-zinc-500">
          {sessionCount} session{sessionCount !== 1 ? "s" : ""} active
        </span>
      </div>

      {/* Game cards */}
      {sessionCount === 0 ? (
        <div className="text-xs text-zinc-600 py-4 text-center">No active sessions</div>
      ) : (
        <div className="flex flex-wrap gap-4">
          {sessions.map((s) => (
            <GameCard
              key={s.session_id}
              session={s}
              serverUrl={serverUrl}
              elapsedSeconds={Math.floor((now - (startTimes[s.session_id] ?? now)) / 1000)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
