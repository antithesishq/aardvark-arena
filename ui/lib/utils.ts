import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const mono = { fontFamily: "var(--font-geist-mono)" };
export const geist = { fontFamily: "var(--font-geist)" };

export const shortId8 = (id: string) => id.slice(0, 8);
export const shortId4 = (id: string) => "#" + id.slice(0, 4);

export function fmtSeconds(s: number) {
  const m = Math.floor(s / 60);
  return `${m}:${String(Math.floor(s % 60)).padStart(2, "0")}`;
}

export function serverHostname(url: string) {
  try { return new URL(url).hostname.toUpperCase(); }
  catch { return url; }
}
