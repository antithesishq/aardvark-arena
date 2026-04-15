"use client";

import { useState } from "react";
import { cancelSession } from "@/lib/api";
import { GameState } from "@/lib/ws";
import { SessionState } from "./GameServerSection";
import { GameBadge } from "./badges";
import { Button } from "@/components/ui/button";
import { TicTacToeBoard } from "./boards/TicTacToeBoard";
import { Connect4Board } from "./boards/Connect4Board";
import { BattleshipBoard } from "./boards/BattleshipBoard";
import { mono, shortId4, fmtSeconds } from "@/lib/utils";

// Status constants from the Go backend
const STATUS_ACTIVE = 0;
const STATUS_P1_WIN = 1;
const STATUS_P2_WIN = 2;
const STATUS_DRAW = 3;
const STATUS_CANCELLED = 4;

function turnCount(state: GameState | null, game: string): number {
  if (!state) return 0;
  const shared = state.Shared as Record<string, unknown>;
  if (game === "tictactoe" || game === "connect4") {
    const cells = shared?.Cells as (number | null)[][] | undefined;
    if (!cells) return 0;
    return cells.flat().filter((c) => c !== null).length;
  }
  if (game === "battleship") {
    const attacks = shared?.Attacks as { P1?: Record<string, number>; P2?: Record<string, number> } | undefined;
    if (!attacks) return 0;
    return Object.keys(attacks.P1 ?? {}).length + Object.keys(attacks.P2 ?? {}).length;
  }
  return 0;
}

interface Props {
  session: SessionState;
  serverUrl: string;
  elapsedSeconds: number;
}

export function GameCard({ session, serverUrl, elapsedSeconds }: Props) {
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);

  const gameState = session.gameState;
  const status = gameState?.Status ?? STATUS_ACTIVE;
  const isFinished = status !== STATUS_ACTIVE;

  async function handleCancel() {
    setCancelling(true);
    setCancelError(null);
    try {
      const res = await cancelSession(serverUrl, session.session_id);
      if (!res.ok) setCancelError(`Error ${res.status}`);
    } catch {
      setCancelError("Network error");
    } finally {
      setCancelling(false);
    }
  }

  return (
    <div data-testid="game-card" data-session-id={session.session_id} className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm p-3 flex flex-col gap-2 w-full aspect-square min-h-0 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-zinc-400" style={mono}>{shortId4(session.session_id)}</span>
        <GameBadge game={session.game} />
      </div>

      {/* Board / Result */}
      <div className="flex-1 min-h-0 flex items-center justify-center overflow-hidden">
        {isFinished ? (
          <div className="flex flex-col items-center gap-1 py-4">
            <span data-testid="game-result" className="text-lg font-bold text-zinc-100" style={mono}>
              {status === STATUS_P1_WIN ? "Player 1 wins!" :
               status === STATUS_P2_WIN ? "Player 2 wins!" :
               status === STATUS_DRAW ? "Draw!" : "Cancelled"}
            </span>
            <span className="text-xs text-zinc-500" style={mono}>
              {turnCount(gameState, session.game)} turns · {fmtSeconds(elapsedSeconds)}
            </span>
          </div>
        ) : !gameState ? (
          <span className="text-xs text-zinc-400">Waiting for state…</span>
        ) : session.game === "tictactoe" ? (
          <TicTacToeBoard shared={gameState.Shared as Parameters<typeof TicTacToeBoard>[0]["shared"]} />
        ) : session.game === "connect4" ? (
          <Connect4Board shared={gameState.Shared as Parameters<typeof Connect4Board>[0]["shared"]} />
        ) : session.game === "battleship" ? (
          <BattleshipBoard shared={gameState.Shared as Parameters<typeof BattleshipBoard>[0]["shared"]} />
        ) : null}
      </div>

      {/* Footer */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between border-t border-zinc-800 pt-2">
          {isFinished ? (
            <span className="text-xs text-zinc-500" style={mono}>
              Game over
            </span>
          ) : (
            <>
              <span className="text-xs text-zinc-400 tabular-nums" style={mono}>
                ⏱ {fmtSeconds(elapsedSeconds)} · Turn {turnCount(gameState, session.game)}
              </span>
              <Button
                size="sm"
                variant="destructive"
                style={mono}
                onClick={handleCancel}
                disabled={cancelling}
                data-testid="cancel-btn"
              >
                {cancelling ? "Cancelling…" : "Cancel"}
              </Button>
            </>
          )}
        </div>
        {cancelError && (
          <span className="text-xs text-red-400" style={mono}>{cancelError}</span>
        )}
      </div>
    </div>
  );
}
