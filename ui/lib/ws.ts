export interface GameState {
  CurrentPlayer: number;
  Status: number;
  Shared: unknown;
}

export type WatchEvent =
  | { type: "health"; active_sessions: number; max_sessions: number; active: boolean }
  | {
      type: "session";
      session_id: string;
      game: "tictactoe" | "connect4" | "battleship";
      players: Record<string, number>;
      state: GameState;
      deadline: string;
    }
  | { type: "session_end"; session_id: string };

/**
 * Opens a server-level watch WebSocket that receives multiplexed events for
 * all sessions on the given game server. Automatically reconnects on close.
 * Returns a cleanup function.
 */
export function watchServer(
  serverUrl: string,
  onEvent: (evt: WatchEvent) => void,
  onConnChange: (connected: boolean) => void,
): () => void {
  let ws: WebSocket | null = null;
  let stopped = false;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  function connect() {
    if (stopped) {return;}
    const wsUrl = serverUrl.replace(/^http/, "ws") + "/watch";
    ws = new WebSocket(wsUrl);

    ws.onopen = () => onConnChange(true);

    ws.onmessage = (e) => {
      try {
        onEvent(JSON.parse(e.data));
      } catch {
        // ignore parse errors
      }
    };

    ws.onclose = () => {
      onConnChange(false);
      if (!stopped) {
        reconnectTimer = setTimeout(connect, 2000);
      }
    };

    ws.onerror = () => {
      // onclose will fire after onerror
    };
  }

  connect();

  return () => {
    stopped = true;
    if (reconnectTimer) {clearTimeout(reconnectTimer);}
    ws?.close();
  };
}
