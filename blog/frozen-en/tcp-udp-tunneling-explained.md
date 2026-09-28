---
title: "How TCP and UDP Tunneling Works — A Protocol Deep Dive"
date: 2026-02-15T14:00:00+03:00
draft: false
description: "How TCP and UDP tunneling works under the hood: multiplexing, connection lifecycle, NAT traversal, packet framing, and hands-on examples with fxTunnel."
tags: [tcp, udp, tunneling, protocols, fxTunnel, networking, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "What is the difference between a TCP tunnel and a UDP tunnel?"
    a: "A TCP tunnel forwards reliable, stream-oriented connections with guaranteed delivery and ordering — ideal for databases, SSH, and Redis. A UDP tunnel forwards datagrams without establishing a connection — ideal for game servers, VoIP, and DNS where speed matters more than guaranteed delivery."
  - q: "Does fxTunnel support both TCP and UDP?"
    a: "Yes. fxTunnel is one of the few tunneling tools that support both protocols. ngrok is limited to TCP and HTTP, Cloudflare Tunnel's free tier only supports HTTP. fxTunnel lets you create HTTP, TCP, and UDP tunnels with a single command."
  - q: "Why is UDP tunneling harder than TCP?"
    a: "UDP is a connectionless protocol with no delivery guarantees. The tunnel must track virtual sessions by source address:port pairs, encapsulate datagrams inside a TCP connection to the server, and handle NAT traversal. TCP tunneling is simpler because the protocol itself provides reliability and ordering."
  - q: "How much latency does a TCP/UDP tunnel add?"
    a: "fxTunnel adds +1–5 ms on top of the base network latency. For TCP services (databases, SSH) this is negligible. For UDP applications (games, VoIP) the overhead is slightly higher due to datagram encapsulation inside TCP, but it remains acceptable for most scenarios."
  - q: "Can I tunnel PostgreSQL or Redis through fxTunnel?"
    a: "Yes. A TCP tunnel is perfect for this. Run `fxtunnel tcp 5432` for PostgreSQL or `fxtunnel tcp 6379` for Redis, and you get a public address to connect from any machine."
---

## TCP and UDP Tunneling: How It Works

Most tunneling tools only handle HTTP. But what if you need to expose a PostgreSQL database, a game server, or a DNS resolver? That is where TCP and UDP tunnels come in. If you are new to the concept, start with [What is tunneling](/blog/what-is-tunneling/).

Below we will walk through how TCP and UDP tunneling works under the hood: connection multiplexing, packet lifecycle, NAT traversal, data framing, and real-world use cases with fxTunnel.

## TCP vs UDP: Key Differences

Why does the protocol matter when you are just forwarding traffic? Because TCP and UDP work in fundamentally different ways, and a tunnel has to account for that.

| Characteristic | TCP | UDP |
|---|---|---|
| **Connection** | Established (3-way handshake) | Connectionless |
| **Delivery guarantee** | Yes (ACKs, retransmission) | No |
| **Packet ordering** | Guaranteed | Not guaranteed |
| **Flow control** | Yes (sliding window) | No |
| **Header size** | 20–60 bytes | 8 bytes |
| **Latency** | Higher (handshake + ACK) | Lower (no connection setup) |
| **Typical use** | Databases, SSH, HTTP, Redis | Games, VoIP, DNS, IoT, video streaming |
| **Behavior on packet loss** | Retransmission | Packet is lost |

In short, TCP gives you reliability at the cost of extra latency; UDP trades reliability for speed. Each protocol requires a different approach to [tunneling](/blog/what-is-tunneling/).

## How TCP Tunneling Works

A TCP tunnel creates a virtual channel between a public port on the server and a local port on your machine. Every incoming connection to the public port is multiplexed through a single control TLS connection between the client and the server. Since TCP already guarantees delivery and ordering, the tunnel just needs to forward byte streams.

### TCP Connection Lifecycle Through a Tunnel

1. **Registration** — the fxTunnel client connects to the server and registers a TCP tunnel for the specified local port.
2. **Port allocation** — the server allocates a public port (e.g., `tunnel.fxtun.dev:41234`).
3. **Incoming connection** — an external client connects to the public port and completes the TCP handshake with the server.
4. **Multiplexing** — the server creates a new logical stream in the multiplexer and forwards data to the fxTunnel client.
5. **Local connection** — the fxTunnel client establishes a TCP connection to the local service and proxies data bidirectionally.
6. **Teardown** — when either side closes the connection, the entire chain is shut down gracefully.

### Framing and Multiplexing TCP Streams

Multiplexing is the core mechanism of TCP tunneling. Dozens or hundreds of independent TCP streams travel through a single physical TLS connection between the client and the server. Each stream has a unique identifier, and data is split into frames with a header:

```
+--------------------------------------+
|  Frame Header                        |
|  +----------+--------+-------------+ |
|  | Stream ID| Length | Flags       | |
|  | (4 bytes)|(4 bytes)| (1 byte)  | |
|  +----------+--------+-------------+ |
|  +----------------------------------+ |
|  |  Payload (up to 64 KB)          | |
|  +----------------------------------+ |
+--------------------------------------+
```

This approach avoids creating a separate TLS connection for every TCP stream, reducing overhead and speeding up connection establishment. For more on this, see [fxTunnel Architecture](/blog/fxtunnel-architecture/).

### TCP Tunneling Diagram

```
External client (psql, redis-cli, ssh)
        |
        | TCP connection
        v
+------------------------------+
|  fxTunnel Server             |
|  Public port: 41234          |
|                              |
|  +------------------------+  |
|  |  Multiplexer           |  |
|  |  Stream #1 -+          |  |
|  |  Stream #2 -+-> TLS   |  |
|  |  Stream #N -+   conn  |  |
|  +------------------------+  |
+--------------+---------------+
               | single TLS connection
               | (all streams multiplexed)
               v
+------------------------------+
|  fxTunnel Client             |
|                              |
|  +------------------------+  |
|  |  Demultiplexer         |  |
|  |  Stream #1 -> :5432    |  |
|  |  Stream #2 -> :5432    |  |
|  |  Stream #N -> :5432    |  |
|  +------------------------+  |
+------------------------------+
               |
               v
        localhost:5432
        (PostgreSQL)
```

Each logical stream corresponds to one TCP connection from an external client. The multiplexer packs data from all streams into a single TLS channel, and the demultiplexer on the client side unpacks it and routes it to the local service.

## How UDP Tunneling Works

UDP tunneling is significantly more complex than TCP. The tunnel must track virtual sessions on its own, encapsulate datagrams inside a reliable TCP connection to the server, and handle NAT traversal correctly. This is why most tunneling tools do not support UDP at all.

### Core Challenges of UDP Tunneling

- **No connection state** — UDP has no handshake and no connection tracking. The tunnel must create virtual sessions based on source address:port pairs.
- **NAT traversal** — without an established connection, NAT tables do not guarantee stable routing. The tunnel solves this by encapsulating datagrams inside TCP.
- **Ordering and packet loss** — UDP guarantees neither ordering nor delivery. Encapsulation inside TCP adds guarantees on the client-server leg, but the end recipient still receives raw UDP.
- **Datagram boundaries** — UDP packets have fixed boundaries that must be preserved during encapsulation. Stream-based TCP has no concept of a "packet," so framing is required.

### Encapsulating UDP Inside TCP

fxTunnel solves the UDP tunneling problem by encapsulating datagrams inside frames on the TCP connection. Each UDP datagram is wrapped in a frame containing metadata: sender address, length, and a virtual session identifier.

```
UDP datagram from a game client
        |
        v
+------------------------------+
|  fxTunnel Server             |
|  Public UDP port: 52001      |
|                              |
|  +------------------------+  |
|  |  Virtual sessions      |  |
|  |  Addr A:1234 -> Sess #1|  |
|  |  Addr B:5678 -> Sess #2|  |
|  +------------+-----------+  |
|               |              |
|  +------------v-----------+  |
|  |  Encapsulation         |  |
|  |  +------+------+-----+ |  |
|  |  |SessID| Len  | Data| |  |
|  |  +------+------+-----+ |  |
|  +------------+-----------+  |
+---------------+--------------+
                | TLS connection (TCP)
                v
+------------------------------+
|  fxTunnel Client             |
|                              |
|  +------------------------+  |
|  |  Decapsulation         |  |
|  |  Sess #1 -> UDP :27015 |  |
|  |  Sess #2 -> UDP :27015 |  |
|  +------------------------+  |
+------------------------------+
                |
                v
        localhost:27015
        (game server)
```

On the server side, incoming UDP datagrams are grouped by virtual session (based on the sender's address and port), encapsulated into frames, and transmitted to the client over the TLS connection. The client decapsulates the frames and sends raw UDP datagrams to the local port. Response datagrams travel back the same way.

### Virtual Session Management

Since UDP has no concept of a "connection," fxTunnel creates virtual sessions with an inactivity timeout. If no datagrams arrive from a specific sender within a configured window (60 seconds by default), the session is removed and resources are freed. This prevents memory leaks when handling a large number of short-lived UDP clients.

## TCP Tunnel Use Cases

Have a service that speaks TCP and needs to be reachable from outside your local network? Here are the most common scenarios.

### Databases (PostgreSQL, MySQL)

```bash
# Forward PostgreSQL
fxtunnel tcp 5432
# -> tunnel.fxtun.dev:41234

# Connect from a remote machine
psql -h tunnel.fxtun.dev -p 41234 -U myuser mydb
```

This gives a teammate or CI server access to your local database without deploying anything or touching network configuration.

### SSH Access

```bash
# Forward SSH
fxtunnel tcp 22
# -> tunnel.fxtun.dev:43210

# Connect to a machine behind NAT
ssh -p 43210 user@tunnel.fxtun.dev
```

A TCP tunnel for SSH lets you reach a machine behind NAT without a static IP or port forwarding. Perfect for Raspberry Pi units and home servers.

### Redis and Cache Servers

```bash
# Forward Redis
fxtunnel tcp 6379
# -> tunnel.fxtun.dev:44567

# Connect from any machine
redis-cli -h tunnel.fxtun.dev -p 44567
```

## UDP Tunnel Use Cases

Real-time applications care about latency, not about guaranteed delivery of every single packet. That is where UDP tunnels shine.

### Game Servers

```bash
# Forward a game server (e.g., Counter-Strike, Minecraft)
fxtunnel udp 27015
# -> tunnel.fxtun.dev:52001

# Players connect through the public address
```

Game servers use UDP because losing a single packet with player coordinates is far less disruptive than the latency caused by retransmission.

### DNS Servers

```bash
# Forward a local DNS server
fxtunnel udp 53
# -> tunnel.fxtun.dev:53001

# Test DNS queries
dig @tunnel.fxtun.dev -p 53001 example.com
```

### VoIP and IoT

VoIP (SIP, RTP), video streaming, and IoT devices all rely on UDP because latency matters more than the occasional lost packet. You can forward any UDP port to the internet without configuring NAT.

## How fxTunnel Handles Both Protocols

fxTunnel multiplexes TCP streams and encapsulated UDP datagrams through a single control TLS connection. This is a key architectural decision: a single client can hold HTTP, TCP, and UDP tunnels simultaneously, all sharing one encrypted channel to the server. For the full picture, see [fxTunnel Architecture](/blog/fxtunnel-architecture/).

```
+---------------------------------------------+
|  Single TLS connection: client <-> server    |
|                                              |
|  +-------------+ +-----------+ +-----------+ |
|  | HTTP streams| |TCP streams| |UDP frames | |
|  | (multiplex) | |(multiplex)| |(encapsul.)| |
|  +-------------+ +-----------+ +-----------+ |
|                                              |
|  Control plane: registration, heartbeat,     |
|  tunnel management                           |
+---------------------------------------------+
```

Benefits of a single connection:

- **Single authentication point** — the token is verified once at connect time.
- **Minimal overhead** — no need to establish a separate TLS connection for each tunnel.
- **Firewall-friendly** — all traffic flows through one outbound port.
- **Fast reconnection** — one connection is restored, and all tunnels resume automatically.

For a deeper comparison with [ngrok and Cloudflare Tunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/), see our side-by-side article. The short version: fxTunnel is one of the few tools with full UDP support alongside HTTP and TCP.

## Hands-On Examples with fxTunnel

### Installation

```bash
# Install fxTunnel (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash
```

### TCP Tunnels

```bash
# PostgreSQL — access your local database
fxtunnel tcp 5432

# Redis — access your local cache
fxtunnel tcp 6379

# SSH — remote access to a machine behind NAT
fxtunnel tcp 22
```

### UDP Tunnels

```bash
# Game server — players connect through a public address
fxtunnel udp 27015

# DNS — test your local DNS server
fxtunnel udp 53
```

After launch, fxTunnel prints the public address:

```
fxTunnel v1.x — tunnel is active
Protocol:    TCP
Public addr: tunnel.fxtun.dev:41234
Forwarding:  tunnel.fxtun.dev:41234 -> localhost:5432

Press Ctrl+C to stop
```

For more on exposing local services, see [How to expose localhost to the internet](/blog/expose-localhost-to-internet/).

## FAQ

### What is the difference between a TCP tunnel and a UDP tunnel?

A TCP tunnel carries stream-oriented connections that guarantee delivery and ordering, which makes it the right choice for databases, SSH, and Redis. A UDP tunnel carries datagrams with no connection setup, trading reliability for speed -- a better fit for game servers, VoIP, and DNS.

### Does fxTunnel support both TCP and UDP?

Yes -- HTTP, TCP, and UDP are all available through the same CLI. Most competitors only cover HTTP and TCP (ngrok) or HTTP alone on their free tier (Cloudflare Tunnel).

### Why is UDP tunneling harder than TCP?

Because UDP has no concept of a connection. The tunnel has to invent virtual sessions (keyed by source address:port), wrap each datagram in a frame on the TCP link to the server, and deal with NAT traversal. TCP tunneling is simpler -- the protocol already handles reliability and ordering for you.

### How much latency does a TCP/UDP tunnel add?

Expect roughly +1--5 ms on top of the base network latency. For TCP services like databases and SSH that is barely noticeable. UDP applications (games, VoIP) see a slightly higher overhead because datagrams are encapsulated inside TCP, but the penalty stays acceptable for most use cases.

### Can I tunnel PostgreSQL or Redis through fxTunnel?

Absolutely. Run `fxtunnel tcp 5432` for PostgreSQL or `fxtunnel tcp 6379` for Redis and you will get a public address that any remote client can connect to.
