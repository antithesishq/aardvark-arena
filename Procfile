matchmaker: go run ./cmd/matchmaker -addr=:8080 -gameserver=http://localhost:8081 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 -match-interval=500ms -session-timeout=5m -monitor-interval=500ms
gameserver: go run ./cmd/gameserver -addr=:8081 -matchmaker=http://localhost:8080 -token=a1b2c3d4-e5f6-7890-abcd-ef1234567890 -turn-timeout=30s -max-sessions=8
swarm: go run ./cmd/swarm -n=21 -matchmaker=http://localhost:8080 -move-delay=300ms -poll-interval=300ms
ui: cd ui && npm run dev
