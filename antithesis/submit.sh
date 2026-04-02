#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 [--duration minutes]" >&2
  echo "  Requires ANTITHESIS_REPOSITORY env var to be set." >&2
}

DURATION="15"

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

if [[ -z "${ANTITHESIS_REPOSITORY:-}" ]]; then
  echo "Error: ANTITHESIS_REPOSITORY env var is required." >&2
  usage
  exit 1
fi

export REPOSITORY="$ANTITHESIS_REPOSITORY"

# Build all images via docker compose
docker compose -f antithesis/config/docker-compose.yaml build

GIT_REV="$(git rev-parse HEAD)"

# Submit test run (snouty pushes images and builds config image automatically)
snouty run \
  --webhook basic_test \
  --config antithesis/config \
  --test-name 'aardvark-arena' \
  --description "[${USER}] ${GIT_REV}" \
  --duration "$DURATION" \
  --recipients 'alex.carcoana@antithesis.com'
