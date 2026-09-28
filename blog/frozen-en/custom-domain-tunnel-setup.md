---
title: "How to Set Up a Custom Domain for Your Development Tunnel"
date: 2026-03-11T14:00:00+03:00
draft: false
description: "Step-by-step guide to configuring a custom domain for fxTunnel: DNS setup, CNAME records, HTTPS certificates, and benefits for team development workflows."
tags: [custom-domain, tunnel, dns, cname, https, fxTunnel, developer-tools, team-workflow]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I use my own domain with fxTunnel?"
    a: "Yes. On the plan from $5/mo you can connect up to 5 custom domains. You point a CNAME record to tunnel.fxtun.dev, run fxtunnel http 8080 --domain myapp.example.com, and your tunnel is accessible at https://myapp.example.com with a valid TLS certificate provisioned automatically."
  - q: "Do I need to configure HTTPS certificates manually?"
    a: "No. fxTunnel provisions and renews TLS certificates automatically via Let's Encrypt the first time traffic hits your custom domain. You do not need to generate, upload, or manage certificates — everything is handled by the tunnel server."
  - q: "What happens if my DNS propagation is slow?"
    a: "DNS propagation typically takes 1 to 30 minutes, but in rare cases it can take up to 48 hours. Until the CNAME resolves to tunnel.fxtun.dev, the custom domain will not work. You can verify propagation with dig myapp.example.com CNAME or an online DNS checker. During this time, you can keep using your default fxtun.dev subdomain."
  - q: "Can multiple team members share a single custom domain?"
    a: "Yes. The team lead configures the domain once, and any team member with the correct token can start a tunnel on that domain. This is ideal for shared staging environments, QA testing, and demo workflows. On the plan from $10/mo you can run 10+ tunnels simultaneously."
  - q: "What is the difference between a custom domain and the default fxtun.dev subdomain?"
    a: "The default subdomain (like abc.fxtun.dev) is assigned automatically and works on all plans including free. A custom domain (like dev.example.com) uses your own DNS and looks professional in webhook configurations, OAuth redirect URIs, and client demos. Custom domains are available from $5/mo."
---

## Why Use a Custom Domain for Your Development Tunnel?

A custom domain turns your [local tunnel](/blog/what-is-tunneling/) into a stable, professional development endpoint. Instead of `abc.fxtun.dev`, your tunnel responds at `dev.yourcompany.com` — a URL that never changes, works across team members, and looks credible in client demos. Setup takes about 5 minutes.

Ever tried registering a random-looking subdomain as an OAuth redirect URI? Or pasting one into a Stripe webhook config, only to have it break when you restart the tunnel? The default `fxtun.dev` subdomains work well for solo development, but as soon as you integrate with third-party services, collaborate with a team, or present work to clients, they start creating friction.

Here is what a custom domain solves:

- **Stable webhook URLs** — services like Stripe, GitHub, and Telegram keep working after tunnel restarts because the domain stays the same. No need to update endpoint configurations. More on this in [Webhook Testing with a Tunnel](/blog/webhook-testing-with-tunnel/).
- **OAuth and SSO integration** — providers like Google, Auth0, and Okta allow you to register `https://dev.yourcompany.com/auth/callback` as a permanent redirect URI, eliminating the need to update it during development.
- **Professional demos** — show `https://demo.yourcompany.com` to a client instead of a random subdomain. It builds trust and looks intentional.
- **Team consistency** — every developer on the team connects to the same domain. QA can bookmark the URL. Documentation references stay valid.
- **Automatic HTTPS** — fxTunnel provisions a valid TLS certificate for your custom domain via Let's Encrypt. No manual certificate management. The [HTTPS on Localhost](/blog/https-localhost-development/) article explains the mechanics.

## Prerequisites

Before configuring a custom domain, make sure you have the following:

1. **fxTunnel CLI installed** — if you have not installed it yet, run the quick install command.
2. **A paid fxTunnel plan** — custom domains are available from $5/mo at [fxtun.dev](https://fxtun.dev).
3. **A registered domain** — you need access to the DNS settings of a domain you own (e.g., `example.com`).
4. **fxTunnel account token** — you will use this to authenticate the CLI.

```bash
# Install fxTunnel (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify installation
fxtunnel --version

# Authenticate with your account
fxtunnel auth login
```

## Step-by-Step: Configuring a Custom Domain

The whole process boils down to three steps: add a CNAME record at your DNS provider, register the domain in your fxTunnel account, and start a tunnel with the `--domain` flag. The TLS certificate is provisioned automatically on first request.

### Step 1. Add a CNAME Record in DNS

Log in to your DNS provider (Cloudflare, Route 53, GoDaddy, Namecheap, etc.) and add a CNAME record that points your desired subdomain to `tunnel.fxtun.dev`.

| Record type | Name | Value | TTL |
|---|---|---|---|
| CNAME | `dev` | `tunnel.fxtun.dev` | 300 (5 min) |

If you want to use `dev.example.com`, the Name field is `dev`. If you want `tunnel.example.com`, the Name is `tunnel`. You can use any subdomain you like.

**Important:** CNAME records work only for subdomains, not for the root domain (`example.com`). If you need to use a root domain, check whether your DNS provider supports CNAME flattening (Cloudflare does) or ALIAS records.

#### DNS Configuration Examples by Provider

**Cloudflare:**

1. Go to **DNS** > **Records** > **Add record**.
2. Type: `CNAME`, Name: `dev`, Target: `tunnel.fxtun.dev`.
3. Proxy status: **DNS only** (gray cloud). Do not use Cloudflare proxy — fxTunnel handles TLS termination.
4. TTL: Auto.

**AWS Route 53:**

1. Open your hosted zone.
2. Click **Create record**.
3. Record name: `dev`, Record type: `CNAME`, Value: `tunnel.fxtun.dev`, TTL: `300`.

**Namecheap:**

1. Go to **Domain List** > **Advanced DNS**.
2. Add a new record: Type `CNAME`, Host `dev`, Value `tunnel.fxtun.dev`, TTL `5 min`.

**GoDaddy:**

1. Go to **My Products** > **DNS**.
2. Add record: Type `CNAME`, Name `dev`, Value `tunnel.fxtun.dev`, TTL `600`.

### Step 2. Register the Domain in fxTunnel

After creating the DNS record, register the domain in your fxTunnel account. This tells the server to accept traffic for your custom domain and provision a TLS certificate.

```bash
# Register your custom domain
fxtunnel domain add dev.example.com

# Verify the domain (checks DNS)
fxtunnel domain verify dev.example.com
```

Expected output:

```
Domain: dev.example.com
Status: verified
CNAME:  tunnel.fxtun.dev ✓
TLS:    certificate will be provisioned on first request
```

If the DNS record has not propagated yet, you will see:

```
Domain: dev.example.com
Status: pending
CNAME:  not found — check your DNS settings
```

Wait a few minutes and try again. DNS propagation typically takes 1 to 30 minutes.

### Step 3. Start a Tunnel with Your Custom Domain

Once the domain is verified, start a tunnel with the `--domain` flag:

```bash
# Start a tunnel on your custom domain
fxtunnel http 8080 --domain dev.example.com
```

Output:

```
fxTunnel v1.x — tunnel is active
Custom domain: https://dev.example.com
Forwarding:    https://dev.example.com -> http://localhost:8080
TLS:           certificate provisioned (Let's Encrypt)
```

Open `https://dev.example.com` in your browser. You will see a green padlock — the TLS certificate is valid and trusted. All traffic from `https://dev.example.com` is now forwarded to your `localhost:8080`.

### Step 4. Verify the Setup

Run a quick check to make sure everything works end-to-end:

```bash
# Check DNS resolution
dig dev.example.com CNAME +short
# Expected: tunnel.fxtun.dev.

# Test HTTPS
curl -I https://dev.example.com
# Expected: HTTP/2 200

# Check certificate details
openssl s_client -connect dev.example.com:443 -servername dev.example.com < /dev/null 2>/dev/null | openssl x509 -noout -subject -issuer -dates
# Expected:
# subject=CN = dev.example.com
# issuer=O = Let's Encrypt
# notBefore=...
# notAfter=...
```

## TLS Certificate Management

fxTunnel handles TLS certificates for custom domains automatically. When the first request hits your custom domain, the server provisions a certificate from Let's Encrypt using the ACME protocol. Certificates are renewed automatically before expiration. You never need to touch a certificate file.

### How It Works

1. You point `dev.example.com` to `tunnel.fxtun.dev` via CNAME.
2. The fxTunnel server receives the first HTTPS request for `dev.example.com`.
3. The server uses the ACME HTTP-01 challenge to prove domain ownership to Let's Encrypt.
4. Let's Encrypt issues a certificate, and the server stores it.
5. The certificate is renewed automatically every 60 days (Let's Encrypt certificates are valid for 90 days).

This is the same approach used by CDN providers and [modern tunneling tools](/blog/best-tunneling-tools-2026/) alike. The difference is that you do not need to configure anything -- it just works.

### Certificate Provisioning Timeline

| Event | Time |
|---|---|
| CNAME propagation | 1-30 minutes (typically under 5) |
| First certificate issuance | 10-30 seconds after first request |
| Certificate renewal | Automatic, 30 days before expiration |
| Certificate type | RSA 2048-bit or ECDSA P-256 |
| Issuer | Let's Encrypt |

### Troubleshooting Certificate Issues

| Problem | Cause | Solution |
|---|---|---|
| Certificate not issued | CNAME not propagated | Wait for DNS propagation, verify with `dig` |
| Invalid certificate warning | Cloudflare proxy enabled | Disable Cloudflare proxy (use DNS-only mode) |
| Certificate expired | Renewal failed (rare) | Run `fxtunnel domain verify` to trigger renewal |
| Wrong domain on cert | Wildcard conflict | Ensure no conflicting DNS records |

## Custom Domain vs. Default Subdomain

The free tier gives you a persistent `fxtun.dev` subdomain that works well for personal development. A custom domain adds branding, stability, and team workflow benefits. Here is how they compare.

| Feature | Default subdomain (Free) | Custom domain (from $5/mo) |
|---|---|---|
| **URL format** | `abc.fxtun.dev` | `dev.example.com` |
| **Persistent across restarts** | Yes | Yes |
| **TLS certificate** | Automatic | Automatic (Let's Encrypt) |
| **Branding** | fxTunnel branded | Your domain |
| **OAuth redirect URIs** | Works, but may need updating | Register once, never change |
| **Webhook endpoints** | Stable but not branded | Professional, permanent |
| **Client demos** | Looks temporary | Looks production-ready |
| **Team sharing** | Each developer gets own URL | Shared URL for the whole team |
| **Inspector + Replay** | Not included | Included |
| **Max tunnels** | 1 | Up to 5 (up to 10+ on $10/mo plan) |

## Advanced Configuration

### Multiple Custom Domains

On the plan from $5/mo, you can register up to 5 custom domains. On the plan from $10/mo, you can have 10+ tunnels running simultaneously. This is useful for microservices architectures where each service needs its own public endpoint.

```bash
# Register multiple domains
fxtunnel domain add api.example.com
fxtunnel domain add app.example.com
fxtunnel domain add admin.example.com

# Start tunnels for each service
fxtunnel http 3000 --domain api.example.com
fxtunnel http 3001 --domain app.example.com
fxtunnel http 3002 --domain admin.example.com
```

Each tunnel gets its own TLS certificate and forwards traffic independently.

### Wildcard Subdomains

If your application uses dynamic subdomains (e.g., `tenant1.app.example.com`, `tenant2.app.example.com`), you can set up a wildcard CNAME:

| Record type | Name | Value | TTL |
|---|---|---|---|
| CNAME | `*.app` | `tunnel.fxtun.dev` | 300 |

Then register the wildcard domain in fxTunnel:

```bash
fxtunnel domain add "*.app.example.com"
fxtunnel http 8080 --domain "*.app.example.com"
```

Any request to `tenant1.app.example.com`, `tenant2.app.example.com`, etc. will be forwarded to your localhost. The `Host` header is preserved, so your application can determine which tenant is making the request.

### Using Custom Domains with Docker Containers

Custom domains work seamlessly with [Docker containers and tunnels](/blog/docker-tunnel-expose-containers/). Point the tunnel at the container port instead of a local process:

```bash
# Start your application in Docker
docker run -d -p 8080:80 my-web-app

# Open a tunnel with a custom domain
fxtunnel http 8080 --domain dev.example.com
```

The workflow is identical — the tunnel does not care whether the traffic originates from a bare-metal process or a container.

### Environment-Specific Domains

A common pattern is to use different subdomains for different environments:

```bash
# Development
fxtunnel http 3000 --domain dev.example.com

# Staging
fxtunnel http 3000 --domain staging.example.com

# QA
fxtunnel http 3000 --domain qa.example.com
```

Each subdomain needs its own CNAME record and domain registration. This gives teams clear separation between environments while keeping everything accessible through stable URLs.

## Team Workflows with Custom Domains

Where custom domains really shine is in team settings. A consistent, branded URL simplifies collaboration, reduces configuration drift, and makes it easy for non-developers -- QA, product managers, clients -- to access the application without asking "what is the URL today?"

### Shared Development Environment

The team lead configures the custom domain once. Any authorized developer can then start a tunnel on that domain:

```bash
# Developer A starts the tunnel in the morning
fxtunnel http 8080 --domain dev.example.com --token fxt_team_token

# Developer B takes over in the afternoon
# (Developer A stops the tunnel first)
fxtunnel http 8080 --domain dev.example.com --token fxt_team_token
```

QA bookmarks `https://dev.example.com` and always sees the latest version of the application — regardless of who is running the tunnel.

### Webhook Integrations for the Team

With a custom domain, webhook URLs remain stable regardless of which developer is running the tunnel. Set up the webhook once in Stripe, GitHub, or Telegram, and it keeps working when different team members take over:

```bash
# Stripe webhook endpoint (configured once)
# URL: https://dev.example.com/webhooks/stripe

# Any developer can receive webhooks by starting the tunnel
fxtunnel http 8080 --domain dev.example.com
```

No more "webhook stopped working because the URL changed" — the domain is permanent.

### CI/CD Preview Environments

Custom domains work well with [CI/CD preview environments](/blog/cicd-tunnel-preview-environments/). Your CI pipeline can start a tunnel on a branch-specific subdomain:

```bash
# In your CI script
BRANCH_SLUG=$(echo $CI_BRANCH | tr '/' '-' | tr '[:upper:]' '[:lower:]')
fxtunnel http 8080 --domain "${BRANCH_SLUG}.preview.example.com"
```

This creates preview URLs like `feature-login.preview.example.com` for each branch — fully accessible over HTTPS with a valid certificate.

## fxTunnel Custom Domain Pricing

Custom domains are available starting from $5/mo. There are no per-request charges or bandwidth limits.

| Plan | Price | Custom domains | Tunnels | Inspector + Replay |
|---|---|---|---|---|
| **Free** | $0 | No | 1 | No |
| **Pro** | from $5/mo | Up to 5 | Up to 5 | Yes |
| **Team** | from $10/mo | Unlimited | 10+ | Yes |

All plans include:

- HTTP, TCP, and UDP tunnel support
- No traffic or connection limits
- Persistent URLs across restarts
- [TLS 1.3 encryption](/blog/tls-13-tunnel-security/)

The Pro plan adds custom domains and the request Inspector with Replay — a web interface for viewing all HTTP requests passing through the tunnel, with the ability to resend any request with one click. The Team plan adds support for 10+ concurrent tunnels and priority support, which is ideal for larger development teams.

Sign up at [fxtun.dev](https://fxtun.dev) to get started. The source code is available on [GitHub](https://github.com/mephistofox/fxtun.dev).

## Comparison: Custom Domains Across Tunneling Tools

If custom domains matter for your workflow, here is how fxTunnel stacks up against [ngrok and Cloudflare Tunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Feature | fxTunnel | ngrok | Cloudflare Tunnel |
|---|---|---|---|
| **Custom domain price** | from $5/mo | from $10/mo (paid plans) | Free (requires Cloudflare DNS) |
| **Certificate management** | Automatic (Let's Encrypt) | Automatic | Automatic (Cloudflare CA) |
| **Wildcard domains** | Yes | Paid plans only | Yes |
| **Setup complexity** | CNAME + 1 CLI command | CNAME + dashboard config | Cloudflare DNS migration required |
| **TCP/UDP with custom domain** | TCP yes, UDP yes (port-based) | TCP only (paid) | HTTP only |
| **Inspector + Replay** | from $5/mo | from $10/mo | No built-in inspector |
| **Open source** | Yes | No | Partial (cloudflared is OSS) |
| **Free tier limits** | No limits on traffic | 1 agent, rate limits | No limits on traffic |

fxTunnel starts at a lower price than ngrok, supports more protocols than Cloudflare Tunnel, and is fully open-source. Unlike Cloudflare Tunnel, you do not need to migrate your entire DNS to Cloudflare -- just add a single CNAME record with any provider.

## Security Considerations

A custom domain connected to a tunnel is safe for development use -- fxTunnel encrypts all traffic via [TLS 1.3](/blog/tls-13-tunnel-security/), provisions certificates automatically, and isolates tunnels from each other. That said, a few habits go a long way:

1. **Do not expose production data** — tunnels are for development and testing. Do not connect a tunnel to a production database or service.
2. **Use token authentication** — always authenticate with `--token` to prevent unauthorized tunnel creation on your domain.
3. **Rotate tokens** — if a team member leaves, generate a new token and distribute it to remaining team members.
4. **Monitor with Inspector** — the Inspector (from $5/mo) shows all incoming requests. Use it to detect unexpected traffic.
5. **Disable tunnels when not in use** — stop the tunnel when you are done developing. The custom domain will simply stop resolving to any server.

```bash
# Generate a new team token
fxtunnel token generate --name "team-q1-2026"

# Revoke an old token
fxtunnel token revoke fxt_old_token_here

# List active tokens
fxtunnel token list
```

## Troubleshooting

### Domain Not Resolving

```bash
# Check if CNAME is set correctly
dig dev.example.com CNAME +short
# Expected: tunnel.fxtun.dev.

# If empty, the DNS record has not propagated yet
# Wait 5-30 minutes and try again
```

### ERR_CONNECTION_REFUSED

The domain resolves but nothing is listening. Make sure the tunnel is running:

```bash
fxtunnel http 8080 --domain dev.example.com
# Make sure your local server is also running on port 8080
```

### SSL Certificate Error

If Cloudflare proxy is enabled, you will see certificate errors because Cloudflare terminates TLS before fxTunnel can. Disable the proxy (set to DNS-only mode in Cloudflare).

### Domain Already In Use

If you see "domain already registered by another account," it means someone else has already claimed this domain in fxTunnel. Make sure you are using the correct account, or contact support.

### Tunnel Starts But Custom Domain Does Not Work

Check the domain verification status:

```bash
fxtunnel domain list
fxtunnel domain verify dev.example.com
```

If the status is "pending," the CNAME has not been detected yet. Double-check your DNS settings and wait for propagation.

## FAQ

### Can I use my own domain with fxTunnel?

Absolutely. Paid plans let you connect custom domains -- up to 5 on the Pro plan. Add a CNAME record pointing to `tunnel.fxtun.dev`, then launch your tunnel with `fxtunnel http 8080 --domain myapp.example.com`. fxTunnel provisions a valid TLS certificate automatically, so `https://myapp.example.com` works right away.

### Do I need to configure HTTPS certificates manually?

No -- certificates are fully automated. When the first request arrives at your custom domain, fxTunnel obtains a certificate from Let's Encrypt and handles all renewals going forward. You never touch a certificate file. Read more about the [tunnel server architecture](/blog/fxtunnel-architecture/).

### What happens if my DNS propagation is slow?

Propagation usually finishes within a few minutes, though it can occasionally take up to 48 hours. While the CNAME has not resolved to `tunnel.fxtun.dev` yet, your custom domain will not respond. Check progress with `dig myapp.example.com CNAME` or an online DNS checker. In the meantime, your default `fxtun.dev` subdomain keeps working.

### Can multiple team members share a single custom domain?

Yes. Once the team lead sets up the domain, anyone with the right token can launch a tunnel on it. This works well for shared staging, QA testing, and demo workflows. The Team plan supports 10+ simultaneous tunnels if you need them.

### What is the difference between a custom domain and the default fxtun.dev subdomain?

A default subdomain like `abc.fxtun.dev` is assigned automatically and works on every plan, including free. A custom domain like `dev.example.com` uses your own DNS, which makes a real difference when you configure [OAuth redirect URIs](/blog/oauth-callback-localhost-tunnel/), set up webhooks, or demo to clients -- it looks intentional rather than temporary.
