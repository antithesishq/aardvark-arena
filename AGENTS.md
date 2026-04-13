# Aardvark Arena

Go project for educational purposes. Keep code simple and clean.

## Commands

- `go build ./...` - verify project builds (no binaries generated)
- `go run ./cmd/<name>` - run a binary
- `go test ./...` - run tests
- `golangci-lint run` - lint and check for issues

## Running locally

- `hivemind` — starts everything (matchmaker, 2 game servers, 100-player swarm, UI)
- Open http://localhost:3001 to see the dashboard

## Frontend (ui/)

- Next.js + React + TypeScript + Tailwind + shadcn/ui
- `cd ui && npm install && npm run dev` — start dev server
- `cd ui && npm run build` — production build

## UI property tests (Bombadil)

- `cd ui && npm run bombadil` — headless run (1 min, exits on first violation)
- `cd ui && npm run bombadil:headed` — headed run for debugging
- `ui/chrome/chrome-wrapper.sh` auto-installs and runs Chrome for Testing, passing arguments transparently
- Spec lives in `ui/bombadil.spec.ts`
