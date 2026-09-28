---
title: "Traffic Inspector: Debug HTTP Requests in Real Time"
date: 2026-03-01T14:00:00+03:00
draft: false
description: "Built-in Traffic Inspector in fxTunnel: view HTTP requests in real time, replay with one click, debug webhooks and mobile APIs. From $5/mo."
tags: [inspector, debugging, http, webhooks, fxTunnel, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "What is Traffic Inspector in fxTunnel?"
    a: "Traffic Inspector is a built-in web interface for viewing all HTTP requests passing through an fxTunnel tunnel. It shows headers, request and response body, status codes, timing, and payload size in real time. Available on plans from $5/mo."
  - q: "How does fxTunnel Inspector compare to ngrok Inspector?"
    a: "Both tools show HTTP traffic in real time. The key difference is price: fxTunnel offers Inspector with Replay from $5/mo, while ngrok charges $8/mo for similar functionality. fxTunnel also supports filtering by method, path, and status code out of the box."
  - q: "Can I use Inspector to debug webhooks?"
    a: "Yes, this is one of the primary use cases. Inspector records every incoming webhook with full body and headers. The Replay feature lets you resend any webhook with one click — no need to re-trigger the event in Stripe, GitHub, or Telegram."
  - q: "Does Inspector work with TCP and UDP tunnels?"
    a: "Inspector is designed for HTTP traffic because HTTP is a structured protocol with headers, body, and status codes. For TCP and UDP tunnels, Inspector is not available — these protocols transmit raw byte streams without HTTP semantics."
  - q: "Do I need to install additional software to use Inspector?"
    a: "No. Inspector is a web interface that opens directly in your browser. After starting a tunnel with fxtunnel http, you get an Inspector link in the terminal. No additional software, extensions, or plugins required."
---

## Why You Need a Traffic Inspector

You integrate Stripe, receive webhooks from GitHub, test a mobile app API -- and at some point something breaks. Where did the request go wrong? The built-in Traffic Inspector in fxTunnel shows every HTTP request in real time: headers, body, status code, and timing -- without a single extra line of code.

Adding logging is not always an option. A third-party service sends a webhook — you do not control the request format. A mobile app sends requests over HTTPS — intercepting them without a proxy is not straightforward. Even in your own code, manually logging every header and body means noise in the terminal and wasted time.

Inspector solves this: the [tunnel](/blog/what-is-tunneling/) passes all traffic through itself, and Inspector records every request and response in a clean web interface. You see exactly what your server sees — without modifying code.

## What fxTunnel Inspector Shows

Traffic Inspector displays full information about every HTTP request and response. The interface updates in real time — new requests appear instantly without a page reload.

For every incoming request, Inspector shows:

- **Method and path** — `POST /api/webhook`, `GET /healthcheck`
- **Headers** — all HTTP headers including `Content-Type`, `Authorization`, `Stripe-Signature`
- **Request body** — JSON, form-data, XML, or raw bytes with syntax highlighting
- **Query parameters** — parsed `?key=value` pairs in a readable table
- **Size and timestamp** — body weight and exact time of receipt with millisecond precision

Inspector also records your server's response: status code, headers, body, and processing time in milliseconds. You can see how long each phase took — transmission through the tunnel and processing on your server:

```
Client -> fxTunnel server -> tunnel -> localhost -> your server -> response -> back
          |<-- DNS + TLS -->|<-- tunnel latency -->|<-- server processing -->|
```

## How to Start Inspector

Inspector is available on paid fxTunnel plans from $5/mo. Getting started takes 10 seconds.

```bash
fxtunnel http 8080
```

In the terminal you will see:

```
fxTunnel v1.x — tunnel is active
Public URL:   https://my-app.fxtun.dev
Inspector:    https://my-app.fxtun.dev/_inspector
Forwarding:   https://my-app.fxtun.dev -> http://localhost:8080
```

Open the Inspector link in your browser and send a test request:

```bash
curl -X POST https://my-app.fxtun.dev/api/test \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"id": 42}}'
```

The request instantly appears in Inspector — with full body, headers, and your server's response.

## Replay: Resend Any Request with One Click

Replay lets you resend any recorded request to your localhost with a single click. No need to re-trigger an event in a third-party service, copy curl commands, or write test scripts.

1. Find the request you need in the Inspector list.
2. Click the **Replay** button.
3. fxTunnel resends an exact copy of the request — with the same headers and body.
4. The new response appears alongside — you see the result immediately.

### Why Replay is essential

A typical debugging cycle without Replay:

```
1. Stripe sends a webhook -> server crashes with an error
2. You fix the code
3. You need to create another test payment in Stripe
4. Wait for the webhook... Repeat steps 2-4 five more times
```

With Replay:

```
1. Stripe sends a webhook -> server crashes with an error
2. You fix the code
3. Click Replay in Inspector -> see the result instantly
```

For complex integrations where reproducing an event is difficult (a Stripe transaction dispute, a GitHub merge event, a payment gateway callback), Replay is the difference between hours and minutes of debugging.

## Debugging Webhooks with Inspector

[Webhooks](/blog/webhook-testing-with-tunnel/) are the primary use case for Inspector. A third-party service sends a request to your server, and you do not control its contents. Inspector lets you see every webhook before your code processes it.

### Stripe: signature verification

Stripe signs every webhook with the `Stripe-Signature` header. If your server responds with `400 Bad Request`, Inspector shows:

```
POST /webhook HTTP/1.1
Content-Type: application/json
Stripe-Signature: t=1709294400,v1=5257a869e7eceb...

{"id": "evt_1abc", "type": "payment_intent.succeeded", ...}

--- Response ---
HTTP/1.1 400 Bad Request
{"error": "Webhook signature verification failed"}
```

You see: the request arrived, the signature is present, but verification failed. The problem is in your code, not Stripe. Fix it, click Replay, see `200 OK`.

### GitHub: event format

GitHub sends the event type in the `X-GitHub-Event` header and the signature in `X-Hub-Signature-256`. Inspector shows both:

```
POST /webhook HTTP/1.1
X-GitHub-Event: push
X-Hub-Signature-256: sha256=abc123...

{"ref": "refs/heads/main", "commits": [...]}
```

### Telegram: bot updates

Telegram does not sign webhooks but sends a complex `Update` structure. Inspector lets you examine the full payload:

```json
{
  "update_id": 123456789,
  "message": {
    "message_id": 42,
    "from": {"id": 100, "first_name": "Aleksei"},
    "chat": {"id": 100, "type": "private"},
    "text": "/start"
  }
}
```

You see the exact format your handler receives — no guessing.

## Debugging Mobile App API Calls

Inspector is invaluable when [testing mobile applications](/blog/mobile-app-testing-tunnel/) through a tunnel. The phone sends requests to the fxTunnel public URL, and Inspector records each one — without a proxy on the phone, without Charles Proxy, without modifying code.

```bash
# Start your API server and tunnel
node server.js  # listens on port 3000
fxtunnel http 3000
# -> Public URL: https://mobile-api.fxtun.dev
# -> Inspector:  https://mobile-api.fxtun.dev/_inspector
```

In the mobile app, set `https://mobile-api.fxtun.dev` as the base API URL. Inspector shows every request:

```
GET /api/v1/user/profile HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
User-Agent: MyApp/2.1.0 (iPhone; iOS 18.3)

--- Response ---
HTTP/1.1 200 OK
{"id": 42, "name": "Aleksei", "email": "dev@example.com"}
```

If the app gets an error, you see both the request and the response and can determine whether the problem is on the client side or the server side.

## Comparison with Alternatives

Inspector is not the only way to view HTTP traffic, but it fits into your existing workflow with minimal setup. Here is how it stacks up against [other tools](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Tool | Request viewing | Replay | Price | Limitations |
|---|---|---|---|---|
| **fxTunnel Inspector** | Real time, headers + body + timing | One click | from $5/mo | HTTP traffic through tunnel only |
| **ngrok Inspector** | Real time | Yes | from $8/mo | Tied to ngrok |
| **Postman** | Outgoing requests only | Manual resend | Free / $14/mo | Cannot see incoming webhooks |
| **curl** | Manual execution | Manual resend | Free | No GUI, no history |
| **Chrome DevTools** | Browser only | No | Free | Cannot see server or mobile requests |
| **Charles Proxy** | All HTTP traffic | Manual resend | $50 | Requires proxy and certificate setup |

The main thing Inspector has going for it: zero setup friction. Start a tunnel, open Inspector, and you are already seeing traffic.

## Filtering and Searching Requests

During active development, dozens of requests pass through Inspector. Filtering helps you find the right one quickly.

Inspector supports filtering by several criteria:

- **By method** — show only `POST`, only `GET`, or any other method
- **By path** — find all requests to `/api/webhook` or `/healthcheck`
- **By status code** — filter for errors only (`4xx`, `5xx`) or successes only (`2xx`)
- **By time** — requests from the last 5 minutes, hour, or a custom interval

Content search is also available across headers and request bodies. For example, find all failed Stripe webhooks:

```
Method: POST | Path: /webhook | Status: 4xx, 5xx
-> 3 requests found
```

Open each one, examine the body and response, identify the cause, and use Replay to verify the fix.

## Practical Workflow: Debug, Replay, Fix, Verify

Inspector fits into the development cycle as an observation and replay tool.

### 1. Debug — discover the problem

Bug report: "Stripe webhook is not being processed." Open Inspector:

```
POST /webhook -> 500 Internal Server Error
{"error": "Cannot read property 'amount' of undefined"}
```

### 2. Replay — reproduce

Click Replay. The same request hits your server again. If needed, set a breakpoint in your IDE and click Replay once more — the debugger will stop on the right line.

### 3. Fix — correct the code

After examining the request body, you see that Stripe sends `amount_received`, not `amount`:

```javascript
// Before:
const amount = event.data.object.amount;

// After:
const amount = event.data.object.amount_received;
```

### 4. Verify — confirm the fix

Click Replay. Inspector shows `POST /webhook -> 200 OK`. Problem solved in minutes, not hours.

## Plans and Availability

Traffic Inspector is available on paid fxTunnel plans:

| Plan | Price | Inspector | Replay | Filtering |
|---|---|---|---|---|
| **Free** | $0 | No | No | — |
| **Pro** | from $5/mo | Yes | Yes | Yes |
| **Team** | from $10/mo | Yes | Yes | Yes |

For reference: ngrok starts at $8/mo for Inspector with Replay, and Postman charges $14/mo for advanced monitoring.

Inspector does not replace unit tests, APM systems, or server logs -- it complements them. Its job is to give you instant visibility into HTTP traffic during development. The comparison table in [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) has more on pricing and feature differences.

## FAQ

### What is Traffic Inspector in fxTunnel?

It is a web-based dashboard that records every HTTP request flowing through your tunnel. You see headers, request and response bodies, status codes, timing, and payload sizes -- all updating live in the browser. Inspector is included on paid plans starting at $5/mo.

### How does fxTunnel Inspector compare to ngrok Inspector?

Functionally they are similar -- both display HTTP traffic in real time with replay capability. The pricing differs: fxTunnel includes Inspector with Replay from $5/mo, ngrok from $8/mo. fxTunnel also ships with built-in filtering by method, path, and status code.

### Can I use Inspector to debug webhooks?

Absolutely -- this is one of its strongest use cases. Every incoming [webhook](/blog/webhook-testing-with-tunnel/) is recorded with full body and headers. When you need to retry, hit Replay and the same payload is sent to your server again. No need to re-trigger events in Stripe, GitHub, or Telegram.

### Does Inspector work with TCP and UDP tunnels?

No. Inspector relies on the structure of HTTP (headers, body, status codes) to present meaningful data. TCP and UDP tunnels carry raw byte streams with no HTTP semantics, so there is nothing for Inspector to parse. The [fxTunnel architecture](/blog/fxtunnel-architecture/) article explains how different protocol modes work.

### Do I need to install additional software to use Inspector?

No. It runs entirely in the browser. When you start a tunnel with `fxtunnel http`, the terminal prints an Inspector URL. Open it, and you are done -- nothing else to install.
