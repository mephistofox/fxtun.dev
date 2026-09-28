---
title: "HTTPS on Localhost for Development — Setup in 2 Minutes"
date: 2026-02-13T14:00:00+03:00
draft: false
description: "How to set up HTTPS on localhost in 2 minutes: fxTunnel (automatic TLS), mkcert, and self-signed certificates. Method comparison, code examples, and FAQ."
tags: [https, localhost, ssl, tls, fxTunnel, security, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why do I need HTTPS on localhost?"
    a: "Many modern APIs and browser features require HTTPS: OAuth 2.0 redirects, Service Workers, Secure Cookies, and the Geolocation API. Without HTTPS you cannot test these features locally, and the browser will block mixed content when loading resources from an HTTP page."
  - q: "What is the fastest way to get HTTPS on localhost?"
    a: "fxTunnel is the fastest way. The command fxtunnel http 3000 creates a public URL like https://abc.fxtun.dev with a valid TLS certificate in 30 seconds. No certificate generation, no CA setup, no application reconfiguration required."
  - q: "Will the browser show a warning when using fxTunnel?"
    a: "No. fxTunnel provides a URL with a valid TLS certificate signed by a public CA. The browser trusts this certificate, so there are no warnings. This is unlike self-signed certificates, where Chrome shows NET::ERR_CERT_AUTHORITY_INVALID."
  - q: "Can I use mkcert and fxTunnel at the same time?"
    a: "Yes, but for different scenarios. fxTunnel is the right choice when you need a public HTTPS address for webhooks, OAuth, or demos. mkcert is better when you need HTTPS specifically on localhost without public access, such as testing Service Workers offline. Both tools are free."
  - q: "Is a self-signed certificate secure?"
    a: "For local development, yes — the traffic is encrypted in exactly the same way. But the browser does not trust self-signed certificates and shows a warning. Some APIs (Fetch, Service Workers) may refuse to work. For development, mkcert or fxTunnel are better choices — both solve the problem without browser warnings."
---

## Why Do You Need HTTPS on Localhost?

Ever tried testing an OAuth redirect on plain HTTP and watched the provider reject your callback? That is one of many cases where HTTPS on localhost is not optional. Service Workers, Secure Cookies, the Geolocation API, mixed content rules — modern browsers lock down features behind a secure context. You can fix this in 2 minutes with a [local tunnel](/blog/what-is-tunneling/) or a local CA tool.

Here are the specific situations where HTTPS on localhost is required:

- **OAuth 2.0 and OpenID Connect** — most providers (Google, GitHub, Auth0) require HTTPS for redirect URIs, even during development.
- **Service Workers** — browsers only register Service Workers on pages loaded over HTTPS (with a `localhost` exception in some browsers, but not all).
- **Secure Cookies** — the `Secure` attribute on a cookie means the browser will only send it over HTTPS. If you are building authentication with secure cookies, you need TLS.
- **Mixed Content** — if your frontend loads over HTTPS (staging or production) but your API responds over HTTP, the browser will block the requests.
- **Geolocation, Camera, Microphone** — these APIs require a secure context, which is provided by HTTPS.
- **Webhook testing** — services like Stripe and Telegram send webhooks only to HTTPS addresses. More details in [Webhook Testing with a Tunnel](/blog/webhook-testing-with-tunnel/).

## Comparison of Methods: How to Get HTTPS on Localhost

There are four main approaches, each with its own trade-offs.

| Method | Setup time | Browser warnings | Public access | Cost | Best for |
|---|---|---|---|---|---|
| **fxTunnel** | 30 seconds | None | Yes | Free (paid from $5/mo) | OAuth, webhooks, demos, quick HTTPS |
| **mkcert** | 2 minutes | None | No | Free | Service Workers, local testing |
| **Self-signed certificate** | 3-5 minutes | Yes | No | Free | Basic encryption, API tests |
| **Reverse proxy + Let's Encrypt** | 15-30 minutes | None | Yes (domain required) | Server + domain needed | Staging, pre-production |

## Method 1: fxTunnel — HTTPS in 30 Seconds

Run one command, get a public URL with a valid TLS certificate. No certificate generation, no local CA setup, no web server reconfiguration. HTTPS works automatically because the tunnel's public URL is `https://`.

### Why Is fxTunnel the Simplest Option?

The fxTunnel public URL (`https://abc.fxtun.dev`) is protected by a TLS certificate issued by a public certificate authority. The browser trusts this certificate just like any other HTTPS website. You do not need to install anything into the system trust store, bypass warnings, or restart the browser. For more on how this works, see [fxTunnel Architecture](/blog/fxtunnel-architecture/).

### Installation and Launch

```bash
# Install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Start a tunnel to your local server on port 3000
fxtunnel http 3000
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://my-app.fxtun.dev
Forwarding:  https://my-app.fxtun.dev → http://localhost:3000
```

That is it. `https://my-app.fxtun.dev` is your HTTPS address. Open it in a browser — the padlock icon is green, the certificate is valid.

### Example: OAuth Redirect with fxTunnel

```bash
# Start your application with OAuth
node app.js
# -> Server listening on port 3000

# Open a tunnel
fxtunnel http 3000
# -> https://my-app.fxtun.dev -> localhost:3000
```

Now enter `https://my-app.fxtun.dev/auth/callback` as the redirect URI in your OAuth provider settings (Google, GitHub, Auth0). The redirect will work without errors — the URL is valid, the certificate is trusted.

### Example: Testing a Service Worker Through the Tunnel

```javascript
// Service Worker registration — works on the tunnel's HTTPS address
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js')
    .then(reg => console.log('SW registered:', reg.scope))
    .catch(err => console.error('SW registration failed:', err));
}
```

Open `https://my-app.fxtun.dev` — the Service Worker will register without errors because the page is loaded over HTTPS.

### What Is Included in the fxTunnel Free Tier

- No limits on traffic or connections
- Public HTTPS URL with a valid certificate
- HTTP, TCP, and UDP tunnels
- Persistent URL (does not change on restart)

Paid plans from $5/mo add custom domains (up to 5 tunnels), Inspector, and Replay for request debugging. You can see how this stacks up against other options in the [tunneling tools comparison](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

## Method 2: mkcert — A Local CA for Trusted Certificates

mkcert is a tool for creating locally trusted TLS certificates. It installs a root CA into the system trust store, after which all generated certificates are automatically considered trusted. Use it when you need HTTPS specifically on `localhost` or `127.0.0.1` without public access.

### Installing mkcert

```bash
# macOS
brew install mkcert
mkcert -install

# Linux (Ubuntu/Debian)
sudo apt install libnss3-tools
brew install mkcert   # or download the binary from GitHub
mkcert -install

# Windows
choco install mkcert
mkcert -install
```

The `mkcert -install` command creates a local root CA and adds it to the trust store of your OS and browser.

### Generating a Certificate for localhost

```bash
# Create a certificate for localhost and 127.0.0.1
mkcert localhost 127.0.0.1 ::1

# Output:
# Created a new certificate valid for the following names:
#  - "localhost"
#  - "127.0.0.1"
#  - "::1"
# The certificate is at "./localhost+2.pem"
# The key is at "./localhost+2-key.pem"
```

### Using mkcert with Node.js

```javascript
const https = require('https');
const fs = require('fs');
const express = require('express');

const app = express();

app.get('/', (req, res) => {
  res.json({ secure: true, protocol: req.protocol });
});

const server = https.createServer({
  key: fs.readFileSync('./localhost+2-key.pem'),
  cert: fs.readFileSync('./localhost+2.pem'),
}, app);

server.listen(3000, () => {
  console.log('HTTPS server: https://localhost:3000');
});
```

### Using mkcert with Vite / React / Next.js

```bash
# Vite
vite --https --host \
  --ssl-cert ./localhost+2.pem \
  --ssl-key ./localhost+2-key.pem

# Next.js (experimental)
# next.config.js — use the local-ssl-proxy package
# or environment variables for a custom server
```

### When Is mkcert Better Than fxTunnel?

- You need HTTPS on `localhost` without a public URL (working offline).
- You are testing Service Workers without an internet connection.
- A corporate firewall blocks external connections.

For everything else, a tunnel is simpler: no certificate generation, no file paths to configure, and no application changes.

## Method 3: Self-Signed Certificate

A self-signed certificate is a TLS certificate that you generate yourself with OpenSSL. It encrypts traffic but is not signed by a trusted CA, so the browser will show a `NET::ERR_CERT_AUTHORITY_INVALID` warning. This method is suitable for API testing with curl or Postman but inconvenient for browser-based work.

### Generating a Self-Signed Certificate

```bash
# Generate a certificate and key (valid for 365 days)
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout localhost-key.pem \
  -out localhost-cert.pem \
  -days 365 \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

### Using It with Node.js

```javascript
const https = require('https');
const fs = require('fs');

const server = https.createServer({
  key: fs.readFileSync('./localhost-key.pem'),
  cert: fs.readFileSync('./localhost-cert.pem'),
}, (req, res) => {
  res.writeHead(200);
  res.end('HTTPS works (self-signed)');
});

server.listen(3000, () => {
  console.log('Server: https://localhost:3000');
  console.log('Browser will show a warning — this is expected');
});
```

### Problems with Self-Signed Certificates

- **Browser warning** — Chrome, Firefox, and Safari block the page and require manual confirmation.
- **Fetch API and Service Workers** — may refuse to work on a page with an untrusted certificate.
- **OAuth providers** — will not accept a redirect URI with a self-signed certificate.
- **Team usage** — every developer needs to add the certificate to their own trust store.

### How to Bypass the Warning (Development Only)

```bash
# curl: skip certificate verification
curl -k https://localhost:3000

# Node.js: disable verification (DO NOT USE IN PRODUCTION!)
NODE_TLS_REJECT_UNAUTHORIZED=0 node client.js
```

These workarounds are fine for API testing but do not help in the browser. For HTTPS without warnings, use a tunnel or mkcert.

## Which Method Should You Choose?

It comes down to what you are building. Need public HTTPS quickly? Use fxTunnel. Need local HTTPS without the internet? Use mkcert. A self-signed certificate is a fallback when the first two are unavailable for some reason.

### fxTunnel — choose it when you:

- Need HTTPS in 30 seconds with no certificate setup
- Are testing OAuth redirects with external providers
- Are receiving webhooks from Stripe, GitHub, or Telegram
- Want to show a project to a colleague or client
- Need to [expose localhost](/blog/expose-localhost-to-internet/) to the outside world

### mkcert — choose it when you:

- Need HTTPS on `localhost` without a public URL
- Work offline or behind a strict firewall
- Are testing Service Workers without internet
- Want full control over certificates

### Self-signed certificate — choose it when you:

- Are testing APIs with curl or Postman
- Do not need a browser
- Cannot install mkcert in your environment

## Common Mistakes When Setting Up HTTPS on Localhost

### ERR_CERT_AUTHORITY_INVALID

The browser does not trust the certificate. Solution: use fxTunnel (valid certificate automatically) or mkcert (local CA). A self-signed certificate will always trigger this error.

### Mixed Content: blocked loading

The page loaded over HTTPS but requests resources over HTTP. Solution: make sure all API requests use HTTPS. When using fxTunnel, all requests through the tunnel are automatically protected by TLS.

### CORS Errors After Switching to HTTPS

The port or protocol changed, so the origin does not match. Solution: update the CORS settings on your server to include the new origin (`https://localhost:3000` or the tunnel URL).

### Service Worker Fails to Register

Service Workers require a secure context. Most browsers make an exception for `http://localhost`, but not all of them. Solution: use HTTPS via fxTunnel or mkcert.

## FAQ

### Why do I need HTTPS on localhost?

A growing list of browser features and APIs are locked behind a secure context: OAuth 2.0 redirects, Service Workers, Secure Cookies, the Geolocation API, and more. Without HTTPS you simply cannot test them locally. On top of that, browsers block mixed content when an HTTP page tries to load resources from an HTTPS origin.

### What is the fastest way to get HTTPS on localhost?

A single command — `fxtunnel http 3000` — gives you a public URL like `https://abc.fxtun.dev` with a valid TLS certificate in about 30 seconds. There is nothing to generate or configure on your side. [Installation](/blog/what-is-tunneling/) is one command as well.

### Will the browser show a warning when using fxTunnel?

No, because the tunnel URL carries a valid TLS certificate signed by a public CA. Browsers treat it like any other HTTPS site — no warnings, no clicking through scary screens. Contrast that with self-signed certificates, where Chrome shows `NET::ERR_CERT_AUTHORITY_INVALID`.

### Can I use mkcert and fxTunnel at the same time?

Absolutely — they solve different problems. Use fxTunnel when you need a publicly reachable HTTPS address for [webhooks](/blog/webhook-testing-with-tunnel/), OAuth callbacks, or demos. Use mkcert when you need HTTPS on `localhost` without any public exposure, like testing Service Workers offline. Both are free.

### Is a self-signed certificate secure?

The encryption itself is identical — your traffic is protected just as well. The problem is trust: browsers do not recognize the issuer and show a warning. Some APIs (Fetch, Service Workers) may refuse to work altogether. For day-to-day development, mkcert or fxTunnel are more practical since they both avoid browser warnings entirely.
