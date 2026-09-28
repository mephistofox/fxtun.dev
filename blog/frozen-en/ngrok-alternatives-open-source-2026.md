---
title: "Open-Source ngrok Alternatives 2026 — Ranked"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Best open-source ngrok alternatives in 2026: fxTunnel, Bore, frp, Rathole, LocalTunnel, Zrok. Compared by protocols, features, and self-hosting."
tags: [ngrok, open-source, alternative, self-hosted, tunneling, fxTunnel, comparison, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "What is the best open-source alternative to ngrok in 2026?"
    a: "fxTunnel is the best open-source ngrok alternative in 2026. It supports HTTP, TCP, and UDP, offers a free SaaS tier with no limits, a traffic inspector with replay from $5/mo, custom domains, and full self-hosting via a single Go binary. No other open-source tunnel matches this combination of features."
  - q: "Can I self-host an ngrok alternative on my own server?"
    a: "Yes. fxTunnel, Bore, frp, Rathole, LocalTunnel, and zrok all support self-hosting. fxTunnel is the easiest to deploy — a single Go binary with HTTP, TCP, UDP, TLS, and traffic inspector built in. frp and Rathole require manual TOML/INI configuration. Bore is TCP-only. zrok requires deploying OpenZiti infrastructure."
  - q: "Which open-source tunnels support TCP and UDP?"
    a: "fxTunnel supports HTTP, TCP, and UDP. frp supports HTTP, TCP, UDP, and additional protocols (STCP, XTCP). Rathole supports TCP and UDP but lacks HTTP-specific features. Bore is TCP-only. LocalTunnel and zrok support HTTP and TCP but not UDP."
  - q: "Is frp a good ngrok alternative?"
    a: "frp is a powerful open-source reverse proxy with support for HTTP, TCP, UDP, and advanced features like load balancing and dashboards. However, it requires manual TOML configuration for both client and server, has no SaaS option, and does not offer a traffic inspector with replay. fxTunnel is easier to set up and provides SaaS + self-hosting flexibility."
  - q: "What is the difference between self-hosted and SaaS tunnels?"
    a: "A self-hosted tunnel runs entirely on your own infrastructure — you control the server, data, and uptime. A SaaS tunnel provides a managed public server so you can start tunneling immediately without deploying anything. fxTunnel is the only open-source tool that offers both: a free SaaS tier and a self-hostable server binary."
---

## Why Open-Source ngrok Alternatives Matter in 2026

Why would you pick an open-source tunnel over ngrok? Three reasons that proprietary tools cannot match: full code auditability, self-hosting on your own infrastructure, and freedom from vendor lock-in. In 2026, several open-source tunnels have matured to the point where they match or exceed ngrok's functionality -- while remaining free and transparent.

ngrok itself is fully closed-source. You cannot audit what happens to your traffic, you cannot deploy the server component on your own VPS, and you are entirely dependent on ngrok Inc.'s pricing and infrastructure decisions. The free tier is limited to a single agent with rate limits, and scaling to a team of five costs $40/mo or more. For many developers and organizations, this is a non-starter.

If you need an [open-source tunnel](/blog/what-is-tunneling/) that you can self-host, audit, and extend -- this article ranks the top six options in 2026. We compare fxTunnel, Bore, frp, Rathole, LocalTunnel, and zrok across protocols, features, ease of setup, security, and self-hosting. Our [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) article covers proprietary tools as well.

## The 6 Best Open-Source ngrok Alternatives — Overview

Before diving into comparison tables and code examples, here is what each tool is, what language it is written in, and where it shines.

### 1. fxTunnel — Full-Featured Open-Source Tunnel in Go

fxTunnel is an [open-source tunneling tool](/blog/fxtunnel-architecture/) written in Go. It supports HTTP, TCP, and UDP, offers a free SaaS tier with no traffic caps, and can be self-hosted as a single binary. The traffic inspector with replay and custom domains are available on paid plans.

fxTunnel is the only open-source tunnel that combines a managed SaaS with a fully self-hostable server. You can start with the free SaaS tier and move to self-hosting when your needs change — without switching tools.

```bash
# Install fxTunnel (no dependencies)
curl -fsSL https://fxtun.ru/install.sh | bash

# Or via Go
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest

# HTTP tunnel
fxtunnel http 8080
# → https://abc123.fxtun.dev → localhost:8080

# TCP tunnel (PostgreSQL, SSH)
fxtunnel tcp 5432

# UDP tunnel (game server, VoIP, DNS)
fxtunnel udp 27015

# Custom domain (from $5/mo)
fxtunnel http 8080 --domain app.example.com

# Traffic inspector (from $5/mo)
fxtunnel http 8080 --inspect
```

### 2. Bore — Minimal TCP Tunnel in Rust

Bore is a lightweight TCP tunneling utility written in Rust by Erik Zhang. It does one thing — forwards TCP ports — and does it with a minimal binary, zero dependencies, and a single-command launch. Bore is ideal when you need raw TCP forwarding and nothing else.

However, Bore does not support HTTP as a distinct protocol type, has no HTTPS termination, no UDP, no traffic inspector, and no SaaS. Self-hosting is the only option, and you need to configure a reverse proxy (nginx, Caddy) for HTTPS yourself.

```bash
# Install Bore (requires Rust toolchain)
cargo install bore-cli

# Start the server (on your VPS)
bore server --min-port 1024 --max-port 65535

# Forward local port 8080 via your server
bore local 8080 --to your-server.com

# Forward local port 5432 (PostgreSQL) via the public server
bore local 5432 --to bore.pub
```

### 3. frp — Feature-Rich Reverse Proxy in Go

frp (Fast Reverse Proxy) is a mature open-source reverse proxy written in Go. It supports HTTP, HTTPS, TCP, UDP, and additional protocols like STCP (secret TCP) and XTCP (peer-to-peer TCP). frp has a web dashboard, load balancing, and a plugin system.

frp is one of the most powerful open-source tunneling tools available. However, it requires manual TOML configuration for both the client (`frpc.toml`) and the server (`frps.toml`), has no SaaS option, and does not provide a traffic inspector with replay. The learning curve is steeper than fxTunnel or Bore.

```bash
# Download frp (Linux amd64)
wget https://github.com/fatedier/frp/releases/latest/download/frp_0.61.1_linux_amd64.tar.gz
tar -xzf frp_0.61.1_linux_amd64.tar.gz
cd frp_0.61.1_linux_amd64

# Server configuration (frps.toml)
cat > frps.toml <<EOF
bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
webServer.addr = "0.0.0.0"
webServer.port = 7500
EOF

# Start the server
./frps -c frps.toml

# Client configuration (frpc.toml)
cat > frpc.toml <<EOF
serverAddr = "your-server.com"
serverPort = 7000

[[proxies]]
name = "web"
type = "http"
localPort = 8080
customDomains = ["app.example.com"]
EOF

# Start the client
./frpc -c frpc.toml
```

### 4. Rathole — Lightweight Tunnel in Rust with Noise Protocol

Rathole is a high-performance tunneling tool written in Rust. It uses the Noise Protocol Framework for encryption (the same cryptographic protocol used by WireGuard). Rathole supports TCP and UDP and is designed for efficiency — it has a smaller memory footprint and lower latency than frp in many benchmarks.

Rathole excels at raw performance and security but lacks HTTP-specific features (custom domains, host-based routing, TLS termination), has no traffic inspector, no SaaS, and requires TOML configuration for both client and server.

```bash
# Download Rathole (Linux amd64)
wget https://github.com/rapiz1/rathole/releases/latest/download/rathole-x86_64-unknown-linux-gnu.zip
unzip rathole-x86_64-unknown-linux-gnu.zip

# Server configuration (server.toml)
cat > server.toml <<EOF
[server]
bind_addr = "0.0.0.0:2333"

[server.services.web]
type = "tcp"
bind_addr = "0.0.0.0:8080"
token = "your-secret-token"

[server.services.dns]
type = "udp"
bind_addr = "0.0.0.0:5353"
token = "your-secret-token"
EOF

# Start the server
./rathole server.toml

# Client configuration (client.toml)
cat > client.toml <<EOF
[client]
remote_addr = "your-server.com:2333"

[client.services.web]
type = "tcp"
local_addr = "127.0.0.1:8080"
token = "your-secret-token"

[client.services.dns]
type = "udp"
local_addr = "127.0.0.1:5353"
token = "your-secret-token"
EOF

# Start the client
./rathole client.toml
```

### 5. LocalTunnel — HTTP Tunnel in Node.js

LocalTunnel is an open-source tool for creating HTTP tunnels, written in Node.js. It was once a popular [free ngrok alternative](/blog/free-ngrok-alternatives-2026/), but the project is effectively unmaintained since 2020. The public server (`localtunnel.me`) is frequently overloaded, and the codebase has accumulated unresolved issues.

LocalTunnel remains useful for quick throwaway demos when you need an HTTP URL in seconds and don't care about reliability or additional protocols.

```bash
# Install LocalTunnel (requires Node.js)
npm install -g localtunnel

# Start an HTTP tunnel
lt --port 8080

# Request a specific subdomain
lt --port 8080 --subdomain my-app
# → https://my-app.loca.lt
```

### 6. zrok — Peer-to-Peer Tunneling on OpenZiti

zrok is an open-source tunneling solution built on the OpenZiti Zero Trust networking platform. It supports public and private sharing, HTTP and TCP, and peer-to-peer connections that bypass central servers entirely.

zrok is the most architecturally ambitious tool on this list, but also the most complex. Self-hosting requires deploying OpenZiti infrastructure, the learning curve is steep, and UDP is not supported.

```bash
# Install zrok (Linux)
curl -sSf https://get.openziti.io/install.bash | bash

# Register and activate
zrok invite
zrok enable

# Public HTTP share
zrok share public localhost:8080

# Private share (peer-to-peer, no public URL)
zrok share private localhost:8080
```

## Comparison Table: 6 Open-Source ngrok Alternatives

Here is how the six tools stack up side by side.

| Criterion | fxTunnel | Bore | frp | Rathole | LocalTunnel | zrok |
|---|---|---|---|---|---|---|
| **Language** | Go | Rust | Go | Rust | Node.js | Go |
| **Protocols** | HTTP, TCP, UDP | TCP only | HTTP, TCP, UDP, STCP, XTCP | TCP, UDP | HTTP only | HTTP, TCP |
| **HTTPS / TLS** | Yes, out of the box | No (needs reverse proxy) | Yes (manual config) | Noise Protocol | Public server only | Yes |
| **Traffic inspector** | Yes (from $5/mo) | No | Dashboard (basic) | No | No | No |
| **Request replay** | Yes (from $5/mo) | No | No | No | No | No |
| **SaaS (managed server)** | Yes (free, no limits) | No | No | No | Yes (unreliable) | Yes (limited free) |
| **Self-hosting** | Yes (single binary) | Yes (single binary) | Yes (binary + TOML) | Yes (binary + TOML) | Yes (Node.js) | Yes (OpenZiti infra) |
| **Custom domains** | Yes (any DNS, from $5/mo) | No | Yes (manual config) | No | Subdomains only | Yes (self-host) |
| **CLI simplicity** | 1 command | 1 command | Config files required | Config files required | 1 command | Multiple steps |
| **Active development** | Yes | Minimal | Yes | Moderate | No (since 2020) | Yes |
| **Configuration** | Zero-config CLI | Minimal flags | TOML (client + server) | TOML (client + server) | Minimal flags | YAML + OpenZiti |
| **GitHub stars (2026)** | Growing | ~9k | ~88k | ~10k | ~16k | ~3k |

## Self-Hosting Comparison: What It Takes to Deploy Each Tool

Self-hosting is one of the primary reasons developers choose open-source tunnels. But the effort required varies dramatically between tools. Below is a detailed breakdown of what it takes to deploy each tool on your own server.

### fxTunnel Self-Hosting

fxTunnel's server is a single Go binary. Download it, run it, and you have a fully functional tunnel server with HTTP, TCP, UDP, and TLS support. No config files required for basic operation.

```bash
# Download and run the fxTunnel server
curl -fsSL https://fxtun.dev/install-server.sh | bash
fxtunnel-server --domain tunnel.example.com

# Client connects to your server
fxtunnel http 8080 --server tunnel.example.com
```

**Effort:** 5 minutes. Single binary, built-in TLS, no config files for basic setup.

### Bore Self-Hosting

Bore's server is equally simple to run — a single Rust binary with minimal flags. However, you only get TCP forwarding. For HTTPS, you need to deploy nginx or Caddy in front of Bore.

```bash
# Run the Bore server
bore server --min-port 1024 --max-port 65535

# Add nginx for HTTPS (additional step)
# → configure nginx reverse proxy + Let's Encrypt
```

**Effort:** 5 minutes for TCP. 30+ minutes if you need HTTPS (reverse proxy + certificates).

### frp Self-Hosting

frp requires TOML configuration for both the server and every client. The server supports HTTP vhosts, HTTPS, TCP, and UDP, but each service must be explicitly configured.

```bash
# Server: create frps.toml, configure ports, vhosts, dashboard
# Client: create frpc.toml for each service
# TLS: configure TLS certificates in frps.toml

./frps -c frps.toml
```

**Effort:** 15-30 minutes. Configuration-heavy but well-documented.

### Rathole Self-Hosting

Rathole requires TOML configuration similar to frp. Each service needs explicit client and server entries with matching tokens. Noise Protocol encryption is built in, so no external TLS setup is needed for the tunnel itself.

```bash
# Server: create server.toml with services and tokens
# Client: create client.toml with matching services and tokens

./rathole server.toml
```

**Effort:** 15-20 minutes. Simpler than frp but no HTTP-specific features.

### LocalTunnel Self-Hosting

LocalTunnel's server requires Node.js and npm. The project is unmaintained, and the self-hosted server documentation is outdated.

```bash
# Install and run LocalTunnel server
git clone https://github.com/localtunnel/server.git
cd server
npm install
bin/server --port 1234 --domain tunnel.example.com
```

**Effort:** 10-15 minutes, but you inherit an unmaintained Node.js codebase.

### zrok Self-Hosting

zrok requires deploying the full OpenZiti infrastructure first — a controller, edge routers, and the zrok front-end. This is the most complex self-hosting setup on this list.

```bash
# Step 1: Deploy OpenZiti (controller + edge router)
# Step 2: Configure zrok front-end
# Step 3: Enable and invite users

# This process takes 30-60+ minutes and requires
# understanding OpenZiti's architecture
```

**Effort:** 30-60+ minutes. Requires understanding OpenZiti's network model.

### Self-Hosting Summary

| Tool | Time to deploy | Dependencies | TLS built-in | Config files |
|---|---|---|---|---|
| **fxTunnel** | 5 min | None (Go binary) | Yes | None (zero-config) |
| **Bore** | 5 min (TCP) / 30+ min (HTTPS) | None (Rust binary) | No | None |
| **frp** | 15-30 min | None (Go binary) | Yes (manual) | TOML (server + client) |
| **Rathole** | 15-20 min | None (Rust binary) | Noise Protocol | TOML (server + client) |
| **LocalTunnel** | 10-15 min | Node.js, npm | No | Minimal |
| **zrok** | 30-60+ min | OpenZiti infra | Yes | YAML + OpenZiti config |

## Feature Deep-Dive

### Protocol Support

Protocol support is the most important differentiator between open-source tunnels. If you need to [expose a database, SSH session, or game server](/blog/tcp-udp-tunneling-explained/), HTTP-only tools are not enough.

- **fxTunnel** — HTTP, TCP, UDP. All three protocols with a single CLI command.
- **frp** — HTTP, TCP, UDP, plus STCP (secret TCP) and XTCP (peer-to-peer TCP). The most protocols of any tool, but each requires config file entries.
- **Rathole** — TCP and UDP. No HTTP-specific features (no vhosts, no custom domains via the tool itself).
- **Bore** — TCP only. No HTTP features, no UDP.
- **LocalTunnel** — HTTP only. No TCP, no UDP.
- **zrok** — HTTP and TCP. No UDP.

### Traffic Inspector and Replay

A traffic inspector lets you view, search, and debug every request passing through your tunnel. Replay lets you re-send a request with one click — essential for [webhook testing](/blog/webhook-testing-with-tunnel/) and debugging.

- **fxTunnel** -- full traffic inspector with replay from $5/mo; request history is not capped.
- **frp** — a basic web dashboard showing active connections and traffic stats, but no request-level inspection or replay.
- **All others** — no inspector, no replay.

If you are testing webhooks from Stripe, GitHub, or Telegram, fxTunnel's inspector saves hours of debugging compared to manually tailing logs.

### Custom Domains

Custom domains are critical for [production use behind NAT](/blog/expose-localhost-to-internet/) and for demo environments that need stable, branded URLs.

- **fxTunnel** -- custom domains via any DNS provider, on paid plans.
- **frp** — custom domains via manual configuration in `frpc.toml`. Requires DNS setup and TLS certificate management.
- **zrok** — custom domains in self-hosted mode.
- **LocalTunnel** — subdomain requests only (`--subdomain`), not custom domains.
- **Bore, Rathole** — no custom domain support.

### Security

Security is one of the primary reasons to choose open-source, but the level of built-in security varies between tools. Our [TLS 1.3 tunnel security](/blog/tls-13-tunnel-security/) article digs deeper into the encryption side.

| Aspect | fxTunnel | Bore | frp | Rathole | LocalTunnel | zrok |
|---|---|---|---|---|---|---|
| **Encryption** | TLS 1.3 | None (raw TCP) | TLS (manual) | Noise Protocol | Public server TLS | TLS + OpenZiti |
| **Auth** | Token | Shared secret | Token/OIDC | Shared token | None | OpenZiti identity |
| **Code audit** | Go (easy) | Rust (easy) | Go (easy) | Rust (easy) | Node.js (easy) | Go + OpenZiti |
| **Active patches** | Yes | Rare | Yes | Moderate | No | Yes |

Rathole stands out with its Noise Protocol encryption — the same cryptographic framework used by WireGuard. However, fxTunnel's TLS 1.3 with automatic certificate management is more practical for most developers.

## Decision Matrix: Which Open-Source Tunnel to Choose

Your choice depends on your specific requirements. Below are concrete recommendations for common scenarios.

### Choose fxTunnel if you need:

- **A universal open-source tunnel** — HTTP + TCP + UDP, SaaS + self-hosting, inspector + replay.
- **The fastest setup** — one command, 30 seconds, no config files.
- **[Webhook testing](/blog/webhook-testing-with-tunnel/)** — free tier, inspector with replay for debugging.
- **Team use** — free SaaS with no per-agent limits.
- **Production behind NAT** — custom domains, TLS, stable infrastructure.
- **Self-hosting with minimal effort** — single Go binary, zero config.
- **[Docker container exposure](/blog/docker-tunnel-expose-containers/)** — all protocols, one command.

### Choose frp if you need:

- **Maximum configurability** — TOML config, plugins, load balancing.
- **Advanced protocols** — STCP, XTCP for internal networking.
- **A mature project** — 88k+ GitHub stars, large community, battle-tested.
- **You are comfortable with config files** — frp requires manual setup but offers fine-grained control.

### Choose Rathole if you need:

- **Raw performance** — minimal memory, low latency, Rust efficiency.
- **Noise Protocol encryption** — WireGuard-level cryptography.
- **TCP + UDP forwarding** — without HTTP-specific overhead.
- **A lightweight alternative to frp** — similar concept, smaller footprint.

### Choose Bore if you need:

- **Minimal TCP forwarding** — one binary, one command.
- **Rust ecosystem** — you already use Rust and want consistency.
- **No HTTP features needed** — pure TCP is enough.

### Choose LocalTunnel if you need:

- **A throwaway HTTP tunnel** — quick demo, 30-second lifespan.
- **Node.js ecosystem** — you already have npm and want zero friction.
- **No reliability requirements** — you can tolerate drops and timeouts.

### Choose zrok if you need:

- **Zero Trust networking** — OpenZiti's identity-based security model.
- **Private peer-to-peer sharing** — no public URL, direct tunnels between peers.
- **You are already using OpenZiti** — zrok integrates natively.

## Open-Source Tunnel Pricing in 2026

One of the biggest advantages of open-source tunnels is cost. Below is a pricing comparison for common scenarios.

| Scenario | fxTunnel | Bore | frp | Rathole | LocalTunnel | zrok |
|---|---|---|---|---|---|---|
| **Solo developer** | $0 | $3-5/mo (VPS) | $3-5/mo (VPS) | $3-5/mo (VPS) | $0 (unreliable) | $0 (limited) |
| **Team of 5** | $0 | $3-5/mo (VPS) | $3-5/mo (VPS) | $3-5/mo (VPS) | $0 (unreliable) | $3-5/mo (VPS) |
| **Custom domains** | From $5/mo | N/A | VPS cost only | N/A | N/A | VPS cost only |
| **Inspector + replay** | From $5/mo | N/A | N/A | N/A | N/A | N/A |
| **10+ tunnels** | From $10/mo | VPS cost only | VPS cost only | VPS cost only | $0 (unreliable) | VPS cost only |
| **Full self-hosting** | VPS cost only | VPS cost only | VPS cost only | VPS cost only | VPS cost only | VPS cost only |

fxTunnel's SaaS eliminates the need for a VPS entirely for most use cases. The free tier covers tunnels, traffic, and connections without caps. Paid plans add custom domains, the traffic inspector with replay, and priority support.

Self-hosting any tool requires a VPS ($3-5/mo from providers like Hetzner or DigitalOcean), plus the time cost of maintenance, updates, and TLS certificate management. fxTunnel's SaaS handles all of this for you.

## Migrating from ngrok to an Open-Source Alternative

If you are currently using ngrok and want to switch to an open-source tool, fxTunnel offers the smoothest migration path. The CLI commands are nearly identical, and you gain TCP/UDP support, no rate limits, and full source code access.

```bash
# Step 1: Install fxTunnel
curl -fsSL https://fxtun.ru/install.sh | bash

# Step 2: Replace ngrok commands
# Before: ngrok http 8080
# After:
fxtunnel http 8080

# Step 3: Use TCP and UDP (not available in ngrok)
fxtunnel tcp 5432
fxtunnel udp 27015

# Step 4: Custom domain (from $5/mo)
fxtunnel http 8080 --domain app.example.com

# Step 5: Traffic inspector (from $5/mo)
fxtunnel http 8080 --inspect
# → http://localhost:7070
```

The entire migration takes under two minutes. More scenarios are covered in the [free ngrok alternatives 2026](/blog/free-ngrok-alternatives-2026/) guide.

## When to Self-Host vs When to Use SaaS

Every team faces this question when choosing a tunnel. Here is a framework for deciding.

**Choose SaaS (fxTunnel free tier) when:**

- You want to start in 30 seconds with no infrastructure setup.
- Your team does not have a dedicated DevOps engineer.
- You need stable uptime without managing servers.
- You are evaluating tunneling and want to try before committing to self-hosting.

**Choose self-hosting when:**

- Your organization requires data to stay on-premise (compliance, HIPAA, SOC 2).
- You need full control over logs, traffic, and server configuration.
- You already have a VPS or bare-metal server available.
- You want to customize the tunnel server behavior beyond what SaaS offers.

The client commands are identical for SaaS and self-hosting, so moving between modes requires no changes to your scripts or CI/CD pipelines.

## Try fxTunnel in 60 Seconds

```bash
# Install
curl -fsSL https://fxtun.ru/install.sh | bash

# HTTP tunnel
fxtunnel http 8080

# TCP tunnel
fxtunnel tcp 5432

# UDP tunnel
fxtunnel udp 27015
```

The free tier has no traffic caps. Custom domains and the traffic inspector start from $5/mo. For a comparison with [proprietary solutions](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) (ngrok, Cloudflare Tunnel), see our dedicated article, or check the [Bore, fxTunnel, and LocalTunnel](/blog/bore-vs-fxtunnel-vs-localtunnel/) head-to-head review.

## FAQ

### What is the best open-source alternative to ngrok in 2026?

Based on our comparison, fxTunnel comes out on top. It covers HTTP, TCP, and UDP, has a free SaaS tier, a traffic inspector with replay on paid plans, custom domains, and a single Go binary you can self-host. No other open-source tunnel offers the same breadth.

### Can I self-host an ngrok alternative on my own server?

All six tools in this comparison support self-hosting. fxTunnel has the lowest setup friction -- one Go binary that bundles HTTP, TCP, UDP, TLS, and the traffic inspector. frp and Rathole need TOML config files for both sides. Bore covers TCP only. zrok pulls in the full OpenZiti infrastructure.

### Which open-source tunnels support TCP and UDP?

Three of the six: fxTunnel (HTTP, TCP, UDP), frp (HTTP, TCP, UDP, plus STCP and XTCP), and Rathole (TCP and UDP, but no HTTP-specific features). Bore is TCP-only. LocalTunnel and zrok handle HTTP and TCP but not UDP.

### Is frp a good ngrok alternative?

It is a solid choice if you value configurability. frp supports HTTP, TCP, UDP, load balancing, and dashboards. The trade-off is manual TOML configuration on both client and server, no SaaS option, and no request-level traffic inspector. fxTunnel is simpler to get running and offers both SaaS and self-hosting.

### What is the difference between self-hosted and SaaS tunnels?

With self-hosted, you run the server on your own infrastructure and control the data, uptime, and configuration. SaaS gives you a managed public server so you can start tunneling immediately with zero setup. fxTunnel is the only open-source tool that supports both modes -- you can start on the free SaaS and move to self-hosting later without changing your workflow.
