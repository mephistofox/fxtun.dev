---
title: "WebSocket Through a Tunnel — Real-Time on Localhost"
date: 2026-03-21T14:00:00+03:00
draft: false
description: "How WebSockets work through a tunnel: HTTP Upgrade, fxTunnel support, chat app example, Socket.IO integration, and latency tips."
tags: [websocket, tunnel, realtime, socket.io, fxTunnel, localhost, developer-tools, http-upgrade]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Does fxTunnel support WebSocket connections?"
    a: "Yes. fxTunnel fully supports WebSocket connections through its HTTP tunnel. The tunnel transparently handles the HTTP Upgrade handshake and then proxies the bidirectional WebSocket frames between the client and your local server. No special configuration is needed — just run `fxtunnel http 8080`."
  - q: "How much latency does a WebSocket tunnel add?"
    a: "fxTunnel adds +1–5 ms of latency on top of the base network round-trip time. For most real-time applications — chats, notifications, dashboards, collaborative editors — this overhead is imperceptible. For latency-critical use cases like competitive multiplayer games, consider a direct connection or a UDP tunnel."
  - q: "Can I use Socket.IO through a fxTunnel tunnel?"
    a: "Yes. Socket.IO works through fxTunnel out of the box. Socket.IO starts with HTTP long-polling and then upgrades to WebSocket — fxTunnel handles both transports transparently. Make sure your Socket.IO client connects to the public tunnel URL instead of localhost."
  - q: "Why is WebSocket better than HTTP polling for real-time apps?"
    a: "HTTP polling creates a new request every few seconds, wasting bandwidth and adding latency. WebSocket establishes a single persistent connection where the server can push data instantly. This reduces latency from seconds to milliseconds, cuts bandwidth usage by 10–100x, and eliminates unnecessary server load."
  - q: "Do I need a paid plan to use WebSocket tunnels in fxTunnel?"
    a: "No. WebSocket support is included in the free tier with no connection or bandwidth limits. Paid plans from $5/mo add custom domains, the request inspector with replay, and other features — but WebSocket tunneling works on every plan."
---

## WebSocket Through a Tunnel: Real-Time Apps on Localhost

WebSocket is the backbone of real-time web applications -- chats, live dashboards, collaborative editors, notifications, and multiplayer games. But developing WebSocket apps locally creates a familiar problem: your `localhost:8080` is not reachable from the internet, so you cannot test with external clients, mobile devices, or third-party services. A [tunnel](/blog/what-is-tunneling/) gives your local server a public URL that fully supports WebSocket connections.

Below we look at how WebSocket works through a tunnel, how fxTunnel handles the HTTP Upgrade mechanism, and walk through building a real-time chat with `ws` and Socket.IO -- all running on localhost and accessible from anywhere.

## How WebSocket Works: A Quick Refresher

WebSocket is a communication protocol that provides full-duplex, bidirectional communication over a single TCP connection. Unlike HTTP, where the client sends a request and waits for a response, WebSocket allows both the client and server to send messages at any time without waiting.

The protocol starts as an ordinary HTTP request. The client sends a special `Upgrade` header, and the server responds with `101 Switching Protocols`. After that, the connection is no longer HTTP — it becomes a persistent WebSocket channel.

### The HTTP Upgrade Handshake

```
Client → Server (HTTP Request):
GET /chat HTTP/1.1
Host: abc123.fxtun.dev
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13

Server → Client (HTTP Response):
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=

--- Connection upgraded ---
--- Bidirectional WebSocket frames flow freely ---
```

After the upgrade, both sides exchange binary or text frames over the same TCP connection. There are no more HTTP requests or responses — just raw frames.

### WebSocket vs HTTP Polling

Many developers start with HTTP polling before discovering WebSocket. Here is how the two approaches compare:

| Characteristic | HTTP Polling | WebSocket |
|---|---|---|
| **Connection** | New request every N seconds | Single persistent connection |
| **Direction** | Client pulls from server | Bidirectional (push + pull) |
| **Latency** | 1–10 seconds (poll interval) | < 50 ms (instant push) |
| **Bandwidth** | High (repeated headers) | Low (small frames) |
| **Server load** | High (many short connections) | Low (one long connection) |
| **Complexity** | Simple to implement | Slightly more complex |
| **Use cases** | Simple dashboards, status checks | Chat, gaming, collaboration |

For anything that requires real-time interaction — chat messages, cursor positions, live price feeds — WebSocket is the right choice. HTTP polling wastes bandwidth and adds seconds of latency on every update.

## The Problem: WebSocket on Localhost

When building a real-time application, you naturally start on localhost. Your WebSocket server listens on `ws://localhost:8080`, and your client connects to it directly. This works fine for solo development, but breaks the moment you need to:

- **Test from a mobile device** — your phone cannot reach `localhost` on your development machine.
- **Share a preview** — a colleague or a client cannot open your local URL.
- **Integrate with external services** — services like Slack, Discord bots, or IoT devices need a public endpoint.
- **Test across networks** — WebSocket behavior can differ between local and remote connections due to proxies, firewalls, and NAT.
- **Demo without deploying** — you want to show a working prototype without pushing code to a server.

A tunnel solves all of these. The [expose localhost to the internet](/blog/expose-localhost-to-internet/) guide covers the general approach in detail.

## How WebSocket Works Through a fxTunnel Tunnel

Do you need a special flag to enable WebSocket? No. fxTunnel supports WebSocket natively through its HTTP tunnel. When an external client connects to your public tunnel URL and sends an HTTP Upgrade request, fxTunnel forwards the handshake to your local server, and after the upgrade, it proxies WebSocket frames bidirectionally.

### Connection Flow Diagram

```
Browser / Mobile / External Client
        |
        | 1. HTTP GET with Upgrade: websocket
        v
+------------------------------+
|  fxTunnel Server             |
|  https://abc123.fxtun.dev    |
|                              |
|  2. Forward Upgrade request  |
|     through multiplexer      |
|                              |
|  5. Proxy WebSocket frames   |
|     bidirectionally           |
+--------------+---------------+
               | TLS connection (multiplexed)
               v
+------------------------------+
|  fxTunnel Client             |
|                              |
|  3. Forward Upgrade to       |
|     localhost:8080            |
|                              |
|  4. Local server responds    |
|     101 Switching Protocols  |
|                              |
|  5. Proxy WebSocket frames   |
|     bidirectionally           |
+------------------------------+
               |
               v
        localhost:8080
        (your WebSocket server)
```

The key insight is that after the HTTP Upgrade handshake completes, the connection becomes a raw TCP stream carrying WebSocket frames. fxTunnel's multiplexer treats it as a long-lived HTTP stream — no different from a chunked response or a server-sent events connection. For details on how multiplexing works, see [fxTunnel Architecture](/blog/fxtunnel-architecture/).

### What Happens Under the Hood

1. **External client** sends an HTTP GET request with `Upgrade: websocket` to `https://abc123.fxtun.dev/chat`.
2. **fxTunnel server** receives the request, sees the Upgrade header, and forwards the entire request through the multiplexer to the fxTunnel client. Critically, the server does not terminate the Upgrade — it passes it through.
3. **fxTunnel client** opens a connection to `localhost:8080` and forwards the Upgrade request.
4. **Your local server** responds with `101 Switching Protocols`.
5. **The response flows back** through the chain: local server -> fxTunnel client -> fxTunnel server -> external client.
6. **Both sides** now exchange WebSocket frames. The tunnel proxies frames in both directions without inspecting or modifying them.
7. **When either side closes** the WebSocket, the close frame propagates through the entire chain, and the multiplexer stream is released.

This process is fully transparent. Your local server sees a normal WebSocket connection with standard headers. The external client sees a normal WebSocket endpoint at a public HTTPS URL with valid TLS — exactly what browsers and mobile apps require.

## Building a Real-Time Chat: Step by Step

Let us build a simple but complete real-time chat application and expose it through fxTunnel. This demonstrates the full workflow: local development, tunnel setup, and testing from multiple devices.

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installation
fxtunnel --version
```

### Step 2. Create the WebSocket Server

We will use the `ws` library — the most popular WebSocket implementation for Node.js.

```bash
mkdir ws-chat && cd ws-chat
npm init -y
npm install ws
```

```javascript
// server.js
const http = require('http');
const fs = require('fs');
const WebSocket = require('ws');

// Create an HTTP server for serving the HTML page
const server = http.createServer((req, res) => {
  if (req.url === '/' || req.url === '/index.html') {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(fs.readFileSync('index.html'));
  } else {
    res.writeHead(404);
    res.end('Not found');
  }
});

// Create a WebSocket server attached to the HTTP server
const wss = new WebSocket.Server({ server });

// Track connected clients
const clients = new Set();

wss.on('connection', (ws, req) => {
  const clientId = req.headers['x-forwarded-for'] || req.socket.remoteAddress;
  console.log(`Client connected: ${clientId}`);
  clients.add(ws);

  // Send welcome message
  ws.send(JSON.stringify({
    type: 'system',
    message: `Welcome! ${clients.size} user(s) online.`,
    timestamp: new Date().toISOString(),
  }));

  // Broadcast to all other clients
  broadcast(ws, {
    type: 'system',
    message: 'A new user joined the chat.',
    timestamp: new Date().toISOString(),
  });

  // Handle incoming messages
  ws.on('message', (data) => {
    const parsed = JSON.parse(data);
    console.log(`Message from ${clientId}: ${parsed.message}`);

    // Broadcast the message to all clients (including sender)
    const outgoing = {
      type: 'message',
      user: parsed.user || 'Anonymous',
      message: parsed.message,
      timestamp: new Date().toISOString(),
    };

    for (const client of clients) {
      if (client.readyState === WebSocket.OPEN) {
        client.send(JSON.stringify(outgoing));
      }
    }
  });

  // Handle disconnection
  ws.on('close', () => {
    clients.delete(ws);
    console.log(`Client disconnected: ${clientId}`);
    broadcast(null, {
      type: 'system',
      message: 'A user left the chat.',
      timestamp: new Date().toISOString(),
    });
  });
});

function broadcast(excludeWs, data) {
  for (const client of clients) {
    if (client !== excludeWs && client.readyState === WebSocket.OPEN) {
      client.send(JSON.stringify(data));
    }
  }
}

server.listen(8080, () => {
  console.log('Chat server running on http://localhost:8080');
});
```

### Step 3. Create the Client HTML

```html
<!-- index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>WebSocket Chat</title>
  <style>
    body { font-family: sans-serif; max-width: 600px; margin: 40px auto; }
    #messages { border: 1px solid #ccc; height: 400px; overflow-y: auto; padding: 10px; }
    .system { color: #888; font-style: italic; }
    .message { margin: 4px 0; }
    #form { display: flex; gap: 8px; margin-top: 8px; }
    #form input { flex: 1; padding: 8px; }
    #form button { padding: 8px 16px; }
  </style>
</head>
<body>
  <h1>WebSocket Chat</h1>
  <div id="messages"></div>
  <form id="form">
    <input id="user" placeholder="Your name" value="" />
    <input id="input" placeholder="Type a message..." autocomplete="off" />
    <button type="submit">Send</button>
  </form>
  <script>
    // Connect to WebSocket — works with both localhost and tunnel URL
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${location.host}`);

    const messages = document.getElementById('messages');

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const div = document.createElement('div');

      if (data.type === 'system') {
        div.className = 'system';
        div.textContent = data.message;
      } else {
        div.className = 'message';
        div.textContent = `[${data.user}]: ${data.message}`;
      }

      messages.appendChild(div);
      messages.scrollTop = messages.scrollHeight;
    };

    ws.onclose = () => {
      const div = document.createElement('div');
      div.className = 'system';
      div.textContent = 'Connection closed.';
      messages.appendChild(div);
    };

    document.getElementById('form').onsubmit = (e) => {
      e.preventDefault();
      const input = document.getElementById('input');
      const user = document.getElementById('user').value || 'Anonymous';

      if (input.value.trim()) {
        ws.send(JSON.stringify({ user, message: input.value }));
        input.value = '';
      }
    };
  </script>
</body>
</html>
```

Notice that the client code uses `location.host` to determine the WebSocket URL. This means it works transparently whether you access it at `localhost:8080` or through the tunnel at `abc123.fxtun.dev` — no hardcoded URLs.

### Step 4. Run the Server and Create a Tunnel

```bash
# Terminal 1: Start the chat server
node server.js
# -> Chat server running on http://localhost:8080

# Terminal 2: Create a tunnel
fxtunnel http 8080
```

fxTunnel prints the public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://ws-chat.fxtun.dev
Forwarding:  https://ws-chat.fxtun.dev -> http://localhost:8080

Press Ctrl+C to stop
```

### Step 5. Test

1. Open `https://ws-chat.fxtun.dev` in your browser — the chat loads and the WebSocket connects.
2. Open the same URL on your phone — you see the second user joining.
3. Send messages — they appear instantly on both devices.
4. Share the URL with a colleague — they can join the chat from their own network.

The WebSocket connection is fully encrypted (WSS) on the public side thanks to the tunnel's TLS certificate. Your local server runs plain `ws://` — the tunnel handles the TLS termination.

## Socket.IO Through a Tunnel

Socket.IO is the most popular real-time framework for Node.js. It adds automatic reconnection, rooms, namespaces, and fallback to HTTP long-polling when WebSocket is unavailable. Socket.IO works through fxTunnel without any special configuration.

### How Socket.IO Uses WebSocket

Socket.IO uses a two-phase connection strategy:

1. **Phase 1: HTTP long-polling** — Socket.IO starts with regular HTTP requests to establish the connection and exchange metadata.
2. **Phase 2: WebSocket upgrade** — once the connection is established, Socket.IO upgrades to WebSocket for better performance.

```
Socket.IO Connection Timeline:

Client                    Tunnel                    Server
  |                         |                         |
  |--- HTTP POST (poll) --->|--- HTTP POST (poll) --->|
  |<-- HTTP 200 (sid) ------|<-- HTTP 200 (sid) ------|
  |                         |                         |
  |--- HTTP GET (poll) ---->|--- HTTP GET (poll) ---->|
  |<-- HTTP 200 (data) -----|<-- HTTP 200 (data) -----|
  |                         |                         |
  |--- GET Upgrade: ws ---->|--- GET Upgrade: ws ---->|
  |<-- 101 Switching -------|<-- 101 Switching -------|
  |                         |                         |
  |<===== WebSocket =======>|<===== WebSocket =======>|
  |   (bidirectional)       |   (bidirectional)       |
```

fxTunnel handles both phases transparently — the HTTP polling requests and the WebSocket upgrade. This is important because some tunneling tools break Socket.IO by failing to forward the Upgrade header correctly.

### Socket.IO Server Example

```bash
npm install express socket.io
```

```javascript
// socketio-server.js
const express = require('express');
const http = require('http');
const { Server } = require('socket.io');

const app = express();
const server = http.createServer(app);
const io = new Server(server, {
  cors: { origin: '*' },
});

app.get('/', (req, res) => {
  res.sendFile(__dirname + '/socketio-client.html');
});

// Track online users
let onlineCount = 0;

io.on('connection', (socket) => {
  onlineCount++;
  console.log(`User connected (${onlineCount} online)`);

  // Notify everyone about the new count
  io.emit('online-count', onlineCount);

  // Handle chat messages
  socket.on('chat-message', (data) => {
    console.log(`${data.user}: ${data.message}`);
    // Broadcast to everyone (including sender)
    io.emit('chat-message', {
      user: data.user,
      message: data.message,
      timestamp: new Date().toISOString(),
    });
  });

  // Handle typing indicator
  socket.on('typing', (user) => {
    socket.broadcast.emit('typing', user);
  });

  // Handle disconnection
  socket.on('disconnect', () => {
    onlineCount--;
    console.log(`User disconnected (${onlineCount} online)`);
    io.emit('online-count', onlineCount);
  });
});

server.listen(8080, () => {
  console.log('Socket.IO server running on http://localhost:8080');
});
```

### Socket.IO Client Example

```html
<!-- socketio-client.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Socket.IO Chat</title>
  <script src="/socket.io/socket.io.js"></script>
</head>
<body>
  <h1>Socket.IO Chat <span id="count"></span></h1>
  <div id="messages" style="height:300px; overflow-y:auto; border:1px solid #ccc; padding:10px;"></div>
  <p id="typing" style="color:#888; height:20px;"></p>
  <form id="form">
    <input id="user" placeholder="Your name" />
    <input id="input" placeholder="Message..." autocomplete="off" />
    <button type="submit">Send</button>
  </form>
  <script>
    // Socket.IO auto-detects the server URL from the page origin
    const socket = io();

    socket.on('online-count', (count) => {
      document.getElementById('count').textContent = `(${count} online)`;
    });

    socket.on('chat-message', (data) => {
      const div = document.createElement('div');
      div.textContent = `[${data.user}]: ${data.message}`;
      document.getElementById('messages').appendChild(div);
    });

    socket.on('typing', (user) => {
      document.getElementById('typing').textContent = `${user} is typing...`;
      setTimeout(() => {
        document.getElementById('typing').textContent = '';
      }, 2000);
    });

    document.getElementById('input').addEventListener('input', () => {
      const user = document.getElementById('user').value || 'Anonymous';
      socket.emit('typing', user);
    });

    document.getElementById('form').onsubmit = (e) => {
      e.preventDefault();
      const input = document.getElementById('input');
      const user = document.getElementById('user').value || 'Anonymous';
      if (input.value.trim()) {
        socket.emit('chat-message', { user, message: input.value });
        input.value = '';
      }
    };
  </script>
</body>
</html>
```

### Running Socket.IO Through fxTunnel

```bash
# Terminal 1: Start the server
node socketio-server.js

# Terminal 2: Create a tunnel
fxtunnel http 8080
```

Socket.IO automatically detects the server URL from `window.location`, so the client code works without changes whether you access it via `localhost:8080` or `https://abc123.fxtun.dev`.

## Latency Considerations for WebSocket Tunnels

Latency is the most important factor for real-time applications. A tunnel adds an extra network hop between the client and your server, which increases round-trip time. Understanding this overhead helps you decide when a tunnel is appropriate and when it is not.

### Where the Latency Comes From

{{< diagram >}}
Without tunnel:
Client ──── (network) ──── Server
RTT: ~50 ms (same city)

With tunnel:
Client ──── (network) ──── fxTunnel Server ──── (TLS mux) ──── fxTunnel Client ──── Server
RTT: ~50 ms + 1-5 ms overhead
{{< /diagram >}}

The tunnel adds three sources of latency:

1. **Extra network hop** — data travels client -> tunnel server -> tunnel client -> local server (and back). If the tunnel server is close to both endpoints, this adds minimal latency.
2. **TLS encryption/decryption** — each frame is encrypted on one end and decrypted on the other. Modern hardware handles TLS 1.3 with negligible overhead.
3. **Multiplexer framing** — data is wrapped in multiplexer frames for transport. This adds a few bytes of overhead per frame.

### Measured Overhead

| Scenario | Without tunnel | With fxTunnel | Overhead |
|---|---|---|---|
| Same city (client & server) | 5–15 ms | 6–20 ms | +1–5 ms |
| Same country | 20–50 ms | 21–55 ms | +1–5 ms |
| Cross-continent | 100–200 ms | 101–205 ms | +1–5 ms |

The +1-5 ms overhead is constant regardless of the base latency. For most real-time applications — chat, notifications, dashboards, collaborative editing — this overhead is unnoticeable.

### When Tunnel Latency Matters

For the vast majority of real-time web applications, a tunnel's latency overhead is negligible. Chat messages that arrive in 22 ms instead of 20 ms are indistinguishable to users. However, for some specific use cases, every millisecond counts:

- **Competitive multiplayer games** — frame-perfect input requires sub-10ms latency. Use a [UDP tunnel](/blog/tcp-udp-tunneling-explained/) instead.
- **High-frequency trading** — not a typical localhost dev scenario, but latency-sensitive by nature.
- **Audio/video real-time processing** — consider WebRTC with direct peer connections for production.

For development and testing, the tunnel overhead is always acceptable. You are testing functionality, not measuring production latency.

## WebSocket Tunnel vs Polling: Bandwidth Comparison

One of WebSocket's biggest advantages over polling becomes even more significant through a tunnel. Polling generates far more traffic, which matters both for bandwidth costs and for tunnel performance.

### Traffic Comparison: Chat with 100 Messages

```
HTTP Polling (every 2 seconds, 30-second session):
  15 poll requests × ~500 bytes headers = 7,500 bytes overhead
  100 messages × ~200 bytes each        = 20,000 bytes payload
  Total: ~27,500 bytes
  Connections opened: 15

WebSocket:
  1 Upgrade request                     = ~500 bytes
  100 messages × ~210 bytes (frame)     = 21,000 bytes payload
  Total: ~21,500 bytes
  Connections opened: 1
```

WebSocket uses 22% less bandwidth in this simple example. In real-world scenarios with frequent small updates (typing indicators, cursor positions, presence), the difference grows to 10-100x because polling sends full HTTP headers with every request.

Through a tunnel, this difference is amplified. Each HTTP poll request goes through the full tunnel chain (TLS encryption, multiplexer framing, forwarding). A single WebSocket connection goes through the chain once and then streams data efficiently.

## Debugging WebSocket Connections Through a Tunnel

Debugging real-time applications requires specialized tools. fxTunnel's Inspector records all HTTP traffic, including the WebSocket Upgrade handshake. For deeper WebSocket frame inspection, combine it with browser DevTools.

### Using Browser DevTools

1. Open DevTools (F12) in your browser.
2. Go to the **Network** tab.
3. Filter by **WS** to see WebSocket connections.
4. Click on the WebSocket connection to see individual frames.
5. The **Messages** tab shows sent and received frames in real time.

### Using fxTunnel Inspector

The fxTunnel Inspector captures the initial HTTP Upgrade request, which is useful for debugging connection failures:

- **401/403 errors** — authentication issues before the upgrade.
- **400 Bad Request** — malformed Upgrade headers.
- **502 Bad Gateway** — local server is not running or is not accepting WebSocket connections.
- **Connection timeout** — the local server is not responding to the Upgrade request.

The Replay feature lets you re-send the Upgrade request to test different scenarios without reconnecting the client. The [Traffic Inspector](/blog/traffic-inspector-debug-requests/) article covers the full workflow.

### Common Issues and Solutions

| Problem | Cause | Solution |
|---|---|---|
| WebSocket connection fails | Local server not running | Start your server before creating the tunnel |
| `ws://` connection rejected | Mixed content (HTTPS page, WS protocol) | Use `wss://` — the tunnel provides TLS automatically |
| Connection drops after 60s | Idle timeout | Implement ping/pong frames (most libraries do this by default) |
| Socket.IO falls back to polling | Upgrade header not forwarded | Verify your tunnel tool supports WebSocket — fxTunnel does |
| CORS errors on connect | Missing CORS headers | Configure `cors: { origin: '*' }` in Socket.IO or add CORS headers |
| Multiple clients get the same data | Broadcasting logic error | Check your server-side broadcast logic |

## Real-World Use Cases

WebSocket tunnels are not just for chat demos. Here are practical scenarios where developers use WebSocket through fxTunnel every day.

### Live Dashboards and Monitoring

Push real-time metrics to a dashboard running on localhost. Share the tunnel URL with your team so everyone can watch deployment progress, server metrics, or CI/CD pipeline status in real time.

```javascript
// Push server metrics every second
setInterval(() => {
  const metrics = {
    cpu: os.loadavg()[0],
    memory: process.memoryUsage().heapUsed / 1024 / 1024,
    uptime: process.uptime(),
    timestamp: Date.now(),
  };
  io.emit('metrics', metrics);
}, 1000);
```

### Collaborative Editing

Build a collaborative text editor or whiteboard. Each user's cursor position and edits are broadcast to all connected clients. The tunnel lets you test with multiple users on different devices and networks — essential for finding synchronization bugs that only appear with real network latency.

### IoT Device Testing

An IoT device sends sensor data over WebSocket to your local server. By tunneling the WebSocket endpoint, the device can connect from any network without deploying your server. The [IoT Tunneling](/blog/iot-tunnel-devices-cloud/) article covers this pattern in more depth.

### Mobile App Development

Your mobile app connects to a WebSocket API on your development machine. The tunnel gives you a stable HTTPS URL that works from both the iOS simulator and a physical Android device on a different WiFi network. For more, see [Mobile App Testing with a Tunnel](/blog/mobile-app-testing-tunnel/).

## Pricing and Plans

WebSocket support is included on every fxTunnel plan, including the free tier.

| Plan | Price | WebSocket Features |
|---|---|---|
| **Free** | $0 | Full WebSocket support, no connection or bandwidth limits |
| **Pro** | from $5/mo | Custom domains, Inspector (view Upgrade requests), Replay |
| **Team** | from $10/mo | 10+ concurrent tunnels, priority support |

For most development workflows, the free tier covers everything you need. If you are debugging tricky connection issues, the Inspector on the Pro plan captures the Upgrade handshake and shows exactly where things go wrong. Teams running multiple WebSocket microservices can use the Team plan for 10+ simultaneous tunnels.

## FAQ

### Does fxTunnel support WebSocket connections?

Yes -- WebSocket works through fxTunnel's HTTP tunnel with zero extra configuration. The tunnel handles the HTTP Upgrade handshake transparently and then proxies frames in both directions between the external client and your local server. Just run `fxtunnel http 8080` and connect.

### How much latency does a WebSocket tunnel add?

Expect roughly +1-5 ms on top of whatever the base network round-trip time is. For chats, notifications, dashboards, and collaborative editors that overhead is imperceptible. If you are building something where every millisecond counts, like a competitive multiplayer game, look into a direct connection or a [UDP tunnel](/blog/tcp-udp-tunneling-explained/) instead.

### Can I use Socket.IO through a fxTunnel tunnel?

Yes, and it works out of the box. Socket.IO's two-phase connection -- HTTP long-polling first, then a WebSocket upgrade -- is handled transparently by fxTunnel. The only thing to watch is that your Socket.IO client connects to the public tunnel URL rather than localhost.

### Why is WebSocket better than HTTP polling for real-time apps?

Polling opens a new HTTP request every few seconds, which wastes bandwidth and introduces seconds of delay. A WebSocket connection stays open, letting the server push data the moment it is available. The result is latency measured in milliseconds instead of seconds, and dramatically lower bandwidth and server load.

### Do I need a paid plan to use WebSocket tunnels in fxTunnel?

No -- WebSocket support is part of the free tier, with no caps on connections or bandwidth. Paid plans (from $5/mo) unlock custom domains, the traffic inspector with replay, and other features, but tunneling WebSocket traffic works on every plan.
