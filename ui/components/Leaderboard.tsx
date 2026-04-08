"use client";

import { LeaderboardEntry } from "@/lib/api";
import { mono, geist, shortId8 } from "@/lib/utils";

interface Props {
  entries: LeaderboardEntry[];
}

export function Leaderboard({ entries }: Props) {
  const maxElo = Math.max(...entries.map((e) => e.Elo), 1000);

  return (
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm py-2 px-3 h-[250px] overflow-y-auto">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="text-sm font-semibold text-zinc-200" style={geist}>ELO Leaderboard</div>
          <div className="text-xs text-zinc-400" style={geist}>All-time rankings</div>
        </div>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-[10px] tracking-widest text-zinc-400 uppercase border-b border-zinc-800" style={mono}>
            <th className="text-left pb-2 font-medium">Player</th>
            <th className="text-left pb-2 font-medium">ELO</th>
            <th className="text-right pb-2 font-medium">W/L</th>
          </tr>
        </thead>
        <tbody>
          {entries.length === 0 && (
            <tr>
              <td colSpan={3} className="py-4 text-center text-zinc-400 text-xs" style={geist}>
                No players yet
              </td>
            </tr>
          )}
          {entries.map((e) => (
            <tr key={e.PlayerID} className="border-b border-zinc-800/50 last:border-0">
              <td className="py-2.5 text-zinc-200" style={mono}>{shortId8(e.PlayerID)}</td>
              <td className="py-2.5">
                <div className="flex items-center gap-2">
                  <span className="text-zinc-300 tabular-nums w-12" style={mono}>{e.Elo}</span>
                  <div className="w-28 h-1 bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-violet-500 rounded-full"
                      style={{ width: `${(e.Elo / maxElo) * 100}%` }}
                    />
                  </div>
                </div>
              </td>
              <td className="py-2.5 text-right text-zinc-400 tabular-nums" style={mono}>
                {e.Wins}/{e.Losses}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
