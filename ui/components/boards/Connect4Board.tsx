"use client";

// Cells is [7 columns][6 rows], row 0 is bottom. null = empty, 0 = P1, 1 = P2
interface Connect4Shared {
  Cells: (number | null)[][];
}

interface Props {
  shared: Connect4Shared;
}

export function Connect4Board({ shared }: Props) {
  const cells = shared.Cells;

  return (
    <div className="grid gap-1.5 p-2 bg-zinc-800 rounded" style={{ gridTemplateColumns: "repeat(7, 1fr)" }}>
      {/* Render rows top-to-bottom (row 5 is top visually, row 0 is bottom) */}
      {[5, 4, 3, 2, 1, 0].map((row) =>
        [0, 1, 2, 3, 4, 5, 6].map((col) => {
          const val = cells[col]?.[row];
          return (
            <div
              key={`${row}-${col}`}
              className="aspect-square rounded-full bg-zinc-700 flex items-center justify-center"
            >
              {val === 0 && <div className="w-full h-full rounded-full bg-violet-500" />}
              {val === 1 && <div className="w-full h-full rounded-full bg-zinc-500" />}
            </div>
          );
        })
      )}
    </div>
  );
}
