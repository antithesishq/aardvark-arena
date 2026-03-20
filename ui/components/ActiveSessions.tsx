"use client";

import { ActiveSession } from "@/lib/api";
import { GameBadgeShort } from "./badges";

const mono = { fontFamily: "var(--font-geist-mono)" };
const geist = { fontFamily: "var(--font-geist)" };

function shortId(id: string) {
  return "#" + id.slice(0, 4);
}

function fmtElapsed(createdAt: string) {
  const s = Math.floor((Date.now() - new Date(createdAt).getTime()) / 1000);
  const m = Math.floor(s / 60);
  const sec = s % 60;
  return `${m}:${String(sec).padStart(2, "0")}`;
}

function serverLabel(serverUrl: string) {
  try {
    const u = new URL(serverUrl);
    return u.hostname.toUpperCase();
  } catch {
    return serverUrl;
  }
}

interface Props {
  sessions: ActiveSession[];
}

export function ActiveSessions({ sessions }: Props) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded py-2 px-3">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="text-sm font-semibold text-zinc-200" style={geist}>Active Sessions</div>
          <div className="text-xs text-zinc-500" style={geist}>In-progress game sessions</div>
        </div>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-[10px] tracking-widest text-zinc-500 uppercase border-b border-zinc-800" style={mono}>
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
              <td colSpan={6} className="py-4 text-center text-zinc-600 text-xs" style={geist}>
                No active sessions
              </td>
            </tr>
          )}
          {sessions.map((s) => (
            <tr key={s.session_id} className="border-b border-zinc-800/50 last:border-0">
              <td className="py-2.5" style={mono}><span className="text-zinc-400">{shortId(s.session_id)}</span></td>
              <td className="py-2.5 text-zinc-300" style={geist}>
                {s.player_ids.map((id) => id.slice(0, 8)).join(" vs ")}
              </td>
              <td className="py-2.5"><GameBadgeShort game={s.game} /></td>
              <td className="py-2.5 text-zinc-400 text-xs" style={mono}>{serverLabel(s.server)}</td>
              <td className="py-2.5 text-xs text-zinc-300 tabular-nums" style={mono}>{fmtElapsed(s.created_at)}</td>
              <td className="py-2.5 text-right">
                <span style={mono} className="px-2 py-0.5 text-[10px] font-bold text-red-400 border border-red-800 rounded cursor-default">
                  × CANCEL
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
