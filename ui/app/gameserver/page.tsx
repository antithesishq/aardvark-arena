"use client";

import { useEffect, useState } from "react";
import { fetchServers } from "@/lib/api";
import { GameServerSection } from "@/components/GameServerSection";
import { mono } from "@/lib/utils";

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

  useEffect(() => {
    fetchServers()
      .then(setServers)
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <div className="max-w-7xl mx-auto px-6">
      {error && (
        <div className="text-xs text-red-400 py-2 mb-2" style={mono}>
          Cannot fetch server list from matchmaker: {error}
        </div>
      )}
      {servers.map((url, i) => (
        <GameServerSection
          key={url}
          serverUrl={url}
          label={serverLabel(url, i)}
        />
      ))}
    </div>
  );
}
