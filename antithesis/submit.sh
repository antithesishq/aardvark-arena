#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 [--duration minutes] [--registry <registry>]" >&2
  echo "  Registry can also be set via ANTITHESIS_REPOSITORY env var." >&2
}

DURATION="15"
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

if [[ -z "$REGISTRY" ]]; then
  echo "Error: --registry is required." >&2
  usage
  exit 1
fi

export ANTITHESIS_REPOSITORY="$REGISTRY"

SERVICE_IMAGE="${REGISTRY}/aardvark-arena-service:latest"
PLAYER_IMAGE="${REGISTRY}/aardvark-arena-player:latest"
HEALTH_CHECKER_IMAGE="${REGISTRY}/aardvark-arena-health-checker:latest"

# Build and push images (snouty --config handles the config image automatically)
docker build --platform linux/amd64 -f antithesis/Dockerfile --target service        -t "$SERVICE_IMAGE"        .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target player         -t "$PLAYER_IMAGE"         .
docker build --platform linux/amd64 -f antithesis/Dockerfile --target health-checker -t "$HEALTH_CHECKER_IMAGE" .
docker push "$SERVICE_IMAGE"
docker push "$PLAYER_IMAGE"
docker push "$HEALTH_CHECKER_IMAGE"

GIT_REV="$(git rev-parse HEAD)"
RUN_DESCRIPTION="[${USER}] ${GIT_REV}"

# Submit test run (snouty builds and pushes config image from antithesis/docker-compose.yaml)
snouty run \
  --webhook basic_test \
  --config antithesis \
  --test-name 'aardvark-arena' \
  --description "$RUN_DESCRIPTION" \
  --duration "$DURATION" \
  --recipients 'alex.carcoana@antithesis.com'
