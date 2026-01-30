FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /matchmaker ./cmd/matchmaker
RUN CGO_ENABLED=1 go build -o /gameserver ./cmd/gameserver

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /matchmaker /gameserver /usr/local/bin/
