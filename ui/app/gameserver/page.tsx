"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchServers } from "@/lib/api";
import { GameServerSection, ServerHealth } from "@/components/GameServerSection";
import { cn, mono } from "@/lib/utils";

const statusStyles = {
  connected:    "bg-emerald-900/60 text-emerald-400 border-emerald-700",
  full:         "bg-amber-900/60 text-amber-400 border-amber-700",
  disconnected: "bg-red-900/60 text-red-400 border-red-700",
};

function serverLabel(url: string, index: number): string {
  try {
    const u = new URL(url);
    return u.port ? `${u.hostname.toUpperCase()}:${u.port}` : u.hostname.toUpperCase();
  } catch {
    return `GS-${String(index + 1).padStart(2, "0")}`;
  }
}

export default function GameServerPage() {
  const [servers, setServers] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [healthMap, setHealthMap] = useState<Record<string, ServerHealth>>({});

  useEffect(() => {
    fetchServers()
      .then((s) => {
        setServers(s);
        if (s.length > 0) setSelected(s[0]);
      })
      .catch((e) => setError(String(e)));
  }, []);

  const handleHealthChange = useCallback((url: string, health: ServerHealth) => {
    setHealthMap((prev) => {
      const existing = prev[url];
      if (
        existing &&
        existing.connected === health.connected &&
        existing.active === health.active &&
        existing.max === health.max
      ) {
        return prev;
      }
      return { ...prev, [url]: health };
    });
  }, []);

  return (
    <div className="max-w-7xl mx-auto px-6">
      {error && (
        <div className="text-xs text-red-400 py-2 mb-2" style={mono}>
          Cannot fetch server list from matchmaker: {error}
        </div>
      )}

      {/* Server tabs */}
      {servers.length > 0 && (
        <div className="flex items-stretch gap-0 border-b border-zinc-800 mb-4 overflow-x-auto">
          {servers.map((url, i) => {
            const label = serverLabel(url, i);
            const h = healthMap[url];
            const isActive = url === selected;

            return (
              <button
                key={url}
                onClick={() => setSelected(url)}
                className={cn(
                  "flex items-center gap-2.5 px-4 py-2.5 -mb-px text-xs font-bold tracking-widest border-b-2 transition-colors cursor-pointer whitespace-nowrap",
                  isActive
                    ? "text-zinc-100 border-violet-500"
                    : "text-zinc-400 border-transparent hover:text-zinc-300 hover:border-zinc-600"
                )}
                style={mono}
              >
                <span>{label}</span>
                {h && (
                  <span
                    style={mono}
                    className={cn(
                      "px-2 py-0.5 text-[10px] font-bold tabular-nums border rounded",
                      !h.connected
                        ? statusStyles.disconnected
                        : h.degraded
                          ? statusStyles.full
                          : statusStyles.connected,
                    )}
                  >
                    {String(h.active).padStart(String(h.max).length, "\u2007")}/{h.max}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      )}

      {/* All server sections — hidden ones still maintain WebSocket connections */}
      {servers.map((url, i) => (
        <GameServerSection
          key={url}
          serverUrl={url}
          label={serverLabel(url, i)}
          hidden={url !== selected}
          onHealthChange={(h) => handleHealthChange(url, h)}
        />
      ))}
    </div>
  );
}
