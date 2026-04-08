"use client";

// Cells is a 3x3 grid: null = empty, 0 = P1 (X), 1 = P2 (O)
interface TicTacToeShared {
  Cells: (number | null)[][];
}

interface Props {
  shared: TicTacToeShared;
}

export function TicTacToeBoard({ shared }: Props) {
  const cells = shared.Cells;

  return (
    <div className="inline-grid gap-px bg-zinc-700 rounded overflow-hidden" style={{ gridTemplateColumns: "repeat(3, 3rem)", gridTemplateRows: "repeat(3, 3rem)" }}>
      {[0, 1, 2].map((row) =>
        [0, 1, 2].map((col) => {
          const val = cells[col]?.[row];
          return (
            <div
              key={`${row}-${col}`}
              className="bg-zinc-900 flex items-center justify-center text-2xl font-bold"
            >
              {val === 0 && <span className="text-violet-400">X</span>}
              {val === 1 && <span className="text-zinc-400">O</span>}
            </div>
          );
        })
      )}
    </div>
  );
}
