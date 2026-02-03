#!/usr/bin/env bash
set -euo pipefail

DURATION="${1:-60}"
REGISTRY="us-central1-docker.pkg.dev/molten-verve-216720/orbitinghail-repository"
SERVICE_IMAGE="${REGISTRY}/aardvark-arena-service:latest"
PLAYER_IMAGE="${REGISTRY}/aardvark-arena-player:latest"
HEALTH_CHECKER_IMAGE="${REGISTRY}/aardvark-arena-health-checker:latest"
CONFIG_IMAGE="${REGISTRY}/aardvark-arena-config:latest"

# Build images
docker build --platform linux/amd64 -f antithesis/Dockerfile.service -t "$SERVICE_IMAGE" .
docker build --platform linux/amd64 -f antithesis/Dockerfile.player -t "$PLAYER_IMAGE" .
docker build --platform linux/amd64 -f antithesis/Dockerfile.health-checker -t "$HEALTH_CHECKER_IMAGE" .
docker build --platform linux/amd64 -f antithesis/Dockerfile.config -t "$CONFIG_IMAGE" .

# Push images
docker push "$SERVICE_IMAGE"
docker push "$PLAYER_IMAGE"
docker push "$HEALTH_CHECKER_IMAGE"
docker push "$CONFIG_IMAGE"

# Submit test run
snouty run \
  --webhook basic_test \
  --antithesis.test_name 'aardvark-arena' \
  --antithesis.description 'aardvark-arena antithesis test' \
  --antithesis.config_image "$CONFIG_IMAGE" \
  --antithesis.images "$SERVICE_IMAGE;$PLAYER_IMAGE;$HEALTH_CHECKER_IMAGE" \
  --antithesis.duration "$DURATION" \
  --antithesis.report.recipients 'antithesis-results@orbitinghail.dev'
