---
title: "Bore vs fxTunnel vs LocalTunnel — 2026 Compared"
date: 2026-03-03T14:00:00+03:00
draft: false
description: "Bore vs fxTunnel vs LocalTunnel: protocols, HTTPS, inspector, self-hosting, and CLI compared. Find the best open-source tunnel for you."
tags: [bore, localtunnel, fxTunnel, open-source, comparison, tunneling, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Which open-source tunnel is best — Bore, LocalTunnel, or fxTunnel?"
    a: "fxTunnel is the most feature-rich open-source tunnel in 2026. Unlike Bore (TCP only, no HTTPS, no inspector) and LocalTunnel (HTTP only, unreliable public server), fxTunnel supports HTTP, TCP, and UDP, offers a traffic inspector with replay, a SaaS free tier, and self-hosting."
  - q: "Can Bore handle HTTPS tunnels?"
    a: "No. Bore only supports raw TCP tunnels without TLS termination. For HTTPS, you need to configure a reverse proxy (e.g., nginx or Caddy) in front of Bore yourself. fxTunnel supports HTTPS out of the box with no additional setup."
  - q: "Why is LocalTunnel unreliable?"
    a: "LocalTunnel's public server (localtunnel.me) is frequently overloaded and unmaintained — the project has not received significant updates since 2020. You can self-host a LocalTunnel server, but it requires a Node.js infrastructure. fxTunnel offers a stable SaaS and a Go-based self-hosting option."
  - q: "Which open-source tunnel supports UDP?"
    a: "Of the three tools compared here, only fxTunnel supports UDP tunnels. Bore handles TCP only, and LocalTunnel handles HTTP only. If you need UDP for game servers, VoIP, or IoT, fxTunnel is the only option."
  - q: "Can I self-host Bore, LocalTunnel, or fxTunnel?"
    a: "Yes, all three tools support self-hosting. Bore is a minimal Rust server for TCP. LocalTunnel is a Node.js server but without active maintenance. fxTunnel is a Go server with full functionality: HTTP, TCP, UDP, traffic inspector, and TLS. fxTunnel is the best self-hosting choice thanks to a single binary and active development."
---

## Why Open-Source Tunnels Matter for Developers

Why reach for an open-source tunnel instead of a proprietary one? Three reasons: you can audit the code yourself, host it on your own infrastructure, and walk away without vendor lock-in.

When a [tunneling tool](/blog/what-is-tunneling/) is open source, you can verify that your traffic is not being intercepted, logged, or shared with third parties. For teams handling sensitive data — medical, financial, or personal — this is not just a benefit but a requirement. Unlike [proprietary tools such as ngrok](/blog/ngrok-vs-cloudflare-vs-fxtunnel/), open-source tools let you inspect every line of code.

Self-hosting also eliminates dependency on someone else's infrastructure. If the tool's public server goes down or changes its terms, you simply keep running on your own.

## Overview of Each Tool

Before diving into comparison tables, let's examine each tool individually — its purpose, language, capabilities, and limitations.

### Bore — Minimal TCP Tunnel in Rust

Bore is a TCP tunneling utility written in Rust. Created by Erik Zhang, the project positions itself as a minimal ngrok alternative: a single binary, zero dependencies, and a one-command launch.

**What Bore does well:**

- Minimal binary size and zero runtime dependencies.
- Fast compilation and installation via `cargo`.
- Simple self-hosting: the server component starts with a single command.
- Great for forwarding TCP ports (SSH, databases) without added complexity.

**Bore's limitations:**

- **TCP only** — no HTTP tunnels as a distinct type, no UDP.
- **No HTTPS termination** — Bore does not handle TLS. For HTTPS, you need an external reverse proxy.
- **No traffic inspector** — no web UI for viewing or replaying requests.
- **No SaaS** — self-hosting only. You need your own server with a public IP.
- **Minimal documentation** — a GitHub README and nothing more.

```bash
# Install Bore
cargo install bore-cli

# Start the server (on your VPS)
bore server --min-port 1024 --max-port 65535

# Start the client — forward port 8080
bore local 8080 --to your-server.com
```

### LocalTunnel — HTTP Tunnel in Node.js

LocalTunnel is an open-source tool for creating HTTP tunnels, written in Node.js. It was once a popular [free ngrok alternative](/blog/free-ngrok-alternatives-2026/), but today the project is effectively unmaintained.

**What LocalTunnel does well:**

- Simple installation via `npm`.
- A single command to start an HTTP tunnel.
- Ability to request a specific subdomain (`--subdomain`).
- A public server is available — no VPS needed to get started.

**LocalTunnel's limitations:**

- **HTTP only** — no TCP or UDP support.
- **Unreliable public server** — `localtunnel.me` is frequently unavailable, slow, or returns errors. Many developers report timeouts and dropped connections.
- **Not actively maintained** — the last significant updates date back to 2020. Issues and PRs on GitHub accumulate without response.
- **No traffic inspector** — no way to view or replay requests.
- **No client-side HTTPS termination** — HTTPS is provided only on the public server side.
- **Node.js dependency** — requires Node.js and npm to be installed.

```bash
# Install LocalTunnel
npm install -g localtunnel

# Start an HTTP tunnel
lt --port 8080

# Start with a custom subdomain
lt --port 8080 --subdomain my-app
```

### fxTunnel — Full-Featured Open-Source Tunnel in Go

fxTunnel is an [open-source tunneling tool](/blog/fxtunnel-architecture/) written in Go. It supports HTTP, TCP, and UDP, offers both a SaaS free tier and a self-hosting option, and includes a traffic inspector with replay on paid plans.

**What fxTunnel does well:**

- **HTTP + TCP + UDP** — the only tool of the three that supports all protocols.
- **SaaS + self-hosting** — a free tier included, and the server component can be deployed on your own VPS.
- **Traffic inspector with replay** — a web UI for viewing and replaying requests (paid plans).
- **HTTPS out of the box** — TLS termination without additional configuration.
- **Single Go binary** — cross-platform, no external dependencies.
- **Active development** — regular updates and bug fixes.

**fxTunnel's limitations:**

- **Younger project** — the ecosystem and documentation are smaller than ngrok's. Under active development.

```bash
# Install fxTunnel
curl -fsSL https://fxtun.ru/install.sh | bash

# Or via Go
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest

# HTTP tunnel
fxtunnel http 8080

# TCP tunnel (e.g., PostgreSQL)
fxtunnel tcp 5432

# UDP tunnel (e.g., game server)
fxtunnel udp 27015
```

## Comparison Table: Bore vs fxTunnel vs LocalTunnel

Here is how the three tools stack up side by side. Bore covers the basics for TCP forwarding, LocalTunnel is showing its age, and fxTunnel fills the gaps both leave open.

| Criterion | Bore | LocalTunnel | fxTunnel |
|---|---|---|---|
| **Language** | Rust | Node.js | Go |
| **Protocols** | TCP | HTTP | HTTP, TCP, UDP |
| **HTTPS** | No (needs reverse proxy) | Public server only | Yes, out of the box |
| **Traffic inspector** | No | No | Yes (from $5/mo) |
| **Request replay** | No | No | Yes (from $5/mo) |
| **SaaS (public server)** | No | Yes (unreliable) | Yes (stable, free tier) |
| **Self-hosting** | Yes | Yes | Yes |
| **Active development** | Minimal | No (since 2020) | Yes |
| **Custom domains** | No | Subdomains | Yes (any DNS, from $5/mo) |
| **Installation** | `cargo install` | `npm install -g` | `curl` or `go install` |

## Installation and Launch — CLI Commands Compared

How quickly can you go from zero to a working tunnel? Below are the commands for all three [tunneling tools](/blog/best-tunneling-tools-2026/) side by side.

### Installation

```bash
# Bore (requires Rust toolchain)
cargo install bore-cli

# LocalTunnel (requires Node.js)
npm install -g localtunnel

# fxTunnel (no dependencies)
curl -fsSL https://fxtun.ru/install.sh | bash
```

### Starting an HTTP Tunnel

```bash
# Bore — no direct HTTP tunnel support,
# requires TCP tunnel + reverse proxy on the server
bore local 8080 --to your-server.com

# LocalTunnel
lt --port 8080

# fxTunnel
fxtunnel http 8080
```

### Starting a TCP Tunnel

```bash
# Bore
bore local 5432 --to your-server.com

# LocalTunnel — TCP not supported

# fxTunnel
fxtunnel tcp 5432
```

### Starting a UDP Tunnel

```bash
# Bore — UDP not supported
# LocalTunnel — UDP not supported

# fxTunnel
fxtunnel udp 27015
```

fxTunnel is the only tool of the three that covers all three protocols with a single command and requires no additional reverse proxy or server infrastructure setup (though self-hosting is also available).

## Feature Deep-Dive

### Protocols and HTTPS

Bore operates exclusively at the TCP level. This means that for HTTP traffic, you need to configure a reverse proxy (nginx, Caddy) on the server yourself, and for HTTPS — TLS certificates on top of that. LocalTunnel supports HTTP only and provides HTTPS on the public server side, but you have no control over the certificates.

fxTunnel supports HTTP with automatic HTTPS termination, TCP, and UDP. For developers who need to [forward a database, SSH, or a game server](/blog/what-is-tunneling/), it is the only option of the three that requires no additional tooling.

### Traffic Inspector and Replay

Neither Bore nor LocalTunnel offer a traffic inspector. If you are [testing webhooks](/blog/webhook-testing-with-tunnel/) (Stripe, GitHub, Telegram), you will need external tools for logging and replaying requests.

fxTunnel includes a web UI for traffic inspection with replay (request re-sending) on paid plans from $5/mo. This simplifies debugging and saves time compared to manually setting up request logging.

### Self-Hosting

All three tools support self-hosting, but with varying levels of complexity:

| Aspect | Bore | LocalTunnel | fxTunnel |
|---|---|---|---|
| **Server component** | Rust binary | Node.js server | Go binary |
| **Dependencies** | None | Node.js, npm | None |
| **TLS setup** | Manual (reverse proxy) | Manual | Built-in |
| **Self-hosting docs** | Minimal | Outdated | Up to date |
| **Maintenance** | Minimal | None | Active |

Bore is the simplest to deploy thanks to its minimal footprint, but you only get TCP without TLS. LocalTunnel requires a Node.js environment and is unmaintained. fxTunnel offers a single binary with full functionality, including TLS -- you can read more about how that works in the [architecture overview](/blog/fxtunnel-architecture/).

### Development Activity and Community

Development activity is a critical factor when choosing an open-source tool. An abandoned project means unpatched vulnerabilities, incompatibility with new OS versions, and no new features.

- **Bore** — minimal activity. The project is stable but evolves slowly. New features are rare.
- **LocalTunnel** — effectively abandoned. The last significant commits date back to 2020. Open issues pile up without response.
- **fxTunnel** — active development. Regular releases, fast response to issues, and a growing community.

## Decision Matrix: Which Open-Source Tunnel to Choose

Your choice depends on what you are building. Here are concrete scenarios to help you pick the [right tool](/blog/best-tunneling-tools-2026/).

### Choose fxTunnel — the universal choice:

- **[Webhook testing](/blog/webhook-testing-with-tunnel/)** — HTTP tunnel in 30 seconds, inspector with replay for debugging.
- **TCP use cases** (databases, SSH) — full TCP support with HTTPS.
- **UDP use cases** (game servers, VoIP, IoT) — the only one of the three with UDP.
- **Team collaboration** — SaaS with a free tier, no need for your own server to get started.
- **Production behind NAT** — custom domains, TLS, reliability.
- **Maximum security** — full code audit, self-hosting when needed.

### Choose Bore only if:

- You need **minimal TCP port forwarding** and nothing else.
- You already use **Rust** in your project and want a homogeneous stack.
- You have your **own server with a public IP** and are willing to set up a reverse proxy for HTTPS.
- You **do not need** HTTP tunnels, an inspector, or SaaS.

### Choose LocalTunnel only if:

- You need a **throwaway HTTP tunnel** for a quick demo and can tolerate the public server's instability.
- You already work within the **Node.js ecosystem** and do not want to install additional tools.
- You are **not concerned** about the lack of maintenance and updates.

For most use cases, fxTunnel strikes the right balance: Bore-level simplicity, LocalTunnel-level accessibility, plus features that neither competitor offers.

## Security of Open-Source Tunnels

Security is one of the main reasons to choose an open-source solution. But the mere fact of open source does not guarantee security — implementation details matter.

| Aspect | Bore | LocalTunnel | fxTunnel |
|---|---|---|---|
| **TLS encryption** | No (raw TCP) | Public server only | Yes, out of the box |
| **Code audit** | Possible (Rust) | Possible (Node.js) | Possible (Go) |
| **Active security patches** | Rare | No | Yes |
| **Self-hosting** | Yes | Yes | Yes |
| **Data control** | Full (self-host) | Partial (public server unverifiable) | Full (self-host or audit SaaS code) |

Bore with self-hosting gives you full control over TCP traffic, but without TLS, data is transmitted in plain text — unacceptable for production. LocalTunnel on the public server does not allow you to verify the server side, and the self-hosted version is not updated. fxTunnel offers TLS out of the box, regular security updates, and full control with self-hosting.

## Try fxTunnel in 60 Seconds

Three protocols, a traffic inspector, SaaS and self-hosting, active development -- if you want to try it out, the setup takes about a minute.

```bash
# Install and launch
curl -fsSL https://fxtun.ru/install.sh | bash
fxtunnel http 8080
```

Curious how fxTunnel compares to proprietary alternatives? We put it up against [ngrok and Cloudflare Tunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) in a separate article, and there is also a broader [2026 tunneling tools ranking](/blog/best-tunneling-tools-2026/).

## FAQ — Frequently Asked Questions About Bore, LocalTunnel, and fxTunnel

If you are new to [tunneling](/blog/what-is-tunneling/) or looking for [free ngrok alternatives](/blog/free-ngrok-alternatives-2026/), those articles are a good starting point. Below are the questions we hear most often about these three open-source tools.

### Which open-source tunnel is best -- Bore, LocalTunnel, or fxTunnel?

It depends on your needs, but fxTunnel covers the widest range of use cases. Bore limits you to raw TCP without HTTPS or an inspector; LocalTunnel only handles HTTP and its public server is unreliable. fxTunnel handles HTTP, TCP, and UDP, ships a traffic inspector with replay, and works both as a SaaS and self-hosted.

### Can Bore handle HTTPS tunnels?

Not on its own. Bore passes raw TCP traffic, so you would need to put a reverse proxy like nginx or Caddy in front of it to terminate TLS. fxTunnel handles HTTPS automatically -- no extra configuration needed.

### Why is LocalTunnel unreliable?

The public server at `localtunnel.me` is frequently overloaded, and the project has been largely dormant since 2020. Self-hosting is possible but requires a Node.js stack and comes with no ongoing maintenance. fxTunnel provides a stable SaaS alternative alongside a Go-based self-hosting option.

### Which open-source tunnel supports UDP?

Among these three, only fxTunnel. Bore is TCP-only, and LocalTunnel is HTTP-only. If your workload involves game servers, VoIP, or IoT over UDP, fxTunnel is the one to use.

### Can I self-host Bore, LocalTunnel, or fxTunnel?

All three support self-hosting. Bore gives you a minimal Rust binary for TCP. LocalTunnel requires Node.js and is no longer actively maintained. fxTunnel ships as a single Go binary with HTTP, TCP, UDP, a traffic inspector, and built-in TLS -- making it the most practical self-hosting option of the three.
