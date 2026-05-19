# -------------------------------------------------------------------
# Antithesis multi-stage Dockerfile for aardvark-arena
# Stages: instrument → build-matchmaker / build-gameserver / build-player
#       → matchmaker / gameserver / workload
# -------------------------------------------------------------------

# ---- Stage: instrument ----
# Run the Antithesis Go instrumentor on the entire source tree.
FROM golang:1.25-bookworm AS instrument

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Run the instrumentor for full coverage instrumentation + assertion cataloging.
RUN mkdir /instrumented && \
    go tool antithesis-go-instrumentor -skip_test_files=true /src /instrumented

# ---- Stage: build-matchmaker ----
FROM golang:1.25-bookworm AS build-matchmaker

WORKDIR /build
COPY --from=instrument /instrumented/customer/ ./
COPY --from=instrument /instrumented/notifier/ /notifier/

ENV CGO_ENABLED=1
RUN go build -race -o /matchmaker ./cmd/matchmaker

# ---- Stage: build-gameserver ----
FROM golang:1.25-bookworm AS build-gameserver

WORKDIR /build
COPY --from=instrument /instrumented/customer/ ./
COPY --from=instrument /instrumented/notifier/ /notifier/

ENV CGO_ENABLED=1
RUN go build -race -o /gameserver ./cmd/gameserver

# ---- Stage: build-player ----
FROM golang:1.25-bookworm AS build-player

WORKDIR /build
COPY --from=instrument /instrumented/customer/ ./
COPY --from=instrument /instrumented/notifier/ /notifier/

ENV CGO_ENABLED=1
RUN go build -race -o /player ./cmd/player

# ---- Stage: matchmaker (runtime) ----
FROM debian:bookworm-slim AS matchmaker

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

ENV NO_COLOR=1
RUN mkdir -p /data /symbols

COPY --from=build-matchmaker /matchmaker /usr/local/bin/matchmaker
COPY --from=instrument /instrumented/symbols/ /symbols/

ENTRYPOINT ["matchmaker"]
CMD ["-addr=:8080", "-db-path=/data/matchmaker.db", "-match-interval=100ms", "-session-timeout=2m", "-monitor-interval=1s"]

# ---- Stage: gameserver (runtime) ----
FROM debian:bookworm-slim AS gameserver

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

ENV NO_COLOR=1
RUN mkdir -p /symbols

COPY --from=build-gameserver /gameserver /usr/local/bin/gameserver
COPY --from=instrument /instrumented/symbols/ /symbols/

ENTRYPOINT ["gameserver"]
CMD ["-addr=:8081", "-self-url=http://gameserver:8081", "-matchmaker=http://matchmaker:8080", "-turn-timeout=10s", "-max-sessions=6"]

# ---- Stage: workload (runtime) ----
FROM debian:bookworm-slim AS workload

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates bash coreutils curl && \
    rm -rf /var/lib/apt/lists/*

ENV NO_COLOR=1
RUN mkdir -p /symbols /opt/antithesis/test/v1/main

COPY --from=build-player /player /usr/local/bin/player
COPY --from=instrument /instrumented/symbols/ /symbols/
COPY antithesis/setup-complete.sh /usr/local/bin/setup-complete.sh
COPY antithesis/test/ /opt/antithesis/test/v1/

# Entrypoint: emit setup_complete then sleep forever (test commands drive the workload)
ENTRYPOINT ["bash", "-c", "setup-complete.sh && exec sleep infinity"]

# ---- Stage: ui-build ----
# Build the Next.js production bundle. NEXT_PUBLIC_* env vars are inlined at
# build time, so the matchmaker URL must be set here, not at runtime.
FROM node:22-bookworm-slim AS ui-build

WORKDIR /app
COPY ui/package.json ui/package-lock.json ./
RUN npm ci

COPY ui/ ./
ENV NEXT_PUBLIC_MATCHMAKER_URL=http://aardvark-arena-matchmaker:8080
RUN npm run build

# ---- Stage: ui (runtime) ----
FROM node:22-bookworm-slim AS ui

WORKDIR /app
COPY --from=ui-build /app/.next ./.next
COPY --from=ui-build /app/public ./public
COPY --from=ui-build /app/node_modules ./node_modules
COPY --from=ui-build /app/package.json ./package.json
COPY --from=ui-build /app/next.config.ts ./next.config.ts

ENV NODE_ENV=production
EXPOSE 3000
CMD ["npx", "next", "start", "-p", "3000"]

# ---- Stage: ui-workload (runtime) ----
# Runs Bombadil property tests against the UI container. Uses Debian-packaged
# Chromium (bombadil discovers it on PATH) and the bombadil binary downloaded
# from GitHub releases. Bombadil is the container's main process — it isn't
# wrapped in a Test Composer command, so it runs continuously for the whole
# Antithesis run and accumulates property violations as it goes.
FROM node:22-bookworm-slim AS ui-workload

RUN apt-get update && apt-get install -y --no-install-recommends \
      bash ca-certificates curl chromium fonts-liberation && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL -o /usr/local/bin/bombadil \
      https://github.com/antithesishq/bombadil/releases/download/v0.5.0/bombadil-x86_64-linux && \
    chmod +x /usr/local/bin/bombadil

WORKDIR /app
COPY ui/package.json ui/package-lock.json ./
RUN npm ci

COPY ui/bombadil.spec.ts ./bombadil.spec.ts
COPY antithesis/setup-complete.sh /usr/local/bin/setup-complete.sh
COPY antithesis/ui-workload/bootstrap.sh /usr/local/bin/ui-workload-bootstrap.sh

# chromiumoxide emits a noisy "Invalid message" warning at INFO level for
# every CDP frame it can't parse; silence it as the local run does.
ENV RUST_LOG=chromiumoxide=error

ENTRYPOINT ["bash", "/usr/local/bin/ui-workload-bootstrap.sh"]
