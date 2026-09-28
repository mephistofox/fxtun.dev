---
title: "Hello World! Launching the fxTunnel Blog"
date: 2026-01-28T14:00:00+03:00
draft: false
description: "First post on the fxTunnel blog — what is fxTunnel, why we built it, and what to expect: tunneling, DevOps, Go, and network security articles."
tags: ["fxTunnel", "announcement", "tunneling"]
image: ""
author: "fxTunnel Team"
faq:
  - q: "What is fxTunnel?"
    a: "fxTunnel is an open-source SaaS tool for creating secure HTTP, TCP, and UDP tunnels. It lets you expose a local server to the internet with a single command in 30 seconds. Free tier with no limits, paid plans from $5/mo."
  - q: "What topics will the fxTunnel blog cover?"
    a: "The blog covers tunneling architecture and protocols, DevOps practices, Go and Vue.js development, network security, and practical guides for developers working with webhooks, IoT, Docker, and CI/CD."
  - q: "Is fxTunnel free?"
    a: "Yes. fxTunnel has a generous free tier with no limits on traffic, connections, or tunnels. Paid plans start at $5/mo for custom domains, a traffic inspector, and replay."
---

## Why a blog?

We decided to start a blog to share our experience building **fxTunnel** — a service for secure traffic tunneling.

Here you'll find articles about:

- Tunnel architecture and protocols (HTTP, TCP, UDP)
- DevOps practices and infrastructure
- Go, Vue.js and web development
- Network security

## What is fxTunnel?

fxTunnel lets you create secure tunnels to access local services over the internet. It's great for:

- Developing and testing webhooks
- Demoing local projects
- Remote access to IoT devices

### Usage example

```bash
# Create a tunnel to a local server
fxtunnel http 8080
```

After that, your server will be available at a public URL.

> fxTunnel is like a bridge between your localhost and the entire world.

## What's next?

In upcoming articles we'll cover:

1. **Architecture** — how the server and client work
2. **Protocols** — differences between HTTP, TCP and UDP tunnels
3. **Deployment** — how we set up the infrastructure

Stay tuned!
