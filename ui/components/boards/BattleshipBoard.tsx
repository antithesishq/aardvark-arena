"use client";

// Attacks is a map from player index (0 or 1) to a map of Position -> AttackResult
// Position: { X: col, Y: row }
// AttackResult: 0 = Miss, 1 = Hit
interface Position {
  X: number;
  Y: number;
}

type AttackResult = 0 | 1;
type AttackMap = Record<string, AttackResult>; // key is JSON like "{X:0,Y:0}" but actually stored differently

interface BattleshipShared {
  Attacks: Record<string, Record<string, AttackResult>>;
}

interface Props {
  shared: BattleshipShared;
  viewPlayer?: number; // which player's perspective (0 or 1), default 0
}

// Parse the attack maps - Go marshals Position as {X:n, Y:n}
function parseAttacks(raw: Record<string, AttackResult>): Map<string, AttackResult> {
  const result = new Map<string, AttackResult>();
  for (const [key, val] of Object.entries(raw ?? {})) {
    // key format from Go: {"X":2,"Y":3}
    try {
      const pos = JSON.parse(key) as Position;
      result.set(`${pos.X},${pos.Y}`, val);
    } catch {
      // skip unparseable keys
    }
  }
  return result;
}

export function BattleshipBoard({ shared, viewPlayer = 0 }: Props) {
  // Show attacks ON the opponent's board (what the viewPlayer is shooting at)
  const opponentKey = viewPlayer === 0 ? "1" : "0";
  const attacksOnOpponent = parseAttacks(shared.Attacks?.[opponentKey] ?? {});

  const COLS = 10;
  const ROWS = 10;

  return (
    <div className="flex flex-col gap-0.5">
      <div
        className="grid gap-0.5"
        style={{ gridTemplateColumns: `repeat(${COLS}, 1fr)` }}
      >
        {Array.from({ length: ROWS }, (_, row) =>
          Array.from({ length: COLS }, (_, col) => {
            const key = `${col},${row}`;
            const attack = attacksOnOpponent.get(key);
            return (
              <div
                key={key}
                className="aspect-square rounded-sm"
                style={{
                  backgroundColor:
                    attack === 1
                      ? "#ef4444" // hit = red
                      : attack === 0
                      ? "#374151" // miss = dark gray
                      : "#1f2937", // empty = even darker
                  minWidth: 16,
                  minHeight: 16,
                }}
              />
            );
          })
        )}
      </div>
      <div className="text-[9px] text-zinc-600 uppercase tracking-wider text-center mt-1">
        {viewPlayer === 0 ? "P1" : "P2"} VIEW
      </div>
    </div>
  );
}
