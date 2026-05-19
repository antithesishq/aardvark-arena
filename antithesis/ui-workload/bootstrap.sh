#!/usr/bin/env bash
set -euo pipefail

# Wait for the UI server to start serving HTTP. depends_on:service_started
# only blocks on the container starting, not on Next.js binding to :3000.
until curl -sf http://aardvark-arena-ui:3000 >/dev/null; do
  sleep 1
done

setup-complete.sh

# No --time-limit / --exit-on-violation: bombadil keeps exploring and
# accumulating violations for the entire run.
exec bombadil test \
  --headless \
  --no-sandbox \
  --chrome-grant-permissions=local-network,local-network-access,loopback-network \
  http://aardvark-arena-ui:3000 \
  /app/bombadil.spec.ts
