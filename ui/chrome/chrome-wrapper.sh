#!/bin/sh
DIR="$(cd "$(dirname "$0")/.." && pwd)"
CHROME_BIN=$(npx --prefix "$DIR" @puppeteer/browsers install chrome@144 | cut -d' ' -f2-)

if [ -z "$CHROME_BIN" ]; then
  echo "chrome-wrapper: could not resolve Chrome for Testing" >&2
  exit 1
fi

exec "$CHROME_BIN" --disable-features=LocalNetworkAccessChecks "$@"
