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
    <div className="inline-grid gap-1 bg-zinc-800 rounded p-2" style={{ gridTemplateColumns: "repeat(7, 1.5rem)", gridTemplateRows: "repeat(6, 1.5rem)" }}>
      {[5, 4, 3, 2, 1, 0].map((row) =>
        [0, 1, 2, 3, 4, 5, 6].map((col) => {
          const val = cells[col]?.[row];
          return (
            <div
              key={`${row}-${col}`}
              className={`rounded-full ${val === 0 ? "bg-violet-500" : val === 1 ? "bg-zinc-500" : "bg-zinc-700"}`}
            />
          );
        })
      )}
    </div>
  );
}
