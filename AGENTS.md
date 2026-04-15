# Aardvark Arena

Go project for educational purposes. Keep code simple and clean.

## Commands

- `go build ./...` - verify project builds (no binaries generated)
- `go run ./cmd/<name>` - run a binary
- `go test ./...` - run tests
- `golangci-lint run` - lint and check for issues

## Frontend (ui/)

- React + Next.js (App Router), TypeScript, Tailwind, shadcn/ui
- No custom utility fns if a library already does it — prefer idiomatic deps
- `cd ui && npm install && npm run dev` — start dev server
- `cd ui && npm run build` — production build

### Conventions
- Optimize for readability over production-readiness
- Simple > clever. If two approaches exist, pick the shorter one
- Components live in ui/components/, boards in ui/components/boards/
- API calls in ui/lib/api.ts, WebSocket logic in ui/lib/ws.ts

### API targets
- Matchmaker: polls /status and /leaderboard
- Game servers: /sessions list + /session/{sid}/watch WebSocket (spectator, read-only)
- CORS is open (Access-Control-Allow-Origin: *) on all backend endpoints

### Style
- shadcn/ui for UI primitives (Button, Select)
- Tailwind for layout/spacing/styling, no custom CSS files besides globals.css

## UI property tests (Bombadil)

- `cd ui && npm run bombadil` — headless run (1 min, exits on first violation)
- `cd ui && npm run bombadil:headed` — headed run for debugging
- `ui/chrome/chrome-wrapper.sh` auto-installs and runs Chrome for Testing, passing arguments transparently
- Spec lives in `ui/bombadil.spec.ts`
