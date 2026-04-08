# post-fault-game-completion

## Evidence

The `eventually_queue_empty.sh` test driver (`antithesis/test/eventually_queue_empty.sh`) runs two fresh players for 1 session each after all parallel drivers complete. If either player fails to complete, the test exits non-zero.

This validates the full end-to-end flow: queue -> match -> create session -> play game -> report result.

## Instrumentation Status

**FULLY COVERED** — The `eventually_` test command provides this check. No additional SUT-side assertions needed.
