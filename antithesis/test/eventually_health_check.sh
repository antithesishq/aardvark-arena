#!/usr/bin/env bash
# Eventually check: after all drivers and faults have stopped, verify that
# the system recovers and all services are healthy again.
set -euo pipefail

echo "Checking that all services are healthy after testing..."

# Check matchmaker
if ! curl -sf "http://matchmaker:8080/health" > /dev/null 2>&1; then
    echo "FAIL: matchmaker is not healthy"
    exit 1
fi
echo "matchmaker: healthy"

# Check all game servers
for gs in gameserver-1 gameserver-2 gameserver-3; do
    if ! curl -sf "http://${gs}:8081/health" > /dev/null 2>&1; then
        echo "FAIL: ${gs} is not healthy"
        exit 1
    fi
    echo "${gs}: healthy"
done

echo "All services recovered and healthy"
