# fxTunnel

Self-hosted reverse tunneling solution similar to ngrok or serveo.net.

## Features

- **HTTP Tunnels** - Expose local web services with custom subdomains (`myapp.tunnel.example.com`)
- **TCP Tunnels** - Forward any TCP port (SSH, databases, etc.)
- **UDP Tunnels** - Forward UDP traffic
- **Wildcard Domains** - Full `*.domain.com` support via nginx
- **Token Authentication** - Secure access with scoped permissions
- **Auto-Reconnect** - Client automatically reconnects on disconnect
- **Multiplexed Connections** - Efficient yamux-based stream multiplexing

## Quick Start

### Server Setup

1. Build the binaries:
```bash
make build
```

2. Create server config:
```bash
cp configs/server.example.yaml configs/server.yaml
# Edit configs/server.yaml with your settings
```

3. Run the server:
```bash
./bin/fxtunnel-server --config configs/server.yaml
```

### Client Usage

Expose a local HTTP server:
```bash
./bin/fxtunnel http 3000 --server tunnel.example.com:4443 --token sk_xxx
```

Expose with custom subdomain:
```bash
./bin/fxtunnel http 3000 --subdomain myapp --server tunnel.example.com:4443 --token sk_xxx
```

Expose TCP port:
```bash
./bin/fxtunnel tcp 22 --server tunnel.example.com:4443 --token sk_xxx
```

Use config file:
```bash
./bin/fxtunnel --config configs/client.yaml
```

## Architecture

```
                                    INTERNET
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
                    ▼                   ▼                   ▼
              *.domain.com         TCP ports           UDP ports
              (via nginx)         (dynamic)            (dynamic)
                    │                   │                   │
                    └───────────────────┼───────────────────┘
                                        │
                                        ▼
                            ┌───────────────────┐
                            │   fxtunnel-server │
                            │                   │
                            │ • Control Plane   │
                            │ • HTTP Router     │
                            │ • TCP Manager     │
                            │ • UDP Manager     │
                            └─────────┬─────────┘
                                      │
                         Control Connection (TCP)
                         + Yamux Multiplexed Streams
                                      │
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
      ┌──────────────┐       ┌──────────────┐       ┌──────────────┐
      │   Client 1   │       │   Client 2   │       │   Client N   │
      │ webapp:3000  │       │ ssh:22       │       │ dns:53/udp   │
      └──────────────┘       └──────────────┘       └──────────────┘
```

## Nginx Configuration

For wildcard domain support with SSL, configure nginx:

```nginx
server {
    listen 443 ssl http2;
    server_name *.tunnel.example.com;

    ssl_certificate /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

Get wildcard certificate with certbot:
```bash
certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d tunnel.example.com \
  -d *.tunnel.example.com
```

## Configuration

### Server Config (`server.yaml`)

```yaml
server:
  control_port: 4443
  http_port: 8080
  tcp_port_range:
    min: 10000
    max: 20000
  udp_port_range:
    min: 20001
    max: 30000

domain:
  base: "tunnel.example.com"
  wildcard: true

auth:
  enabled: true
  tokens:
    - name: "admin"
      token: "sk_admin_xxx"
      allowed_subdomains: ["*"]
      max_tunnels: 100
```

### Client Config (`client.yaml`)

```yaml
server:
  address: "tunnel.example.com:4443"
  token: "sk_admin_xxx"

tunnels:
  - name: "webapp"
    type: "http"
    local_port: 3000
    subdomain: "myapp"

  - name: "ssh"
    type: "tcp"
    local_port: 22

reconnect:
  enabled: true
  interval: 5s
```

## Environment Variables

Server:
- `FXTUNNEL_SERVER_CONTROL_PORT` - Control plane port
- `FXTUNNEL_SERVER_HTTP_PORT` - HTTP listener port
- `FXTUNNEL_AUTH_ENABLED` - Enable authentication

Client:
- `FXTUNNEL_SERVER_ADDRESS` - Server address
- `FXTUNNEL_SERVER_TOKEN` - Authentication token

## Building

```bash
# Build both binaries
make build

# Build server only
make server

# Build client only
make client

# Run tests
make test

# Clean build artifacts
make clean
```

## Protocol

The protocol uses length-prefixed JSON messages over TCP:

```
┌──────────────┬──────────────────────────────┐
│ Length (4B)  │ JSON Payload                 │
│ big-endian   │                              │
└──────────────┴──────────────────────────────┘
```

Data streams are multiplexed using [yamux](https://github.com/hashicorp/yamux).

## Security

- Use TLS termination at nginx level
- Generate strong random tokens
- Limit subdomains per token
- Set appropriate tunnel limits
- Configure firewall rules for port ranges

## License

MIT
