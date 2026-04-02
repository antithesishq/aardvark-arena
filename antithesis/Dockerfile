FROM golang:1.25 AS go-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Install the Antithesis Go instrumentor.
RUN go get -tool github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor@latest

# Instrument the source once, then build all service binaries from it.
RUN mkdir -p /instrumented /out
RUN go tool antithesis-go-instrumentor /src /instrumented

RUN cp /instrumented/customer/*catalog.go /instrumented/customer/cmd/matchmaker
RUN cp /instrumented/customer/*catalog.go /instrumented/customer/cmd/gameserver
RUN cp /instrumented/customer/*catalog.go /instrumented/customer/cmd/player

WORKDIR /instrumented/customer
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/matchmaker ./cmd/matchmaker
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/gameserver ./cmd/gameserver
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/player ./cmd/player

FROM debian:bookworm-slim AS runtime-base
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates curl \
  && rm -rf /var/lib/apt/lists/*

FROM runtime-base AS service
COPY --from=go-builder /out/matchmaker /out/gameserver /usr/local/bin/
COPY --from=go-builder /instrumented/symbols /symbols

FROM runtime-base AS player
COPY --from=go-builder /out/player /usr/local/bin/
COPY --from=go-builder /instrumented/symbols /symbols
COPY antithesis/test/ /opt/antithesis/test/v1/aardvark-arena/
ENTRYPOINT ["sleep", "infinity"]

FROM runtime-base AS health-checker
COPY antithesis/health-checker.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
