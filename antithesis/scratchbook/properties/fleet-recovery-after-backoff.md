# fleet-recovery-after-backoff

## Evidence

When a game server returns 503 or a temporary transport error, `Fleet.CreateSession` sets `server.retryAt` to a future time (`fleet.go:118-119` for transport errors, `fleet.go:144-151` for 503). On subsequent calls, servers with `retryAt` in the future are excluded (`fleet.go:74`):

```go
if server.retryAt == nil || server.retryAt.Before(now) {
    candidates = append(candidates, server)
}
```

After the backoff period expires, the server becomes a candidate again. The backoff duration is either parsed from the `Retry-After` header (for 503) or defaults to `FailureTimeout` (1 minute).

## Code Paths

- Backoff set: `fleet.go:118-119` (transport error), `fleet.go:144-151` (503 with Retry-After)
- Backoff check: `fleet.go:72-76` — exclude backed-off servers
- Recovery: implicit — time passes, `retryAt.Before(now)` becomes true

## Potential Violation Scenario

If `FailureTimeout` is very long and all servers simultaneously enter backoff, no sessions can be created until the first backoff expires. This is working as designed but means the fleet has a minimum recovery time of `FailureTimeout`. In Antithesis config, this is 1 minute (the default).

A more subtle issue: `retryAt` is never cleared — it persists even after the server successfully handles a request. This means a server that had one failure is always checked against its old `retryAt`, but since `retryAt.Before(now)` will be true after the backoff, this is harmless.

## Instrumentation Status

**NOT COVERED** — R4 confirms "no servers available" is reached, R5 and R6 confirm failures occur. But no assertion verifies that a backed-off server eventually re-enters the candidate pool. A `Sometimes` on the condition "a server with a non-nil retryAt is selected as a candidate" would verify recovery.
