---
title: "fxTunnel Architecture: Secure Tunnels Built with Go"
date: 2026-02-07T14:00:00+03:00
draft: false
description: "fxTunnel architecture deep dive: client-server design, multiplexing, TLS 1.3 security, Go runtime performance, and comparison with ngrok."
tags: [architecture, go, security, protocols, fxTunnel, tunneling, open-source, saas]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why is fxTunnel written in Go instead of Rust or C++?"
    a: "Go strikes the right balance between performance and development speed. Built-in concurrency via goroutines is ideal for handling thousands of simultaneous connections. Compilation into a single static binary with no dependencies makes deployment straightforward on any platform."
  - q: "How does fxTunnel keep tunnels secure?"
    a: "Security is implemented at three layers: TLS encryption of all traffic between client and server, token-based authentication on connect, and tunnel isolation — each tunnel runs in its own context with no access to other tunnels' data."
  - q: "How many concurrent tunnels can a single fxTunnel server handle?"
    a: "On a VPS with 1 vCPU and 1 GB RAM, the fxTunnel server can handle hundreds of concurrent tunnels and thousands of connections. The Go runtime efficiently manages goroutines, using roughly 4 KB of stack per goroutine. Scaling depends primarily on network bandwidth."
  - q: "Can I run fxTunnel without TLS for local development?"
    a: "Yes, you can run the server without TLS in isolated development environments. However, for any use over a public network, TLS encryption is strongly recommended — it protects data from interception and tampering."
  - q: "Is fxTunnel a SaaS or self-hosted solution?"
    a: "fxTunnel is a SaaS-first tool. The cloud service at fxtun.dev works out of the box: a free tier with no limits, from $5/mo for custom domains and request inspector with replay. Self-hosted deployment is also available — the code is fully open source on GitHub."
  - q: "How is fxTunnel better than ngrok and Cloudflare Tunnel?"
    a: "fxTunnel is open source with a completely free tier and no connection limits. Unlike ngrok, the code is open and auditable. Unlike Cloudflare Tunnel, fxTunnel supports TCP and UDP — not just HTTP. Plus it includes a built-in request inspector with replay."
---

## Architecture Overview

fxTunnel is an open-source tool for secure [tunneling](/blog/what-is-tunneling/) of HTTP, TCP, and UDP traffic, built in Go. It follows a client-server model: the client initiates an outbound connection to a public server, and the server routes external traffic back through that connection. Go goroutines multiplex thousands of streams through a single encrypted channel, and the system splits cleanly into three layers — the data plane (traffic forwarding), the control plane (tunnel management), and the transport layer (HTTP, TCP, UDP).

This design keeps fxTunnel lightweight — a single binary with no dependencies — while handling diverse traffic types, from HTTP requests and [webhook testing](/blog/webhook-testing-with-tunnel/) to UDP datagrams for game servers. The SaaS service at [fxtun.dev](https://fxtun.dev) works out of the box, while a self-hosted option is available for those who need full control.

## fxTunnel System Components

Three components make up the system: a CLI client, a routing server, and a control plane. The client opens a TLS connection from inside the network, bypassing NAT and firewalls. The server multiplexes thousands of streams through that connection and routes traffic by domain (HTTP) or port (TCP/UDP). The control plane manages the tunnel lifecycle through heartbeat monitoring.

### Client (CLI)

The fxTunnel client is a CLI application that establishes an encrypted connection to the server and forwards traffic to a local port. Because the connection is initiated from inside the network, NAT and firewalls are not an obstacle. Architecturally, the client performs three jobs: authentication, maintaining the control channel, and data multiplexing.

{{< diagram >}}
┌─────────────────────────────────────────────┐
│  fxTunnel Client (CLI)                      │
│                                             │
│  ┌───────────┐  ┌──────────┐  ┌──────────┐ │
│  │ Auth      │  │ Control  │  │ Data     │ │
│  │ Module    │  │ Channel  │  │ Mux      │ │
│  └───────────┘  └──────────┘  └──────────┘ │
│         │              │             │      │
│         └──────────────┼─────────────┘      │
│                        │                    │
│              ┌─────────▼────────┐           │
│              │  TLS Connection  │           │
│              └──────────────────┘           │
└─────────────────────────────────────────────┘
{{< /diagram >}}

Example client usage:

```bash
# HTTP tunnel — proxies HTTP requests to localhost:8080
fxtunnel http 8080

# TCP tunnel — forwards TCP connections to localhost:5432
fxtunnel tcp 5432

# UDP tunnel — relays UDP datagrams to localhost:27015
fxtunnel udp 27015
```

The client automatically reconnects on connection loss and uses heartbeats for early detection of network issues.

### Server (Routing and Multiplexing)

The server is the central component that accepts client connections and routes external traffic. For each incoming request, the server matches it against a registered tunnel by URL (for HTTP) or port number (for TCP/UDP) and forwards the data to the corresponding client through a multiplexed connection.

{{< diagram >}}
┌──────────────────────────────────────────────┐
│  fxTunnel Server                             │
│                                              │
│  ┌─────────────┐    ┌─────────────────────┐  │
│  │ HTTP Router │    │ TCP/UDP Port Router │  │
│  │ (by domain) │    │ (by port number)    │  │
│  └──────┬──────┘    └─────────┬───────────┘  │
│         │                     │              │
│         └─────────┬───────────┘              │
│                   ▼                          │
│         ┌──────────────────┐                 │
│         │  Session Manager │                 │
│         │  (multiplexer)   │                 │
│         └────────┬─────────┘                 │
│                  ▼                           │
│         ┌──────────────────┐                 │
│         │  Client Pool     │                 │
│         │  (TLS connections)│                │
│         └──────────────────┘                 │
└──────────────────────────────────────────────┘
{{< /diagram >}}

Key server features:

- **Multiplexing** — many logical data streams travel over a single physical TCP connection, reducing the overhead of establishing connections.
- **Domain-based routing** — for HTTP tunnels, the server inspects the `Host` header and routes the request to the correct client.
- **Port-based routing** — for TCP/UDP tunnels, the server allocates a public port and forwards all traffic on that port to the client.
- **Hot reload** — tunnels can be added and removed without restarting the server.

### Control Plane (Registration and Heartbeat)

The control plane manages the tunnel lifecycle: registration, health monitoring, and graceful shutdown. It runs over the same TLS connection as the data plane but uses a separate logical channel.

Tunnel lifecycle:

1. **Authentication** — the client sends a token, and the server validates it.
2. **Registration** — the client requests a tunnel (protocol + local port), and the server allocates a public address.
3. **Heartbeat** — client and server exchange pings every N seconds to detect disconnections.
4. **Teardown** — when a client disconnects, the server releases resources and the public address.

```bash
# Simplified control plane sequence
CLIENT → SERVER: Auth { token: "xxx" }
SERVER → CLIENT: AuthOK { session_id: "abc123" }

CLIENT → SERVER: Register { protocol: "http", local_port: 8080 }
SERVER → CLIENT: Registered { public_url: "https://abc123.fxtun.dev" }

CLIENT ↔ SERVER: Ping / Pong (every 10 seconds)

CLIENT → SERVER: Disconnect { session_id: "abc123" }
SERVER → CLIENT: Disconnected
```

The heartbeat mechanism allows rapid detection of broken connections — if three consecutive pings go unanswered, the server considers the client disconnected and releases the tunnel.

## Tunneling Protocols: HTTP, TCP, UDP

fxTunnel supports three protocols — HTTP, TCP, and UDP. Unlike Cloudflare Tunnel (HTTP only) and ngrok (HTTP + limited TCP), you get full support for all three, including UDP for game servers, DNS, and VoIP. Each protocol has its own traffic-handling model — here is how they [compare to the competition](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Characteristic | HTTP | TCP | UDP |
|---|---|---|---|
| **Routing** | By `Host` header | By allocated port | By allocated port |
| **Connection type** | Request-Response | Persistent (stream) | Datagrams (connectionless) |
| **Custom domain** | Yes | No (port only) | No (port only) |
| **Public-side TLS** | Automatic HTTPS | No (raw TCP) | No (raw UDP) |
| **Typical use cases** | Web apps, APIs, webhooks | Databases, SSH, Redis | Game servers, DNS, VoIP |
| **Multiplexing** | By HTTP request | By TCP stream | By datagram |
| **WebSocket support** | Yes (Upgrade) | — | — |

### How HTTP Tunnels Work

The HTTP tunnel is the most common type. The server accepts HTTPS requests on a public domain, inspects the `Host` header, finds the matching registered tunnel, and forwards the request to the client. The response travels back along the same path.

{{< diagram >}}
Browser → HTTPS → fxTunnel Server → multiplexer → fxTunnel Client → localhost:8080
                  (abc123.fxtun.dev)                                 (your server)
{{< /diagram >}}

HTTP tunnels automatically provide TLS on the public side, support WebSockets (via HTTP Upgrade), and allow custom domains.

### How TCP Tunnels Work

A TCP tunnel allocates a public port on the server and forwards all TCP connections to the client. This is useful for databases, SSH, Redis, and any other TCP-based service.

```bash
# Client forwards PostgreSQL
fxtunnel tcp 5432
# → Public address: tunnel.fxtun.dev:41234

# Connect to the database from a remote machine
psql -h tunnel.fxtun.dev -p 41234 -U myuser mydb
```

Each new TCP connection on the public port creates a separate stream in the multiplexer, ensuring isolation between clients.

### How UDP Tunnels Work

The UDP tunnel is a feature unique to fxTunnel among comparable tools. Since UDP is a connectionless protocol, tunneling requires a special approach: the client encapsulates UDP datagrams inside the TCP connection to the server, and the server unpacks them and sends them from a public UDP port.

```bash
# Client forwards a game server
fxtunnel udp 27015
# → Public address: tunnel.fxtun.dev:52001

# Players connect via the public address
```

The encapsulation adds minimal latency but makes it possible to forward UDP traffic through NAT without configuring port forwarding on your router.

## fxTunnel Tunnel Security

Security rests on three layers: TLS 1.3 encryption of all traffic between client and server, token-based authentication on connect, and complete tunnel isolation. Because the code is open source, any developer can perform a security audit — something you cannot do with closed-source alternatives.

### TLS Encryption

All data between the fxTunnel client and server is transmitted over a TLS connection. This protects traffic from interception and modification at intermediate nodes. For HTTP tunnels, the public side also uses HTTPS — the server can use certificates from Let's Encrypt or any other CA.

{{< diagram >}}
Client ←── TLS 1.3 ──→ Server ←── HTTPS ──→ External client
        (multiplexed)           (public URL)
{{< /diagram >}}

TLS 1.3 is supported with modern cipher suites. Legacy protocols (TLS 1.0, 1.1) are disabled by default.

### Authentication

On connect, the client presents a token that the server validates. This prevents unauthorized tunnel creation. Tokens can be generated on the server side and distributed to users — one per person.

```bash
# Generate a token on the server
fxtunnel server token generate --name "developer-1"
# → Token: fxt_a1b2c3d4e5f6...

# Client uses the token
fxtunnel http 8080 --token fxt_a1b2c3d4e5f6...
```

Tokens are bound to the server and can be revoked at any time without a restart.

### Tunnel Isolation

Each tunnel runs in an isolated context. Client A cannot access Client B's data — the multiplexer strictly separates data streams based on session identifiers. This is critical when multiple developers share a single server.

## Go Runtime Performance in fxTunnel

What does the Go implementation actually cost you in overhead? On a 1-vCPU VPS, expect +1-5 ms added latency, up to 500 Mbps throughput, and 5,000+ concurrent connections. Goroutines use only ~4 KB of stack each, so hundreds of tunnels fit comfortably in ~50 MB of RAM.

### Benchmarks

Performance depends on three things: VPS network bandwidth, CPU capacity for TLS encryption, and multiplexing efficiency. Here are approximate numbers for a typical VPS (1 vCPU, 1 GB RAM):

| Metric | Value |
|---|---|
| **Latency** | +1–5 ms (added to network latency) |
| **HTTP throughput** | Up to 500 Mbps |
| **Concurrent connections** | 5,000+ |
| **Concurrent tunnels** | 500+ |
| **RAM usage (idle)** | ~15 MB |
| **RAM usage (100 tunnels)** | ~50 MB |

### Go Runtime Optimizations

Why does Go work so well here? A few runtime features matter most:

- **Goroutines** — each connection is served by a goroutine that uses roughly 4 KB of stack. This allows handling thousands of connections without heavyweight OS threads.
- **Netpoller** — the built-in I/O multiplexer (epoll on Linux, kqueue on macOS) handles network events efficiently without blocking.
- **Zero-copy where possible** — data is forwarded between connections via `io.Copy`, which uses `splice` on Linux to minimize copies into userspace.
- **Static binary** — compilation into a single file with no dependencies simplifies deployment and reduces cold start time.

```go
// Example: multiplexing with goroutines (simplified)
func (s *Server) handleTunnel(session *Session) {
    for {
        // Accept a new stream from the multiplexer
        stream, err := session.AcceptStream()
        if err != nil {
            return
        }
        // Each stream is handled in its own goroutine
        go func() {
            defer stream.Close()
            // Proxy data between the public
            // and local connections
            localConn, err := net.Dial("tcp", session.LocalAddr)
            if err != nil {
                return
            }
            defer localConn.Close()
            // Bidirectional data copy
            go io.Copy(localConn, stream)
            io.Copy(stream, localConn)
        }()
    }
}
```

The result: thousands of concurrent connections on a single CPU core, with straightforward synchronous-looking code.

## SaaS Service and Pricing

The cloud service at [fxtun.dev](https://fxtun.dev) lets you get started in seconds with no server setup. The free tier covers HTTP, TCP, and UDP tunnels. Paid plans from $5/mo add custom domains and a request inspector with replay, while the $10/mo tier supports 10+ concurrent tunnels for teams.

| Plan | Price | Features |
|---|---|---|
| **Free** | $0 | HTTP/TCP/UDP tunnels, no connection limits |
| **Pro** | from $5/mo | Custom domains, request inspector, replay |
| **Team** | from $10/mo | 10+ concurrent tunnels, priority support |

The **request inspector** is a built-in tool that records every HTTP request passing through the tunnel, complete with body, headers, and timing data. The **replay** feature lets you re-send any request with a single click — perfect for [webhook testing](/blog/webhook-testing-with-tunnel/) and debugging integrations.

## Open Source and Contributing

The full source code lives on [GitHub](https://github.com/mephistofox/fxtun.dev). You can audit the security implementation, propose improvements, or deploy a self-hosted version on your own infrastructure.

How to contribute:

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USER/fxTunnel.git
cd fxTunnel

# Create a branch for your change
git checkout -b feature/my-improvement

# Run the tests
go test ./...

# Submit a pull request on GitHub
```

Bug fixes, documentation improvements, new features, testing, translations — all contributions are welcome regardless of experience level.

Developer links:

- **Repository:** [github.com/mephistofox/fxtun.dev](https://github.com/mephistofox/fxtun.dev)
- **SaaS service:** [fxtun.dev](https://fxtun.dev)
- **Issues:** [github.com/mephistofox/fxtun.dev/issues](https://github.com/mephistofox/fxtun.dev/issues)
- **Comparison with alternatives:** [fxTunnel vs ngrok vs Cloudflare](/blog/ngrok-vs-cloudflare-vs-fxtunnel/)

## FAQ

### Why is fxTunnel written in Go instead of Rust or C++?

Go hits a sweet spot: it is fast enough for production networking and productive enough for rapid iteration. Goroutines handle thousands of concurrent connections without the complexity of manual memory management. And because Go compiles to a single static binary, deploying to any platform is trivial.

### How does fxTunnel keep tunnels secure?

Three layers work together: TLS encrypts everything between client and server, tokens authenticate each connection, and tunnel isolation ensures that every tunnel operates in its own context with no way to reach another tunnel's data.

### How many concurrent tunnels can a single fxTunnel server handle?

A modest VPS (1 vCPU, 1 GB RAM) comfortably runs hundreds of tunnels and thousands of connections. Each goroutine consumes roughly 4 KB of stack, so RAM is rarely the bottleneck — network bandwidth is what limits scaling in practice.

### Can I run fxTunnel without TLS for local development?

You can, as long as you are in a fully isolated environment. For anything that touches a public network, though, keep TLS on — it guards against interception and tampering.

### Is fxTunnel a SaaS or self-hosted solution?

Both. The cloud service at [fxtun.dev](https://fxtun.dev) works out of the box — free tier included, with paid plans from $5/mo for custom domains and the request inspector. If you prefer full control, you can self-host using the open-source code on GitHub.

### How is fxTunnel better than ngrok and Cloudflare Tunnel?

The code is [fully open and auditable](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) — ngrok is closed-source, and Cloudflare only publishes its client. fxTunnel supports TCP and UDP alongside HTTP, while Cloudflare is HTTP-only on the free tier. The built-in request inspector with replay rounds out the developer experience.
