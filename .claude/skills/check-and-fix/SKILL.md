---
name: check-and-fix
description: Run project level linters, formatters, and static analysis tools. Then fix any issues raised.
---

1. run `golangci-lint run --fix` from the root of the project directory
2. run `go fmt ./...` from the root of the project directory
3. run `npm run lint -- --fix` from the `ui/` directory
4. run `npm run format` from the `ui/` directory
5. fix any remaining issues
