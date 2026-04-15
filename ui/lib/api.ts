const MM = process.env.NEXT_PUBLIC_MATCHMAKER_URL ?? "http://localhost:8080";

export interface QueuedPlayer {
  player_id: string;
  elo: number;
  wait_seconds: number;
  game?: "tictactoe" | "connect4" | "battleship";
}

export interface ActiveSession {
  session_id: string;
  server: string;
  game: "tictactoe" | "connect4" | "battleship";
  player_ids: string[];
  created_at: string;
  deadline: string;
}

export interface StatusResponse {
  queue: QueuedPlayer[];
  sessions: ActiveSession[];
}

export interface LeaderboardEntry {
  PlayerID: string;
  Elo: number;
  Wins: number;
  Losses: number;
  Draws: number;
}

export interface SessionSummary {
  session_id: string;
  game: "tictactoe" | "connect4" | "battleship";
}

export interface HealthResponse {
  ActiveSessions: number;
  MaxSessions: number;
  Full: boolean;
  Active: boolean;
}

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`${url} → ${res.status}`);
  }
  return res.json();
}

export interface ServerInfo {
  id: string;
  url: string;
}

export const fetchStatus = () => get<StatusResponse>(`${MM}/status`);
export const fetchLeaderboard = () =>
  get<LeaderboardEntry[]>(`${MM}/leaderboard`);
export const fetchServers = () => get<ServerInfo[]>(`${MM}/servers`);
export const fetchSessions = (serverUrl: string) =>
  get<SessionSummary[]>(`${serverUrl}/sessions`);
export const fetchHealth = (serverUrl: string) =>
  get<HealthResponse>(`${serverUrl}/health`);

// Cancel via the matchmaker — it proxies to the game server and handles the
// case where the session already finished on the game server side.
export const cancelSessionViaMatchmaker = (sessionId: string) =>
  fetch(`${MM}/session/${sessionId}`, { method: "DELETE" });

// Cancel directly on a game server (used from the game server tab).
export const cancelSession = (serverUrl: string, sessionId: string) =>
  fetch(`${serverUrl}/session/${sessionId}`, { method: "DELETE" });

// Drain a game server (stop accepting new sessions).
export const drainServer = (serverUrl: string) =>
  fetch(`${serverUrl}/drain`, { method: "POST" });

// Activate a drained game server (resume accepting new sessions).
export const activateServer = (serverUrl: string) =>
  fetch(`${serverUrl}/activate`, { method: "POST" });

// Cancel all active sessions on a game server.
export const cancelAllSessions = (serverUrl: string) =>
  fetch(`${serverUrl}/sessions`, { method: "DELETE" });
