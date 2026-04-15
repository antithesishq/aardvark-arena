"use client";

// Attacks is { P1: { "x,y": 0|1 }, P2: { "x,y": 0|1 } }
// Keys are "X,Y" strings, values are 0 = Miss, 1 = Hit
type AttackResult = 0 | 1;

interface BattleshipShared {
  Attacks: {
    P1: Record<string, AttackResult>;
    P2: Record<string, AttackResult>;
  };
}

interface Props {
  shared: BattleshipShared;
}

const COLS = 10;
const ROWS = 10;

function MiniGrid({
  attacks,
  label,
}: {
  attacks: Record<string, AttackResult>;
  label: string;
}) {
  return (
    <div className="flex flex-col items-center gap-1 flex-1">
      <div
        className="grid gap-px w-full"
        style={{ gridTemplateColumns: `repeat(${COLS}, 1fr)` }}
      >
        {Array.from({ length: ROWS }, (_, row) =>
          Array.from({ length: COLS }, (_, col) => {
            const key = `${col},${row}`;
            const attack = attacks[key];
            return (
              <div
                key={key}
                className="aspect-square rounded-[2px]"
                style={{
                  backgroundColor:
                    attack === 1
                      ? "#ef4444" // hit = red
                      : attack === 0
                        ? "#6b7280" // miss = gray
                        : "#1e293b", // empty = slate
                }}
              />
            );
          }),
        )}
      </div>
      <span className="text-[8px] text-zinc-500 uppercase tracking-widest">
        {label}
      </span>
    </div>
  );
}

export function BattleshipBoard({ shared }: Props) {
  const attacks = shared.Attacks ?? {};
  return (
    <div className="flex gap-3 items-start w-full max-h-full">
      <MiniGrid attacks={attacks.P1 ?? {}} label="P1" />
      <MiniGrid attacks={attacks.P2 ?? {}} label="P2" />
    </div>
  );
}
