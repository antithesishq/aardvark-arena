import { cn, mono } from "@/lib/utils";

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
        gameColors[game] ?? "bg-zinc-800 text-zinc-300 border-zinc-700",
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
        gameColors[game] ?? "bg-zinc-800 text-zinc-300 border-zinc-700",
      )}
    >
      {gameShort[game] ?? game.slice(0, 3).toUpperCase()}
    </span>
  );
}

const statusStyles: Record<
  string,
  { bg: string; dot: string; pulse?: boolean }
> = {
  connected: {
    bg: "bg-emerald-900/60 text-emerald-400 border-emerald-700",
    dot: "bg-emerald-400",
    pulse: true,
  },
  full: {
    bg: "bg-amber-900/60 text-amber-400 border-amber-700",
    dot: "bg-amber-400",
  },
  draining: {
    bg: "bg-yellow-900/60 text-yellow-400 border-yellow-700",
    dot: "bg-yellow-400",
    pulse: true,
  },
  disconnected: {
    bg: "bg-red-900/60 text-red-400 border-red-700",
    dot: "bg-red-400",
  },
};

export function StatusBadge({
  status,
  label,
}: {
  status: string;
  label: string;
}) {
  const s = statusStyles[status] ?? statusStyles.disconnected;
  return (
    <span
      style={mono}
      className={cn(
        "flex items-center gap-1.5 px-2 py-0.5 text-[10px] font-bold tracking-widest border rounded",
        s.bg,
      )}
    >
      <span
        className={cn(
          "w-1.5 h-1.5 rounded-full",
          s.dot,
          s.pulse && "animate-pulse",
        )}
      />
      {label}
    </span>
  );
}

export function ConnectedBadge({ degraded }: { degraded?: boolean }) {
  return degraded ? (
    <StatusBadge status="full" label="FULL" />
  ) : (
    <StatusBadge status="connected" label="CONNECTED" />
  );
}

export function DisconnectedBadge() {
  return <StatusBadge status="disconnected" label="DISCONNECTED" />;
}
