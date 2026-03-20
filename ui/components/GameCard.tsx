"use client";

import { useEffect, useState } from "react";
import { watchSession, PlayerMsg } from "@/lib/ws";
import { SessionSummary, cancelSession } from "@/lib/api";
import { GameBadge } from "./badges";
import { Button } from "@/components/ui/button";
import { TicTacToeBoard } from "./boards/TicTacToeBoard";
import { Connect4Board } from "./boards/Connect4Board";
import { BattleshipBoard } from "./boards/BattleshipBoard";

function shortId(id: string) {
  return "#" + id.slice(0, 4);
}

interface GameState {
  CurrentPlayer: number;
  Status: number;
  Shared: unknown;
}

function turnCount(state: GameState | null, game: string): number {
  if (!state) return 0;
  const shared = state.Shared as Record<string, unknown>;
  if (game === "tictactoe") {
    const cells = shared?.Cells as (number | null)[][] | undefined;
    if (!cells) return 0;
    return cells.flat().filter((c) => c !== null).length;
  }
  if (game === "connect4") {
    const cells = shared?.Cells as (number | null)[][] | undefined;
    if (!cells) return 0;
    return cells.flat().filter((c) => c !== null).length;
  }
  return 0;
}

interface Props {
  session: SessionSummary;
  serverUrl: string;
  elapsedSeconds: number;
}

function fmtTime(s: number) {
  const m = Math.floor(s / 60);
  const sec = s % 60;
  return `${m}:${String(sec).padStart(2, "0")}`;
}

export function GameCard({ session, serverUrl, elapsedSeconds }: Props) {
  const [lastMsg, setLastMsg] = useState<PlayerMsg | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);
  const gameState = lastMsg?.State as GameState | null;

  useEffect(() => {
    const close = watchSession(serverUrl, session.session_id, setLastMsg);
    return close;
  }, [serverUrl, session.session_id]);

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
    <div className="bg-zinc-900/20 border border-zinc-800 rounded backdrop-blur-sm p-3 flex flex-col gap-2 min-w-[240px]">
      {/* Header */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-zinc-400" style={{ fontFamily: "var(--font-geist-mono)" }}>{shortId(session.session_id)}</span>
        <GameBadge game={session.game} />
      </div>

      {/* Board */}
      <div className="min-h-[120px] flex items-center justify-center">
        {!gameState ? (
          <span className="text-xs text-zinc-400">Connecting…</span>
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
          <span className="text-xs text-zinc-400 tabular-nums" style={{ fontFamily: "var(--font-geist-mono)" }}>
            ⏱ {fmtTime(elapsedSeconds)} · Turn {turnCount(gameState, session.game)}
          </span>
          <Button
            size="sm"
            variant="destructive"
            style={{ fontFamily: "var(--font-geist-mono)" }}
            onClick={handleCancel}
            disabled={cancelling}
          >
            {cancelling ? "Cancelling…" : "Cancel"}
          </Button>
        </div>
        {cancelError && (
          <span className="text-xs text-red-400" style={{ fontFamily: "var(--font-geist-mono)" }}>{cancelError}</span>
        )}
      </div>
    </div>
  );
}
