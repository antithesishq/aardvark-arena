"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { QueuedPlayer } from "@/lib/api";
import { geist } from "@/lib/utils";

const PAD = 20;
const SIZE = 300 + PAD * 2;
const CX = SIZE / 2;
const CY = SIZE / 2;
const R = 147;
const POLL_INTERVAL_MS = 3000;

function dotRadius(n: number) {
  if (n <= 16) return 5;
  return Math.max(1.875, 5 - (n - 16) * 0.15);
}

function dotFill(waitSeconds: number) {
  if (waitSeconds < 60) return "#8b5cf6";
  if (waitSeconds < 120) return "#f59e0b";
  return "#ef4444";
}

/** Find the midpoint of the largest angular gap among existing points. */
function findBestAngle(occupied: number[]): number {
  if (occupied.length === 0) return -Math.PI / 2; // top of circle
  const sorted = [...occupied].sort((a, b) => a - b);
  let bestGap = 0;
  let bestMid = -Math.PI / 2;
  for (let i = 0; i < sorted.length; i++) {
    const curr = sorted[i];
    const next =
      i === sorted.length - 1 ? sorted[0] + 2 * Math.PI : sorted[i + 1];
    const gap = next - curr;
    if (gap > bestGap) {
      bestGap = gap;
      bestMid = curr + gap / 2;
    }
  }
  if (bestMid > Math.PI) bestMid -= 2 * Math.PI;
  return bestMid;
}

interface DisplayPlayer {
  player: QueuedPlayer;
  angle: number;
}

interface Props {
  queue: QueuedPlayer[];
  avgWait?: string;
}

export function PlayerQueueRing({ queue, avgWait }: Props) {
  // All mutable state lives in a ref so tick callbacks never see stale data.
  const ring = useRef({
    displayed: new Map<string, DisplayPlayer>(),
    pendingAdds: [] as QueuedPlayer[],
    pendingRemoves: [] as string[],
    tickTimer: null as ReturnType<typeof setTimeout> | null,
    tickDelay: 200,
  });

  const [, setRenderTick] = useState(0);
  const rerender = useCallback(() => setRenderTick((n) => n + 1), []);

  // Process one pending addition or removal per tick.
  const processOne = useCallback(() => {
    const s = ring.current;
    s.tickTimer = null;

    // Removals first, two at a time (one session / pair).
    if (s.pendingRemoves.length > 0) {
      const pair = s.pendingRemoves.splice(0, 2);
      for (const id of pair) {
        s.displayed.delete(id);
      }
      rerender();
    } else if (s.pendingAdds.length > 0) {
      const player = s.pendingAdds.shift()!;
      if (!s.displayed.has(player.player_id)) {
        const angle = findBestAngle([...s.displayed.values()].map(d => d.angle));
        s.displayed.set(player.player_id, { player, angle });
        rerender();
      }
    }

    const remainingTicks =
      Math.ceil(s.pendingRemoves.length / 2) + s.pendingAdds.length;
    if (remainingTicks > 0) {
      s.tickTimer = setTimeout(processOne, s.tickDelay);
    }
  }, [rerender]);

  // Diff incoming queue vs current ring state on every poll.
  useEffect(() => {
    const s = ring.current;
    const incomingIds = new Set(queue.map((p) => p.player_id));
    const incomingMap = new Map(queue.map((p) => [p.player_id, p]));

    // Everything we're already tracking (displayed + queued to add).
    const trackedIds = new Set([
      ...s.displayed.keys(),
      ...s.pendingAdds.map((p) => p.player_id),
    ]);
    const pendingRemoveSet = new Set(s.pendingRemoves);

    // --- New players ---------------------------------------------------------
    const newPlayers: QueuedPlayer[] = [];
    for (const p of queue) {
      if (!trackedIds.has(p.player_id) && !pendingRemoveSet.has(p.player_id)) {
        newPlayers.push(p);
      }
      // Cancel pending removal if a player reappeared.
      if (pendingRemoveSet.has(p.player_id)) {
        s.pendingRemoves = s.pendingRemoves.filter((id) => id !== p.player_id);
      }
    }

    // --- Departed players ----------------------------------------------------
    const departed: string[] = [];
    for (const id of s.displayed.keys()) {
      if (!incomingIds.has(id) && !pendingRemoveSet.has(id)) {
        departed.push(id);
      }
    }

    // Drop pending adds for players that left the queue before being shown.
    s.pendingAdds = s.pendingAdds.filter((p) => incomingIds.has(p.player_id));

    if (newPlayers.length > 0) s.pendingAdds.push(...newPlayers);
    if (departed.length > 0) s.pendingRemoves.push(...departed);

    // Update wait_seconds (and therefore color) for visible players.
    let changed = false;
    for (const [id, dp] of s.displayed) {
      const fresh = incomingMap.get(id);
      if (fresh && fresh.wait_seconds !== dp.player.wait_seconds) {
        s.displayed.set(id, { ...dp, player: fresh });
        changed = true;
      }
    }
    if (changed) rerender();

    // Kick off the ticker if there's pending work.
    const totalTicks =
      Math.ceil(s.pendingRemoves.length / 2) + s.pendingAdds.length;
    if (totalTicks > 0) {
      s.tickDelay = Math.min(POLL_INTERVAL_MS / totalTicks, 500);
      if (!s.tickTimer) {
        s.tickTimer = setTimeout(processOne, s.tickDelay);
      }
    }
  }, [queue, processOne, rerender]);

  useEffect(() => {
    return () => {
      if (ring.current.tickTimer) clearTimeout(ring.current.tickTimer);
    };
  }, []);

  const players = [...ring.current.displayed.values()];
  const n = players.length;
  const dr = dotRadius(n);

  return (
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm py-2 px-3 flex flex-col">
      <div className="mb-1 flex items-baseline justify-between">
        <div>
          <div className="text-lg font-semibold text-zinc-200" style={geist}>
            Player Queue
          </div>
          <div className="text-xs text-zinc-400" style={geist}>
            Players awaiting match
          </div>
        </div>
        {avgWait && (
          <div className="text-xs text-zinc-400" style={{ ...geist, fontVariantNumeric: "tabular-nums" }}>
            avg wait <span className="text-zinc-200 font-medium">{avgWait}</span>
          </div>
        )}
      </div>

      <div className="relative flex flex-1 items-center justify-center px-4">
        <svg viewBox={`0 0 ${SIZE} ${SIZE}`} className="w-full max-w-[400px]">
          {/* ring track */}
          <circle
            cx={CX}
            cy={CY}
            r={R}
            fill="none"
            stroke="#27272a"
            strokeWidth={4}
          />

          {/* player dots */}
          <AnimatePresence>
            {players.map(({ player, angle }) => {
              const x = CX + R * Math.cos(angle);
              const y = CY + R * Math.sin(angle);

              return (
                <motion.circle
                  key={player.player_id}
                  initial={{ cx: x, cy: y, r: 0, opacity: 0 }}
                  animate={{
                    cx: x,
                    cy: y,
                    r: dr,
                    opacity: 1,
                    fill: dotFill(player.wait_seconds),
                  }}
                  exit={{
                    cx: CX,
                    cy: CY,
                    r: 0,
                    opacity: 0,
                    transition: {
                      cx: { duration: 0.4, ease: "easeIn" },
                      cy: { duration: 0.4, ease: "easeIn" },
                      r: { duration: 0.2, delay: 0.35, ease: "easeIn" },
                      opacity: { duration: 0.2, delay: 0.35 },
                    },
                  }}
                  transition={{
                    r: { type: "spring", stiffness: 400, damping: 8 },
                    opacity: { duration: 0.2 },
                    fill: { duration: 1 },
                  }}
                />
              );
            })}
          </AnimatePresence>
        </svg>

        {/* center disk with queue stats */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none z-10">
          <div className="flex flex-col items-center justify-center w-24 h-24 rounded-full bg-zinc-900 border border-zinc-700">
            <span className="text-2xl font-semibold text-zinc-200" style={geist}>
              {n}
            </span>
            <span className="text-xs text-zinc-500" style={geist}>
              {n === 1 ? "player" : "players"}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
