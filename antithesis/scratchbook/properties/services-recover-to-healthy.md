# services-recover-to-healthy

## Evidence

The `eventually_health_check.sh` test driver (`antithesis/test/eventually_health_check.sh`) runs after all parallel drivers complete. It curls the health endpoints of all 4 services (matchmaker + 3 game servers) and exits non-zero if any are unhealthy.

## Instrumentation Status

**FULLY COVERED** — The `eventually_` test command provides this check. No additional SUT-side assertions needed.
