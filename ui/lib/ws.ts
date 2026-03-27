export interface PlayerMsg {
  Player: number;
  State: unknown;
  Error: string;
}

// watchSession opens a spectator WebSocket and calls onState for each message.
// Returns a cleanup function to close the connection.
export function watchSession(
  serverUrl: string,
  sessionId: string,
  onState: (msg: PlayerMsg) => void
): () => void {
  const wsUrl = serverUrl.replace(/^http/, "ws") + `/session/${sessionId}/watch`;
  const ws = new WebSocket(wsUrl);
  ws.onmessage = (e) => {
    try {
      onState(JSON.parse(e.data));
    } catch {
      // ignore parse errors
    }
  };
  return () => ws.close();
}
