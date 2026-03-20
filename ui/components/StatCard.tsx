import { cn, mono, geist } from "@/lib/utils";

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  valueClass?: string;
}

export function StatCard({ label, value, sub, valueClass }: StatCardProps) {
  return (
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm py-2 px-3 flex flex-col gap-1 min-w-0">
      <span
        className="text-[10px] font-bold tracking-widest text-zinc-400 uppercase"
        style={mono}
      >
        {label}
      </span>
      <span
        className={cn("text-3xl font-bold tabular-nums", valueClass ?? "text-zinc-100")}
        style={mono}
      >
        {value}
      </span>
      {sub && (
        <span className="text-xs text-zinc-400" style={geist}>
          {sub}
        </span>
      )}
    </div>
  );
}
