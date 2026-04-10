import {
  extract,
  always,
  eventually,
  now,
  actions,
} from "@antithesishq/bombadil";

// Re-export defaults: standard properties (uncaught exceptions, console
// errors, etc.) and actions (clicks on semantic elements, navigation).
export * from "@antithesishq/bombadil/defaults";

// ============================================================
// Extractors
// ============================================================

// --- Matchmaker page: Active Sessions table ---

// Full session IDs from the active sessions table rows.
const activeSessionIds = extract((state) => {
  const rows = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='session-row']",
  );
  return Array.from(rows)
    .map((r) => r.dataset.sessionId ?? "")
    .filter(Boolean);
});

// Session ID currently being cancelled (Cancel button says "Cancelling…").
const cancellingSessionId = extract((state) => {
  const rows = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='session-row']",
  );
  for (const row of Array.from(rows)) {
    const btn = row.querySelector("[data-testid='cancel-btn']");
    if (btn?.textContent?.includes("Cancelling")) {
      return row.dataset.sessionId ?? null;
    }
  }
  return null;
});

// Centre-points of clickable Cancel buttons (not already in-flight).
// Stored as [x, y] pairs to satisfy the JSON type constraint.
const cancelButtonPoints = extract((state) => {
  const buttons = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='cancel-btn']",
  );
  const pts: number[][] = [];
  for (const btn of Array.from(buttons)) {
    if (btn.textContent?.trim() !== "Cancel") continue;
    const r = btn.getBoundingClientRect();
    if (r.width > 0 && r.height > 0)
      pts.push([r.left + r.width / 2, r.top + r.height / 2]);
  }
  return pts;
});

// --- Game Server page: Game Cards ---

// All game card session IDs currently in the DOM.
const gameCardIds = extract((state) => {
  const cards = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='game-card']",
  );
  return Array.from(cards)
    .map((c) => c.dataset.sessionId ?? "")
    .filter(Boolean);
});

// Finished game cards whose result text contains a winner
// (i.e. "Player 1 wins!" or "Player 2 wins!").
const finishedWinnerCards = extract((state) => {
  const cards = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='game-card']",
  );
  const out: { [k: string]: string }[] = [];
  for (const card of Array.from(cards)) {
    const el = card.querySelector("[data-testid='game-result']");
    const text = el?.textContent ?? "";
    if (text.includes("wins")) {
      out.push({ id: card.dataset.sessionId ?? "", result: text });
    }
  }
  return out;
});

// --- Navigation link centre-points (stored as [x, y] | null) ---

const matchmakerNavPoint = extract((state) => {
  for (const a of Array.from(
    state.document.querySelectorAll<HTMLElement>("nav a"),
  )) {
    if (a.textContent?.includes("Matchmaker")) {
      const r = a.getBoundingClientRect();
      if (r.width > 0 && r.height > 0)
        return [r.left + r.width / 2, r.top + r.height / 2];
    }
  }
  return null;
});

const gameServerNavPoint = extract((state) => {
  for (const a of Array.from(
    state.document.querySelectorAll<HTMLElement>("nav a"),
  )) {
    if (a.textContent?.includes("Game Servers")) {
      const r = a.getBoundingClientRect();
      if (r.width > 0 && r.height > 0)
        return [r.left + r.width / 2, r.top + r.height / 2];
    }
  }
  return null;
});

// ============================================================
// Properties
// ============================================================

// Property 1 – Cancel removes session from the active sessions list.
//
// When we observe a Cancel button in its "Cancelling…" state the
// corresponding session ID must eventually disappear from the table.
// The matchmaker page polls every 3 s, so 10 s is generous.
export const cancelRemovesSession = always(() => {
  const id = cancellingSessionId.current;
  if (!id) return true;
  return eventually(() => !activeSessionIds.current.includes(id)).within(
    10,
    "seconds",
  );
});

// Property 2 – Finished games show the winner then disappear.
//
// On the game server page, when a card displays "Player N wins!" it
// should linger visibly for a moment and then be removed from the DOM.
// The UI lingers for 2 s; 10 s gives plenty of headroom.
export const winnerShownThenDisappears = always(() => {
  const winners = finishedWinnerCards.current;
  if (winners.length === 0) return true;
  const card = winners[0];
  const id = card.id;
  const result = card.result;
  if (!id) return true;
  // The winner text is visible right now (the extractor proved it);
  // assert the card will eventually be cleaned up.
  return now(() => result.includes("wins")).and(
    eventually(() => !gameCardIds.current.includes(id)).within(10, "seconds"),
  );
});
