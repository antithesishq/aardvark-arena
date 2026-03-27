import { extract, always, eventually } from "@antithesishq/bombadil";
export * from "@antithesishq/bombadil/defaults";

// ─── Extractors ────────────────────────────────────────────────────────────────

// Which nav tab is highlighted vs. what the URL path actually is
const navState = extract((state) => {
  const active = state.document.querySelector<HTMLAnchorElement>(
    "nav a.border-violet-500"
  );
  return {
    activeHref: active?.getAttribute("href") ?? null,
    pathname: state.window.location.pathname,
  };
});

// Whether a "Connecting…" placeholder is visible (WebSocket hasn't delivered state yet)
const connecting = extract((state) =>
  state.document.body.innerText.includes("Connecting…")
);

// "Active Sessions" stat card number vs. the actual number of rows in the sessions table
const sessionCounts = extract((state) => {
  let statValue: number | null = null;
  for (const span of Array.from(state.document.querySelectorAll<HTMLElement>("span"))) {
    if (span.textContent?.trim() === "Active Sessions") {
      const n = parseInt(span.nextElementSibling?.textContent ?? "", 10);
      if (!isNaN(n)) { statValue = n; break; }
    }
  }
  // Find the Active Sessions table specifically (it has an "ELAPSED" column header)
  let rowCount = 0;
  for (const th of Array.from(state.document.querySelectorAll<HTMLElement>("th"))) {
    if (th.textContent?.trim() === "Elapsed") {
      const tbody = th.closest("table")?.querySelector("tbody");
      rowCount = tbody?.querySelectorAll("tr:not(:has(td[colspan]))").length ?? 0;
      break;
    }
  }
  return { statValue, rowCount };
});

// ─── Properties ────────────────────────────────────────────────────────────────

/**
 * The highlighted nav tab must always match the current URL path.
 *
 * Why it's hard in Playwright: you'd have to script every navigation sequence
 * (click, back, forward, direct URL) to catch regressions. Bombadil explores
 * all of them automatically — including sequences no human would think to write.
 */
export const navTabMatchesUrl = always(() => {
  const { activeHref, pathname } = navState.current;
  return activeHref === null || activeHref === pathname;
});

/**
 * "Connecting…" placeholders must always eventually disappear.
 *
 * Why it's hard in Playwright: temporal properties ("this must eventually
 * become false") can't be expressed without polling hacks. In Bombadil,
 * always(eventually(x)) is a first-class formula.
 */
export const connectingAlwaysResolves = always(() =>
  connecting.current
    ? eventually(() => !connecting.current).within(10, "seconds")
    : true
);

/**
 * The "Active Sessions" stat card number must always equal the sessions table row count.
 *
 * Why it's hard in Playwright: you'd wire this assertion into every component
 * test individually. Bombadil checks it after every random action, site-wide,
 * for free.
 */
export const sessionCountMatchesTable = always(() => {
  const { statValue, rowCount } = sessionCounts.current;
  return statValue === null || statValue === rowCount;
});
