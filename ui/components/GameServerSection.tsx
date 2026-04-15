"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { WatchEvent, GameState } from "@/lib/ws";
import { watchServer } from "@/lib/ws";
import { GameCard } from "./GameCard";

const LINGER_MS = 2000;

export interface SessionState {
  session_id: string;
  game: "tictactoe" | "connect4" | "battleship";
  players: Record<string, number>;
  gameState: GameState | null;
}

export interface ServerHealth {
  connected: boolean;
  active: number;
  max: number;
  degraded: boolean;
  enabled: boolean;
}

interface Props {
  serverUrl: string;
  hidden?: boolean;
  onHealthChange?: (health: ServerHealth) => void;
}

export function GameServerSection({
  serverUrl,
  hidden,
  onHealthChange,
}: Props) {
  const [connected, setConnected] = useState(false);
  const [health, setHealth] = useState<{
    active: number;
    max: number;
    serverActive: boolean;
  } | null>(null);
  const [sessions, setSessions] = useState<Map<string, SessionState>>(
    new Map(),
  );
  const [startTimes, setStartTimes] = useState<Record<string, number>>({});

  // Linger tracking for ended sessions
  const lingerTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map(),
  );

  const handleEvent = useCallback((evt: WatchEvent) => {
    if (evt.type === "health") {
      setHealth({
        active: evt.active_sessions,
        max: evt.max_sessions,
        serverActive: evt.active,
      });
    } else if (evt.type === "session") {
      // Ignore updates for sessions already ending — prevents race where a
      // late "session" event cancels the linger timer and the card sticks.
      if (lingerTimers.current.has(evt.session_id)) {
        return;
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
          const rest = { ...prev };
          delete rest[evt.session_id];
          return rest;
        });
      }, LINGER_MS);
      lingerTimers.current.set(evt.session_id, timer);
    }
  }, []);

  useEffect(() => {
    const timers = lingerTimers.current;
    const cleanup = watchServer(serverUrl, handleEvent, (conn) => {
      if (conn) {
        // Clear stale sessions on (re)connect — server will re-send active ones
        setSessions(new Map());
        setStartTimes({});
        for (const t of timers.values()) {
          clearTimeout(t);
        }
        timers.clear();
      }
      setConnected(conn);
    });
    return () => {
      cleanup();
      // Clear all linger timers on unmount
      for (const t of timers.values()) {
        clearTimeout(t);
      }
      timers.clear();
    };
  }, [serverUrl, handleEvent]);

  // Bubble health state up to parent for tab rendering
  useEffect(() => {
    onHealthChange?.({
      connected,
      active: health?.active ?? 0,
      max: health?.max ?? 0,
      degraded: health ? health.active >= health.max : false,
      enabled: health?.serverActive ?? true,
    });
  }, [connected, health, onHealthChange]);

  const [now, setNow] = useState(0);
  useEffect(() => {
    const tick = () => setNow(Date.now());
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, []);

  const sorted = useMemo(
    () =>
      Array.from(sessions.values()).sort(
        (a, b) =>
          (startTimes[a.session_id] ?? 0) - (startTimes[b.session_id] ?? 0),
      ),
    [sessions, startTimes],
  );
  if (hidden) {
    return null;
  }

  return (
    <div>
      {/* Game cards */}
      {sorted.length === 0 ? (
        <div className="text-xs text-zinc-400 py-8 text-center">
          No active sessions
        </div>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(288px,1fr))] gap-4">
          {sorted.map((s) => (
            <GameCard
              key={s.session_id}
              session={s}
              serverUrl={serverUrl}
              elapsedSeconds={Math.floor(
                (now - (startTimes[s.session_id] ?? now)) / 1000,
              )}
            />
          ))}
        </div>
      )}
    </div>
  );
}
