# Build Go server (frontend served separately via nginx/CDN)
FROM golang:1.25-bookworm AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.Version=docker -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -o /fxtunnel-server ./cmd/server

# Runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /fxtunnel-server /usr/local/bin/fxtunnel-server

RUN mkdir -p /data /etc/fxtunnel

RUN groupadd -r fxtunnel && useradd -r -g fxtunnel fxtunnel && chown -R fxtunnel:fxtunnel /data /etc/fxtunnel
USER fxtunnel

VOLUME ["/data"]

EXPOSE 4443 8080 8081

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/usr/local/bin/fxtunnel-server", "--version"]

ENTRYPOINT ["fxtunnel-server"]
CMD ["--config", "/etc/fxtunnel/server.yaml"]
