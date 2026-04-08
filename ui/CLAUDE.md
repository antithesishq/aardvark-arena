@AGENTS.md

# Aardvark Arena UI

## Stack
- React + Next.js (App Router), TypeScript, Tailwind, shadcn/ui
- No custom utility fns if a library already does it — prefer idiomatic deps

## Commands
- `npm run dev` — start dev server
- `npm run build` — production build

## Conventions
- Optimize for readability over production-readiness
- Simple > clever. If two approaches exist, pick the shorter one
- Components live in src/components/, boards in src/components/boards/
- API calls in src/lib/api.ts, WebSocket logic in src/lib/ws.ts

## API targets
- Matchmaker: polls /status and /leaderboard
- Game servers: /sessions list + /session/{sid}/watch WebSocket (spectator, read-only)
- CORS is open (Access-Control-Allow-Origin: *) on all backend endpoints

## Style
- shadcn/ui for all UI primitives (Button, Table, Badge, Card)
- Tailwind for layout/spacing, no custom CSS files
