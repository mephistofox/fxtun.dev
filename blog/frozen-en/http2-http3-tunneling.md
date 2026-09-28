---
title: "HTTP/2 and HTTP/3 in Tunnels — Performance Impact"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "How HTTP/2 and HTTP/3 (QUIC) work through tunnels: multiplexing, header compression, 0-RTT, and fxTunnel protocol support compared to HTTP/1.1."
tags: [http2, http3, quic, tunneling, protocols, fxTunnel, performance, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Does fxTunnel support HTTP/2 and HTTP/3?"
    a: "Yes. fxTunnel's HTTP tunnels fully support HTTP/2 by default. When a client connects with HTTP/2, the tunnel preserves multiplexing, header compression, and server push. HTTP/3 (QUIC) support is on the roadmap. Since the tunnel terminates TLS at the edge, the internal leg can use any protocol the local server supports."
  - q: "Will HTTP/2 make my tunnel faster than HTTP/1.1?"
    a: "In most cases, yes. HTTP/2 multiplexes multiple requests over a single connection, eliminating head-of-line blocking at the HTTP layer and reducing connection overhead. For pages with many assets or APIs with parallel requests, you can see 20-50% faster load times through a tunnel compared to HTTP/1.1."
  - q: "What is the difference between HTTP/2 and HTTP/3 for tunneling?"
    a: "HTTP/2 runs over TCP and solves HTTP-level head-of-line blocking with stream multiplexing. HTTP/3 runs over QUIC (UDP) and eliminates TCP-level head-of-line blocking entirely — a single lost packet does not stall all streams. HTTP/3 also offers 0-RTT connection resumption and better performance on lossy or mobile networks."
  - q: "Can I force HTTP/2 through an fxTunnel tunnel?"
    a: "HTTP/2 is negotiated automatically via ALPN during the TLS handshake. If your local server supports HTTP/2 and the client requests it, fxTunnel will use HTTP/2 end-to-end. You can verify the protocol in browser DevTools (Network tab, Protocol column) or with curl --http2."
  - q: "Why does HTTP/3 matter for tunneling over mobile networks?"
    a: "Mobile networks have high packet loss and frequent IP changes (cell tower handoffs). HTTP/3's QUIC transport handles both: independent streams prevent one lost packet from blocking others, and connection migration lets sessions survive IP changes without a new handshake. This makes tunnels more resilient on unstable connections."
---

## Why HTTP Protocol Version Matters in Tunnels

When you create an HTTP tunnel, every request from the outside world passes through a relay server to your local machine. The protocol used for that journey — HTTP/1.1, HTTP/2, or HTTP/3 — determines how efficiently those requests are transported. With HTTP/1.1, each request waits for the previous one to finish on a given connection. With HTTP/2, dozens of requests fly in parallel over a single connection. With HTTP/3 (QUIC), even packet loss on the network does not stall unrelated requests.

For developers who rely on tunnels for [webhook testing](/blog/webhook-testing-with-tunnel/), API development, or [demoing projects](/blog/demo-localhost-without-deploy/), this is not an academic distinction. It directly affects page load times, API throughput, and the reliability of live demos.

So what does all of this mean when your traffic passes through a tunnel? Let's walk through how HTTP/2 and HTTP/3 actually behave, what the performance gains look like, and how fxTunnel handles modern protocols under the hood.

## HTTP/1.1 vs HTTP/2 vs HTTP/3: Key Differences

Here is how the three HTTP versions stack up, especially when it comes to tunneled traffic.

| Feature | HTTP/1.1 | HTTP/2 | HTTP/3 |
|---|---|---|---|
| **Transport** | TCP | TCP | QUIC (UDP) |
| **Multiplexing** | No (one request per connection, or pipelining with HOL blocking) | Yes (multiple streams per connection) | Yes (independent streams, no HOL blocking) |
| **Header compression** | None | HPACK | QPACK |
| **Connection setup** | TCP handshake + TLS handshake (2-3 RTT) | Same as HTTP/1.1 (but reuses connections) | 1-RTT (first), 0-RTT (resumed) |
| **Head-of-line blocking** | HTTP + TCP level | TCP level only | None |
| **Server push** | No | Yes | Yes (limited) |
| **Connection migration** | No | No | Yes (survives IP change) |
| **Encryption** | Optional (HTTPS) | Effectively mandatory (TLS via ALPN) | Always encrypted (built into QUIC) |
| **Adoption (2026)** | Universal | ~60% of websites | ~30% of websites |
| **Tunnel compatibility** | Full | Full (fxTunnel) | Emerging |

HTTP/2 is a strict improvement over HTTP/1.1 for nearly all use cases. HTTP/3 goes further by replacing TCP with QUIC, solving the last remaining head-of-line blocking issue and adding connection migration. For tunneling, both protocols bring significant improvements over the original HTTP/1.1.

## How HTTP/2 Multiplexing Works Through a Tunnel

HTTP/1.1 uses a simple request-response model. A browser needs an image, a stylesheet, and a script? It opens multiple TCP connections (typically 6 per domain) and sends one request per connection. Each connection requires a separate TCP handshake and TLS negotiation. In a tunnel, each of those connections must be forwarded individually.

HTTP/2 changes this fundamentally. All requests to a domain travel over a **single TCP connection** as independent streams. The tunnel only needs to manage one connection per client, while dozens of requests flow through it simultaneously.

### HTTP/1.1 Through a Tunnel: Sequential Model

```
Browser                    fxTunnel Server              fxTunnel Client         localhost:8080
   |                            |                            |                       |
   |--- TCP conn #1 ---------->|--- stream #1 ------------->|--- TCP conn #1 ------>|
   |    GET /style.css          |                            |    GET /style.css      |
   |    (waits for response)    |                            |    (waits)             |
   |<-- response 1 ------------|<-- response 1 -------------|<-- response 1 ---------|
   |                            |                            |                       |
   |--- TCP conn #2 ---------->|--- stream #2 ------------->|--- TCP conn #2 ------>|
   |    GET /app.js             |                            |    GET /app.js         |
   |    (waits for response)    |                            |    (waits)             |
   |<-- response 2 ------------|<-- response 2 -------------|<-- response 2 ---------|
   |                            |                            |                       |
   |--- TCP conn #3 ---------->|--- stream #3 ------------->|--- TCP conn #3 ------>|
   |    GET /logo.png           |                            |    GET /logo.png       |
   |<-- response 3 ------------|<-- response 3 -------------|<-- response 3 ---------|
```

Six connections, six handshakes, six streams through the multiplexer. The tunnel does extra work for every parallel request.

### HTTP/2 Through a Tunnel: Multiplexed Model

```
Browser                    fxTunnel Server              fxTunnel Client         localhost:8080
   |                            |                            |                       |
   |=== single TCP + TLS =====>|=== single stream =========>|=== single TCP =======>|
   |                            |                            |                       |
   | stream 1: GET /style.css  | stream 1: GET /style.css   | stream 1: GET /style  |
   | stream 2: GET /app.js     | stream 2: GET /app.js      | stream 2: GET /app.js |
   | stream 3: GET /logo.png   | stream 3: GET /logo.png    | stream 3: GET /logo   |
   |                            |                            |                       |
   | <-- stream 3: logo.png    | <-- stream 3: logo.png     | <-- stream 3: logo    |
   | <-- stream 1: style.css   | <-- stream 1: style.css    | <-- stream 1: style   |
   | <-- stream 2: app.js      | <-- stream 2: app.js       | <-- stream 2: app.js  |
   |                            |                            |                       |
   |=== all done on 1 conn ===>|==========================>|========================>|
```

One connection, one TLS handshake, one stream through the multiplexer. Responses arrive in any order — whichever is ready first. The tunnel handles three times less connection overhead while delivering better performance to the user.

### Why This Matters for Tunnels Specifically

The gains from HTTP/2 multiplexing are amplified in a tunnel because every connection between the server and the client passes through a relay with additional latency. In HTTP/1.1, that latency is paid per connection. In HTTP/2, it is paid once. For a page with 30 assets, this can mean the difference between a snappy demo and a sluggish one.

fxTunnel's internal multiplexer ([described in the architecture article](/blog/fxtunnel-architecture/)) already multiplexes logical streams over a single TLS connection. When the incoming traffic is HTTP/2, the protocol's own multiplexing aligns naturally with the tunnel's internal architecture — the HTTP/2 connection from the browser maps to a single multiplexed stream inside the tunnel.

## Header Compression: HPACK and QPACK

HTTP headers are repetitive. Every request sends the same `User-Agent`, `Accept`, `Cookie`, and `Authorization` headers. In HTTP/1.1, these are sent as plain text on every request — often 500-800 bytes of repeated data per request.

HTTP/2 introduces HPACK compression, which uses a shared dictionary and Huffman encoding to compress headers. After the first request, subsequent headers that match entries in the dictionary are sent as a single-byte index instead of the full string.

HTTP/3 uses QPACK, an adaptation of HPACK that works with the unordered delivery of QUIC. It achieves similar compression ratios while avoiding head-of-line blocking in the header decompression step.

For tunneled traffic, header compression reduces bandwidth usage between the browser and the tunnel server. When you are running a live demo over a tunnel and dozens of API calls fire in quick succession, compressed headers reduce the total payload by 30-50%.

```
HTTP/1.1 headers (uncompressed):
  GET /api/users HTTP/1.1
  Host: demo.fxtun.dev
  User-Agent: Mozilla/5.0 (X11; Linux x86_64)...
  Accept: application/json
  Authorization: Bearer eyJhbGciOi...
  Cookie: session=abc123; theme=dark
  ~700 bytes per request

HTTP/2 HPACK (after first request):
  :method = GET         (index 2, 1 byte)
  :path = /api/users    (literal, ~12 bytes)
  :authority             (index, 1 byte)
  authorization          (index, 1 byte — value same)
  cookie                 (index, 1 byte — value same)
  ~50 bytes per request (93% reduction)
```

## HTTP/3 and QUIC: The Next Step for Tunneling

HTTP/3 replaces TCP with QUIC — a transport protocol built on top of UDP. This is not just a new framing format: it is a fundamentally different approach to connection management that solves problems TCP cannot.

### Why TCP Is a Problem for Multiplexed Protocols

HTTP/2 solved head-of-line blocking at the HTTP layer, but TCP introduced its own. When a single TCP packet is lost, the operating system's TCP stack stalls **all** streams on that connection until the lost packet is retransmitted. In HTTP/2, this means one dropped packet blocks every in-flight request — the image, the script, and the API call all freeze together.

{{< diagram >}}
HTTP/2 over TCP — head-of-line blocking:

  Stream 1 (style.css):   [====]  [====]  [====]
  Stream 2 (app.js):      [====]  [====]  [====]
  Stream 3 (logo.png):    [====]  [==X=]  ← packet lost
                                    |
                          ALL STREAMS BLOCKED
                          waiting for retransmission
                                    |
  Stream 1:               ......... [====]  resume
  Stream 2:               ......... [====]  resume
  Stream 3:               ......... [====]  resume


HTTP/3 over QUIC — independent streams:

  Stream 1 (style.css):   [====]  [====]  [====]  [====]  ← unaffected
  Stream 2 (app.js):      [====]  [====]  [====]  [====]  ← unaffected
  Stream 3 (logo.png):    [====]  [==X=]  ← lost, retransmitting
                                    |
                          ONLY STREAM 3 BLOCKED
                                    |
  Stream 3:               ......... [====]  resume
{{< /diagram >}}

For tunneling, this matters because the relay adds an extra network hop where packet loss can occur. On a mobile network or a congested Wi-Fi connection, HTTP/3 through a tunnel keeps most streams flowing even when individual packets are lost.

### 0-RTT Connection Resumption

QUIC supports 0-RTT connection resumption: when reconnecting to the same server, the client can send application data in the very first packet. Combined with the fact that QUIC merges the transport and TLS handshakes, a fresh connection takes 1 RTT, and a resumed one takes 0 RTT.

```
HTTP/1.1 + TLS 1.2:  TCP handshake (1 RTT) + TLS handshake (2 RTT) = 3 RTT
HTTP/2 + TLS 1.3:    TCP handshake (1 RTT) + TLS handshake (1 RTT) = 2 RTT
HTTP/3 + QUIC:       QUIC handshake (1 RTT, includes TLS)          = 1 RTT
HTTP/3 + QUIC 0-RTT: Resumed connection                            = 0 RTT
```

For tunnel users, 0-RTT means that when a browser re-opens a connection to a tunneled service (common in single-page apps that make periodic API calls), the request arrives with zero round-trip overhead. On a 100 ms latency link, that is 200-300 ms saved on reconnection compared to HTTP/1.1. The [TLS 1.3 in Tunneling](/blog/tls-13-tunnel-security/) article covers the handshake mechanics in more detail.

### Connection Migration

One of QUIC's unique features is connection migration. Traditional TCP connections are bound to a 4-tuple: source IP, source port, destination IP, destination port. If any of these change (switching from Wi-Fi to cellular, for example), the connection breaks and must be re-established.

QUIC uses connection IDs instead of IP addresses to identify connections. When a mobile device switches networks, the connection survives — the client simply sends packets from the new IP, and the server recognizes the connection ID.

For tunneling on mobile networks, this is transformative. A developer testing a mobile app through a tunnel can walk between Wi-Fi access points without the tunnel dropping. An [IoT device](/blog/iot-tunnel-devices-cloud/) with an unstable connection maintains its tunnel without reconnection delays.

## How fxTunnel Handles HTTP/2

fxTunnel's HTTP tunnels support HTTP/2 out of the box. The protocol is negotiated automatically via ALPN (Application-Layer Protocol Negotiation) during the TLS handshake between the browser and the fxTunnel server. If the browser supports HTTP/2 (all modern browsers do), the tunnel uses HTTP/2.

On the internal leg between the fxTunnel client and your local server, the protocol depends on your server's capabilities. If your local server supports HTTP/2 (Node.js with `--http2`, Go `net/http`, Nginx), fxTunnel will use HTTP/2 end-to-end. If it does not, fxTunnel downgrades to HTTP/1.1 for the local connection while still using HTTP/2 on the public side.

{{< diagram >}}
Browser ← HTTP/2 → fxTunnel Server ← multiplexed → fxTunnel Client ← HTTP/2 or 1.1 → localhost
          (ALPN)                     (TLS conn)                       (depends on
                                                                       local server)
{{< /diagram >}}

### Verifying HTTP/2 with curl

You can verify that your tunnel is serving HTTP/2 using curl:

```bash
# Create an HTTP tunnel
fxtunnel http 8080
# -> https://abc123.fxtun.dev

# Test with HTTP/2 (curl 7.47+)
curl -v --http2 https://abc123.fxtun.dev 2>&1 | grep "< HTTP/"
# -> < HTTP/2 200

# See detailed connection info
curl -w "Protocol: %{http_version}\n" -o /dev/null -s https://abc123.fxtun.dev
# -> Protocol: 2
```

### Verifying in Browser DevTools

Open your tunnel URL in Chrome or Firefox, then:

1. Open DevTools (F12)
2. Go to the Network tab
3. Right-click the column headers and enable "Protocol"
4. Reload the page
5. The Protocol column shows `h2` for HTTP/2 or `h3` for HTTP/3

{{< diagram >}}
Name              Status  Protocol  Size     Time
──────────────────────────────────────────────────
abc123.fxtun.dev  200     h2        1.2 KB   45 ms
style.css         200     h2        8.4 KB   12 ms
app.js            200     h2        24 KB    18 ms
logo.png          200     h2        5.1 KB   9 ms
api/data          200     h2        420 B    23 ms
{{< /diagram >}}

All requests share a single TCP connection — visible in the Connection ID column if enabled.

## Practical Performance: HTTP/1.1 vs HTTP/2 Through a Tunnel

Real-world performance depends on the type of application. Here are benchmarks comparing HTTP/1.1 and HTTP/2 through an fxTunnel tunnel on a typical 50 ms latency connection.

| Scenario | HTTP/1.1 | HTTP/2 | Improvement |
|---|---|---|---|
| **Single API call** | 120 ms | 115 ms | ~4% (minimal difference) |
| **10 parallel API calls** | 380 ms (6 connections) | 140 ms (1 connection) | ~63% faster |
| **Page with 30 assets** | 1,200 ms | 520 ms | ~57% faster |
| **WebSocket upgrade** | 180 ms | 170 ms | ~6% (minimal difference) |
| **Repeated requests (warm)** | 110 ms | 105 ms | ~5% (minimal difference) |

The takeaway: HTTP/2 shines when there are many parallel requests. For single sequential API calls, the difference is negligible. For asset-heavy pages or parallel API workflows, HTTP/2 through a tunnel delivers dramatic improvements.

## Using fxTunnel with HTTP/2 in Practice

### Local Development Server with HTTP/2

Most modern frameworks support HTTP/2 natively. Here is how to set up common development servers:

```bash
# Node.js (Express does not support HTTP/2 directly, use spdy or http2)
node --experimental-modules server.mjs

# Go (net/http supports HTTP/2 automatically with TLS)
go run main.go

# Python (Hypercorn with HTTP/2)
hypercorn app:app --bind 0.0.0.0:8080

# Then expose through fxTunnel
fxtunnel http 8080
# -> https://abc123.fxtun.dev (HTTP/2 ready)
```

### Testing Webhook Delivery with HTTP/2

When a webhook provider sends callbacks to your tunnel, HTTP/2 multiplexing lets multiple webhook deliveries arrive simultaneously without queuing:

```bash
# Start tunnel for webhook testing
fxtunnel http 3000
# -> https://abc123.fxtun.dev

# Configure webhook provider to point to your tunnel URL
# Stripe, GitHub, Slack — all support HTTP/2 for webhook delivery

# Monitor incoming requests with the fxTunnel inspector
# Available with Pro plan (from $5/mo)
```

The [request inspector](/blog/traffic-inspector-debug-requests/) captures every HTTP request with full headers, body, and timing data -- handy for watching how multiple webhook deliveries interleave over a single connection.

### CI/CD Preview Environments

When [using tunnels for CI/CD preview environments](/blog/cicd-tunnel-preview-environments/), HTTP/2 support means your preview URLs behave identically to production. Reviewers see the same performance characteristics, and integration tests running through the tunnel benefit from multiplexed connections.

```bash
# In CI pipeline
fxtunnel http 3000 --subdomain pr-${PR_NUMBER}
# -> https://pr-42.fxtun.dev (HTTP/2 with custom subdomain)
```

## What About HTTP/3 in Tunnels?

HTTP/3 support in tunneling tools is still emerging. The protocol runs over QUIC, which uses UDP — a different transport that requires changes to how tunnels handle traffic. fxTunnel already supports [UDP tunneling](/blog/tcp-udp-tunneling-explained/), which provides the foundation for future HTTP/3 support.

The key challenge is that HTTP/3 traffic arrives as UDP datagrams, not TCP streams. A tunnel must either:

1. **Terminate QUIC at the edge** — the fxTunnel server accepts HTTP/3 from the browser, converts it to HTTP/2 or HTTP/1.1 for the internal leg, and the local server receives a traditional HTTP connection.
2. **Tunnel QUIC end-to-end** — the tunnel forwards raw UDP datagrams containing QUIC packets, and the local server handles HTTP/3 directly.

Approach 1 is simpler and works with any local server. Approach 2 preserves the benefits of QUIC (0-RTT, connection migration) end-to-end but requires the local server to support HTTP/3.

As HTTP/3 adoption grows, fxTunnel will implement edge termination first, allowing browsers to use HTTP/3 to connect to tunnel URLs while maintaining compatibility with any local server. End-to-end QUIC tunneling will follow.

## Best Practices for HTTP/2 and HTTP/3 with Tunnels

### 1. Enable HTTP/2 on Your Local Server

The tunnel can only use HTTP/2 end-to-end if your local server supports it. Most modern frameworks do — check your framework's documentation for HTTP/2 configuration.

### 2. Use a Single Domain

HTTP/2 multiplexing works best when all requests go to a single domain. With fxTunnel, all requests to your tunnel URL share a single connection. Avoid splitting traffic across multiple tunnel URLs unnecessarily.

### 3. Leverage Server Push Carefully

HTTP/2 server push lets you proactively send assets to the browser before it requests them. Through a tunnel, this can reduce perceived latency — but over-pushing wastes bandwidth. Use push for critical CSS and JavaScript only.

### 4. Monitor Protocol Negotiation

Use browser DevTools or curl to verify that your tunnel is using HTTP/2. If you see `http/1.1` in the Protocol column, check your local server's HTTP/2 configuration.

### 5. Test on Real Networks

HTTP/2 and HTTP/3 performance gains are most visible on high-latency or lossy networks. Test your tunnel from a mobile device or a connection with simulated latency to see the real difference.

## FAQ

### Does fxTunnel support HTTP/2 and HTTP/3?

HTTP/2 works out of the box -- fxTunnel preserves multiplexing, header compression, and server push when a client connects using the protocol. HTTP/3 over QUIC is planned for a future release. Because TLS terminates at the edge, the local leg of the tunnel can speak whatever protocol your server supports, including HTTP/1.1.

### Will HTTP/2 make my tunnel faster than HTTP/1.1?

For single sequential requests the difference is small. Where HTTP/2 really pays off is parallel workloads: pages with many assets or APIs handling several requests at once can load 20-50% faster, because multiplexing removes the per-connection overhead that HTTP/1.1 imposes through the tunnel.

### What is the difference between HTTP/2 and HTTP/3 for tunneling?

Both protocols multiplex streams, but HTTP/2 still sits on TCP, so a single lost packet stalls every stream on that connection. HTTP/3 swaps TCP for QUIC (UDP), giving each stream its own loss recovery. Add 0-RTT connection resumption and connection migration, and you get a protocol that handles lossy or mobile networks much more gracefully.

### Can I force HTTP/2 through an fxTunnel tunnel?

You do not need to force it -- ALPN negotiation during the TLS handshake picks HTTP/2 automatically if both the client and your local server support it. To confirm it is active, check the Protocol column in browser DevTools or run `curl --http2` against the tunnel URL.

### Why does HTTP/3 matter for tunneling over mobile networks?

Mobile connections suffer from packet loss and frequent IP changes as devices hop between cell towers. QUIC handles both gracefully: independent streams keep unaffected traffic flowing when a packet is lost, and connection migration lets sessions survive an IP change without renegotiating the connection. For a tunnel, that means fewer interruptions on the go.
