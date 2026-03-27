"use client";

import { QueuedPlayer } from "@/lib/api";
import { cn, mono, geist, shortId8, fmtSeconds } from "@/lib/utils";
import { Button } from "@/components/ui/button";

function waitColor(s: number) {
  if (s < 60) return "bg-violet-500";
  if (s < 120) return "bg-amber-500";
  return "bg-red-500";
}

interface Props {
  queue: QueuedPlayer[];
}

export function PlayerQueue({ queue }: Props) {
  return (
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm py-2 px-3 h-[250px] overflow-y-auto">
      <div className="mb-3">
        <div className="text-sm font-semibold text-zinc-200" style={geist}>Player Queue</div>
        <div className="text-xs text-zinc-400" style={geist}>Bots awaiting match</div>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-[10px] tracking-widest text-zinc-400 uppercase border-b border-zinc-800" style={mono}>
            <th className="text-left pb-2 font-medium">Bot</th>
            <th className="text-left pb-2 font-medium">ELO</th>
            <th className="text-left pb-2 font-medium">Wait</th>
            <th className="pb-2" />
          </tr>
        </thead>
        <tbody>
          {queue.length === 0 && (
            <tr>
              <td colSpan={4} className="py-4 text-center text-zinc-400 text-xs" style={geist}>
                Queue is empty
              </td>
            </tr>
          )}
          {queue.map((p) => (
            <tr key={p.player_id} className="border-b border-zinc-800/50 last:border-0">
              <td className="py-2.5 text-zinc-200" style={mono}>{shortId8(p.player_id)}</td>
              <td className="py-2.5 text-zinc-400" style={mono}>{p.elo}</td>
              <td className="py-2.5">
                <div className="flex items-center gap-2">
                  <span style={mono} className={cn(
                    "text-xs tabular-nums",
                    p.wait_seconds > 120 ? "text-red-400" : p.wait_seconds > 60 ? "text-amber-400" : "text-zinc-300"
                  )}>
                    {fmtSeconds(p.wait_seconds)}
                  </span>
                  <div className="w-24 h-1 bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className={cn("h-full rounded-full", waitColor(p.wait_seconds))}
                      style={{ width: `${Math.min(100, (p.wait_seconds / 180) * 100)}%` }}
                    />
                  </div>
                </div>
              </td>
              <td className="py-2.5 text-right">
                <Button size="sm" variant="outline" style={mono}>Match</Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
