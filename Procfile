matchmaker: go run ./cmd/matchmaker -addr=:8080 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 -match-interval=2s -session-timeout=5m -monitor-interval=1s
gs-1: go run ./cmd/gameserver -addr=:8081 -self-url=http://localhost:8081 -matchmaker=http://localhost:8080 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 -turn-timeout=30s -max-sessions=20
gs-2: go run ./cmd/gameserver -addr=:8082 -self-url=http://localhost:8082 -matchmaker=http://localhost:8080 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 -turn-timeout=30s -max-sessions=20
swarm: go run ./cmd/swarm -n=100 -matchmaker=http://localhost:8080 -game-length=15s -poll-interval=500ms
ui: cd ui && npm run dev
