---
title: "SSH Tunnel vs ngrok, fxTunnel, Cloudflare — 2026"
date: 2026-02-17T14:00:00+03:00
draft: false
description: "SSH tunnel vs ngrok, fxTunnel, and Cloudflare Tunnel: commands, comparison table, and migration examples. Pick the right tool."
tags: [ssh, tunneling, comparison, fxTunnel, ngrok, developer-tools, networking]
image: ""
author: "fxTunnel Team"
faq:
  - q: "What is the difference between an SSH tunnel and ngrok?"
    a: "An SSH tunnel requires your own server with SSH access, manual setup, and provides no HTTPS or traffic inspector. ngrok and fxTunnel work without your own server, provide a public HTTPS URL in seconds, and offer a web interface for inspecting requests. fxTunnel is free, open source, and supports HTTP, TCP, and UDP."
  - q: "Can I replace an SSH tunnel with fxTunnel?"
    a: "Yes. If you use ssh -L or ssh -R for port forwarding, fxTunnel replaces both with a single command: fxtunnel http 8080 or fxtunnel tcp 5432. You also get automatic HTTPS, a traffic inspector, and no need for your own server."
  - q: "When is an SSH tunnel better than modern tools?"
    a: "An SSH tunnel is the better choice when you already have a server with SSH access, you need simple port forwarding without a public URL, you want zero dependency on third-party services, or you work in a locked-down corporate network where installing external tools is not allowed."
  - q: "Do I need my own server for an SSH tunnel?"
    a: "Yes. An SSH tunnel requires a remote server with SSH access and a public IP address. Modern tools like fxTunnel and ngrok work through a cloud relay server and require no infrastructure of your own."
  - q: "Does an SSH tunnel support HTTPS?"
    a: "No. An SSH tunnel encrypts the connection between your machine and the server but does not issue a TLS certificate for a public URL. For HTTPS over an SSH tunnel you need to manually configure nginx or Caddy with Let's Encrypt. fxTunnel provides HTTPS automatically."
---

## What Is an SSH Tunnel

An SSH tunnel redirects network traffic through an encrypted SSH connection between your machine and a remote server. It is built into the SSH protocol and requires no extra software. SSH tunnels have been around long before ngrok and Cloudflare Tunnel, and developers still rely on them for port forwarding, reaching remote databases, and getting through firewalls. If the concept of tunneling is new to you, start with [What Is Tunneling](/blog/what-is-tunneling/).

SSH supports three types of tunnels:

- **Local forwarding (`ssh -L`)** — forwards a port from the remote server to your local machine. Useful when you need access to a service running on the remote server (for example, a database).
- **Remote forwarding (`ssh -R`)** — forwards a port from your local machine to the remote server. This is the SSH equivalent of a reverse tunnel: the outside world connects to the server, and traffic is redirected to you.
- **Dynamic forwarding (`ssh -D`)** — creates a SOCKS proxy over the SSH connection. Used to route all traffic through the remote server.

## SSH Tunnel Commands Explained

Before we compare SSH with modern alternatives, let us walk through the actual commands. Each one solves a different problem, and seeing the syntax will make the trade-offs obvious.

### Local forwarding (ssh -L)

Redirects a remote server port to your machine. For example, accessing PostgreSQL on a remote server:

```bash
# Syntax: ssh -L [local_port]:[destination]:[dest_port] user@server
ssh -L 5432:localhost:5432 user@server

# PostgreSQL on the server is now available as localhost:5432
psql -h localhost -p 5432 -U myuser mydb
```

### Remote forwarding (ssh -R)

Exposes your local server to the outside world through a remote server. This is the closest analog to what ngrok and fxTunnel do:

```bash
# Syntax: ssh -R [remote_port]:[destination]:[dest_port] user@server
ssh -R 80:localhost:3000 user@server

# http://server:80 now points to your localhost:3000
```

### Dynamic forwarding (ssh -D)

Creates a SOCKS proxy. All traffic routed through the proxy passes through the SSH server:

```bash
# Syntax: ssh -D [port] user@server
ssh -D 1080 user@server

# Configure your browser or application to use SOCKS5 proxy at localhost:1080
curl --socks5 localhost:1080 https://example.com
```

### Useful flags

```bash
# Background mode with no interactive session
ssh -f -N -L 5432:localhost:5432 user@server

# -f — go to background after authentication
# -N — do not execute a remote command (tunnel only)
# -T — do not allocate a TTY (optional, for cleanliness)
```

## Limitations of SSH Tunnels

SSH tunnels are reliable, but they have real limitations that get in the way of modern development workflows. If you regularly test webhooks, share demos, or work on a team, you will hit these walls quickly. For protocol-level details, see [TCP/UDP Tunneling Explained](/blog/tcp-udp-tunneling-explained/).

### You need your own server

An SSH tunnel requires a remote server with a public IP address and SSH access. That means you need to rent a VPS, set up SSH keys, secure the server, and keep it running. For many developers — especially freelancers and beginners — that is too much infrastructure for simple port forwarding.

### No automatic HTTPS

SSH encrypts the connection between client and server, but it does not issue a TLS certificate for a public URL. If you need HTTPS (and you do for webhooks, OAuth, and mobile APIs), you have to manually configure nginx or Caddy with Let's Encrypt on your server.

### No traffic inspector

An SSH tunnel is a black box. You cannot see which requests pass through the tunnel, you cannot replay them, and you cannot debug issues without additional tools like `tcpdump` or Wireshark.

### Manual setup

Every time you connect, you type a long command with ports and addresses. There is no built-in configuration file, no automatic reconnection on disconnect, and no convenient status output.

### No UDP support

SSH runs over TCP and does not support UDP forwarding. If you need UDP (game servers, VoIP, DNS, IoT devices), an SSH tunnel will not work.

## Modern Tunnel Tools Overview

Tools like fxTunnel, ngrok, and Cloudflare Tunnel solve the same problems as SSH tunnels but remove most of the friction: no server needed, HTTPS works out of the box, and setup takes seconds. For a head-to-head comparison of these three, see [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

### fxTunnel

[fxTunnel](/blog/fxtunnel-architecture/) is an open-source tool written in Go. One command, 30 seconds, and you have a public HTTPS URL. Supports HTTP, TCP, and UDP with a free tier included. Paid plans start at $5/mo if you need custom domains, a traffic inspector, or replay.

```bash
fxtunnel http 8080
# -> https://abc123.fxtun.dev -> localhost:8080
```

### ngrok

ngrok is the most well-known tunneling tool. Fast to start, excellent documentation, but proprietary, no UDP support, strict free-tier limits (1 agent, rate limits), and pricing from $8/mo for advanced features.

```bash
ngrok http 8080
```

### Cloudflare Tunnel

Cloudflare Tunnel is free for HTTP but requires your domain to be on Cloudflare DNS, does not support UDP, and setup takes 10+ minutes via YAML configs. TCP is only available on paid plans.

```bash
cloudflared tunnel run my-tunnel
```

## Comparison Table: SSH vs fxTunnel vs ngrok vs Cloudflare

Here is how the four options stack up side by side. The SSH tunnel is the only one with zero dependency on a third-party service; the modern tools trade that independence for convenience. For a broader overview, see [Best Tunneling Tools 2026](/blog/best-tunneling-tools-2026/).

| Criterion | SSH Tunnel | fxTunnel | ngrok | Cloudflare Tunnel |
|---|---|---|---|---|
| **Own server required** | Yes | No | No | No |
| **Setup time** | 5-30 minutes | 30 seconds | 2 minutes | 10+ minutes |
| **HTTPS** | Manual (nginx + LE) | Automatic | Automatic | Automatic |
| **Traffic inspector** | No | Yes (from $5/mo) | Yes (limited) | No |
| **Request replay** | No | Yes (from $5/mo) | From $8/mo | No |
| **Protocols** | TCP | HTTP, TCP, UDP | HTTP, TCP | HTTP (TCP on paid) |
| **UDP** | No | Yes | No | No |
| **Open source** | SSH itself — yes | Fully | No | Partially |
| **Custom domain** | Own server + DNS | Any DNS (from $5/mo) | From $8/mo | CF DNS only |
| **Cost** | VPS from $5/mo | Free | Free (with limits) | Free (HTTP only) |
| **Auto-reconnect** | No (need autossh) | Yes | Yes | Yes |
| **Vendor dependency** | None | None (open source) | Yes | Yes (CF DNS) |

## When an SSH Tunnel Is Still the Right Choice

SSH tunnels have not become obsolete. There are scenarios where they remain the right -- or only -- option. The key advantage is complete independence from external services.

### You already have a server

If you have a VPS or dedicated server with SSH access, an SSH tunnel requires no additional software. The SSH client is pre-installed on Linux and macOS, and Windows ships with a built-in OpenSSH client.

### Simple port forwarding

For one-off tasks like connecting to a remote database or checking an API on a staging server, an SSH tunnel works perfectly: one command, no registration, no account creation.

```bash
# Access Redis on a production server
ssh -L 6379:localhost:6379 deploy@production-server
redis-cli -h localhost -p 6379
```

### Locked-down corporate networks

In companies with strict security policies, installing third-party tools may be forbidden. An SSH client is available everywhere, and outbound SSH traffic is usually allowed even in restrictive corporate environments.

### Full independence

An SSH tunnel depends on no external service. No ngrok servers, no fxTunnel cloud, no Cloudflare DNS — just your server and an SSH client. For security-conscious teams and air-gapped scenarios, that is the deciding factor.

## When to Switch to Modern Tools

If you regularly run into the tasks listed below, an SSH tunnel creates more friction than it removes. For a common scenario, see [How to Expose Localhost to the Internet](/blog/expose-localhost-to-internet/).

### Webhook testing

Stripe, GitHub, and Telegram require a public HTTPS URL to deliver events. An SSH tunnel gives you only an IP:port with no HTTPS. With fxTunnel, one command and the HTTPS URL is ready:

```bash
fxtunnel http 8080
# -> https://wh-test.fxtun.dev (HTTPS out of the box)
```

### Project demos

Need to show a client or teammate your current progress? An SSH tunnel forces them to remember an IP address and port. fxTunnel gives you a readable URL that is easy to share in a messenger.

### Team collaboration

When multiple developers need tunnels at the same time, the SSH approach means either a shared server with a pile of redirections or a separate VPS for each person. fxTunnel works without a server of your own — everyone launches their own tunnel independently.

### HTTPS without manual setup

OAuth, mobile APIs, and payment APIs all require HTTPS. Configuring nginx plus Let's Encrypt on your own server just for testing is overkill. fxTunnel and ngrok provide HTTPS automatically.

### No server available

If you do not have a VPS or dedicated server, an SSH tunnel is simply not an option. Modern tools work through a cloud relay and require no infrastructure.

### UDP needed

Game servers, VoIP, and IoT devices using UDP — SSH does not support UDP forwarding. Of all the tools covered here, only fxTunnel supports UDP. For more details, see [TCP/UDP Tunneling Explained](/blog/tcp-udp-tunneling-explained/).

## Migration: SSH Tunnel to fxTunnel

Switching from SSH to fxTunnel takes about a minute. Below is a side-by-side comparison of commands for common scenarios.

### Expose a web server

```bash
# SSH (requires your own server, no HTTPS)
ssh -R 80:localhost:3000 user@server

# fxTunnel (no server needed, HTTPS automatic)
fxtunnel http 3000
```

### Access a remote database

```bash
# SSH — still an excellent choice for this scenario
ssh -L 5432:localhost:5432 user@db-server

# fxTunnel — when you need external access to a local database
fxtunnel tcp 5432
```

### Forward multiple ports

```bash
# SSH — a separate flag for each port
ssh -L 5432:localhost:5432 -L 6379:localhost:6379 user@server

# fxTunnel — separate tunnels with independent URLs
fxtunnel http 3000 &
fxtunnel tcp 5432 &
fxtunnel udp 27015 &
```

### Webhooks with HTTPS

```bash
# SSH — requires a server + nginx + Let's Encrypt
ssh -R 8080:localhost:3000 user@server
# + configure nginx reverse proxy
# + certbot --nginx -d webhook.example.com

# fxTunnel — one command, HTTPS out of the box
fxtunnel http 3000
# -> https://wh-abc.fxtun.dev (done)
```

## FAQ

### What is the difference between an SSH tunnel and ngrok?

With an SSH tunnel you need your own server, you configure everything manually, and there is no built-in HTTPS or traffic inspector. Tools like ngrok and fxTunnel skip the server entirely, give you a public HTTPS URL in seconds, and let you inspect requests through a web UI. fxTunnel adds open-source code and UDP support on top of that.

### Can I replace an SSH tunnel with fxTunnel?

In most cases, yes. If you currently use `ssh -L` or `ssh -R` for port forwarding, a single `fxtunnel http 8080` or `fxtunnel tcp 5432` does the same job. You also get automatic HTTPS and a traffic inspector without needing your own server.

### When is an SSH tunnel better than modern tools?

When you already have a server with SSH access and just need quick port forwarding without a public URL. It is also the go-to choice if you want zero dependency on third-party services or work in a locked-down corporate network where installing external tools is not an option.

### Do I need my own server for an SSH tunnel?

Yes -- you need a remote machine with SSH access and a public IP. Tools like fxTunnel and ngrok route traffic through a cloud relay, so no personal infrastructure is required.

### Does an SSH tunnel support HTTPS?

Not on its own. SSH encrypts the link between your machine and the server, but it does not issue a TLS certificate for a public URL. If you need HTTPS you will have to set up nginx or Caddy with Let's Encrypt yourself. Modern tools like fxTunnel handle certificate issuance automatically.
