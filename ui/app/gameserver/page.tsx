"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchServers, drainServer, activateServer, cancelAllSessions, type ServerInfo } from "@/lib/api";
import { GameServerSection, ServerHealth } from "@/components/GameServerSection";
import { StatusBadge } from "@/components/badges";
import { Button } from "@/components/ui/button";
import { cn, mono } from "@/lib/utils";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

function serverLabel(server: ServerInfo): string {
  return server.url;
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      id="server-toggle"
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring/30",
        checked ? "bg-emerald-600" : "bg-zinc-700",
      )}
    >
      <span
        className={cn(
          "pointer-events-none block size-4 rounded-full bg-white shadow-lg transition-transform",
          checked ? "translate-x-[18px]" : "translate-x-0.5",
        )}
      />
    </button>
  );
}

export default function GameServerPage() {
  const [servers, setServers] = useState<ServerInfo[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [healthMap, setHealthMap] = useState<Record<string, ServerHealth>>({});

  useEffect(() => {
    fetchServers()
      .then((s) => {
        setServers(s);
        if (s.length > 0) setSelected(s[0].url);
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
        existing.max === health.max &&
        existing.degraded === health.degraded &&
        existing.enabled === health.enabled
      ) {
        return prev;
      }
      return { ...prev, [url]: health };
    });
  }, []);

  const h = selected ? healthMap[selected] : undefined;
  const draining = h ? h.connected && !h.enabled && h.active > 0 : false;

  return (
    <div className="max-w-7xl mx-auto px-6">
      {error && (
        <div className="text-xs text-red-400 py-2 mb-2" style={mono}>
          Cannot fetch server list from matchmaker: {error}
        </div>
      )}

      {/* Server selector */}
      {servers.length > 0 && (
        <div className="flex items-center gap-3 mb-4">
          <Select value={selected} onValueChange={setSelected}>
            <SelectTrigger data-testid="server-select-trigger" style={mono}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {servers.map((s) => (
                <SelectItem key={s.url} value={s.url} data-testid="server-select-item" style={mono}>
                  {serverLabel(s)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Health badge — always shows status */}
          {(() => {
            if (!h) return null;
            if (!h.connected) return <StatusBadge status="disconnected" label="OFFLINE" />;
            return <span data-testid="server-health" data-active={h.active}><StatusBadge status={h.degraded ? "full" : "connected"} label={`${h.active}/${h.max}`} /></span>;
          })()}

          {/* Enabled toggle */}
          {h?.connected && (
            <Toggle
              checked={h.enabled}
              onChange={(checked) => {
                if (!selected) return;
                if (checked) {
                  activateServer(selected);
                } else {
                  drainServer(selected);
                }
              }}
            />
          )}

          {/* Server state badge */}
          {h?.connected && (
            draining
              ? <StatusBadge status="draining" label="DRAINING" />
              : h.enabled
                ? <StatusBadge status="connected" label="ENABLED" />
                : <StatusBadge status="disconnected" label="DISABLED" />
          )}

          {/* Force cancel — only when draining */}
          {draining && (
            <Button size="sm" variant="destructive" style={mono}
              id="force-btn"
              onClick={() => selected && cancelAllSessions(selected)}>
              Force
            </Button>
          )}

          <div className="flex-1" />

          {(() => {
            const healths = Object.values(healthMap);
            const totalActive = healths.reduce((s, h) => s + (h.connected ? h.active : 0), 0);
            const totalMax = healths.reduce((s, h) => s + (h.connected ? h.max : 0), 0);
            if (healths.length === 0) return null;
            return (
              <span className="text-xs text-zinc-400 flex items-center gap-3" style={mono}>
                <span>
                  <span className="text-zinc-500">SERVERS</span>{" "}
                  <span className="tabular-nums">{servers.length}</span>
                </span>
                <span>
                  <span className="text-zinc-500">SESSIONS</span>{" "}
                  <span className="tabular-nums">{totalActive}/{totalMax}</span>
                </span>
              </span>
            );
          })()}
        </div>
      )}

      {/* All server sections — hidden ones still maintain WebSocket connections */}
      {servers.map((s) => (
        <GameServerSection
          key={s.url}
          serverUrl={s.url}
          label={serverLabel(s)}
          hidden={s.url !== selected}
          onHealthChange={(h) => handleHealthChange(s.url, h)}
        />
      ))}
    </div>
  );
}
