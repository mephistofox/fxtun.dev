---
title: "Multiple Tunnels for Microservices — Dev Guide"
date: 2026-03-23T14:00:00+03:00
draft: false
description: "Run multiple tunnels for microservices: one tunnel per service, docker-compose integration, service discovery, and team workflows."
tags: [microservices, tunneling, docker, fxTunnel, developer-tools, multiple-tunnels, service-discovery]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I run multiple tunnels at the same time for free?"
    a: "Yes. fxTunnel's free tier supports multiple simultaneous tunnels with no connection limits. You can open one tunnel per microservice and they all work in parallel. Paid plans from $5/mo add custom domains and request inspector, and from $10/mo you get 10+ concurrent tunnels with priority support."
  - q: "How do microservices discover each other through tunnels?"
    a: "Pass each tunnel URL as an environment variable to the services that need it. For example, the frontend receives REACT_APP_API_URL with the API tunnel URL, and the API receives PAYMENT_SERVICE_URL with the payment service tunnel URL. In docker-compose, you can use a startup script that reads tunnel URLs from log files and injects them into dependent services."
  - q: "Do I need a separate tunnel for every microservice?"
    a: "Only for services that need to be accessed from outside your local network. Internal services that only communicate with other local services can use Docker networking or localhost directly. Create tunnels only for services that external clients, webhooks, or teammates need to reach."
  - q: "Will multiple tunnels slow down my development machine?"
    a: "No. Each fxTunnel process uses roughly 10-15 MB of RAM and minimal CPU. Running 5-10 tunnels simultaneously adds negligible overhead. The fxTunnel client is a single Go binary optimized for low resource usage."
  - q: "Can my team share the same set of tunnels for collaborative development?"
    a: "Yes. Once the tunnels are running, anyone with the public URLs can access your services. Share the URLs in Slack or your team chat. For persistent team setups, use the $10/mo plan which supports 10+ concurrent tunnels and custom domains so each service gets a memorable address."
---

## The Problem: Microservices Need Multiple Entry Points

Modern applications are rarely a single process. A typical stack includes a frontend, an API gateway, several backend services, a database, and maybe a message queue. During local development, each service runs on its own port: the frontend on `localhost:3000`, the API on `localhost:4000`, the payment service on `localhost:4100`, and so on.

This works fine on your machine. But the moment you need external access -- a colleague testing your branch, a [webhook from Stripe](/blog/webhook-testing-with-tunnel/) hitting your payment service, or a mobile app connecting to your API -- you need public URLs. Not one URL, but several, one for each service that receives external traffic.

A single [tunnel](/blog/what-is-tunneling/) will not cut it. You need multiple tunnels running simultaneously, each pointing at a different service. This guide covers how to set that up with fxTunnel -- both on bare metal and with Docker Compose -- and how to wire microservices together through tunnel URLs.

## Architecture: One Tunnel per Service

The idea is straightforward: each microservice that needs external access gets its own tunnel. Internal services that only talk to other local services do not need a tunnel — they use Docker networking or localhost directly.

{{< diagram >}}
                        Internet
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
    ┌─────────────┐ ┌───────────┐ ┌──────────────┐
    │  fxTunnel   │ │ fxTunnel  │ │   fxTunnel   │
    │  Server     │ │ Server    │ │   Server     │
    │ (HTTPS)     │ │ (HTTPS)   │ │   (TCP)      │
    └──────┬──────┘ └─────┬─────┘ └──────┬───────┘
           │              │              │
    ┌──────▼──────┐ ┌─────▼─────┐ ┌──────▼───────┐
    │  Tunnel 1   │ │ Tunnel 2  │ │  Tunnel 3    │
    │  :3000      │ │ :4000     │ │  :5432       │
    └──────┬──────┘ └─────┬─────┘ └──────┬───────┘
           │              │              │
           ▼              ▼              ▼
    ┌─────────────┐ ┌───────────┐ ┌──────────────┐
    │  Frontend   │ │    API    │ │  PostgreSQL  │
    │  (React)    │ │  (Node)   │ │              │
    │  port 3000  │ │  port 4000│ │  port 5432   │
    └─────────────┘ └───────────┘ └──────────────┘
              Developer Machine
{{< /diagram >}}

Each tunnel creates its own public URL. The frontend is available at `https://front-abc.fxtun.dev`, the API at `https://api-xyz.fxtun.dev`, and the database through a [TCP tunnel](/blog/tcp-udp-tunneling-explained/) at `tcp://db-qrs.fxtun.dev:18432`.

### What Needs a Tunnel and What Does Not

Not every service in your stack needs a tunnel. Here is a simple rule:

| Service | Needs a Tunnel? | Why |
|---|---|---|
| Frontend (React, Vue) | Yes | Accessed by browsers, mobile apps, teammates |
| API gateway | Yes | Called by frontend, webhooks, external clients |
| Payment service | Yes | Receives webhooks from Stripe, PayPal |
| Auth service | Maybe | Only if [OAuth callbacks](/blog/oauth-callback-localhost-tunnel/) come from external providers |
| Database | Rarely | Only if a teammate needs direct access |
| Redis / Message queue | No | Internal communication only |
| Worker / Background jobs | No | Processes messages from the queue, no inbound traffic |

## Quick Start: Multiple Tunnels from the CLI

The fastest way to run multiple tunnels is to open several terminal windows and run `fxtunnel` in each one.

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify
fxtunnel --version
```

### Step 2. Start Your Services

```bash
# Terminal 1: Frontend
cd frontend && npm run dev
# → listening on localhost:3000

# Terminal 2: API
cd api && npm run dev
# → listening on localhost:4000

# Terminal 3: Payment service
cd payment-service && go run main.go
# → listening on localhost:4100
```

### Step 3. Open a Tunnel for Each Service

```bash
# Terminal 4: Tunnel for frontend
fxtunnel http 3000

# Terminal 5: Tunnel for API
fxtunnel http 4000

# Terminal 6: Tunnel for payment service
fxtunnel http 4100
```

Each command prints its public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://front-abc.fxtun.dev
Forwarding:  https://front-abc.fxtun.dev → http://localhost:3000
```

Three services, three tunnels, three public URLs — all running simultaneously on the free tier.

## Automation: A Bash Script for Multiple Tunnels

Opening six terminal windows is not practical for daily work. A simple bash script starts all services and tunnels in the background and collects the URLs.

```bash
#!/bin/bash
# start-tunnels.sh — Start multiple tunnels for microservices

set -e

SERVICES=(
  "frontend:3000:http"
  "api:4000:http"
  "payment:4100:http"
  "db:5432:tcp"
)

LOG_DIR="/tmp/tunnels"
mkdir -p "$LOG_DIR"

echo "Starting tunnels..."

for entry in "${SERVICES[@]}"; do
  IFS=':' read -r name port proto <<< "$entry"
  log_file="$LOG_DIR/${name}.log"

  fxtunnel "$proto" "$port" --log-file "$log_file" &
  echo "  $name ($proto:$port) — PID $!"
done

# Wait for URLs to appear
sleep 3

echo ""
echo "=== Tunnel URLs ==="
for entry in "${SERVICES[@]}"; do
  IFS=':' read -r name port proto <<< "$entry"
  log_file="$LOG_DIR/${name}.log"
  url=$(grep -oP 'https?://[a-z0-9-]+\.fxtun\.dev(:[0-9]+)?' "$log_file" 2>/dev/null || echo "pending...")
  echo "  $name → $url"
done

echo ""
echo "Press Ctrl+C to stop all tunnels"
wait
```

Usage:

```bash
chmod +x start-tunnels.sh
./start-tunnels.sh
```

Output:

```
Starting tunnels...
  frontend (http:3000) — PID 12345
  api (http:4000) — PID 12346
  payment (http:4100) — PID 12347
  db (tcp:5432) — PID 12348

=== Tunnel URLs ===
  frontend → https://front-abc.fxtun.dev
  api → https://api-xyz.fxtun.dev
  payment → https://pay-def.fxtun.dev
  db → tcp://db-qrs.fxtun.dev:18432

Press Ctrl+C to stop all tunnels
```

When you press Ctrl+C, all background processes are terminated and the tunnels close.

## Docker Compose: The Production-Ready Approach

For team workflows and reproducible environments, Docker Compose is the right tool. Each microservice and its tunnel are defined as separate services. The entire stack comes up with a single command.

### Full Microservice Stack with Tunnels

```yaml
# docker-compose.yml — Microservices with tunnels
version: "3.8"

services:
  # === Application services ===
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - REACT_APP_API_URL=${API_TUNNEL_URL:-http://localhost:4000}
    depends_on:
      - api

  api:
    build: ./api
    ports:
      - "4000:4000"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp
      - REDIS_URL=redis://redis:6379
      - PAYMENT_SERVICE_URL=http://payment:4100
    depends_on:
      - db
      - redis

  payment:
    build: ./payment-service
    ports:
      - "4100:4100"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
    depends_on:
      - db

  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

  # === Tunnels ===
  tunnel-frontend:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "frontend:3000"]
    depends_on:
      - frontend
    restart: unless-stopped

  tunnel-api:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "api:4000"]
    depends_on:
      - api
    restart: unless-stopped

  tunnel-payment:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "payment:4100"]
    depends_on:
      - payment
    restart: unless-stopped

volumes:
  pgdata:
```

Start everything:

```bash
docker compose up -d

# View tunnel URLs
docker compose logs tunnel-frontend tunnel-api tunnel-payment
```

Output:

```
tunnel-frontend_1 | Public URL: https://front-abc.fxtun.dev → http://frontend:3000
tunnel-api_1      | Public URL: https://api-xyz.fxtun.dev   → http://api:4000
tunnel-payment_1  | Public URL: https://pay-def.fxtun.dev   → http://payment:4100
```

Three tunnels, three public URLs, everything managed by Docker Compose.

### How Internal Communication Works

Services inside Docker Compose communicate through Docker's internal DNS — no tunnels needed for inter-service calls. The API reaches the database at `db:5432`, the API reaches Redis at `redis:6379`, and the API calls the payment service at `payment:4100`. All of this happens inside the Docker network, with zero overhead.

Tunnels are only for traffic coming from outside the Docker network: browsers, mobile apps, webhooks, and teammates.

{{< diagram >}}
 External traffic (via tunnels)          Internal traffic (Docker DNS)
 ──────────────────────────────          ─────────────────────────────
 Browser → tunnel → frontend             frontend → api (http://api:4000)
 Stripe  → tunnel → payment              api → db (postgres://db:5432)
 Mobile  → tunnel → api                  api → redis (redis://redis:6379)
 Teammate→ tunnel → api                  api → payment (http://payment:4100)
{{< /diagram >}}

## Service Discovery: Connecting Microservices Through Tunnel URLs

The trickiest part of running multiple tunnels is wiring services together. Your frontend needs to know the API's public URL to make requests from the browser. Your payment service needs to tell Stripe where to send webhooks. There are several approaches.

### Approach 1: Environment Variables (Simplest)

After starting the tunnels, read the URLs from the logs and pass them as environment variables:

```bash
#!/bin/bash
# start-with-discovery.sh

# Start the stack
docker compose up -d

# Wait for tunnels to initialize
sleep 5

# Extract tunnel URLs from logs
API_URL=$(docker compose logs tunnel-api 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' | head -1)
FRONTEND_URL=$(docker compose logs tunnel-frontend 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' | head -1)
PAYMENT_URL=$(docker compose logs tunnel-payment 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' | head -1)

echo "API:      $API_URL"
echo "Frontend: $FRONTEND_URL"
echo "Payment:  $PAYMENT_URL"

# Restart frontend with the API URL injected
API_TUNNEL_URL=$API_URL docker compose up -d frontend

# Configure Stripe webhook to point at the payment tunnel
echo "Set Stripe webhook URL to: $PAYMENT_URL/webhooks/stripe"
```

### Approach 2: A Shared .env File

Generate a `.env` file that Docker Compose picks up automatically:

```bash
#!/bin/bash
# generate-env.sh — Extract tunnel URLs into .env

docker compose up -d tunnel-api tunnel-frontend tunnel-payment
sleep 5

API_URL=$(docker compose logs tunnel-api 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' | head -1)
FRONTEND_URL=$(docker compose logs tunnel-frontend 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' | head -1)
PAYMENT_URL=$(docker compose logs tunnel-payment 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' | head -1)

cat > .env.tunnels <<EOF
API_TUNNEL_URL=$API_URL
FRONTEND_TUNNEL_URL=$FRONTEND_URL
PAYMENT_TUNNEL_URL=$PAYMENT_URL
EOF

echo "Tunnel URLs written to .env.tunnels"
echo "Restarting services with new URLs..."

# Merge with existing .env
cat .env .env.tunnels > .env.combined && mv .env.combined .env
docker compose up -d
```

### Approach 3: Custom Domains (Paid Plan)

With the fxTunnel Pro plan (from $5/mo), you can assign custom subdomains to your tunnels. This means the URLs are predictable and you can hardcode them in configuration:

```yaml
  tunnel-api:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "api:4000", "--domain", "api.myteam.fxtun.dev"]
    depends_on:
      - api

  tunnel-frontend:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "frontend:3000", "--domain", "app.myteam.fxtun.dev"]
    depends_on:
      - frontend
```

Now `api.myteam.fxtun.dev` and `app.myteam.fxtun.dev` always point at the right services. No URL extraction scripts needed.

## Team Workflows: Sharing Your Microservice Stack

When multiple developers work on the same microservice architecture, everyone needs their own set of tunnels. Here are practical patterns for team collaboration.

### Pattern 1: Each Developer Runs Their Own Stack

The simplest approach: every developer runs `docker compose up` on their machine. Each person gets unique tunnel URLs. They share the URLs in the team chat when they need feedback.

```
 Developer A                     Developer B
 ────────────                    ────────────
 frontend → https://a-front.fxtun.dev    frontend → https://b-front.fxtun.dev
 api      → https://a-api.fxtun.dev      api      → https://b-api.fxtun.dev
 payment  → https://a-pay.fxtun.dev      payment  → https://b-pay.fxtun.dev
```

This works well for small teams (2-5 developers). Each developer's stack is fully isolated. There are no conflicts.

### Pattern 2: Shared Backend, Local Frontend

In many projects, the frontend changes more frequently than the backend. One developer runs the backend with tunnels, and other developers point their local frontends at the shared API.

```bash
# Developer A (runs the full stack)
docker compose up -d
# API: https://shared-api.fxtun.dev

# Developer B (runs only the frontend)
REACT_APP_API_URL=https://shared-api.fxtun.dev npm run dev
```

This reduces resource usage and avoids running the entire stack on every machine.

### Pattern 3: CI/CD Preview Environments

For pull request reviews, combine tunnels with CI/CD. Each PR gets its own set of tunnels via [GitHub Actions](/blog/cicd-tunnel-preview-environments/), and reviewers click a link instead of checking out the branch locally.

## Debugging Multiple Tunnels

When several tunnels are running, figuring out which service got which request matters. The fxTunnel [traffic inspector](/blog/traffic-inspector-debug-requests/) (available on the Pro plan from $5/mo) records every HTTP request passing through each tunnel -- headers, body, and timing included. Replay lets you re-send any request with a single click.

### Viewing Logs Per Service

With Docker Compose, you can view logs for a specific tunnel:

```bash
# View only the API tunnel logs
docker compose logs -f tunnel-api

# View all tunnel logs
docker compose logs -f tunnel-frontend tunnel-api tunnel-payment
```

### Checking Tunnel Status

Verify all tunnels are running:

```bash
# List running tunnel containers
docker compose ps | grep tunnel

# Expected output:
# tunnel-frontend_1   ghcr.io/mephistofox/fxtunnel:latest   Up 2 minutes
# tunnel-api_1         ghcr.io/mephistofox/fxtunnel:latest   Up 2 minutes
# tunnel-payment_1     ghcr.io/mephistofox/fxtunnel:latest   Up 2 minutes
```

### Testing Connectivity

If a tunnel is not forwarding traffic, verify the service is reachable from inside the tunnel container:

```bash
# Shell into the tunnel container
docker compose exec tunnel-api sh

# Test the target service
wget -qO- http://api:4000/health
```

If `wget` fails, the issue is Docker networking, not the tunnel. Ensure both containers are on the same Docker network.

## Advanced: Selective Tunnels with Docker Compose Profiles

You do not always need all tunnels. Docker Compose profiles let you start only the tunnels you need:

```yaml
version: "3.8"

services:
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"

  api:
    build: ./api
    ports:
      - "4000:4000"

  payment:
    build: ./payment-service
    ports:
      - "4100:4100"

  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass

  tunnel-frontend:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "frontend:3000"]
    depends_on:
      - frontend
    profiles:
      - tunnels
      - frontend-tunnel

  tunnel-api:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "api:4000"]
    depends_on:
      - api
    profiles:
      - tunnels
      - api-tunnel

  tunnel-payment:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "payment:4100"]
    depends_on:
      - payment
    profiles:
      - tunnels
      - webhook-tunnel
```

Usage:

```bash
# Start everything without tunnels (local development)
docker compose up -d

# Start with all tunnels
docker compose --profile tunnels up -d

# Start with only the API tunnel (for webhook testing)
docker compose --profile api-tunnel up -d

# Start with only the payment tunnel (for Stripe webhook testing)
docker compose --profile webhook-tunnel up -d
```

Profiles keep your development environment flexible — tunnels are there when you need them and out of the way when you do not.

## Pricing: How Many Tunnels Can You Run?

| Plan | Price | Concurrent Tunnels | Features |
|---|---|---|---|
| **Free** | $0 | Multiple (no hard limit) | HTTP/TCP/UDP, no connection limits |
| **Pro** | from $5/mo | Multiple | Custom domains, request inspector, replay |
| **Team** | from $10/mo | 10+ | Priority support, team management |

The free tier covers most individual developers running 3-5 microservice tunnels. For teams that need predictable custom domains and the traffic inspector for debugging, the Pro plan at $5/mo covers the essentials. The Team plan at $10/mo is designed for larger setups with 10+ concurrent tunnels and multiple team members.

Unlike ngrok, which limits free users to one tunnel, fxTunnel lets you run your entire stack with public URLs at no cost. See the [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) comparison for the full picture.

## Best Practices

### 1. Tunnel Only What Needs External Access

Do not create tunnels for databases, Redis, or message queues unless a teammate specifically needs direct access. Internal services communicate over Docker networking with zero overhead.

### 2. Use a Script or Makefile

Wrap your tunnel startup in a script or Makefile so every developer on the team uses the same setup:

```makefile
# Makefile
.PHONY: dev dev-tunnels

dev:
	docker compose up -d

dev-tunnels:
	docker compose --profile tunnels up -d
	@sleep 3
	@echo "=== Tunnel URLs ==="
	@docker compose logs tunnel-frontend tunnel-api 2>&1 | grep -oP 'https://[a-z0-9-]+\.fxtun\.dev'
```

### 3. Close Tunnels When Done

Open tunnels are public URLs accessible to anyone who knows the address. For [security](/blog/https-localhost-development/), close tunnels when you finish your session:

```bash
# Stop all tunnels but keep services running
docker compose stop tunnel-frontend tunnel-api tunnel-payment

# Or stop everything
docker compose down
```

### 4. Use .env Files for Team Configuration

Keep tunnel-related configuration in `.env` files that are excluded from version control:

```bash
# .env (not committed to git)
STRIPE_WEBHOOK_SECRET=whsec_test_xxx
API_TUNNEL_URL=https://api-xyz.fxtun.dev
```

### 5. Document the Setup

Add a section to your project README explaining how to start the tunnels. Future team members will thank you.

## FAQ

### Can I run multiple tunnels at the same time for free?

Yes -- the free tier has no hard limit on simultaneous tunnels, so you can open one per microservice and they all run in parallel. If you need custom domains or the traffic inspector, those come with the $5/mo plan; the $10/mo plan adds 10+ concurrent tunnels and priority support.

### How do microservices discover each other through tunnels?

The simplest approach is environment variables: pass each tunnel URL to the services that need it. For instance, your frontend gets `REACT_APP_API_URL` pointing at the API tunnel, and the API gets `PAYMENT_SERVICE_URL` pointing at the payment service tunnel. In docker-compose, a startup script can read tunnel URLs from logs and inject them before dependent services start.

### Do I need a separate tunnel for every microservice?

Only for services that receive traffic from outside your local network. Anything that only talks to other local services -- databases, Redis, background workers -- can use Docker networking or localhost directly. Save tunnels for services that external clients, [webhooks](/blog/webhook-testing-with-tunnel/), or teammates need to reach.

### Will multiple tunnels slow down my development machine?

Not noticeably. Each fxTunnel process consumes about 10-15 MB of RAM and very little CPU. Running 5-10 tunnels at once has negligible impact. The client is a single Go binary built for low overhead.

### Can my team share the same set of tunnels for collaborative development?

Yes -- once the tunnels are up, anyone who has the public URLs can access your services. Drop the links in Slack or your team chat. For a more permanent setup, the $10/mo plan gives you 10+ concurrent tunnels with custom domains so each service keeps a memorable, stable address.
