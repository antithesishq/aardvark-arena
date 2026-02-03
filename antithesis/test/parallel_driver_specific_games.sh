#!/usr/bin/env bash
# Parallel driver: a single AI player that plays a smaller batch of sessions.
# Provides a shorter-lived workload that completes and restarts more frequently,
# exercising player join/leave dynamics under fault injection.
set -euo pipefail

NUM_SESSIONS=5

echo "Starting player for ${NUM_SESSIONS} sessions (short batch)"
player -matchmaker="${MATCHMAKER_URL}" -num-sessions="${NUM_SESSIONS}"
echo "Player completed ${NUM_SESSIONS} sessions"
