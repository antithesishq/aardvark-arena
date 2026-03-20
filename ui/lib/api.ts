const MM = process.env.NEXT_PUBLIC_MATCHMAKER_URL ?? "http://localhost:8080";

export interface QueuedPlayer {
  player_id: string;
  elo: number;
  wait_seconds: number;
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
}

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url, { cache: "no-store" });
  if (!res.ok) throw new Error(`${url} → ${res.status}`);
  return res.json();
}

export const fetchStatus = () => get<StatusResponse>(`${MM}/status`);
export const fetchLeaderboard = () => get<LeaderboardEntry[]>(`${MM}/leaderboard`);
export const fetchSessions = (serverUrl: string) =>
  get<SessionSummary[]>(`${serverUrl}/sessions`);
export const fetchHealth = (serverUrl: string) =>
  get<HealthResponse>(`${serverUrl}/health`);
