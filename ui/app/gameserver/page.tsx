"use client";

import { GameServerSection } from "@/components/GameServerSection";

// Game server URLs — configured via env var as a comma-separated list.
// Falls back to a single localhost server for local dev.
const raw = process.env.NEXT_PUBLIC_GAME_SERVER_URLS ?? "http://localhost:8081";
const SERVERS = raw.split(",").map((s) => s.trim()).filter(Boolean);

function serverLabel(url: string, index: number): string {
  try {
    const u = new URL(url);
    return u.hostname.toUpperCase();
  } catch {
    return `GS-${String(index + 1).padStart(2, "0")}`;
  }
}

export default function GameServerPage() {
  return (
    <div className="max-w-7xl mx-auto">
      {SERVERS.map((url, i) => (
        <GameServerSection
          key={url}
          serverUrl={url}
          label={serverLabel(url, i)}
        />
      ))}
    </div>
  );
}
