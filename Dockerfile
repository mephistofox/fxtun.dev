# Stage 1: Build web frontend
FROM node:20-slim AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go server
FROM golang:1.24-bookworm AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /app/web/dist ./internal/web/dist/

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.Version=docker -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -o /fxtunnel-server ./cmd/server

# Stage 3: Runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /fxtunnel-server /usr/local/bin/fxtunnel-server

RUN mkdir -p /data /etc/fxtunnel
VOLUME ["/data"]

EXPOSE 4443 8080 8081

ENTRYPOINT ["fxtunnel-server"]
CMD ["--config", "/etc/fxtunnel/server.yaml"]
