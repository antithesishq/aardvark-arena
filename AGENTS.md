# Aardvark Arena

Go project for educational purposes. Keep code simple and clean.

## Commands

- `go build ./...` - verify project builds (no binaries generated)
- `go run ./cmd/<name>` - run a binary
- `go test ./...` - run tests
- `golangci-lint run` - lint and check for issues

## Frontend (ui/)

- Next.js + React + TypeScript + Tailwind + shadcn/ui
- `cd ui && npm install && npm run dev` — start dev server
- `cd ui && npm run build` — production build
- Bombadil spec: `ui/bombadil-spec.ts` — property-based UI tests
