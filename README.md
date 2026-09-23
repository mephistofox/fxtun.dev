<p align="center">
  <img src="assets/logo.png" alt="fxTunnel Logo" width="120" height="120">
</p>

<h1 align="center">fxTunnel</h1>

<p align="center">
  <strong>Self-hosted reverse tunnel server to expose your localhost to the internet</strong>
</p>

<p align="center">
  <a href="https://github.com/mephistofox/fxtun.dev/releases/latest"><img src="https://img.shields.io/github/v/release/mephistofox/fxtun.dev?style=flat-square&color=brightgreen" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/mephistofox/fxtun.dev"><img src="https://goreportcard.com/badge/github.com/mephistofox/fxtun.dev?style=flat-square" alt="Go Report Card"></a>
  <a href="https://github.com/mephistofox/fxtun.dev/releases"><img src="https://img.shields.io/github/downloads/mephistofox/fxtun.dev/total?style=flat-square&logo=github" alt="Downloads"></a>
  <img src="https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <a href="https://ghcr.io/mephistofox/fxtun.dev"><img src="https://img.shields.io/badge/docker-ghcr.io-blue?style=flat-square&logo=docker" alt="Docker"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT%20with%20Attribution-yellow?style=flat-square" alt="License"></a>
  <a href="https://github.com/mephistofox/fxtun.dev/stargazers"><img src="https://img.shields.io/github/stars/mephistofox/fxtun.dev?style=flat-square&logo=github" alt="Stars"></a>
</p>

<p align="center">
  <a href="https://fxtun.dev">Website</a> &bull;
  <a href="README_RU.md">Русский</a> &bull;
  <a href="https://github.com/mephistofox/fxtun.dev/discussions">Discussions</a>
</p>

---

## What is fxTunnel?

**fxTunnel** is a fast, self-hosted reverse tunneling solution written in Go. It lets you expose local HTTP, TCP, and UDP services to the internet through a server you fully control — no third-party dependencies, no usage limits, no vendor lock-in.

Deploy the server on any VPS, point a wildcard DNS record at it, and your team instantly gets secure public URLs for local development servers, webhook testing, IoT devices, SSH access, and more.

## Comparison

| Feature | fxTunnel | ngrok | Cloudflare Tunnel | frp |
|---|:---:|:---:|:---:|:---:|
| **Self-hosted** | Yes | No | Partial | Yes |
| **Open source** | Yes | No | Client only | Yes |
| **HTTP tunnels** | Yes | Yes | Yes | Yes |
| **TCP tunnels** | Yes | Yes | Yes | Yes |
| **UDP tunnels** | Yes | Paid | No | Yes |
| **Custom subdomains** | Unlimited | 1 free | Via DNS | Manual |
| **Wildcard domains** | Yes | Paid | No | No |
| **Web admin panel** | Built-in | Cloud dashboard | Cloud dashboard | Dashboard plugin |
| **GUI desktop client** | Yes | No | No | No |
| **User management & 2FA** | Built-in | Cloud-based | Cloudflare Access | No |
| **Connection limits** | None | 1 tunnel free | No limit | None |
| **Bandwidth limits** | None | 1 GB/mo free | No limit | None |
| **Price** | **Free** | From $8/mo | Free (requires CF) | **Free** |
| **Multiplexing** | yamux | QUIC | QUIC | Custom |

## Key Features

- **HTTP Tunnels** — Expose local web services at `yourapp.tunnel.example.com` with automatic subdomain routing
- **TCP Tunnels** — Forward any TCP port: SSH, databases, game servers, RDP
- **UDP Tunnels** — Forward UDP traffic for DNS, VoIP, gaming protocols
- **Wildcard Domains** — Full `*.yourdomain.com` support with nginx + Let's Encrypt
- **Web Admin Panel** — Manage users, tokens, domains, and active tunnels from a built-in Vue 3 dashboard
- **User Management** — Registration with invite codes, TOTP two-factor authentication, scoped API tokens
- **Desktop GUI Client** — Cross-platform Wails-based app (Linux, macOS, Windows) with system tray support
- **CLI Client** — Lightweight command-line client with YAML config and auto-reconnect
- **Stream Multiplexing** — Efficient [yamux](https://github.com/hashicorp/yamux)-based multiplexed connections over a single TCP link
- **Docker Ready** — Official container image on GitHub Container Registry
- **Security** — Interstitial warning pages for untrusted tunnel traffic, TLS termination via nginx, token-scoped permissions

## Quick Start

### Install Client

One-liner install (Linux/macOS):

```bash
curl -fsSL https://fxtun.dev/install.sh | sh
```

Or download from [Releases](https://github.com/mephistofox/fxtun.dev/releases).

> **Windows users:** Windows may show a SmartScreen warning ("Windows protected your PC") when running the `.exe` for the first time. This is expected — the binaries are not code-signed yet. All release binaries are automatically scanned with [VirusTotal](https://www.virustotal.com/) during CI — check the scan results linked in each [release](https://github.com/mephistofox/fxtun.dev/releases).
>
> To bypass SmartScreen: click **"More info"** → **"Run anyway"**.

### Client Usage

Expose a local HTTP server:
```bash
fxtunnel http 3000 --server tunnel.example.com:4443 --token sk_your_token
# → https://random-subdomain.tunnel.example.com
```

Use a custom subdomain:
```bash
fxtunnel http 3000 --domain myapp --server tunnel.example.com:4443 --token sk_your_token
# → https://myapp.tunnel.example.com
```

Forward a TCP port (SSH, database, etc.):
```bash
fxtunnel tcp 22 --server tunnel.example.com:4443 --token sk_your_token
```

Forward UDP traffic:
```bash
fxtunnel udp 53 --server tunnel.example.com:4443 --token sk_your_token
```

Use a config file for persistent tunnels:
```bash
fxtunnel --config client.yaml
```

### Server Setup

Install the server via Docker:

```bash
docker run -d \
  --name fxtunnel \
  -p 4443:4443 \
  -p 8080:8080 \
  -p 3000:3000 \
  -p 10000-20000:10000-20000 \
  -v ./data:/app/data \
  -v ./configs/server.yaml:/app/configs/server.yaml \
  ghcr.io/mephistofox/fxtun.dev:latest
```

Or build from source:

```bash
git clone https://github.com/mephistofox/fxtun.dev.git
cd fxtun.dev
make build
./bin/fxtunnel-server --config configs/server.yaml
```

Point a wildcard DNS record to your server:
```
*.tunnel.example.com  →  A  →  YOUR_SERVER_IP
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
                            │   fxtunnel-server  │
                            │                    │
                            │  • HTTP Router     │
                            │  • TCP Manager     │
                            │  • UDP Manager     │
                            │  • Web Admin UI    │
                            │  • REST API        │
                            └─────────┬──────────┘
                                      │
                         yamux-multiplexed TCP
                                      │
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
      ┌──────────────┐       ┌──────────────┐       ┌──────────────┐
      │   Client 1   │       │   Client 2   │       │   Client N   │
      │ webapp:3000  │       │   ssh:22     │       │ dns:53/udp   │
      └──────────────┘       └──────────────┘       └──────────────┘
```

## Configuration

### Server (`server.yaml`)

```yaml
server:
  control_port: 4443      # Client tunnel connections
  http_port: 8080         # HTTP tunnel traffic
  tcp_port_range:
    min: 10000
    max: 20000
  udp_port_range:
    min: 20001
    max: 30000

domain:
  base: "tunnel.example.com"
  wildcard: true

web:
  port: 3000              # Admin panel & API

auth:
  jwt_secret: "change-me"
  totp_key: "change-me"

database:
  path: "./data/fxtunnel.db"
```

### Client (`client.yaml`)

```yaml
server:
  address: "tunnel.example.com:4443"
  token: "sk_your_token"

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

### Environment Variables

All config values can be set via environment variables with `FXTUNNEL_` prefix:

```bash
export FXTUNNEL_AUTH_JWT_SECRET="your-secret"
export FXTUNNEL_SERVER_CONTROL_PORT=4443
export FXTUNNEL_DATABASE_PATH="./data/fxtunnel.db"
```

## Nginx + SSL

For production use with HTTPS, configure nginx as a TLS-terminating reverse proxy:

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
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

Get a wildcard certificate:
```bash
certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d tunnel.example.com \
  -d *.tunnel.example.com
```

## Building from Source

```bash
make build          # Build CLI client + server
make server         # Build server only
make client         # Build CLI client only
make gui            # Build desktop GUI client (requires Wails)
make web            # Build Vue 3 admin panel
make test           # Run tests
make build-all      # Full build: web + server + all platform clients
```

**Requirements:** Go 1.24+, Node.js 20+ (for web UI and GUI client)

## Protocol

fxTunnel uses a custom length-prefixed JSON protocol over TCP, with [yamux](https://github.com/hashicorp/yamux) stream multiplexing:

```
┌──────────────┬──────────────────────────────┐
│ Length (4B)   │ JSON Payload                 │
│ big-endian    │                              │
└──────────────┴──────────────────────────────┘
```

## Contributing

Contributions are welcome! Please open an issue first to discuss what you would like to change.

## License

MIT with Attribution Requirement — see [LICENSE](LICENSE).

Any use, deployment, or distribution must include visible attribution:
- **GitHub:** [github.com/mephistofox/fxtun.dev](https://github.com/mephistofox/fxtun.dev)
- **Website:** [fxtun.dev](https://fxtun.dev)
