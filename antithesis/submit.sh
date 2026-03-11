#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 [--duration seconds] [--registry <registry>]" >&2
  echo "  Note: without --registry, this script runs in build-only mode." >&2
}

DURATION="60"
REGISTRY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --duration)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --duration" >&2
        usage
        exit 2
      fi
      DURATION="$2"
      shift 2
      ;;
    --registry)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --registry" >&2
        usage
        exit 2
      fi
      REGISTRY="$2"
      shift 2
      ;;
    --help)
      usage
      exit 0
      ;;
    -*)
      echo "Unknown option: $1" >&2
      usage
      exit 2
      ;;
    *)
      echo "Unexpected argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ -n "$REGISTRY" ]]; then
  SERVICE_IMAGE="${REGISTRY}/aardvark-arena-service:latest"
  PLAYER_IMAGE="${REGISTRY}/aardvark-arena-player:latest"
  HEALTH_CHECKER_IMAGE="${REGISTRY}/aardvark-arena-health-checker:latest"
  CONFIG_IMAGE="${REGISTRY}/aardvark-arena-config:latest"
else
  SERVICE_IMAGE="aardvark-arena-service"
  PLAYER_IMAGE="aardvark-arena-player"
  HEALTH_CHECKER_IMAGE="aardvark-arena-health-checker"
  CONFIG_IMAGE="aardvark-arena-config"
fi

# Build images
docker build --platform linux/amd64 -f antithesis/Dockerfile --target service -t "$SERVICE_IMAGE" .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target player -t "$PLAYER_IMAGE" .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target health-checker -t "$HEALTH_CHECKER_IMAGE" .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target config -t "$CONFIG_IMAGE" .

if [[ -z "$REGISTRY" ]]; then
  echo "Build complete (no --registry): skipped push and test run."
  exit 0
fi

# Push images (required before snouty run)
docker push "$SERVICE_IMAGE"
docker push "$PLAYER_IMAGE"
docker push "$HEALTH_CHECKER_IMAGE"
docker push "$CONFIG_IMAGE"

GIT_REV="$(git rev-parse HEAD)"
RUN_DESCRIPTION="aardvark-arena antithesis test (rev ${GIT_REV})"

# Submit test run
snouty run \
  --webhook basic_test \
  --antithesis.test_name 'aardvark-arena' \
  --antithesis.description "$RUN_DESCRIPTION" \
  --antithesis.config_image "$CONFIG_IMAGE" \
  --antithesis.images "$SERVICE_IMAGE;$PLAYER_IMAGE;$HEALTH_CHECKER_IMAGE" \
  --antithesis.duration "$DURATION" \
  --antithesis.report.recipients 'alex.carcoana@antithesis.com'
