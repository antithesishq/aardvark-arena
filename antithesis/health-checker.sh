#!/usr/bin/env bash
set -euo pipefail

# Wait for matchmaker health endpoint
echo "Waiting for matchmaker..."
until curl -sf http://matchmaker:8080/health > /dev/null 2>&1; do
    sleep 1
done
echo "Matchmaker is healthy."

# Wait for all game servers
for gs in gameserver-1 gameserver-2 gameserver-3; do
    echo "Waiting for ${gs}..."
    until curl -sf "http://${gs}:8081/health" > /dev/null 2>&1; do
        sleep 1
    done
    echo "${gs} is healthy."
done

echo "All services healthy. Emitting setup_complete."

# Emit setup_complete signal
echo '{"antithesis_setup": {"status": "complete", "details": {"message": "All services healthy and ready for testing"}}}' \
    >> "${ANTITHESIS_OUTPUT_DIR:-/tmp}/sdk.jsonl"
