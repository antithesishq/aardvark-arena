#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHROME="$SCRIPT_DIR/.chrome/chrome/mac_arm-130.0.6723.58/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
DEBUG_PORT=9222
BOMBADIL_ARGS=("$@")

if [ ! -f "$CHROME" ]; then
  echo "Chrome not found. Run: npx @puppeteer/browsers install chrome@130.0.6723.58 --path .chrome" >&2
  exit 1
fi

# Launch Chrome with remote debugging
"$CHROME" \
  --remote-debugging-port=$DEBUG_PORT \
  --no-first-run \
  --no-default-browser-check \
  --disable-background-networking \
  --user-data-dir="$(mktemp -d)" \
  "${BOMBADIL_ARGS[@]:+}" \
  &>/dev/null &
CHROME_PID=$!

cleanup() { kill "$CHROME_PID" 2>/dev/null || true; wait "$CHROME_PID" 2>/dev/null || true; }
trap cleanup EXIT

# Wait for the debugger to be ready
for i in $(seq 1 30); do
  if curl -s "http://localhost:$DEBUG_PORT/json/version" &>/dev/null; then
    break
  fi
  sleep 0.2
done

bombadil test-external \
  --remote-debugger "http://localhost:$DEBUG_PORT" \
  --chrome-grant-permissions=local-network,local-network-access,loopback-network \
  --time-limit 1m \
  --exit-on-violation \
  http://localhost:8000 \
  bombadil.spec.ts
