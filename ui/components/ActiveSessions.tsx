"use client";

import { useState } from "react";
import { ActiveSession, cancelSessionViaMatchmaker } from "@/lib/api";
import { GameBadgeShort } from "./badges";
import { Button } from "@/components/ui/button";
import { mono, geist, shortId4, fmtSeconds, serverHostname } from "@/lib/utils";

interface Props {
  sessions: ActiveSession[];
}

function CancelButton({ session }: { session: ActiveSession }) {
  const [cancelling, setCancelling] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleCancel() {
    setCancelling(true);
    setError(null);
    try {
      const res = await cancelSessionViaMatchmaker(session.session_id);
      if (!res.ok) setError(`${res.status}`);
    } catch {
      setError("Network error");
    } finally {
      setCancelling(false);
    }
  }

  return (
    <div className="flex flex-col items-end gap-0.5">
      <Button size="sm" variant="destructive" style={mono} onClick={handleCancel} disabled={cancelling}>
        {cancelling ? "Cancelling…" : "Cancel"}
      </Button>
      {error && <span className="text-[10px] text-red-400" style={mono}>{error}</span>}
    </div>
  );
}

export function ActiveSessions({ sessions }: Props) {
  return (
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm py-2 px-3 h-[350px] overflow-y-auto">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="text-sm font-semibold text-zinc-200" style={geist}>Active Sessions</div>
          <div className="text-xs text-zinc-400" style={geist}>In-progress game sessions</div>
        </div>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-[10px] tracking-widest text-zinc-400 uppercase border-b border-zinc-800" style={mono}>
            <th className="text-left pb-2 font-medium">Session</th>
            <th className="text-left pb-2 font-medium">Players</th>
            <th className="text-left pb-2 font-medium">Game</th>
            <th className="text-left pb-2 font-medium">Server</th>
            <th className="text-left pb-2 font-medium">Elapsed</th>
            <th className="pb-2" />
          </tr>
        </thead>
        <tbody>
          {sessions.length === 0 && (
            <tr>
              <td colSpan={6} className="py-4 text-center text-zinc-400 text-xs" style={geist}>
                No active sessions
              </td>
            </tr>
          )}
          {sessions.map((s) => (
            <tr key={s.session_id} className="border-b border-zinc-800/50 last:border-0">
              <td className="py-2.5" style={mono}><span className="text-zinc-400">{shortId4(s.session_id)}</span></td>
              <td className="py-2.5 text-zinc-300" style={geist}>
                {s.player_ids.map((id) => id.slice(0, 8)).join(" vs ")}
              </td>
              <td className="py-2.5"><GameBadgeShort game={s.game} /></td>
              <td className="py-2.5 text-zinc-400 text-xs" style={mono}>{serverHostname(s.server)}</td>
              <td className="py-2.5 text-xs text-zinc-300 tabular-nums" style={mono}>{fmtSeconds(Math.floor((Date.now() - new Date(s.created_at).getTime()) / 1000))}</td>
              <td className="py-2.5 text-right">
                <CancelButton session={s} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
