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

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ENV NO_COLOR=1
RUN mkdir -p /data /symbols

COPY --from=build-matchmaker /matchmaker /usr/local/bin/matchmaker
COPY --from=instrument /instrumented/symbols/ /symbols/

ENTRYPOINT ["matchmaker"]
CMD ["-addr=:8080", "-gameserver=http://gameserver:8081", "-db-path=/data/matchmaker.db", "-match-interval=100ms", "-session-timeout=2m", "-monitor-interval=1s"]

# ---- Stage: gameserver (runtime) ----
FROM debian:bookworm-slim AS gameserver

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ENV NO_COLOR=1
RUN mkdir -p /symbols

COPY --from=build-gameserver /gameserver /usr/local/bin/gameserver
COPY --from=instrument /instrumented/symbols/ /symbols/

ENTRYPOINT ["gameserver"]
CMD ["-addr=:8081", "-matchmaker=http://matchmaker:8080", "-turn-timeout=10s", "-max-sessions=6"]

# ---- Stage: workload (runtime) ----
FROM debian:bookworm-slim AS workload

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates bash coreutils && \
    rm -rf /var/lib/apt/lists/*

ENV NO_COLOR=1
RUN mkdir -p /symbols /opt/antithesis/test/v1/main

COPY --from=build-player /player /usr/local/bin/player
COPY --from=instrument /instrumented/symbols/ /symbols/
COPY antithesis/setup-complete.sh /usr/local/bin/setup-complete.sh
COPY antithesis/test/ /opt/antithesis/test/v1/

# Entrypoint: emit setup_complete then sleep forever (test commands drive the workload)
ENTRYPOINT ["bash", "-c", "setup-complete.sh && exec sleep infinity"]
