# max-sessions-reached — Game Server Hits Capacity Limit

## Evidence

`SessionManager.CreateSession` (`internal/gameserver/session.go:64-66`) checks capacity:
```go
if len(s.sessions) >= s.cfg.MaxSessions {
    return &ErrMaxSessions{RetryAt: minDeadline}
}
```

The Fleet interprets 503 with Retry-After and backs off (`internal/matchmaker/fleet.go:125-136`).

## Relevant Code Paths
- `internal/gameserver/session.go:60-78` — Capacity check in CreateSession
- `internal/matchmaker/fleet.go:125-136` — Fleet handles 503 response
- `internal/gameserver/server.go:90-113` — handleCreateSession returns 503

## SUT Instrumentation
- **Missing**: `Reachable` assertion when `ErrMaxSessions` is returned.
