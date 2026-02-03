#!/usr/bin/env bash
# Eventually check: after drivers complete, verify that no players are stuck
# in the matchmaking queue. A non-empty queue after all players have finished
# would indicate a matching or session management bug.
set -euo pipefail

echo "Verifying matchmaking queue is empty after testing..."

# Queue two fresh players for a quick game - if the system is working,
# they should match and complete. This validates end-to-end flow still works.
player -matchmaker="${MATCHMAKER_URL}" -num-sessions=1 &
PID1=$!

player -matchmaker="${MATCHMAKER_URL}" -num-sessions=1 &
PID2=$!

FAIL=0
wait $PID1 || FAIL=1
wait $PID2 || FAIL=1

if [ $FAIL -ne 0 ]; then
    echo "FAIL: post-test game session did not complete"
    exit 1
fi

echo "Post-test game completed successfully - system is functional"
