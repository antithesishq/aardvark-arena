import { cn } from "@/lib/utils";

const mono = { fontFamily: "var(--font-geist-mono)" };

const gameColors: Record<string, string> = {
  tictactoe: "bg-violet-900 text-violet-200 border-violet-700",
  connect4: "bg-sky-900 text-sky-200 border-sky-700",
  battleship: "bg-amber-900 text-amber-200 border-amber-700",
};

const gameLabels: Record<string, string> = {
  tictactoe: "TICTACTOE",
  connect4: "CONNECT4",
  battleship: "BATTLESHIP",
};

export function GameBadge({ game }: { game: string }) {
  return (
    <span
      style={mono}
      className={cn(
        "px-2 py-0.5 text-[10px] font-bold tracking-widest border rounded",
        gameColors[game] ?? "bg-zinc-800 text-zinc-300 border-zinc-700"
      )}
    >
      {gameLabels[game] ?? game.toUpperCase()}
    </span>
  );
}

const gameShort: Record<string, string> = {
  tictactoe: "TTT",
  connect4: "C4",
  battleship: "BSHIP",
};

export function GameBadgeShort({ game }: { game: string }) {
  return (
    <span
      style={mono}
      className={cn(
        "px-1.5 py-0.5 text-[10px] font-bold tracking-wider border rounded",
        gameColors[game] ?? "bg-zinc-800 text-zinc-300 border-zinc-700"
      )}
    >
      {gameShort[game] ?? game.slice(0, 3).toUpperCase()}
    </span>
  );
}

export function OnlineBadge() {
  return (
    <span style={mono} className="flex items-center gap-1.5 px-2 py-0.5 text-[10px] font-bold tracking-widest bg-emerald-900/60 text-emerald-400 border border-emerald-700 rounded">
      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
      ONLINE
    </span>
  );
}

export function DegradedBadge() {
  return (
    <span style={mono} className="flex items-center gap-1.5 px-2 py-0.5 text-[10px] font-bold tracking-widest bg-amber-900/60 text-amber-400 border border-amber-700 rounded">
      <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
      DEGRADED
    </span>
  );
}

export function RunningBadge({ count }: { count: number }) {
  return (
    <span style={mono} className="px-2 py-0.5 text-[10px] font-bold tracking-widest bg-emerald-900/60 text-emerald-400 border border-emerald-700 rounded">
      {count} RUNNING
    </span>
  );
}
