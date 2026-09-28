# fxTunnel User Guide

Complete guide to the fxTunnel client — from installation to advanced usage.

---

## Table of Contents

- [Installation](#installation)
- [Authentication](#authentication)
- [Quick Start](#quick-start)
- [HTTP Tunnels](#http-tunnels)
- [TCP Tunnels](#tcp-tunnels)
- [UDP Tunnels](#udp-tunnels)
- [Subdomain Management](#subdomain-management)
- [Custom Domains](#custom-domains)
- [Configuration File](#configuration-file)
- [Daemon Mode](#daemon-mode)
- [Traffic Inspector](#traffic-inspector)
- [Warning Page](#warning-page)
- [HTTP Headers](#http-headers)
- [Reconnection](#reconnection)
- [Security Presets](#security-presets)
- [Updating the Client](#updating-the-client)
- [Limits](#limits)
- [FAQ](#faq)

---

## Installation

### Linux / macOS

One-line install:

```bash
curl -fsSL https://fxtun.dev/install.sh | sh
```

The script auto-detects your OS and architecture, downloads the binary to `~/.local/bin/`, and creates `fxtunnel` and `fxtun` symlinks.

**Supported platforms:**
- Linux: amd64, arm64
- macOS: amd64 (Intel), arm64 (Apple Silicon)

### Windows

PowerShell (run as administrator):

```powershell
irm https://fxtun.dev/install.ps1 | iex
```

Or download the `.exe` manually from the [downloads page](https://fxtun.dev/downloads).

> **SmartScreen:** Windows may show a "Windows protected your PC" warning on first run. The binaries are not yet code-signed. Click **"More info"** → **"Run anyway"**.

### GUI Client

A desktop application with a graphical interface is available for Linux and Windows on the [downloads page](https://fxtun.dev/downloads).

### Verify Installation

```bash
fxtunnel version
```

---

## Authentication

You need an API token to use fxTunnel. Get one from your [dashboard](https://fxtun.dev/login) → "Tokens" section.

### Save Token

Interactive mode (choose between token entry or browser auth):

```bash
fxtunnel login
```

Or provide the token directly:

```bash
fxtunnel login -t sk_your_token
```

The token is stored in the system keyring:
- **macOS:** Keychain
- **Linux:** Secret Service (GNOME Keyring / KDE Wallet)
- **Windows:** Credential Manager

### Remove Token

```bash
fxtunnel logout
```

### Use Without Saving

Pass the token on every call via flag or environment variable:

```bash
fxtunnel http 3000 --token sk_your_token

# Or via env
export FXTUNNEL_SERVER_TOKEN=sk_your_token
fxtunnel http 3000
```

---

## Quick Start

### HTTP Tunnel

Expose a local web server on port 3000:

```bash
fxtunnel http 3000
```

Output:
```
Tunnel established!
HTTP: https://ab1cd2ef.fxtun.dev
Forwarding to localhost:3000
Inspector: http://127.0.0.1:4040
```

Now `https://ab1cd2ef.fxtun.dev` is publicly accessible and proxies traffic to your `localhost:3000`.

### TCP Tunnel

Expose an SSH server:

```bash
fxtunnel tcp 22
```

Output:
```
Tunnel established!
TCP: fxtun.dev:51234
Forwarding to localhost:22
```

Connect: `ssh user@fxtun.dev -p 51234`

### UDP Tunnel

Expose a local DNS server:

```bash
fxtunnel udp 53
```

---

## HTTP Tunnels

```bash
fxtunnel http <port> [flags]
```

### Custom Subdomain

```bash
fxtunnel http 3000 --domain myapp
# → https://myapp.fxtun.dev
```

`--domain` and `--subdomain` are aliases. If not specified, a random subdomain is generated.

### Basic Auth

Protect your tunnel with a username and password:

```bash
fxtunnel http 3000 --auth user:MySecurePass123
```

Requirements: password must be at least 8 characters, username cannot contain `:`.

### IP Allowlist

Restrict access to specific IPs or subnets:

```bash
fxtunnel http 3000 --allow-ip 203.0.113.10 --allow-ip 10.0.0.0/8
```

The `--allow-ip` flag can be repeated. Supports individual IPs and CIDR notation.

### Auto-Close

Automatically close the tunnel after a period of inactivity:

```bash
fxtunnel http 3000 --auto-close 30m
```

Valid range: `1m` to `24h`. The timer resets on each request.

### Max Lifetime

Force-close the tunnel after a set duration, regardless of activity:

```bash
fxtunnel http 3000 --max-lifetime 8h
```

Valid range: `1m` to `7d`.

### Combining Flags

```bash
fxtunnel http 3000 \
  --domain myapp \
  --auth admin:StrongPass123 \
  --allow-ip 10.0.0.0/8 \
  --auto-close 1h \
  --max-lifetime 24h
```

### All HTTP Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--domain` | `-d` | Subdomain | Auto |
| `--subdomain` | | Alias for --domain | Auto |
| `--auth` | | Basic Auth (user:password) | None |
| `--allow-ip` | | Allowed IP/CIDR (repeatable) | All IPs |
| `--auto-close` | | Close on idle (1m–24h) | None |
| `--max-lifetime` | | Max lifetime (1m–7d) | None |
| `--preset` | | Security preset | None |

---

## TCP Tunnels

```bash
fxtunnel tcp <port> [flags]
```

### Auto-Assigned Port

```bash
fxtunnel tcp 22
# → fxtun.dev:51234 (assigned by server)
```

### Specific Port

```bash
fxtunnel tcp 22 --remote-port 2222
# → fxtun.dev:2222
```

### Examples

SSH access:
```bash
fxtunnel tcp 22 --remote-port 2222
# Connect: ssh user@fxtun.dev -p 2222
```

PostgreSQL:
```bash
fxtunnel tcp 5432 --remote-port 15432 --allow-ip 10.0.0.0/8
# Connect: psql -h fxtun.dev -p 15432 -U myuser mydb
```

### Blocked Ports

On the free plan, certain remote ports are blocked for TCP tunnels:

| Port | Service |
|------|---------|
| 22 | SSH |
| 25 | SMTP |
| 53 | DNS |
| 135 | MSRPC |
| 139 | NetBIOS |
| 445 | SMB |
| 3306 | MySQL |
| 5432 | PostgreSQL |
| 6379 | Redis |
| 11211 | Memcached |
| 27017 | MongoDB |

These restrictions apply to the **remote** port. Paid plans have no restrictions.

### All TCP Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--remote-port` | `-r` | Remote port (0 = auto) | 0 |
| `--allow-ip` | | Allowed IP/CIDR (repeatable) | All IPs |
| `--auto-close` | | Close on idle (1m–24h) | None |
| `--max-lifetime` | | Max lifetime (1m–7d) | None |

---

## UDP Tunnels

```bash
fxtunnel udp <port> [flags]
```

Syntax and flags are identical to TCP. Use for DNS, VoIP, game protocols, and other UDP services.

```bash
# DNS server
fxtunnel udp 53 --remote-port 5353

# Game server
fxtunnel udp 27015 --auto-close 2h
```

---

## Subdomain Management

### List Reserved Subdomains

```bash
fxtunnel domains list
```

### Reserve a Subdomain

```bash
fxtunnel domains add myapp
# → Reserved: myapp → https://myapp.fxtun.dev
```

A reserved subdomain is locked to your account — no one else can claim it.

### Check Availability

```bash
fxtunnel domains check myapp
```

### Release a Subdomain

```bash
fxtunnel domains remove myapp
```

### Naming Rules

- Length: 3–32 characters
- Allowed characters: `a-z`, `0-9`, `-` (hyphen)
- Must start and end with a letter or digit
- Case-insensitive (converted to lowercase)

### Reserved Names

The following subdomains are reserved by the system:

`api`, `www`, `admin`, `mail`, `smtp`, `imap`, `pop`, `ftp`, `ns1`–`ns4`, `autoconfig`, `autodiscover`, `_dmarc`, `status`, `metrics`, `grafana`

---

## Custom Domains

You can map your own domain to a tunnel.

### Add a Domain

```bash
fxtunnel domains custom add mydomain.com --target myapp
```

`--target` is the subdomain to route traffic to. It must be reserved beforehand.

### DNS Setup

**For a subdomain** (e.g., `tunnel.mydomain.com`):
```
CNAME  tunnel.mydomain.com  →  myapp.fxtun.dev
```

**For a root domain** (e.g., `mydomain.com`):
```
A  mydomain.com  →  fxTunnel server IP
```

### Verify DNS

```bash
fxtunnel domains custom verify mydomain.com
```

After successful verification, a TLS certificate is provisioned automatically.

### List and Remove

```bash
fxtunnel domains custom list
fxtunnel domains custom remove mydomain.com
```

---

## Configuration File

Use a configuration file to manage multiple tunnels.

### Create via Wizard

```bash
fxtunnel init
```

The interactive wizard creates `fxtunnel.yaml` in the current directory.

### Full Example

```yaml
# fxtunnel.yaml

server:
  address: "fxtun.dev:4443"       # Server address
  token: "sk_your_token"          # API token
  insecure: false                  # Skip TLS verification
  tls_verify: true                 # Verify server certificate
  compression: true                # Enable zstd compression

tunnels:
  - name: "webapp"                 # Tunnel name (for logs)
    type: "http"                   # Type: http, tcp, udp
    local_port: 3000               # Local port
    subdomain: "myapp"             # Subdomain (HTTP only)
    basic_auth: "user:Password123" # Basic Auth (HTTP only)
    allow_ips:                     # IP restriction
      - "10.0.0.0/8"
    auto_close: "1h"              # Idle timeout
    max_lifetime: "8h"            # Max lifetime

  - name: "ssh"
    type: "tcp"
    local_port: 22
    remote_port: 2222              # Remote port (TCP/UDP)

  - name: "dns"
    type: "udp"
    local_port: 53
    remote_port: 5353

reconnect:
  enabled: true                    # Auto-reconnect
  interval: 5s                     # Initial interval
  max_attempts: 0                  # 0 = infinite

inspect:
  enabled: true                    # Enable inspector
  addr: "127.0.0.1:4040"          # Inspector address
  max_entries: 1000                # Max buffered entries
  max_body_size: 262144            # Max body size (256 KB)

logging:
  level: "info"                    # debug, info, warn, error
  format: "console"                # console, json
```

### Running with Config

```bash
# Auto-find fxtunnel.yaml in current directory
fxtunnel

# Or specify path
fxtunnel --config path/to/config.yaml
```

### Settings Priority

Settings are applied in ascending priority:

1. **Defaults** (hardcoded)
2. **Config file** (fxtunnel.yaml)
3. **Environment variables** (FXTUNNEL_*)
4. **CLI flags** (--token, --server, ...)

### Config File Search Order

If `--config` is not specified, the client looks for:

1. `fxtunnel.yaml` in the current directory
2. `client.yaml` in the current directory
3. `client.yaml` in `./configs/`
4. `~/.fxtunnel/client.yaml`

### Environment Variables

All config options can be set via `FXTUNNEL_` prefixed environment variables:

```bash
export FXTUNNEL_SERVER_ADDRESS="fxtun.dev:4443"
export FXTUNNEL_SERVER_TOKEN="sk_your_token"
export FXTUNNEL_SERVER_COMPRESSION="false"
export FXTUNNEL_RECONNECT_INTERVAL="10s"
export FXTUNNEL_RECONNECT_MAX_ATTEMPTS="5"
export FXTUNNEL_INSPECT_ENABLED="true"
export FXTUNNEL_INSPECT_ADDR="127.0.0.1:4041"
export FXTUNNEL_LOGGING_LEVEL="debug"
export FXTUNNEL_LOGGING_FORMAT="json"
```

---

## Daemon Mode

Daemon mode runs tunnels in the background and lets you manage them separately.

### Start

```bash
# Background (detaches from terminal)
fxtunnel up

# Foreground (useful for systemd or debugging)
fxtunnel up --foreground
```

The daemon reads configuration from `fxtunnel.yaml` and opens all defined tunnels.

### Status

```bash
fxtunnel status
```

```
Daemon running (PID 12345)
Server: fxtun.dev:4443
  HTTP: https://myapp.fxtun.dev → localhost:3000
  TCP: fxtun.dev:2222 → localhost:22
  Uptime: 2h 15m
```

### Stop

```bash
fxtunnel down
```

---

## Traffic Inspector

The inspector captures and displays HTTP traffic flowing through your tunnels. Useful for debugging APIs, testing webhooks, and analyzing requests.

### Web Interface

By default, the inspector is available at:

```
http://127.0.0.1:4040
```

Open in your browser to view requests in real time.

If port 4040 is busy, the inspector tries ports 4041–4049.

### Disable Inspector

```bash
fxtunnel http 3000 --no-inspect
```

Or in config:

```yaml
inspect:
  enabled: false
```

### Inspector REST API

#### Status

```bash
curl http://127.0.0.1:4040/api/status
```

#### List Requests

```bash
curl http://127.0.0.1:4040/api/requests/http
```

Query parameters:
- `tunnel_id` — filter by tunnel ID
- `method` — HTTP method (GET, POST, ...)
- `status` — status code (200, 404, ...)
- `search` — full-text search
- `limit` — max results (default 100)
- `offset` — pagination offset

#### Request Details

```bash
curl http://127.0.0.1:4040/api/requests/http/{id}
```

Returns full details: request/response headers, body, duration.

#### Replay Request

```bash
curl -X POST http://127.0.0.1:4040/api/requests/http \
  -H "Content-Type: application/json" \
  -d '{"id": "request-uuid"}'
```

Re-sends the request through the tunnel. The result is captured as a new entry.

#### Live Stream (SSE)

```bash
curl http://127.0.0.1:4040/api/requests/http/stream
```

Server-Sent Events stream — receive new requests in real time:

```
data: {"id":"...","method":"GET","path":"/api/users","status":200,...}

data: {"id":"...","method":"POST","path":"/webhook","status":201,...}
```

#### Delete Entries

```bash
curl -X DELETE "http://127.0.0.1:4040/api/requests/http?tunnel_id=xxx"
```

#### List Active Tunnels

```bash
curl http://127.0.0.1:4040/api/tunnels
```

### Inspector Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `inspect.enabled` | Enable/disable | `true` |
| `inspect.addr` | Address and port | `127.0.0.1:4040` |
| `inspect.max_entries` | Max buffered entries | `1000` |
| `inspect.max_body_size` | Max request/response body size | `262144` (256 KB) |

---

## Warning Page

On first browser visit to an HTTP tunnel, a warning page (interstitial) is displayed. It warns that the content is provided by a third party and is not verified — a phishing protection measure.

### When It Appears

The page is shown if **all** conditions are met:
- Request method is `GET`
- Response Content-Type is `text/html`
- Using a subdomain (not a custom domain)
- User is not an admin

### When It Does NOT Appear

- **Custom domains** — no warning on custom domains
- **POST/PUT/DELETE and other methods** — only GET triggers the interstitial
- **API requests** — JSON, XML, and other non-HTML responses pass through
- **Cookie** — clicking "Continue" sets a `_fxt_consent_<subdomain>` cookie for 12 hours
- **`X-FxTunnel-Skip-Warning` header** — bypasses the warning (see below)
- **Admin accounts** — admins never see warnings

### Skip via Header

Add the `X-FxTunnel-Skip-Warning` header with any non-empty value:

```bash
curl -H "X-FxTunnel-Skip-Warning: true" https://myapp.fxtun.dev
```

This is useful for:
- Automated scripts and CI/CD pipelines
- Webhook testing (Stripe, GitHub, Telegram, etc.)
- Monitoring and health checks
- Any programmatic access where the interstitial gets in the way

### Skip via Cookie

When visiting in a browser, click "Continue to site" — the cookie lasts 12 hours.

---

## HTTP Headers

### Headers Set by the Server

| Header | Description |
|--------|-------------|
| `X-Trace-Id` | Unique request ID (16 hex characters) for debugging |
| `X-Forwarded-For` | Client IP address |
| `X-Forwarded-Proto` | Protocol (`http`) |
| `X-Forwarded-Host` | Original Host header |

The server **strips** incoming `X-Forwarded-*` headers and sets its own — this prevents spoofing.

### Headers for Clients

| Header | Description |
|--------|-------------|
| `X-FxTunnel-Skip-Warning: true` | Skip the warning page |

### Internal Headers

| Header | Description |
|--------|-------------|
| `X-FxTunnel-Hop` | Loop prevention for edge routing (set automatically) |

---

## Reconnection

The client automatically reconnects when the connection is lost.

### Behavior

1. Connection drops
2. Wait for `interval` (default: 5 seconds)
3. Attempt reconnection
4. On failure — exponential backoff: `interval × 1.5`, max 2 minutes
5. Retry up to `max_attempts` (0 = infinite)

### Configuration

```yaml
reconnect:
  enabled: true       # Enable auto-reconnect
  interval: 5s        # Initial interval
  max_attempts: 0     # 0 = infinite
```

### After Reconnection

- Tunnels are recreated with the same configuration
- Subdomains and ports are preserved (if reserved)
- Traffic statistics reset
- `auto-close` and `max-lifetime` timers restart

---

## Security Presets

Presets are ready-made configuration bundles for common scenarios.

### List Available Presets

```bash
fxtunnel presets
```

### openclaw

Quick secure sharing with auto-generated Basic Auth:

```bash
fxtunnel http 3000 --preset openclaw
```

```
Preset 'openclaw' credentials:
  Username: admin
  Password: x7kP9qM2lL5nR8sT

Tunnel established!
HTTP: https://ab1cd2ef.fxtun.dev
```

A new random password (16 characters) is generated on each run. If you explicitly set `--auth`, your values override the preset.

---

## Updating the Client

```bash
fxtunnel update
```

The client checks for a newer version on the server and updates automatically.

---

## Limits

### Plan Limits

| Feature | Free | Base ($5) | Pro ($10) | Business ($15) |
|---------|------|-----------|-----------|----------------|
| Tunnels | 3 | 5 | 15 | 50 |
| Reserved subdomains | 0 | 5 | 15 | 50 |
| Custom domains | 0 | 1 | 5 | 50 |
| API tokens | 1 | 5 | 10 | 50 |
| Inspector | No | Unlimited | Unlimited | Unlimited |

### Rate Limiting

| Protocol | Default Limit |
|----------|---------------|
| HTTP | 3,600 requests/min per tunnel |
| TCP | 1,800 connections/min per tunnel |
| UDP | 10,000 packets/sec per tunnel |

An additional per-IP limit applies — 1/10 of the tunnel limit.

When exceeded, HTTP requests receive a `429 Too Many Requests` response.

### Inspector Body Size

By default, the inspector captures the first 256 KB of request/response bodies. Configurable via `inspect.max_body_size`.

---

## Global CLI Flags

These flags are available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Config file path | Auto-detect |
| `--server` | `-s` | Server address (host:port) | fxtun.dev:4443 |
| `--token` | `-t` | API token | From keyring |
| `--log-level` | | Log level | warn |
| `--log-format` | | Log format (console/json) | console |
| `--inspect-addr` | | Inspector address | 127.0.0.1:4040 |
| `--no-inspect` | | Disable inspector | false |

---

## FAQ

### How do I get a permanent URL?

Reserve a subdomain:

```bash
fxtunnel domains add myapp
fxtunnel http 3000 --domain myapp
# Always https://myapp.fxtun.dev
```

Without a reservation, another user could claim the subdomain.

### How do I share access with a teammate?

Option 1 — Basic Auth:
```bash
fxtunnel http 3000 --domain demo --auth team:SharedPass123
# Share the URL and password
```

Option 2 — IP allowlist:
```bash
fxtunnel http 3000 --allow-ip 203.0.113.10
```

Option 3 — Preset:
```bash
fxtunnel http 3000 --preset openclaw
# Share the generated credentials
```

### How do I use it with Docker?

Point to the host address instead of localhost:

```bash
# Docker container listening on port 8080
fxtunnel http 8080
```

If the container is on a different network interface:

```yaml
tunnels:
  - name: "docker-app"
    type: "http"
    local_addr: "172.17.0.2"    # Container IP
    local_port: 8080
    subdomain: "myapp"
```

### How do I debug webhooks?

1. Start a tunnel with the inspector:
```bash
fxtunnel http 3000 --domain webhooks
```

2. Set the URL `https://webhooks.fxtun.dev/webhook` in the service settings (Stripe, GitHub, Telegram, etc.)

3. Open the inspector at `http://127.0.0.1:4040` — all incoming webhooks appear in real time

4. Use Replay to re-send a request:
```bash
curl -X POST http://127.0.0.1:4040/api/requests/http \
  -H "Content-Type: application/json" \
  -d '{"id": "request-id-from-inspector"}'
```

### How do I proxy WebSocket?

WebSocket connections are proxied automatically — no extra configuration needed:

```bash
fxtunnel http 3000 --domain ws-app
# ws://ws-app.fxtun.dev and wss://ws-app.fxtun.dev work out of the box
```

### How do I run it on system startup?

Create a systemd service:

```ini
# /etc/systemd/system/fxtunnel.service
[Unit]
Description=fxTunnel Client
After=network.target

[Service]
Type=simple
ExecStart=/home/user/.local/bin/fxtunnel up --foreground
Restart=always
RestartSec=5
User=user
WorkingDirectory=/home/user/project

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now fxtunnel
```

### Inspector not available?

Make sure your plan supports the inspector (available on paid plans). On the free plan, the inspector is disabled server-side.

If port 4040 is busy, check the startup output — the client will show the actual port (4041–4049).

### Error "port is blocked for security reasons"?

Certain TCP ports are blocked on the free plan (SSH, databases, etc.). Upgrade to a paid plan or use a different remote port.
