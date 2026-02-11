#!/usr/bin/env bash
# Parallel driver: probabilistically malicious player. It intentionally behaves
# badly on some turns while still often playing valid games.
set -euo pipefail

NUM_SESSIONS=20

echo "Starting evil player for ${NUM_SESSIONS} sessions"
player \
  -matchmaker="${MATCHMAKER_URL}" \
  -num-sessions="${NUM_SESSIONS}" \
  -evil \
  -evil-chaos-rate=0.35 \
  -evil-out-of-turn-rate=0.20 \
  -evil-malformed-rate=0.55 \
  -evil-extra-connect-rate=0.18 \
  -evil-queue-abandon-rate=0.16
echo "Player (evil mode) completed ${NUM_SESSIONS} sessions"
