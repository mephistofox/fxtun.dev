---
title: "Share Your API with Partners — No Deployment"
date: 2026-03-13T14:00:00+03:00
draft: false
description: "Share your local API with partners and clients via fxTunnel: secure access, authentication, rate limiting, and real-world integration scenarios."
tags: [api, sharing, partners, integration, fxTunnel, tunnel, developer-tools, collaboration]
image: ""
author: "fxTunnel Team"
faq:
  - q: "How do I share a local API with an external partner without deploying?"
    a: "Run fxtunnel http 8080 — you get a public HTTPS URL like https://partner-api.fxtun.dev in seconds. Send this URL to your partner. They can call your API endpoints as if they were hitting a production server. No deployment, no staging server, no DNS changes required."
  - q: "Is it safe to expose my local API through a tunnel?"
    a: "For development and integration testing, yes. fxTunnel encrypts all traffic via TLS 1.3. Add authentication to your API endpoints, use test data instead of production databases, and close the tunnel when the integration session is over. Do not use a tunnel as a production API gateway."
  - q: "Can multiple partners access my tunnel at the same time?"
    a: "Yes. A single fxTunnel tunnel supports multiple concurrent connections. Each partner can send requests to the same public URL simultaneously. If you need to give different partners separate endpoints, open multiple tunnels to different ports — from $10/mo you get 10+ simultaneous tunnels."
  - q: "Does the tunnel URL change when I restart my computer?"
    a: "In fxTunnel SaaS, the URL stays the same across restarts — even on the free tier. With a custom domain (from $5/mo), the URL is fully under your control, for example api.yourcompany.com. Partners do not need to update their integration code after you restart."
  - q: "Can I use a tunnel to share a GraphQL API?"
    a: "Yes. GraphQL runs over HTTP, so fxtunnel http 4000 exposes your GraphQL endpoint at https://your-tunnel.fxtun.dev/graphql. Partners can use GraphQL Playground, Apollo Studio, or any GraphQL client. Inspector (from $5/mo) shows all queries and mutations in real time."
---

## The Problem: Your Partner Needs API Access, But You Have Not Deployed Yet

You are building an API. A partner team, an external contractor, or a client's frontend developer needs to start integrating with your endpoints today. But your API lives on `localhost:8080`. There is no staging server, no CI/CD pipeline, and deploying just for an integration test feels like overkill.

The traditional approach looks like this: spin up a cloud instance, configure a reverse proxy, set up SSL certificates, deploy your code, manage environment variables, and pray nothing breaks. This takes hours — sometimes days if the infrastructure is not ready. And every time you change a line of code, you need to redeploy.

Meanwhile, the partner is blocked. They cannot write their integration code without a live API to test against. The project timeline slips. Everyone loses time.

This is the core problem: sharing a local API with an external party should be simple, but the deployment overhead makes it painful.

## The Solution: A Tunnel Exposes Your Local API in 30 Seconds

A [local tunnel](/blog/what-is-tunneling/) creates a public HTTPS URL that points directly at your localhost. Run one command, and your partner can call your API from anywhere in the world. No deployment, no server provisioning, no DNS configuration.

fxTunnel is a SaaS tunnel built for this exact workflow. A single command -- `fxtunnel http 8080` -- gives you a stable public URL with automatic TLS encryption. The [Inspector](/blog/traffic-inspector-debug-requests/) (from $5/mo) lets both you and your partner see every request in real time, which makes debugging integration issues much faster.

How it works:

1. You start your API server locally on port 8080.
2. You run `fxtunnel http 8080` in a second terminal.
3. fxTunnel establishes an encrypted connection to a public relay server.
4. The relay server issues a public URL like `https://partner-api.fxtun.dev`.
5. Your partner sends requests to that URL — they are forwarded to your `localhost:8080`.

The partner sees a regular HTTPS API. They do not need to know it is running on your laptop. For them, it behaves like any other remote API.

## Step-by-Step: Sharing Your API with a Partner

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Or via go install
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest

# Verify
fxtunnel --version
```

Source code and documentation: [github.com/mephistofox/fxtun.dev](https://github.com/mephistofox/fxtun.dev).

### Step 2. Start Your Local API

Make sure your API server is running and listening on the correct port. Here is a minimal Express example:

```javascript
// api-server.js
const express = require('express');
const app = express();

app.use(express.json());

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Product catalog API
app.get('/api/v1/products', (req, res) => {
  res.json([
    { id: 1, name: 'Widget A', price: 29.99, stock: 150 },
    { id: 2, name: 'Widget B', price: 49.99, stock: 75 },
  ]);
});

app.get('/api/v1/products/:id', (req, res) => {
  res.json({ id: req.params.id, name: 'Widget A', price: 29.99, stock: 150 });
});

app.post('/api/v1/orders', (req, res) => {
  console.log('New order:', req.body);
  res.status(201).json({
    orderId: 'ord_' + Date.now(),
    status: 'created',
    items: req.body.items,
  });
});

app.listen(8080, () => {
  console.log('API server listening on port 8080');
});
```

```bash
node api-server.js
# -> API server listening on port 8080
```

### Step 3. Open a Tunnel

```bash
fxtunnel http 8080
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://partner-api.fxtun.dev
Forwarding:  https://partner-api.fxtun.dev -> http://localhost:8080
```

### Step 4. Share the URL with Your Partner

Send `https://partner-api.fxtun.dev` to your partner. They can immediately start calling your endpoints:

```bash
# Partner tests your API
curl https://partner-api.fxtun.dev/health
# -> {"status":"ok","timestamp":"2026-03-13T11:00:00.000Z"}

curl https://partner-api.fxtun.dev/api/v1/products
# -> [{"id":1,"name":"Widget A","price":29.99,"stock":150}, ...]

curl -X POST https://partner-api.fxtun.dev/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"items": [{"productId": 1, "quantity": 2}]}'
# -> {"orderId":"ord_1710320400000","status":"created","items":[...]}
```

That is it. The partner has full access to your API. You see their requests in your terminal, and you can debug in real time.

## Real-World Scenarios for API Sharing via Tunnel

Sharing a local API through a tunnel is not a niche trick -- it is a workflow pattern that comes up all the time. Here are five scenarios where a tunnel eliminates deployment friction and unblocks partner teams.

### Scenario 1: B2B Integration Development

You are building a payment processing API. A partner company needs to integrate their checkout flow with your endpoints. Their frontend team needs to test the full request-response cycle: create orders, process payments, handle errors.

Without a tunnel, you deploy to staging and give them the URL. Every code change requires a redeployment. With a tunnel:

```bash
# Your local API with payment endpoints
fxtunnel http 8080
# -> https://payments-dev.fxtun.dev

# Partner's frontend developer hits your endpoints directly
# You fix a bug, the change is live instantly — no redeploy
```

### Scenario 2: Third-Party Contractor Building a Frontend

You hired a freelance React developer to build a dashboard for your API. They need to connect to your endpoints to build the UI, but your API is not deployed anywhere yet.

```bash
# Your API on port 8080
fxtunnel http 8080
# -> https://dashboard-api.fxtun.dev

# The contractor sets this in their .env:
# REACT_APP_API_URL=https://dashboard-api.fxtun.dev
```

The contractor works against your live API. You iterate on endpoints together in real time. When you change a response format, the contractor sees the change immediately.

### Scenario 3: Mobile App Team in Another Office

Your company has a mobile team in a different city. They are building an iOS/Android app that consumes your API. They need to test against real endpoints, not mocked data. We cover this workflow in [mobile app testing with tunnels](/blog/mobile-app-testing-tunnel/).

```bash
# API server running locally
fxtunnel http 3000
# -> https://mobile-backend.fxtun.dev

# Mobile developers configure their app:
# API_BASE_URL = "https://mobile-backend.fxtun.dev"
```

### Scenario 4: Client Review of API Before Contract Signing

A potential client wants to evaluate your API before signing a contract. They want to send real requests and see the responses. You do not want to deploy a staging instance just for a sales demo.

```bash
fxtunnel http 8080 --domain api-demo.yourcompany.com
# -> https://api-demo.yourcompany.com
```

With a custom domain (from $5/mo), the client sees a professional URL instead of a random subdomain. The [how to demo your local project without deploying](/blog/demo-localhost-without-deploy/) article covers this approach.

### Scenario 5: Webhook Callback Testing Between Two APIs

Your partner's system needs to send callbacks (webhooks) to your API when events happen on their side. This is the reverse of outgoing API calls — you need to receive requests. A tunnel handles this naturally because it makes your localhost publicly reachable.

```bash
fxtunnel http 8080
# -> https://callback-receiver.fxtun.dev

# Give this URL to your partner for webhook delivery
# They configure: callback_url = "https://callback-receiver.fxtun.dev/api/v1/callbacks"
```

The [webhook testing with a tunnel](/blog/webhook-testing-with-tunnel/) article covers this scenario in depth.

## Securing Your Shared API

Sharing an API with partners means exposing it to the internet, and security matters even during development. fxTunnel encrypts all traffic with TLS, but application-level security is your responsibility. More on this topic in the guide on [exposing localhost safely](/blog/expose-localhost-to-internet/).

### API Key Authentication

The simplest approach: require an API key in every request. Generate a key, share it with your partner, and reject requests without it.

```javascript
// Middleware: API key authentication
const API_KEY = process.env.API_KEY || 'partner-test-key-2026';

function authenticateApiKey(req, res, next) {
  const key = req.headers['x-api-key'];
  if (!key || key !== API_KEY) {
    return res.status(401).json({ error: 'Invalid or missing API key' });
  }
  next();
}

// Apply to all /api routes
app.use('/api', authenticateApiKey);
```

Your partner includes the key in every request:

```bash
curl -H "X-API-Key: partner-test-key-2026" \
  https://partner-api.fxtun.dev/api/v1/products
```

### JWT Bearer Token Authentication

For more complex setups, use JWT tokens. This allows you to embed partner identity, permissions, and expiration into the token.

```javascript
const jwt = require('jsonwebtoken');
const SECRET = process.env.JWT_SECRET || 'dev-secret-change-me';

// Generate a token for the partner
app.post('/auth/token', (req, res) => {
  const { partnerId, secret } = req.body;
  if (secret !== 'shared-secret') {
    return res.status(401).json({ error: 'Invalid credentials' });
  }
  const token = jwt.sign(
    { partnerId, role: 'partner', exp: Math.floor(Date.now() / 1000) + 86400 },
    SECRET
  );
  res.json({ token });
});

// Verify token middleware
function authenticateJWT(req, res, next) {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Missing Bearer token' });
  }
  try {
    const decoded = jwt.verify(authHeader.split(' ')[1], SECRET);
    req.partner = decoded;
    next();
  } catch (err) {
    return res.status(403).json({ error: 'Invalid or expired token' });
  }
}

app.use('/api', authenticateJWT);
```

### Rate Limiting

Protect your local machine from accidental overload. A partner running a load test against your tunnel can crash your dev environment.

```javascript
const rateLimit = require('express-rate-limit');

const apiLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 100,            // 100 requests per minute per IP
  message: { error: 'Too many requests. Limit: 100/min.' },
});

app.use('/api', apiLimiter);
```

### Security Checklist for API Sharing

| Measure | Why It Matters | Implementation |
|---|---|---|
| API key or JWT auth | Prevents unauthorized access | Middleware on all `/api` routes |
| Rate limiting | Protects against overload | `express-rate-limit` or equivalent |
| Test data only | Prevents real data exposure | Seed DB with fake records |
| Close tunnel after session | Limits exposure window | `Ctrl+C` when done |
| HTTPS (automatic) | Encrypts data in transit | Built into fxTunnel |
| No hardcoded secrets | Prevents credential leaks | Use `.env` + `.gitignore` |
| Restrict admin routes | Prevents privilege escalation | Separate auth for admin endpoints |

## Using Inspector to Debug Partner Requests

When a partner reports "your API returns an error," you need to see exactly what they sent. The built-in [Traffic Inspector](/blog/traffic-inspector-debug-requests/) in fxTunnel (from $5/mo) shows every request and response in real time.

```bash
fxtunnel http 8080
```

```
fxTunnel v1.x — tunnel is active
Public URL:   https://partner-api.fxtun.dev
Inspector:    https://partner-api.fxtun.dev/_inspector
Forwarding:   https://partner-api.fxtun.dev -> http://localhost:8080
```

Open the Inspector link. When your partner sends a request, you see it immediately:

```
POST /api/v1/orders HTTP/1.1
X-API-Key: partner-test-key-2026
Content-Type: application/json

{"items": [{"product_id": "abc", "quantity": -1}]}

--- Response ---
HTTP/1.1 400 Bad Request
{"error": "Quantity must be a positive integer"}
```

Now you know exactly what went wrong. The partner sent `product_id` as a string instead of a number, and `quantity` was negative. You can give them precise feedback instead of guessing.

The Replay feature lets you resend the partner's exact request after you fix a bug. No need to ask the partner to retry — click Replay, verify the fix, then let them know.

## Sharing Multiple APIs Simultaneously

What if your partner needs access to a REST API, a WebSocket server, and a background worker's admin panel -- each running on a different port? Real-world integrations often involve multiple services, and tunnels handle this naturally.

### Two Tunnels, Two Services

```bash
# Terminal 1 — REST API on port 8080
fxtunnel http 8080
# -> https://rest-api.fxtun.dev

# Terminal 2 — WebSocket server on port 8081
fxtunnel http 8081
# -> https://ws-server.fxtun.dev
```

Give your partner both URLs. REST calls go to `https://rest-api.fxtun.dev`, WebSocket connections go to `https://ws-server.fxtun.dev`.

### Docker Compose + Multiple Tunnels

If your services run in [Docker containers](/blog/docker-tunnel-expose-containers/), expose each container's port through a separate tunnel:

```yaml
# docker-compose.yml
services:
  api:
    build: ./api
    ports:
      - "8080:8080"
  auth:
    build: ./auth
    ports:
      - "8081:8081"
  gateway:
    build: ./gateway
    ports:
      - "3000:3000"
```

```bash
# Open tunnels to each service
fxtunnel http 8080   # -> https://api.fxtun.dev
fxtunnel http 8081   # -> https://auth.fxtun.dev
fxtunnel http 3000   # -> https://gateway.fxtun.dev
```

With the plan from $10/mo, you get 10+ simultaneous tunnels — enough for complex microservice architectures.

## Comparison: Tunnel vs Other API Sharing Methods

There are several ways to give a partner access to your API. Each method has trade-offs in setup time, cost, and freshness of the code being tested.

| Method | Setup Time | Cost | Always Up-to-Date | HTTPS | Auth Support |
|---|---|---|---|---|---|
| **fxTunnel** | 30 seconds | Free (custom domain from $5/mo) | Yes (live code) | Built-in | Your app handles it |
| Deploy to staging | 30-120 minutes | $10-100/mo | After each deploy | Needs config | Full control |
| Deploy to Vercel/Railway | 5-15 minutes | Free with limits | After each push | Yes | Platform-specific |
| API mock (Postman, Stoplight) | 1-4 hours | Free / $14+/mo | No (static mock) | Yes | Built-in |
| VPN to your network | 30-60 minutes | VPN cost | Yes | Depends on config | Network-level |
| SSH tunnel | 10-30 minutes | SSH server needed | Yes | Manual TLS | None built-in |

A tunnel wins when you need immediate access to live, actively changing code without deployment overhead. We compare tunneling tools in more detail in the [2026 roundup](/blog/best-tunneling-tools-2026/).

### When a Tunnel Is the Right Choice

- **Early development** — the API changes multiple times per day and deploying after every change is wasteful.
- **Short integration sprints** — a partner needs access for days or weeks, not months.
- **Proof of concept** — you want the partner to validate the API design before investing in infrastructure.
- **Freelance projects** — no budget for a dedicated staging server.

### When a Tunnel Is Not Enough

- **Production traffic** — a tunnel is for development, not production. Deploy your API properly for live users.
- **SLA requirements** — if the partner expects 99.9% uptime, use a proper deployment.
- **Large teams with CI/CD** — if you already have staging infrastructure, use it.

## Framework-Specific Examples

### Python FastAPI

```python
# main.py
from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel
from typing import Optional

app = FastAPI(title="Partner API", version="0.1.0")

API_KEY = "partner-test-key-2026"

async def verify_api_key(x_api_key: str = Header(...)):
    if x_api_key != API_KEY:
        raise HTTPException(status_code=401, detail="Invalid API key")

@app.get("/api/v1/products")
async def list_products(x_api_key: str = Header(...)):
    await verify_api_key(x_api_key)
    return [
        {"id": 1, "name": "Widget A", "price": 29.99},
        {"id": 2, "name": "Widget B", "price": 49.99},
    ]

class OrderRequest(BaseModel):
    product_id: int
    quantity: int

@app.post("/api/v1/orders", status_code=201)
async def create_order(order: OrderRequest, x_api_key: str = Header(...)):
    await verify_api_key(x_api_key)
    return {"order_id": f"ord_{order.product_id}_{order.quantity}", "status": "created"}
```

```bash
uvicorn main:app --reload --port 8080
fxtunnel http 8080
# -> https://fastapi-partner.fxtun.dev
# Partner opens https://fastapi-partner.fxtun.dev/docs for interactive Swagger
```

FastAPI's automatic Swagger UI is a killer feature for partner integrations. The partner gets interactive API documentation at `https://your-tunnel.fxtun.dev/docs` without any extra setup.

### Go (net/http)

```go
// main.go
package main

import (
    "encoding/json"
    "log"
    "net/http"
)

const apiKey = "partner-test-key-2026"

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("X-API-Key")
        if key != apiKey {
            http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode([]map[string]interface{}{
        {"id": 1, "name": "Widget A", "price": 29.99},
        {"id": 2, "name": "Widget B", "price": 49.99},
    })
}

func main() {
    http.HandleFunc("/api/v1/products", authMiddleware(productsHandler))
    log.Println("API server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

```bash
go run main.go
fxtunnel http 8080
# -> https://go-api.fxtun.dev
```

### Ruby on Rails

```bash
rails server -p 8080
fxtunnel http 8080
# -> https://rails-api.fxtun.dev
```

Rails API mode works seamlessly through the tunnel. If your partner needs to see the API structure, consider adding `rswag` for Swagger documentation.

## Tips for Smooth Partner Integrations

### Provide API Documentation Alongside the Tunnel URL

A tunnel gives your partner access. Documentation tells them what to do with it. Share an OpenAPI spec, a Postman collection, or at minimum a list of endpoints with example requests.

```bash
# Export an OpenAPI spec if your framework supports it
# FastAPI: GET /openapi.json
# Express + swagger-jsdoc: GET /api-docs.json

curl https://partner-api.fxtun.dev/openapi.json > api-spec.json
# Share this file with your partner
```

### Use a Custom Domain for Professional Appearance

A random subdomain like `https://a3f7b.fxtun.dev` works perfectly, but `https://api.yourcompany.com` inspires more confidence in a B2B context. Custom domains are available from $5/mo.

```bash
fxtunnel http 8080 --domain api.yourcompany.com
```

### Keep the Tunnel Running During Business Hours

Your partner is blocked if the tunnel is down. During active integration periods, keep the tunnel running throughout business hours. With fxTunnel SaaS, the URL stays the same even after a restart, so brief interruptions do not require URL changes.

### Communicate Rate Limits and Constraints

Tell your partner upfront:
- What authentication is required (API key, JWT, etc.)
- What rate limits are in place (e.g., 100 requests/minute)
- What data is test data and should not be relied upon
- What hours the tunnel will be available

### Use Version Prefixes in URLs

Always use versioned API paths like `/api/v1/...`. This way, when you make breaking changes, you can introduce `/api/v2/...` without disrupting the partner's existing integration.

## Pricing: What It Costs

Visit [fxtun.dev](https://fxtun.dev) for full details. Here is the overview.

| Plan | Price | What You Get |
|---|---|---|
| **Free** | $0 | Unlimited traffic, unlimited connections, stable URL |
| **Pro** | from $5/mo | Custom domains, Inspector + Replay, up to 5 tunnels |
| **Team** | from $10/mo | 10+ simultaneous tunnels, team features |

For partner API sharing, the free tier is enough for most cases. Upgrade to Pro if you want a custom domain (professional appearance for B2B integrations) or Inspector (to debug partner requests visually). The Team plan makes sense when you have multiple microservices that all need tunnels simultaneously.

If you are evaluating alternatives, we cover [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) and [free ngrok alternatives](/blog/free-ngrok-alternatives-2026/) in separate articles.

## FAQ

### How do I share a local API with an external partner without deploying?

Just run `fxtunnel http 8080`. You will get a public HTTPS URL like `https://partner-api.fxtun.dev` within seconds. Send it to your partner and they can start calling your endpoints immediately -- no staging server, no DNS changes, no deployment pipeline needed.

### Is it safe to expose my local API through a tunnel?

For development and integration testing -- yes. The connection is encrypted with [TLS 1.3](/blog/tls-13-tunnel-security/). On your side, protect API endpoints with authentication, use test data rather than production databases, and shut down the tunnel when the session ends. A tunnel is not meant to serve as a production API gateway.

### Can multiple partners access my tunnel at the same time?

Yes -- a single tunnel handles multiple concurrent connections without any special configuration. Different partners can send requests to the same URL in parallel. If you want to give each partner a separate endpoint, open multiple tunnels on different ports.

### Does the tunnel URL change when I restart my computer?

No. In fxTunnel SaaS, your URL persists across restarts, even on the free tier. If you use a custom domain (from $5/mo), the URL is entirely under your control -- something like `api.yourcompany.com`. Your partners never need to update their integration code.

### Can I use a tunnel to share a GraphQL API?

Yes. Since GraphQL runs over HTTP, `fxtunnel http 4000` exposes your endpoint at `https://your-tunnel.fxtun.dev/graphql`. Partners can query it with GraphQL Playground, Apollo Studio, or any other client. The [Inspector](/blog/traffic-inspector-debug-requests/) (from $5/mo) is especially handy here -- you can watch queries and mutations arrive in real time.
