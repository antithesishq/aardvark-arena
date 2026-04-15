import {
  extract,
  always,
  eventually,
  actions,
  weighted,
  type Action,
  type ActionGenerator,
} from "@antithesishq/bombadil";
// Re-export default properties (uncaught exceptions, console errors, etc.)
// but NOT default actions — we define our own below.
export {
  noHttpErrorCodes,
  noUncaughtExceptions,
  noUnhandledPromiseRejections,
  noConsoleErrors,
} from "@antithesishq/bombadil/defaults";

// ============================================================
// Extractors
// ============================================================

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

// Centre-points of Cancel buttons on active (not finished) game cards.
// Stored as [x, y] pairs to satisfy the JSON type constraint.
const cancelButtonPoints = extract((state) => {
  const buttons = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='cancel-btn']",
  );
  const pts: number[][] = [];
  for (const btn of Array.from(buttons)) {
    if (btn.textContent?.trim() !== "Cancel") {
      continue;
    }
    const r = btn.getBoundingClientRect();
    if (r.width > 0 && r.height > 0) {
      pts.push([r.left + r.width / 2, r.top + r.height / 2]);
    }
  }
  return pts;
});

// --- Game Server page: Server controls ---

// Centre-point of the enabled/disabled toggle (role="switch").
const serverTogglePoint = extract((state) => {
  const el = state.document.getElementById("server-toggle");
  if (!el) {
    return null;
  }
  const r = el.getBoundingClientRect();
  if (r.width > 0 && r.height > 0) {
    return [r.left + r.width / 2, r.top + r.height / 2];
  }
  return null;
});

// Whether the selected server's toggle is enabled (aria-checked="true").
const serverIsEnabled = extract((state) => {
  const el = state.document.getElementById("server-toggle");
  if (!el) {
    return null;
  }
  return el.getAttribute("aria-checked") === "true";
});

// Active session count for the selected server, from the health badge data attribute.
const activeSessionCount = extract((state) => {
  const el = state.document.querySelector<HTMLElement>(
    "[data-testid='server-health']",
  );
  if (!el) {
    return null;
  }
  const v = el.dataset.active;
  return v != null ? parseInt(v, 10) : null;
});

// Centre-point of the Force button (visible only when a server is draining).
const forceButtonPoint = extract((state) => {
  const el = state.document.getElementById("force-btn");
  if (!el) {
    return null;
  }
  const r = el.getBoundingClientRect();
  if (r.width > 0 && r.height > 0) {
    return [r.left + r.width / 2, r.top + r.height / 2];
  }
  return null;
});

// --- Game Server page: Server select dropdown ---

// The text of the currently selected server (from the select trigger).
const selectedServerText = extract((state) => {
  const trigger = state.document.querySelector<HTMLElement>(
    "[data-testid='server-select-trigger']",
  );
  if (!trigger) {
    return null;
  }
  return trigger.textContent?.trim() || null;
});

// Centre-point of the server select trigger button.
const serverSelectTriggerPoint = extract((state) => {
  const el = state.document.querySelector<HTMLElement>(
    "[data-testid='server-select-trigger']",
  );
  if (!el) {
    return null;
  }
  const r = el.getBoundingClientRect();
  if (r.width > 0 && r.height > 0) {
    return [r.left + r.width / 2, r.top + r.height / 2];
  }
  return null;
});

// Centre-points of visible server select items (only present when dropdown is open).
const serverSelectItemPoints = extract((state) => {
  const items = state.document.querySelectorAll<HTMLElement>(
    "[data-testid='server-select-item']",
  );
  const pts: number[][] = [];
  for (const item of Array.from(items)) {
    const r = item.getBoundingClientRect();
    if (r.width > 0 && r.height > 0) {
      pts.push([r.left + r.width / 2, r.top + r.height / 2]);
    }
  }
  return pts;
});

// --- Navigation link centre-points (stored as [x, y] | null) ---

const matchmakerNavPoint = extract((state) => {
  for (const a of Array.from(
    state.document.querySelectorAll<HTMLElement>("nav a"),
  )) {
    if (a.textContent?.includes("Matchmaker")) {
      const r = a.getBoundingClientRect();
      if (r.width > 0 && r.height > 0) {
        return [r.left + r.width / 2, r.top + r.height / 2];
      }
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
      if (r.width > 0 && r.height > 0) {
        return [r.left + r.width / 2, r.top + r.height / 2];
      }
    }
  }
  return null;
});

// True until the page shell has rendered (nav visible). We don't wait for
// data — the backend may be unreachable in some environments, and we still
// want to exercise navigation and check the no-error-code properties.
const isLoading = extract((state) => {
  return state.document.querySelector("nav") === null;
});

const canGoBack = extract((state) => {
  return state.navigationHistory.back.length > 0;
});

// ============================================================
// Properties
// ============================================================

// Property 1 – Finished games show the winner then disappear.
//
// On the game server page, when a card displays "Player N wins!" it
// should linger visibly for a moment and then be removed from the DOM.
// The UI lingers for 2 s; 10 s gives plenty of headroom.
export const winnerShownThenDisappears = always(() => {
  const winners = finishedWinnerCards.current;
  if (winners.length === 0) {
    return true;
  }
  const card = winners[0];
  const id = card.id;
  if (!id) {
    return true;
  }
  // The extractor already confirmed the winner text is visible; assert cleanup.
  return eventually(() => !gameCardIds.current.includes(id)).within(
    5,
    "seconds",
  );
});

// Property 2 – Bombadil explores more than one game server.
//
// Eventually we observe a selected server, and eventually after that
// a different server is selected. Fails if the workload never switches
// away from the initially selected server.
export const exploresMultipleServers = eventually(() => {
  const server = selectedServerText.current;
  if (!server) {
    return false;
  }
  return eventually(() => {
    const other = selectedServerText.current;
    return !!other && other !== server;
  });
});

// Property 3 – Enabled servers recover and receive sessions.
//
// Whenever the selected server is enabled but has 0 active sessions,
// it should eventually either gain sessions or be disabled again.
// Re-disabling is an acceptable exit (bombadil may toggle it off).
export const serverRecoversAfterEnable = always(() => {
  if (serverIsEnabled.current !== true || activeSessionCount.current !== 0) {
    return true;
  }
  return eventually(
    () => (activeSessionCount.current ?? 0) > 0 || !serverIsEnabled.current,
  ).within(30, "seconds");
});

// ============================================================
// Actions
// ============================================================

// Custom action generator: short-circuit to Wait when the page is still
// loading, otherwise use a weighted mix that lets data settle before acting.
export const explore = actions(() => {
  // Nothing on screen yet — wait for the first poll/render to complete.
  if (isLoading.current) {
    return ["Wait" as Action];
  }

  const center = { x: 512, y: 384 };
  const weightedActions: [number, Action | ActionGenerator][] = [
    [6, "Wait"], // frequently pause so data has time to update
    [2, "Reload"], // occasionally reload to exercise the polling path
    [1, { ScrollDown: { origin: center, distance: 200 } }],
    [1, { ScrollUp: { origin: center, distance: 200 } }],
  ];

  // Cancel active game cards.
  const cancelActions: Action[] = [];
  for (const [x, y] of cancelButtonPoints.current) {
    cancelActions.push({ Click: { name: "cancel-game", point: { x, y } } });
  }
  if (cancelActions.length > 0) {
    weightedActions.push([2, actions(() => cancelActions)]);
  }

  // Toggle server enabled/disabled.
  const togglePt = serverTogglePoint.current;
  if (togglePt) {
    weightedActions.push([
      2,
      {
        Click: {
          name: "server-toggle",
          point: { x: togglePt[0], y: togglePt[1] },
        },
      },
    ]);
  }

  // Force-cancel all sessions on a draining server.
  const forcePt = forceButtonPoint.current;
  if (forcePt) {
    weightedActions.push([
      3,
      {
        Click: {
          name: "force-cancel",
          point: { x: forcePt[0], y: forcePt[1] },
        },
      },
    ]);
  }

  // Go back if there is history.
  if (canGoBack.current) {
    weightedActions.push([1, "Back"]);
  }

  // Open the server select dropdown.
  const selectPt = serverSelectTriggerPoint.current;
  if (selectPt) {
    weightedActions.push([
      2,
      {
        Click: {
          name: "server-select-open",
          point: { x: selectPt[0], y: selectPt[1] },
        },
      },
    ]);
  }

  // Pick a server from the open dropdown.
  const selectItems = serverSelectItemPoints.current;
  if (selectItems.length > 0) {
    const itemActions: Action[] = selectItems.map(([x, y]) => ({
      Click: { name: "server-select-item", point: { x, y } },
    }));
    weightedActions.push([6, actions(() => itemActions)]);
  }

  // Navigate between pages via the nav links.
  const mmPt = matchmakerNavPoint.current;
  if (mmPt) {
    weightedActions.push([
      2,
      { Click: { name: "matchmaker-nav", point: { x: mmPt[0], y: mmPt[1] } } },
    ]);
  }

  const gsPt = gameServerNavPoint.current;
  if (gsPt) {
    weightedActions.push([
      2,
      { Click: { name: "gameserver-nav", point: { x: gsPt[0], y: gsPt[1] } } },
    ]);
  }

  return weighted(weightedActions).generate();
});
