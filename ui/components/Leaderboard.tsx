"use client";

import { LeaderboardEntry } from "@/lib/api";
import { mono, geist, shortId8 } from "@/lib/utils";

function playerStatus(
  id: string,
  queuedIds?: Set<string>,
  playingIds?: Set<string>,
): { label: string; color: string } {
  if (playingIds?.has(id)) return { label: "playing", color: "text-green-400" };
  if (queuedIds?.has(id)) return { label: "queued", color: "text-violet-400" };
  return { label: "offline", color: "text-zinc-600" };
}

interface Props {
  entries: LeaderboardEntry[];
  queuedIds?: Set<string>;
  playingIds?: Set<string>;
}

export function Leaderboard({ entries, queuedIds, playingIds }: Props) {
  return (
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm py-2 px-3 ">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="text-lg font-semibold text-zinc-200" style={geist}>
            ELO Leaderboard
          </div>
          <div className="text-xs text-zinc-400" style={geist}>
            Top 10 players
          </div>
        </div>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr
            className="text-[10px] tracking-widest text-zinc-400 uppercase border-b border-zinc-800"
            style={mono}
          >
            <th className="text-left pb-2 font-medium">Player</th>
            <th className="text-left pb-2 font-medium">ELO</th>
            <th className="text-left pb-2 font-medium">W/L</th>
            <th className="text-right pb-2 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          {entries.length === 0 && (
            <tr>
              <td
                colSpan={4}
                className="py-4 text-center text-zinc-400 text-xs"
                style={geist}
              >
                No players yet
              </td>
            </tr>
          )}
          {entries.map((e) => {
            const status = playerStatus(e.PlayerID, queuedIds, playingIds);
            return (
              <tr
                key={e.PlayerID}
                className="border-b border-zinc-800/50 last:border-0"
              >
                <td className="py-2.5 text-zinc-200" style={mono}>
                  {shortId8(e.PlayerID)}
                </td>
                <td
                  className="py-2.5 text-zinc-300 tabular-nums"
                  style={mono}
                >
                  {e.Elo}
                </td>
                <td
                  className="py-2.5 text-zinc-400 tabular-nums"
                  style={mono}
                >
                  {e.Wins}/{e.Losses}
                </td>
                <td
                  className={`py-2.5 text-right text-xs ${status.color}`}
                  style={mono}
                >
                  {status.label}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
