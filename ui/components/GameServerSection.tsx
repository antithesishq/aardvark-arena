"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { WatchEvent, GameState } from "@/lib/ws";
import { watchServer } from "@/lib/ws";
import { GameCard } from "./GameCard";
import { ConnectedBadge, DisconnectedBadge } from "./badges";
import { mono } from "@/lib/utils";

const LINGER_MS = 2000;

export interface SessionState {
  session_id: string;
  game: "tictactoe" | "connect4" | "battleship";
  players: Record<string, number>;
  gameState: GameState | null;
}

interface Props {
  serverUrl: string;
  label: string;
}

export function GameServerSection({ serverUrl, label }: Props) {
  const [connected, setConnected] = useState(false);
  const [health, setHealth] = useState<{ active: number; max: number } | null>(null);
  const [sessions, setSessions] = useState<Map<string, SessionState>>(new Map());
  const [startTimes, setStartTimes] = useState<Record<string, number>>({});

  // Linger tracking for ended sessions
  const lingerTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const handleEvent = useCallback((evt: WatchEvent) => {
    if (evt.type === "health") {
      setHealth({ active: evt.active_sessions, max: evt.max_sessions });
    } else if (evt.type === "session") {
      // Cancel any pending linger removal
      const timer = lingerTimers.current.get(evt.session_id);
      if (timer) {
        clearTimeout(timer);
        lingerTimers.current.delete(evt.session_id);
      }

      setSessions((prev) => {
        const next = new Map(prev);
        next.set(evt.session_id, {
          session_id: evt.session_id,
          game: evt.game,
          players: evt.players,
          gameState: evt.state,
        });
        return next;
      });
      setStartTimes((prev) =>
        prev[evt.session_id] ? prev : { ...prev, [evt.session_id]: Date.now() },
      );
    } else if (evt.type === "session_end") {
      // Linger: keep the card visible briefly after session ends
      const timer = setTimeout(() => {
        lingerTimers.current.delete(evt.session_id);
        setSessions((prev) => {
          const next = new Map(prev);
          next.delete(evt.session_id);
          return next;
        });
        setStartTimes((prev) => {
          const { [evt.session_id]: _, ...rest } = prev;
          return rest;
        });
      }, LINGER_MS);
      lingerTimers.current.set(evt.session_id, timer);
    }
  }, []);

  useEffect(() => {
    const cleanup = watchServer(serverUrl, handleEvent, setConnected);
    return () => {
      cleanup();
      // Clear all linger timers on unmount
      for (const t of lingerTimers.current.values()) clearTimeout(t);
      lingerTimers.current.clear();
    };
  }, [serverUrl, handleEvent]);

  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const sorted = useMemo(
    () => Array.from(sessions.values()).sort((a, b) => a.session_id.localeCompare(b.session_id)),
    [sessions],
  );
  const isDegraded = health ? health.active >= health.max : false;

  return (
    <div className="mb-8">
      {/* Section header */}
      <div className="flex items-center gap-3 mb-3 border-b border-zinc-800 pb-2">
        <span className="text-sm font-bold text-zinc-300 tracking-widest" style={mono}>
          {label}
        </span>
        {connected ? (
          isDegraded ? <ConnectedBadge degraded /> : <ConnectedBadge />
        ) : (
          <DisconnectedBadge />
        )}
        <span className="ml-auto text-xs text-zinc-400">
          {sorted.length} session{sorted.length !== 1 ? "s" : ""} active
        </span>
      </div>

      {/* Game cards */}
      {sorted.length === 0 ? (
        <div className="text-xs text-zinc-400 py-4 text-center">No active sessions</div>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(288px,1fr))] gap-4">
          {sorted.map((s) => (
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
