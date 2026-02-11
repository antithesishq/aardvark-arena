#!/usr/bin/env bash
# Parallel driver: a single AI player that plays through multiple game sessions.
# Test Composer runs many of these concurrently across client containers,
# creating realistic matchmaking pressure and concurrent game execution.
set -euo pipefail

NUM_SESSIONS=20

echo "Starting player for ${NUM_SESSIONS} sessions"
player -matchmaker="${MATCHMAKER_URL}" -num-sessions="${NUM_SESSIONS}"
echo "Player completed ${NUM_SESSIONS} sessions"
