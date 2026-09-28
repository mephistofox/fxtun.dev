---
title: "Tunnel Security Checklist — Audit Guide for Devs"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Security audit checklist for tunnels: TLS verification, auth tokens, port exposure, monitoring, and a 15-point guide for production-grade security."
tags: [security, tunneling, checklist, audit, fxTunnel, developer-tools, tls]
image: ""
author: "fxTunnel Team"
faq:
  - q: "How many security checks should I run before exposing a tunnel to the internet?"
    a: "At minimum, verify five critical areas: TLS encryption is active (TLS 1.3), authentication tokens are configured and not hardcoded, only the required port is exposed, firewall rules restrict access, and monitoring or logging is enabled. The full checklist in this article covers 15 points for production-grade security."
  - q: "Can I use a tunnel safely in a CI/CD pipeline?"
    a: "Yes, but you must treat the tunnel token as a secret — store it in your CI/CD secrets manager (GitHub Secrets, GitLab CI variables), never commit it to the repository, and ensure the tunnel process is terminated after the pipeline completes. fxTunnel's CLI makes this straightforward with the --token flag."
  - q: "Is it safe to expose a database through a tunnel?"
    a: "A TCP tunnel to a database is acceptable for development and testing if TLS is active and the tunnel uses token-based authentication. Never expose a production database this way. For development, use test credentials, restrict access with firewall rules, and shut down the tunnel when you are done."
  - q: "How do I verify that my tunnel uses TLS 1.3 and not an older version?"
    a: "Run openssl s_client -connect your-tunnel.fxtun.dev:443 and check the Protocol line in the output. It should read TLSv1.3. You can also inspect the certificate in your browser via the lock icon. fxTunnel uses TLS 1.3 by default with no configuration required."
  - q: "What is the difference between a security checklist and a penetration test for tunnels?"
    a: "A security checklist is a systematic review of configuration and best practices — encryption, authentication, port exposure, logging. A penetration test actively probes for vulnerabilities by simulating attacks. Both are important: the checklist prevents common misconfigurations, while a pentest uncovers issues that only surface under adversarial conditions."
---

## Why a Security Checklist Matters for Tunnels

A tunnel opens a direct path from the public internet to your local machine. Every misconfiguration — an unencrypted connection, a leaked token, an overly permissive firewall rule — becomes an attack vector. Unlike a typical web deployment behind a reverse proxy with multiple layers of defense, a tunnel is often the only layer standing between an attacker and your local service.

So where do most tunnel security incidents actually come from? Not protocol vulnerabilities -- human error. A developer forgot to rotate a token, a tunnel stayed running overnight with a production database behind it, or a firewall allowed traffic from any source. A structured checklist eliminates these mistakes systematically.

This article provides a 15-point security audit checklist for tunnels, with practical commands, configuration examples, and a comparison of secure vs insecure practices. It applies to any tunneling tool, but the examples use [fxTunnel](/blog/fxtunnel-architecture/) — an open-source tunneling tool for HTTP, TCP, and UDP. For background on how tunnels work, see [What Is Tunneling](/blog/what-is-tunneling/).

## The 15-Point Tunnel Security Checklist

Below is the complete checklist. Each point includes what to check, why it matters, and how to verify it. The checklist is ordered from most critical (encryption and authentication) to operational (monitoring and lifecycle).

### 1. Verify TLS Encryption Is Active

**Why:** Without TLS, all traffic between your machine and the relay server travels in plain text. Anyone on the network path — your ISP, a Wi-Fi operator, an attacker — can read and modify it. Our [TLS 1.3 in Tunneling](/blog/tls-13-tunnel-security/) article goes deeper into why this matters.

**How to check:**

```bash
# Check the TLS version for your tunnel endpoint
openssl s_client -connect your-tunnel.fxtun.dev:443 2>/dev/null | grep "Protocol"
# Expected output: Protocol  : TLSv1.3

# Verify the full certificate chain
openssl s_client -connect your-tunnel.fxtun.dev:443 -showcerts 2>/dev/null | \
  openssl x509 -noout -subject -issuer -dates
```

fxTunnel encrypts all connections with TLS 1.3 by default — no configuration required.

### 2. Confirm TLS 1.3 (Not TLS 1.2 or Older)

**Why:** TLS 1.2 supports legacy cipher suites that have known weaknesses (RC4, 3DES, CBC). TLS 1.3 eliminates these entirely and enforces forward secrecy on every session.

**How to check:**

```bash
# Force TLS 1.3 and confirm the connection succeeds
openssl s_client -connect your-tunnel.fxtun.dev:443 -tls1_3 2>/dev/null | grep "Protocol"
# Expected: Protocol  : TLSv1.3

# Verify that TLS 1.1 is rejected
openssl s_client -connect your-tunnel.fxtun.dev:443 -tls1_1 2>&1 | grep -i "error\|alert"
# Expected: handshake failure or protocol version error
```

### 3. Use Authentication Tokens

**Why:** Without authentication, anyone who discovers your relay server can create tunnels. Tokens ensure that only authorized clients can connect.

**How to check:**

```bash
# Generate a token on the server
fxtunnel server token generate --name "dev-alice"
# -> Token: fxt_a1b2c3d4e5f6...

# Use the token when creating a tunnel
fxtunnel http 8080 --token fxt_a1b2c3d4e5f6...

# Verify: try connecting without a token — should fail
fxtunnel http 8080
# -> Error: authentication required
```

### 4. Never Hardcode Tokens in Source Code

**Why:** A token committed to a Git repository is a token leaked to everyone with access to the repository — including public GitHub if you push by mistake.

**How to check:**

```bash
# Search your codebase for leaked tokens
grep -rn "fxt_" . --include="*.sh" --include="*.yml" --include="*.yaml" \
  --include="*.env" --include="*.json" --include="Makefile"

# Correct approach: use environment variables
export FXTUNNEL_TOKEN="fxt_a1b2c3d4e5f6..."
fxtunnel http 8080
# fxTunnel reads FXTUNNEL_TOKEN from the environment automatically
```

Store tokens in your CI/CD secrets manager (GitHub Actions secrets, GitLab CI variables, AWS Secrets Manager) and never in version control.

### 5. Expose Only the Required Port

**Why:** Every open port is an attack surface. If you need to tunnel port 8080, do not tunnel your entire machine. A single-port tunnel limits the blast radius if something goes wrong.

**How to check:**

```bash
# Correct: expose only the port your application uses
fxtunnel http 8080

# Incorrect: exposing multiple ports "just in case"
# fxtunnel tcp 22    <- SSH exposed to the internet!
# fxtunnel tcp 5432  <- PostgreSQL exposed to the internet!
```

Only create tunnels for ports that need external access at that moment.

### 6. Configure Firewall Rules on the Tunnel Host

**Why:** A firewall is your second line of defense. Even if a tunnel is compromised, firewall rules prevent lateral movement within your network.

**How to check:**

```bash
# UFW (Ubuntu/Debian) — allow only outbound tunnel traffic
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow out 443/tcp comment "fxTunnel TLS connection"
sudo ufw enable
sudo ufw status verbose

# iptables — restrict incoming connections to localhost only
sudo iptables -A INPUT -i lo -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -j DROP

# Verify no unexpected ports are listening
ss -tlnp | grep -v "127.0.0.1"
```

### 7. Use Test Data, Never Production Data

**Why:** A tunnel creates a public entry point to your local service. If that service runs against a production database with real user data, a compromised tunnel means a data breach.

**How to check:**

- Verify your application connects to a test/development database, not production.
- Use test API keys (Stripe test mode, sandbox environments).
- Use fake user data (no real emails, no real payment details).

```bash
# Check your database connection string
env | grep DATABASE_URL
# Should be: DATABASE_URL=postgres://localhost:5432/myapp_test
# Not:       DATABASE_URL=postgres://prod-db.example.com/myapp_prod
```

### 8. Shut Down Tunnels When Not in Use

**Why:** A running tunnel is an open door. The longer it stays open, the greater the window for an attacker to discover and exploit it.

**How to check:**

```bash
# List all running fxTunnel processes
ps aux | grep fxtunnel

# Stop the tunnel when you are done
# Press Ctrl+C in the terminal, or:
kill $(pgrep fxtunnel)

# In CI/CD: use a timeout or explicit cleanup step
timeout 300 fxtunnel http 8080 --token "$FXTUNNEL_TOKEN" &
TUNNEL_PID=$!
# ... run your tests ...
kill $TUNNEL_PID
```

### 9. Validate Webhook Signatures

**Why:** Even with TLS, you should verify that incoming requests actually come from the service you expect (Stripe, GitHub, Telegram). Webhook signatures provide this guarantee. If you are setting up webhooks through a tunnel, the [Webhook Testing with a Tunnel](/blog/webhook-testing-with-tunnel/) walkthrough covers the full workflow.

**How to check (Node.js example):**

```javascript
const crypto = require('crypto');

function verifyStripeSignature(payload, sigHeader, secret) {
  const elements = sigHeader.split(',');
  const timestamp = elements.find(e => e.startsWith('t=')).slice(2);
  const signature = elements.find(e => e.startsWith('v1=')).slice(3);

  const signedPayload = `${timestamp}.${payload}`;
  const expected = crypto
    .createHmac('sha256', secret)
    .update(signedPayload)
    .digest('hex');

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}
```

### 10. Monitor Tunnel Traffic with the Request Inspector

**Why:** You cannot secure what you cannot see. Monitoring incoming requests lets you detect unexpected traffic patterns, malicious payloads, and unauthorized access attempts.

**How to check:**

fxTunnel includes a built-in request inspector that logs every HTTP request with full headers, body, and timing. On paid plans (from $5/mo), the inspector also supports replay — letting you re-send any request for debugging.

```bash
# Start a tunnel with the inspector enabled
fxtunnel http 8080 --inspect

# The inspector UI is available in your browser
# -> Inspector: http://127.0.0.1:4040
```

Review the inspector output regularly for:
- Unexpected source IPs or User-Agent strings
- Requests to paths you did not create
- Abnormally large request bodies
- Rapid-fire requests (possible brute-force attempts)

### 11. Rotate Tokens Regularly

**Why:** Tokens can leak through logs, screenshots, terminal history, or shared environments. Regular rotation limits the window of exposure.

**How to check:**

```bash
# Revoke an old token
fxtunnel server token revoke --name "dev-alice"

# Generate a new token
fxtunnel server token generate --name "dev-alice-2026-q2"
# -> Token: fxt_x9y8z7w6v5u4...

# Update the environment variable or secrets manager
export FXTUNNEL_TOKEN="fxt_x9y8z7w6v5u4..."
```

Rotate tokens at least quarterly, or immediately if you suspect a leak.

### 12. Restrict Tunnel Access by IP (When Possible)

**Why:** If you know which IP addresses need to access your tunnel (your office, a CI/CD runner, a specific client), restricting access by IP eliminates all other traffic.

**How to check:**

```bash
# Restrict at the firewall level (server side)
sudo ufw allow from 203.0.113.10 to any port 443 comment "Office IP"
sudo ufw allow from 198.51.100.0/24 to any port 443 comment "CI/CD subnet"
sudo ufw deny 443/tcp

# Or restrict at the application level with a reverse proxy
# nginx example:
# location / {
#     allow 203.0.113.10;
#     allow 198.51.100.0/24;
#     deny all;
#     proxy_pass http://localhost:8080;
# }
```

### 13. Use Custom Domains with Verified Certificates

**Why:** A random subdomain like `abc123.fxtun.dev` is fine for quick testing. For client demos, webhook integrations that validate the sender, or long-running tunnels, a custom domain with a proper certificate builds trust and allows certificate pinning.

fxTunnel supports custom domains on paid plans (from $5/mo). The certificate is issued automatically via Let's Encrypt.

```bash
# Use a custom domain for your tunnel
fxtunnel http 8080 --domain demo.yourcompany.com

# Verify the certificate matches your domain
openssl s_client -connect demo.yourcompany.com:443 2>/dev/null | \
  openssl x509 -noout -subject
# Expected: subject=CN = demo.yourcompany.com
```

### 14. Audit Open-Source Code Before Deployment

**Why:** Open source means you can verify the security implementation. Closed-source tools like ngrok require you to trust the vendor. With fxTunnel, the code is on [GitHub](https://github.com/mephistofox/fxtun.dev) — audit it before you deploy.

**What to audit:**

- TLS configuration: minimum version, cipher suites, certificate validation.
- Token handling: storage, comparison (timing-safe), revocation.
- Data flow: does the relay server log traffic content? (fxTunnel does not.)
- Dependencies: are they up to date? Are there known CVEs?

```bash
# Clone and audit the fxTunnel source
git clone https://github.com/mephistofox/fxtun.dev.git
cd fxtun.dev

# Check TLS configuration
grep -rn "tls.Config" --include="*.go"

# Check for known vulnerabilities in dependencies
go list -m all | nancy sleuth
# or
govulncheck ./...
```

### 15. Document Your Security Configuration

**Why:** Security without documentation is security that degrades over time. When team members change, undocumented configurations get lost, tokens stop being rotated, and firewall rules become stale.

**What to document:**

- Which ports are tunneled and why.
- Where tokens are stored (secrets manager, environment).
- Token rotation schedule.
- Firewall rules and their purpose.
- Who has access to create tunnels.
- Incident response procedure: what to do if a token leaks or a tunnel is compromised.

## Secure vs Insecure Practices: Comparison Table

| Area | Insecure Practice | Secure Practice |
|---|---|---|
| **Encryption** | HTTP tunnel without TLS; accepting TLS 1.0/1.1 | TLS 1.3 enforced; verify with `openssl s_client` |
| **Authentication** | No token; shared token across team | Per-user tokens; stored in secrets manager |
| **Token storage** | Hardcoded in source code or `.env` committed to Git | Environment variable or CI/CD secrets; `.env` in `.gitignore` |
| **Port exposure** | Multiple ports tunneled simultaneously | Only the required port; tunnel stopped when idle |
| **Firewall** | No firewall rules; all ports open | `ufw` or `iptables` restricting access by IP and port |
| **Data** | Production database behind the tunnel | Test database with fake data |
| **Monitoring** | No logging; blind to incoming traffic | Request inspector enabled; logs reviewed regularly |
| **Tunnel lifecycle** | Tunnel running 24/7 unattended | Tunnel started on demand; killed after use or via timeout |
| **Token rotation** | Same token for months or years | Quarterly rotation; immediate rotation on suspected leak |
| **Code audit** | Closed-source tunnel tool; trust the vendor | Open-source tool (fxTunnel); code audited before deployment |
| **Webhook validation** | Accept all incoming requests blindly | Verify webhook signatures (HMAC, RSA) |
| **Domain** | Random subdomain for client-facing demos | Custom domain with verified certificate |
| **Documentation** | No written security policy | Security configuration documented and maintained |
| **Dependencies** | Outdated tunnel client with known CVEs | Latest version; dependencies scanned with `govulncheck` |

## Applying the Checklist: A Practical Walkthrough

Let's walk through the checklist for an fxTunnel setup from start to finish.

### Step 1: Install and Verify the Client

```bash
# Install fxTunnel
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installed version
fxtunnel version
```

### Step 2: Configure Authentication

```bash
# Set the token via environment variable (never hardcode)
export FXTUNNEL_TOKEN="fxt_a1b2c3d4e5f6..."

# Verify the token is not in any tracked files
grep -rn "fxt_" . --include="*.sh" --include="*.yml" --include="*.env"
# Expected: no output
```

### Step 3: Start the Tunnel with Minimal Exposure

```bash
# Start an HTTP tunnel to your dev server
fxtunnel http 3000 --inspect
```

### Step 4: Verify TLS

```bash
# In another terminal, verify TLS 1.3
openssl s_client -connect your-tunnel.fxtun.dev:443 -tls1_3 2>/dev/null | grep "Protocol"
# -> Protocol  : TLSv1.3
```

### Step 5: Apply Firewall Rules

```bash
# Lock down the host — allow only outbound tunnel traffic
sudo ufw default deny incoming
sudo ufw allow out 443/tcp comment "fxTunnel"
sudo ufw enable
```

### Step 6: Monitor and Shut Down

```bash
# Check the inspector at http://127.0.0.1:4040 for unexpected traffic

# When done, stop the tunnel
# Press Ctrl+C or:
kill $(pgrep fxtunnel)

# Verify no tunnel processes remain
ps aux | grep fxtunnel
```

## Security Checklist for CI/CD Pipelines

Tunnels in CI/CD environments (preview deployments, integration tests with external APIs) require additional care. The [CI/CD Tunnel Preview Environments](/blog/cicd-tunnel-preview-environments/) article covers this in detail.

```yaml
# GitHub Actions example: secure tunnel in CI/CD
name: Integration Tests
on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start application
        run: npm start &

      - name: Create tunnel with timeout
        env:
          FXTUNNEL_TOKEN: ${{ secrets.FXTUNNEL_TOKEN }}
        run: |
          # Install fxTunnel
          curl -fsSL https://fxtun.ru/install.sh | bash

          # Start tunnel with a 5-minute timeout
          timeout 300 fxtunnel http 3000 &
          sleep 5  # Wait for the tunnel to initialize

      - name: Run integration tests
        run: npm test

      # Tunnel automatically dies after the job ends
      # or after the 300-second timeout — whichever comes first
```

Key CI/CD security rules:

1. **Store the token in CI/CD secrets** — never in the workflow file.
2. **Use a timeout** — prevent tunnels from running indefinitely if a job hangs.
3. **Generate per-pipeline tokens** — rotate tokens per environment or per branch.
4. **Clean up explicitly** — do not rely on process termination alone; verify with `ps`.

## fxTunnel Plans and Security Features

fxTunnel provides security features across all plans, with advanced capabilities on paid tiers:

| Feature | Free ($0) | Pro (from $5/mo) | Team (from $10/mo) |
|---|---|---|---|
| **TLS 1.3 encryption** | Yes | Yes | Yes |
| **Token authentication** | Yes | Yes | Yes |
| **HTTP/TCP/UDP tunnels** | Yes | Yes | Yes |
| **Request inspector** | Basic | Full + replay | Full + replay |
| **Custom domains** | No | Yes | Yes |
| **Concurrent tunnels** | 1 | Up to 5 | 10+ |
| **Priority support** | No | No | Yes |

The free plan covers TLS 1.3 encryption and token authentication with no traffic caps. Pro (from $5/mo) adds custom domains and the full request inspector with replay, and Team (from $10/mo) supports 10+ concurrent tunnels.

## FAQ

### How many security checks should I run before exposing a tunnel to the internet?

Start with five non-negotiable checks: confirm TLS 1.3 is active, make sure authentication tokens are set up and not embedded in code, open only the port you actually need, lock down firewall rules, and turn on logging or monitoring. The 15-point checklist above gives you a thorough audit path from there.

### Can I use a tunnel safely in a CI/CD pipeline?

You can, as long as you treat the tunnel token like any other secret. Store it in your CI/CD secrets manager (GitHub Secrets, GitLab CI variables), keep it out of the repository, and make sure the tunnel process gets killed when the pipeline finishes. With fxTunnel, the `--token` flag or the `FXTUNNEL_TOKEN` environment variable makes this easy to wire up.

### Is it safe to expose a database through a tunnel?

For development and testing -- yes, provided TLS is on and token-based authentication is in place. A production database is a different story: never expose one through a tunnel. When working locally, stick to test credentials, layer in firewall rules, and shut down the tunnel the moment you are done.

### How do I verify that my tunnel uses TLS 1.3 and not an older version?

The quickest check is `openssl s_client -connect your-tunnel.fxtun.dev:443` -- look for `TLSv1.3` in the `Protocol` line. You can also click the lock icon in your browser to inspect the certificate. fxTunnel enforces TLS 1.3 by default, so no extra configuration is needed.

### What is the difference between a security checklist and a penetration test for tunnels?

Think of them as complementary. A checklist walks through configuration and known good practices -- encryption, auth, port exposure, logging -- to prevent common mistakes. A penetration test actively tries to break things by simulating real attacks. You need both: the checklist catches misconfigurations, while the pentest reveals weaknesses that only show up under adversarial pressure.
