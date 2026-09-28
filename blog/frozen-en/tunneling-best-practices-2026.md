---
title: "Tunneling Best Practices 2026 — Developer Guide"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Tunneling best practices: security, performance, team workflows, CI/CD integration, monitoring, and 20 actionable tips for developers in 2026."
tags: [tunneling, best-practices, security, performance, ci-cd, fxTunnel, developer-tools, guide]
image: ""
author: "fxTunnel Team"
faq:
  - q: "How many tunnels can I run simultaneously on the free tier?"
    a: "fxTunnel's free tier has no limits on the number of tunnels, traffic, or connections. You can run as many tunnels as you need. Paid plans from $5/mo add custom domains, the traffic inspector, and replay. From $10/mo you get 10+ tunnels with all features."
  - q: "Should I use a tunnel in production?"
    a: "Tunnels are designed primarily for development and testing. For production, use a proper deployment behind a reverse proxy. That said, fxTunnel from $5/mo supports custom domains and stable long-running tunnels, making it viable for staging, demos, and internal tools."
  - q: "How do I keep tunnel URLs stable across restarts?"
    a: "By default, fxTunnel assigns a random subdomain on each start. On paid plans from $5/mo, you can use a custom domain via any DNS provider, giving you a permanent URL that survives restarts."
  - q: "Can I use tunnels in CI/CD pipelines?"
    a: "Yes. Install fxTunnel in your CI runner, start the tunnel in the background, capture the URL, and use it for E2E tests or preview environments. See the CI/CD section in this guide for ready-to-use scripts."
  - q: "What is the biggest security mistake when using tunnels?"
    a: "The most common mistake is leaving a tunnel running after you are done testing. Every active tunnel is an open entry point into your local machine. Always stop the tunnel with Ctrl+C when you no longer need it."
---

## Why Best Practices Matter for Tunneling

A [localhost tunnel](/blog/what-is-tunneling/) is one of the simplest tools in a developer's toolkit — run one command and your local server is on the internet. But simplicity is deceptive. A carelessly configured tunnel can expose sensitive data, slow down your workflow, or break in the middle of a client demo. In 2026, tunnels have become a standard part of the development process: teams use them for [webhook testing](/blog/webhook-testing-with-tunnel/), [CI/CD preview environments](/blog/cicd-tunnel-preview-environments/), [mobile testing](/blog/mobile-app-testing-tunnel/), and [IoT device access](/blog/iot-tunnel-devices-cloud/). The more you rely on tunnels, the more important it is to use them correctly.

This guide covers 20 best practices organized into four categories: security, performance, team workflows, and CI/CD integration. Each practice includes a concrete action you can take today. Code examples use [fxTunnel](https://fxtun.dev) — an open-source tunneling tool with HTTP, TCP, and UDP support — but the principles apply to any tunneling solution.

## Quick Reference: All 20 Practices at a Glance

Here is every practice at a glance. Use it as a checklist.

| # | Category | Practice | Priority |
|---|---|---|---|
| 1 | Security | Always use TLS-encrypted tunnels | Critical |
| 2 | Security | Stop tunnels when idle | Critical |
| 3 | Security | Never expose production data | Critical |
| 4 | Security | Validate webhook signatures | High |
| 5 | Security | Use environment variables for tokens | High |
| 6 | Security | Prefer open-source tools | Medium |
| 7 | Security | Keep the client updated | Medium |
| 8 | Performance | Match the tunnel protocol to your workload | High |
| 9 | Performance | Use health checks before opening tunnels | High |
| 10 | Performance | Pick a relay server close to your users | Medium |
| 11 | Performance | Monitor latency and throughput | Medium |
| 12 | Performance | Use the traffic inspector for debugging | High |
| 13 | Team | Standardize tunnel usage with a wrapper script | High |
| 14 | Team | Document tunnel URLs in shared channels | Medium |
| 15 | Team | Use custom domains for stable URLs | Medium |
| 16 | Team | Separate tunnels per microservice | High |
| 17 | CI/CD | Automate tunnel creation in pipelines | High |
| 18 | CI/CD | Generate preview URLs for every PR | High |
| 19 | CI/CD | Run E2E tests against tunnel URLs | Medium |
| 20 | CI/CD | Clean up tunnels automatically | Critical |

---

## Security Best Practices (Practices 1-7)

Security is the foundation. A tunnel creates an entry point from the internet into your local machine -- treat it with the same seriousness as a production server. The [TLS 1.3 in Tunneling](/blog/tls-13-tunnel-security/) article digs deeper into the encryption side.

### Practice 1: Always Use TLS-Encrypted Tunnels

Every byte of data between your machine and the relay server must be encrypted. Without TLS, anyone on the network path — your ISP, a Wi-Fi operator, an attacker — can read your traffic. This includes API tokens, session cookies, and request bodies.

fxTunnel encrypts all connections with TLS 1.3 by default. You do not need to configure anything:

```bash
# TLS is automatic — every tunnel is encrypted
fxtunnel http 8080
# -> https://abc123.fxtun.dev (TLS 1.3)
```

Verify the TLS version for any tunnel:

```bash
# Check TLS version
openssl s_client -connect your-tunnel.fxtun.dev:443 2>/dev/null | grep "Protocol"
# -> Protocol  : TLSv1.3
```

**Action:** If your current tool does not encrypt traffic by default, switch to one that does.

### Practice 2: Stop Tunnels When Idle

Every running tunnel is an open door. When you stop working — close it. This sounds obvious, but forgotten tunnels are the number one security issue in teams that use tunneling daily.

```bash
# Start a tunnel, do your work, then stop it
fxtunnel http 8080
# Press Ctrl+C when done

# For scripted tunnels, use trap to ensure cleanup
fxtunnel http 8080 &
TUNNEL_PID=$!
trap "kill $TUNNEL_PID 2>/dev/null" EXIT
```

**Action:** Add a `trap` to every script that starts a tunnel. Set a reminder if you tend to forget.

### Practice 3: Never Expose Production Data

A tunnel should point at a development server with test data, test API keys, and test user accounts. Never tunnel into a production database, a production API, or a service that handles real user data.

```bash
# Good: tunnel to a dev server with test data
DATABASE_URL="postgres://dev:dev@localhost:5432/testdb" npm run dev
fxtunnel http 3000

# Bad: tunneling into production
# fxtunnel tcp 5432  # pointing at production PostgreSQL — NEVER DO THIS
```

**Action:** Create a checklist for your team that explicitly forbids tunneling to production services.

### Practice 4: Validate Webhook Signatures

Even with TLS, always validate webhook signatures on your server. Stripe, GitHub, Telegram, and other services sign their webhook payloads. Validating signatures ensures that the request actually came from the expected service, not from an attacker who discovered your tunnel URL.

```bash
# Start a tunnel for webhook testing
fxtunnel http 8080
# -> https://wh-dev.fxtun.dev

# In your server code, always verify the signature:
# Stripe:  stripe.webhooks.constructEvent(body, sig, secret)
# GitHub:  crypto.timingSafeEqual(expectedSig, actualSig)
```

The [webhook testing guide](/blog/webhook-testing-with-tunnel/) walks through the full setup.

**Action:** Add signature validation to every webhook endpoint before you start testing through a tunnel.

### Practice 5: Use Environment Variables for Tokens

Never hardcode tunnel tokens, API keys, or secrets in scripts or configuration files. Use environment variables.

```bash
# Store your fxTunnel token in the environment
export FXTUNNEL_TOKEN="your-token-here"

# The client reads it automatically
fxtunnel http 8080

# In CI/CD, use secrets
# GitHub Actions:
#   env:
#     FXTUNNEL_TOKEN: ${{ secrets.FXTUNNEL_TOKEN }}
```

**Action:** Audit your scripts and config files for hardcoded tokens. Move them to environment variables or a secrets manager.

### Practice 6: Prefer Open-Source Tools

An open-source tunnel lets you verify the TLS implementation, inspect data handling, and audit the codebase. With a closed-source tool, you trust the vendor's word. fxTunnel is fully open on [GitHub](https://github.com/mephistofox/fxtun.dev) — every line of code is auditable.

The [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) and [tunneling tools roundup](/blog/best-tunneling-tools-2026/) articles compare open-source and closed-source options side by side.

**Action:** Check whether your current tunneling tool is open source. If not, evaluate open-source alternatives.

### Practice 7: Keep the Client Updated

Every fxTunnel release includes an updated Go runtime with the latest security patches for `crypto/tls`. Outdated clients may have known vulnerabilities.

```bash
# Update fxTunnel to the latest version
curl -fsSL https://fxtun.ru/install.sh | bash

# Or via Go
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest

# Check current version
fxtunnel --version
```

**Action:** Set a monthly reminder to update your tunnel client. In CI/CD, always install the latest version.

---

## Performance Best Practices (Practices 8-12)

A tunnel adds latency because traffic goes through a relay server. These practices help you minimize the overhead and get the best performance from your tunnels.

### Practice 8: Match the Tunnel Protocol to Your Workload

fxTunnel supports three protocols: HTTP, TCP, and UDP. Choosing the right one avoids unnecessary overhead.

```bash
# Web application — use HTTP tunnel (optimized for HTTP traffic)
fxtunnel http 3000

# Database access — use TCP tunnel
fxtunnel tcp 5432

# Game server or IoT sensor data — use UDP tunnel
fxtunnel udp 27015
```

HTTP tunnels are optimized for web servers and APIs. TCP fits [databases](/blog/docker-tunnel-expose-containers/), SSH, and other stream protocols. UDP is the right choice for real-time, latency-sensitive workloads. The [TCP and UDP tunneling](/blog/tcp-udp-tunneling-explained/) article covers the differences in depth.

**Action:** Review your tunnel commands. Make sure you are using the right protocol for each service.

### Practice 9: Use Health Checks Before Opening Tunnels

Do not start a tunnel until your application is ready to accept requests. A tunnel pointing at a server that is still booting will serve errors to anyone who visits the URL.

```bash
#!/bin/bash
# start-with-tunnel.sh — start app, wait for health, then tunnel

# Start the application
npm run dev &
APP_PID=$!

# Wait for the app to be ready
echo "Waiting for app to start..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:3000/health > /dev/null 2>&1; then
    echo "App is ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "App failed to start within 60 seconds."
    kill $APP_PID 2>/dev/null
    exit 1
  fi
  sleep 2
done

# Start the tunnel
fxtunnel http 3000 &
TUNNEL_PID=$!

# Cleanup on exit
trap "kill $APP_PID $TUNNEL_PID 2>/dev/null" EXIT

wait $APP_PID
```

**Action:** Wrap your tunnel startup in a script that waits for a health check response.

### Practice 10: Pick a Relay Server Close to Your Users

If you are demoing to a client in Europe and the relay server is in Asia, every request adds 200+ ms of latency. With fxTunnel SaaS, the relay is automatically selected. For self-hosted deployments, place the server geographically close to the people who will access the tunnel.

```bash
# SaaS — relay selection is automatic
fxtunnel http 8080

# Self-hosted — deploy the server near your users
fxtunnel server --listen :443 --tls-cert cert.pem --tls-key key.pem
```

**Action:** For self-hosted setups, deploy the relay in the same region as your primary audience.

### Practice 11: Monitor Latency and Throughput

If your tunnel feels slow, measure it. A simple curl timing test reveals whether the issue is the tunnel or your application.

```bash
# Measure response time through the tunnel
curl -o /dev/null -s -w "DNS: %{time_namelookup}s\nConnect: %{time_connect}s\nTLS: %{time_appconnect}s\nTotal: %{time_total}s\n" https://your-tunnel.fxtun.dev/api/health

# Compare with direct localhost access
curl -o /dev/null -s -w "Total: %{time_total}s\n" http://localhost:8080/api/health
```

The difference between the two measurements is the tunnel overhead. Typically, it is 20-80 ms for a well-configured tunnel.

**Action:** Add latency measurements to your tunnel testing workflow.

### Practice 12: Use the Traffic Inspector for Debugging

fxTunnel's built-in traffic inspector lets you see every request and response passing through the tunnel in real time. This is faster than setting up a separate proxy like mitmproxy or Charles.

```bash
# Start a tunnel with the inspector enabled
fxtunnel http 8080
# Inspector available at the fxTunnel dashboard

# Replay a specific request for debugging
# Use the replay button in the inspector UI
```

The inspector shows headers, bodies, status codes, and timing for every request. Replay lets you re-send a request without triggering the original event again — perfect for debugging webhook handlers. The [Traffic Inspector article](/blog/traffic-inspector-debug-requests/) covers the feature in detail.

**Action:** Use the inspector instead of adding debug logs to your application when troubleshooting tunnel issues.

---

## Team Workflow Best Practices (Practices 13-16)

When multiple developers on a team use tunnels, coordination matters. These practices prevent conflicts and confusion.

### Practice 13: Standardize Tunnel Usage with a Wrapper Script

Instead of every developer running raw `fxtunnel` commands with different flags, create a project-level wrapper script that standardizes the configuration.

```bash
#!/bin/bash
# scripts/tunnel.sh — project tunnel launcher
# Usage: ./scripts/tunnel.sh [service] [port]

SERVICE=${1:-"web"}
PORT=${2:-"3000"}

# Ensure the app is running
if ! curl -sf http://localhost:$PORT/health > /dev/null 2>&1; then
  echo "Error: no service running on port $PORT"
  echo "Start it first: npm run dev"
  exit 1
fi

echo "Starting tunnel for $SERVICE on port $PORT..."
fxtunnel http $PORT --log-file "/tmp/tunnel-${SERVICE}.log" &
TUNNEL_PID=$!

# Wait for the URL
for i in $(seq 1 10); do
  TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' "/tmp/tunnel-${SERVICE}.log" 2>/dev/null || true)
  if [ -n "$TUNNEL_URL" ]; then
    echo "Tunnel active: $TUNNEL_URL -> localhost:$PORT"
    break
  fi
  sleep 1
done

trap "kill $TUNNEL_PID 2>/dev/null; echo 'Tunnel stopped.'" EXIT
wait $TUNNEL_PID
```

Add this script to your repository so every team member uses the same workflow.

**Action:** Create a `scripts/tunnel.sh` in your project and commit it to the repo.

### Practice 14: Document Tunnel URLs in Shared Channels

When you open a tunnel for a demo or review, post the URL in your team's Slack channel (or equivalent). Include what the tunnel points to, how long it will be active, and who started it.

A simple format:

```
Tunnel active:
  URL:     https://abc123.fxtun.dev
  Service: frontend (port 3000)
  Who:     @alice
  Expires: ~30 minutes
```

**Action:** Agree on a format and a channel where tunnel URLs are shared.

### Practice 15: Use Custom Domains for Stable URLs

Random subdomains like `abc123.fxtun.dev` change every time you restart the tunnel. For demos, QA, and integrations that need a stable URL, use a custom domain. fxTunnel supports custom domains via any DNS provider on paid plans.

```bash
# With a custom domain configured
fxtunnel http 3000
# -> https://dev.yourcompany.com -> localhost:3000
```

This is especially useful for OAuth callback URLs, which require a fixed redirect URI. The [OAuth callback guide](/blog/oauth-callback-localhost-tunnel/) walks through the setup.

**Action:** Set up a custom domain for your most-used tunnel if you are on a paid plan.

### Practice 16: Separate Tunnels per Microservice

If your project has multiple services (frontend, backend API, worker), create a separate tunnel for each instead of routing everything through one. This makes debugging easier and keeps URLs predictable.

```bash
# Terminal 1: Frontend
fxtunnel http 3000
# -> https://fe-xyz.fxtun.dev

# Terminal 2: Backend API
fxtunnel http 4000
# -> https://api-xyz.fxtun.dev

# Terminal 3: Admin panel
fxtunnel http 5000
# -> https://admin-xyz.fxtun.dev
```

The free tier does not limit the number of simultaneous tunnels. Paid plans add named tunnels with custom domains.

**Action:** Map out your services and assign a tunnel to each one.

---

## CI/CD Best Practices (Practices 17-20)

Tunnels in CI/CD pipelines unlock preview environments, E2E testing against real URLs, and automated webhook integration tests. The [CI/CD preview environments guide](/blog/cicd-tunnel-preview-environments/) covers the full workflow.

### Practice 17: Automate Tunnel Creation in Pipelines

Install fxTunnel in your CI runner and start tunnels programmatically. This is the foundation for all CI/CD tunnel workflows.

```yaml
# .github/workflows/tunnel-test.yml
name: Tunnel Integration Test

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start application
        run: |
          docker compose up -d --build
          for i in $(seq 1 30); do
            curl -sf http://localhost:8080/health && break
            sleep 2
          done

      - name: Install fxTunnel
        run: curl -fsSL https://fxtun.ru/install.sh | bash

      - name: Create tunnel
        id: tunnel
        run: |
          fxtunnel http 8080 --log-file /tmp/tunnel.log &
          echo "tunnel_pid=$!" >> "$GITHUB_OUTPUT"
          sleep 5
          TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' /tmp/tunnel.log)
          echo "url=$TUNNEL_URL" >> "$GITHUB_OUTPUT"

      - name: Run tests against tunnel URL
        run: |
          curl -sf ${{ steps.tunnel.outputs.url }}/api/health
          echo "Tunnel is working: ${{ steps.tunnel.outputs.url }}"

      - name: Cleanup
        if: always()
        run: |
          kill ${{ steps.tunnel.outputs.tunnel_pid }} || true
          docker compose down
```

**Action:** Add a tunnel step to your CI pipeline and verify it works with a simple health check.

### Practice 18: Generate Preview URLs for Every PR

The most powerful CI/CD use case for tunnels: automatic preview environments. Every pull request gets a public URL with the running application. Reviewers click the link instead of checking out the branch locally.

```yaml
# Key step: comment the preview URL on the PR
- name: Comment PR with preview URL
  uses: actions/github-script@v7
  with:
    script: |
      const url = '${{ steps.tunnel.outputs.url }}';
      await github.rest.issues.createComment({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: context.issue.number,
        body: `## Preview\n\n| | |\n|---|---|\n| **URL** | ${url} |\n| **Commit** | \`${context.sha.substring(0, 7)}\` |`,
      });
```

The [CI/CD preview environments article](/blog/cicd-tunnel-preview-environments/) has the full workflow with Docker Compose, cleanup, and GitLab CI examples.

**Action:** Set up a preview environment workflow for your most active repository.

### Practice 19: Run E2E Tests Against Tunnel URLs

A tunnel gives you a public HTTPS URL that behaves like a real deployment. Use it for E2E testing with Playwright, Cypress, or Selenium.

```bash
# Start tunnel, then run E2E tests
fxtunnel http 3000 --log-file /tmp/tunnel.log &
sleep 5
TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' /tmp/tunnel.log)

# Playwright
npx playwright test --base-url="$TUNNEL_URL"

# Cypress
npx cypress run --config baseUrl="$TUNNEL_URL"
```

Testing against a tunnel URL catches issues that localhost testing misses: CORS problems, mixed content warnings, TLS certificate handling, and real DNS resolution.

**Action:** Add a tunnel-based E2E test step to your CI pipeline.

### Practice 20: Clean Up Tunnels Automatically

A forgotten tunnel in CI is both a security risk and a resource waste. Always clean up.

```bash
#!/bin/bash
# ci-tunnel-wrapper.sh — start tunnel with automatic cleanup

cleanup() {
  echo "Cleaning up tunnel..."
  kill $TUNNEL_PID 2>/dev/null
  docker compose down 2>/dev/null
}
trap cleanup EXIT INT TERM

# Start services and tunnel
docker compose up -d --build
fxtunnel http 8080 --log-file /tmp/tunnel.log &
TUNNEL_PID=$!

# Wait for tunnel URL
sleep 5
TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' /tmp/tunnel.log)
echo "Tunnel: $TUNNEL_URL"

# Run your tests or wait for manual review
"$@"

# Cleanup happens automatically via trap
```

In GitHub Actions, use `concurrency` with `cancel-in-progress: true` and a separate cleanup workflow triggered on PR close.

**Action:** Verify that every CI job that creates a tunnel also destroys it — even when the job fails.

---

## Environment Variable Patterns for Tunnel Workflows

Here is a reference for the environment variables that streamline tunnel usage across different contexts.

```bash
# Local development
export FXTUNNEL_TOKEN="your-token"          # Auth token for paid features
export TUNNEL_PORT="8080"                    # Default port
export TUNNEL_LOG="/tmp/tunnel.log"          # Log file path

# CI/CD (GitHub Actions)
# Store FXTUNNEL_TOKEN as a repository secret
# Reference in workflow:
#   env:
#     FXTUNNEL_TOKEN: ${{ secrets.FXTUNNEL_TOKEN }}

# Docker Compose
# Pass the tunnel URL as an env variable to other services:
#   environment:
#     - API_URL=${TUNNEL_URL}

# .env file (DO NOT commit to git)
FXTUNNEL_TOKEN=your-token
TUNNEL_PORT=8080
```

Add `.env` to your `.gitignore` to prevent accidental commits of tokens.

## Choosing the Right Plan for Your Workflow

fxTunnel offers three tiers that match different workflow needs:

| Feature | Free | From $5/mo | From $10/mo |
|---|---|---|---|
| HTTP/TCP/UDP tunnels | Unlimited | Up to 5 named | 10+ named |
| Traffic & connections | No limits | No limits | No limits |
| Custom domains | No | Yes (any DNS) | Yes (any DNS) |
| Traffic inspector | No | Yes | Yes |
| Replay | No | Yes | Yes |
| Good fit | Solo dev, testing | Small teams, demos | Growing teams, CI/CD |

Solo developers and basic testing are covered by the free tier. Teams that need stable URLs and webhook debugging will find the $5/mo plan fits well. If you are running 10+ tunnels in CI/CD pipelines, the $10/mo plan scales with that workflow.

## Common Anti-Patterns to Avoid

These are mistakes we see repeatedly. Avoid them.

### Anti-Pattern: Tunneling to Production

Never create a tunnel pointing at a production database, API, or service. Use test environments exclusively.

### Anti-Pattern: Sharing Tunnel Tokens in Code

Tokens in source code end up in git history, CI logs, and screenshots. Always use environment variables or a secrets manager.

### Anti-Pattern: Running Tunnels 24/7

A tunnel is a temporary tool. If you need permanent access, deploy your application properly behind a [reverse proxy](/blog/reverse-proxy-vs-tunnel/) with proper TLS and monitoring.

### Anti-Pattern: One Tunnel for Everything

Routing multiple services through a single tunnel makes debugging painful. Use one tunnel per service.

### Anti-Pattern: Ignoring Logs

Tunnel client logs contain connection errors, reconnection events, and URL assignments. Read them when something goes wrong.

```bash
# Always use a log file for background tunnels
fxtunnel http 8080 --log-file /tmp/tunnel.log &

# Check logs when debugging
tail -f /tmp/tunnel.log
```

## Putting It All Together: A Complete Project Setup

Here is a real-world example that combines multiple practices into a single developer workflow.

```bash
#!/bin/bash
# dev-tunnel-setup.sh — complete tunnel workflow for a multi-service project
set -euo pipefail

# Configuration
FRONTEND_PORT=3000
BACKEND_PORT=4000
LOG_DIR="/tmp/tunnels"
mkdir -p "$LOG_DIR"

# Cleanup handler
cleanup() {
  echo "Stopping all tunnels..."
  kill $FE_TUNNEL_PID $BE_TUNNEL_PID 2>/dev/null || true
  echo "Done."
}
trap cleanup EXIT INT TERM

# Verify services are running
echo "Checking services..."
curl -sf http://localhost:$FRONTEND_PORT > /dev/null || { echo "Frontend not running on port $FRONTEND_PORT"; exit 1; }
curl -sf http://localhost:$BACKEND_PORT/health > /dev/null || { echo "Backend not running on port $BACKEND_PORT"; exit 1; }

# Start tunnels
echo "Starting tunnels..."
fxtunnel http $FRONTEND_PORT --log-file "$LOG_DIR/frontend.log" &
FE_TUNNEL_PID=$!

fxtunnel http $BACKEND_PORT --log-file "$LOG_DIR/backend.log" &
BE_TUNNEL_PID=$!

# Wait for URLs
sleep 5
FE_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' "$LOG_DIR/frontend.log" || echo "pending")
BE_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' "$LOG_DIR/backend.log" || echo "pending")

echo ""
echo "========================================="
echo "  Tunnels Active"
echo "========================================="
echo "  Frontend: $FE_URL -> localhost:$FRONTEND_PORT"
echo "  Backend:  $BE_URL -> localhost:$BACKEND_PORT"
echo "========================================="
echo ""
echo "Press Ctrl+C to stop all tunnels."

# Keep alive
wait
```

This script enforces health checks (Practice 9), uses log files (anti-pattern avoidance), separates tunnels per service (Practice 16), and ensures cleanup on exit (Practice 20).

## FAQ

### How many tunnels can I run simultaneously on the free tier?

There is no hard limit -- run as many as you need. The free tier does not restrict tunnel count, traffic, or connections. Paid plans layer on custom domains, a traffic inspector, and replay starting at $5/mo.

### Should I use a tunnel in production?

Tunnels are built for development and testing. Production workloads belong behind a proper reverse proxy. That said, for staging environments, internal tools, and long-running demos, the paid plans (from $5/mo) support custom domains and stable tunnels that work well in those roles.

### How do I keep tunnel URLs stable across restarts?

By default, each restart gets a new random subdomain. If you need a permanent address, set up a custom domain through your DNS provider -- available on paid plans from $5/mo. The URL then survives restarts and reconnections.

### Can I use tunnels in CI/CD pipelines?

Yes -- install fxTunnel in your CI runner, spin up the tunnel in the background, grab the URL from the log output, and pass it to your E2E tests or preview environment. The CI/CD section above has copy-paste scripts for GitHub Actions.

### What is the biggest security mistake when using tunnels?

Leaving a tunnel running after you have finished. Every active tunnel is an open entry point to your machine. Get in the habit of hitting `Ctrl+C` when the session is over, and use `trap` in scripts to ensure cleanup even if something fails.
