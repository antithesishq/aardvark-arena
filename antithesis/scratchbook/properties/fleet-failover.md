# fleet-failover — Fleet Uses Available Servers

## Evidence

`Fleet.CreateSession` (`internal/matchmaker/fleet.go:68-143`) gathers candidates by filtering out servers whose `retryAt` is in the future. It shuffles candidates and tries each one. On 503 response, it sets `retryAt` from the Retry-After header. On temporary network error, it sets `retryAt` to `FailureTimeout` (1 minute).

The test (`internal/matchmaker/fleet_test.go:39-63`) verifies the fleet skips a 503 server and uses the second candidate.

## Failure Scenario

1. **All servers in retry**: If all servers recently failed, `candidates` is empty and `ErrNoServersAvailable` is returned. The match is dropped silently.
2. **retryAt clock skew**: `retryAt` uses `time.Now()`. In Antithesis, time behavior is controlled. If `time.Now()` returns unexpected values, retry logic could break.
3. **Retry-After header parsing**: If the game server returns a non-integer Retry-After, `strconv.Atoi` fails and the server gets the default `FailureTimeout`.

## Relevant Code Paths
- `internal/matchmaker/fleet.go:68-143` — `CreateSession`
- `internal/matchmaker/fleet.go:21` — `FailureTimeout` (1 minute)
- `internal/matchmaker/fleet_test.go` — Fleet tests

## SUT Instrumentation
- **Missing**: `AlwaysOrUnreachable` assertion that successful CreateSession returns a server not currently in retry.
