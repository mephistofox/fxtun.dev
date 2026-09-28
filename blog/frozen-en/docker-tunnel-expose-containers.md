---
title: "Docker + Tunnel: Expose Containers Without Port Forwarding"
date: 2026-02-19T14:00:00+03:00
draft: false
description: "How to expose Docker containers to the internet with fxTunnel: two approaches (host and in-container), docker-compose examples, HTTP and TCP tunnels."
tags: [docker, tunneling, containers, fxTunnel, devops, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I expose a Docker container to the internet without router port forwarding?"
    a: "Yes. fxTunnel establishes an outbound connection to a public server and receives an HTTPS address that points straight at the container port. NAT and firewalls are not a problem because the connection is initiated from inside your network. No router port forwarding is needed."
  - q: "Should I run fxTunnel on the host or inside a container?"
    a: "For quick testing, running fxTunnel on the host is simpler — one command, no extra configuration. For reproducible environments and CI/CD, adding fxTunnel as a service in docker-compose is better — the tunnel starts and stops together with the rest of your containers."
  - q: "How do I expose multiple containers at once?"
    a: "Run a separate tunnel for each service. On the host, run multiple fxtunnel commands with different ports. In docker-compose, add multiple tunnel services, each with its own port. fxTunnel supports multiple simultaneous tunnels on the free tier."
  - q: "Does a TCP tunnel work for accessing a database in Docker?"
    a: "Yes. The command fxtunnel tcp 5432 creates a TCP tunnel to the PostgreSQL port. You will receive a public address like tcp://xxx.fxtun.dev:12345 that any client can connect to. This is useful for collaborative debugging or accessing a dev database from external tools."
  - q: "Is it safe to expose Docker containers through a tunnel?"
    a: "For development and testing, yes. fxTunnel encrypts traffic via TLS. Do not use a tunnel for production containers with real data. Close the tunnel when you are done and avoid exposing services with sensitive data without additional protection."
---

## The Problem: Docker Containers Are Isolated from the Outside World

You have a container running locally, everything works on `localhost:8080`, and then someone asks for a link. Docker containers live in an isolated network -- even with `-p 8080:80`, the service is only reachable on your machine. To show a project to a teammate, receive a [webhook](/blog/webhook-testing-with-tunnel/) from Stripe, or test a mobile app on a real device, you need a public URL. A [tunnel](/blog/what-is-tunneling/) gets you there in 30 seconds -- no router port forwarding, no static IP, no deployment.

A standard Docker port mapping looks like this:

```bash
docker run -p 8080:80 my-web-app
```

The container listens on port 80, Docker maps it to port 8080 on the host. You open `http://localhost:8080` and everything works. But this address is only reachable by you. Neither a colleague nor an external service can reach it.

## Why Traditional Port Forwarding Does Not Help

Router port forwarding sounds like it should work, but in practice it is difficult, insecure, and often impossible. Your ISP may give you a shared IP (CGNAT), a corporate network will not let you near the router, and an open port is a permanent security hole. For a deeper comparison, see [How to Expose Localhost to the Internet](/blog/expose-localhost-to-internet/).

Common obstacles:

- **CGNAT / shared IP** — the ISP shares one public IP among dozens of customers. Port forwarding is impossible.
- **Corporate network** — router access is locked down by security policies.
- **Dynamic IP** — the address changes every time you reconnect. You need DDNS, which adds another point of failure.
- **Security** — an open port on the router is visible to the entire internet. Without TLS and authentication, it is an open invitation for attacks.

A tunnel bypasses all of these limitations: the client establishes an outbound connection to a public server, which assigns an [HTTPS address](/blog/https-localhost-development/) pointing at your container. NAT, firewalls, and dynamic IPs are no longer a problem.

## Solution 1: fxTunnel on the Host (Simplest Approach)

The fastest path: run fxTunnel directly on the host. The container maps its port via `-p`, and fxTunnel creates a tunnel to that port. Two commands, done.

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify
fxtunnel --version
```

### Step 2. Start the Container with a Port Mapping

```bash
# Run the container with a port mapped to the host
docker run -d -p 8080:80 --name my-app nginx
```

Now nginx is available at `localhost:8080`.

### Step 3. Create a Tunnel

```bash
# Open a tunnel to port 8080 on the host
fxtunnel http 8080
```

You will see the public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://docker-demo.fxtun.dev
Forwarding:  https://docker-demo.fxtun.dev → http://localhost:8080
```

Done. `https://docker-demo.fxtun.dev` now points at nginx inside the container. Anyone with this URL can open your application in a browser.

### When to Use This Approach

- Quick testing and demos — one command, minimal configuration.
- Local development — fxTunnel is already installed on the machine, no need to modify `Dockerfile` or `docker-compose.yml`.
- One-off tasks — show a project to a colleague, receive a webhook, test an integration.

## Solution 2: fxTunnel Inside a Docker Container

For reproducible environments, CI/CD pipelines, and team workflows, it is better to add fxTunnel as a separate service in docker-compose. The tunnel starts and stops together with the rest of the containers, the configuration lives in code, and every developer gets an identical environment.

### docker-compose.yml with fxTunnel

```yaml
version: "3.8"

services:
  web:
    image: nginx
    ports:
      - "8080:80"

  tunnel:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "web:80"]
    depends_on:
      - web
```

Start everything:

```bash
docker compose up
```

fxTunnel inside the container reaches the `web` service by name (Docker DNS), bypassing the need for a host port mapping. The tunnel prints the public URL in the logs.

### How Containers See Each Other

Docker Compose creates a shared network for all services in the file. The `tunnel` container can reach the `web` container by service name: `web:80`. This is Docker's internal DNS resolution. Port 80 is the port inside the container, not on the host.

## Practical Scenarios

### Scenario 1: HTTP Tunnel for a Web Application

Say you are building a React, Vue, or Next.js app in Docker and want to share it with a colleague or client.

```yaml
version: "3.8"

services:
  frontend:
    build: .
    ports:
      - "3000:3000"
    volumes:
      - .:/app
    environment:
      - NODE_ENV=development

  tunnel:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "frontend:3000"]
    depends_on:
      - frontend
```

```bash
docker compose up
# tunnel_1  | Public URL: https://xyz.fxtun.dev
# tunnel_1  | Forwarding: https://xyz.fxtun.dev → http://frontend:3000
```

Send the link `https://xyz.fxtun.dev` to your client — they will see your application without a deployment.

### Scenario 2: TCP Tunnel for a Database

Want to give a teammate access to a dev PostgreSQL database running in Docker? A [TCP tunnel](/blog/tcp-udp-tunneling-explained/) exposes the database port through a public address.

On the host (if the port is mapped):

```bash
# PostgreSQL mapped to host: docker run -p 5432:5432 postgres
fxtunnel tcp 5432
```

In docker-compose:

```yaml
version: "3.8"

services:
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    ports:
      - "5432:5432"

  db-tunnel:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["tcp", "db:5432"]
    depends_on:
      - db
```

```bash
docker compose up
# db-tunnel_1 | TCP tunnel active
# db-tunnel_1 | tcp://db-demo.fxtun.dev:18432 → tcp://db:5432
```

Your teammate connects to the database:

```bash
psql -h db-demo.fxtun.dev -p 18432 -U postgres -d myapp
```

### Scenario 3: Multiple Tunnels for Microservices

Working with a microservice stack -- frontend, API, database? Each service gets its own tunnel. In docker-compose, just add a separate tunnel service for each.

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
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp

  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp

  tunnel-frontend:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "frontend:3000"]
    depends_on:
      - frontend

  tunnel-api:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "api:4000"]
    depends_on:
      - api

  tunnel-db:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["tcp", "db:5432"]
    depends_on:
      - db
```

```bash
docker compose up
# tunnel-frontend_1 | https://front-xyz.fxtun.dev → http://frontend:3000
# tunnel-api_1      | https://api-xyz.fxtun.dev   → http://api:4000
# tunnel-db_1       | tcp://db-xyz.fxtun.dev:19432 → tcp://db:5432
```

Three public addresses, three services — everything comes up with a single `docker compose up`.

## Docker Compose + fxTunnel: Ready-to-Use Recipe

Below is a complete docker-compose template for a typical web application with an API, a database, and tunnels. Copy it, change the service names and ports to match your project.

```yaml
version: "3.8"

services:
  # === Your application ===
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp
    depends_on:
      - db

  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    volumes:
      - pgdata:/var/lib/postgresql/data

  # === Tunnels ===
  tunnel-app:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "app:8080"]
    depends_on:
      - app
    restart: unless-stopped

  tunnel-db:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["tcp", "db:5432"]
    depends_on:
      - db
    restart: unless-stopped

volumes:
  pgdata:
```

Commands:

```bash
# Start the full stack
docker compose up -d

# View tunnel logs (public URLs will be printed here)
docker compose logs tunnel-app tunnel-db

# Stop everything
docker compose down
```

## Tips for Working with Docker Networking and Tunnels

### host.docker.internal

If fxTunnel is running inside a container and you need to reach a service on the host machine (not in Docker), use the special address `host.docker.internal`:

```yaml
  tunnel:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "host.docker.internal:3000"]
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

The `extra_hosts` directive is required on Linux. On macOS and Windows, `host.docker.internal` works out of the box.

### Default Bridge Network

Docker Compose automatically creates a bridge network for all services in the file. Containers can reach each other by service name. If you use `docker run` without Compose, containers are placed in the default `bridge` network and must address each other by IP, not by name. In that case, running fxTunnel on the host (Solution 1) is simpler.

### Multiple Networks

If your containers belong to different Docker networks, make sure the fxTunnel container is connected to the same network as the target service:

```yaml
services:
  web:
    image: nginx
    networks:
      - frontend

  tunnel:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "web:80"]
    networks:
      - frontend

networks:
  frontend:
```

### Verifying Connectivity from Inside a Container

If the tunnel cannot connect to the target service, check network connectivity:

```bash
# Shell into the tunnel container
docker compose exec tunnel sh

# Check if the target service is reachable
wget -qO- http://web:80
```

## FAQ

### Can I expose a Docker container to the internet without router port forwarding?

Yes. fxTunnel opens an outbound connection to a public relay server and gets back an HTTPS address pointing at your container port. Because the connection originates from inside your network, NAT and firewalls stay out of the way. More on [how tunnels work](/blog/what-is-tunneling/).

### Should I run fxTunnel on the host or inside a container?

It depends on your workflow. For a quick demo, running it on the host is fastest -- one command, nothing to configure. If you need a reproducible setup or CI/CD integration, add it as a docker-compose service so the tunnel lifecycle matches your containers.

### How do I expose multiple containers at once?

Run a separate tunnel per service. On the host, that means multiple `fxtunnel` commands with different ports. In docker-compose, add a tunnel service for each target. Multiple simultaneous tunnels work on the free tier.

### Does a TCP tunnel work for accessing a database in Docker?

Yes. `fxtunnel tcp 5432` creates a [TCP tunnel](/blog/tcp-udp-tunneling-explained/) to the PostgreSQL port and gives you a public address like `tcp://xxx.fxtun.dev:12345`. Any standard database client can connect to it, which is handy for collaborative debugging or hooking up external tools to a dev database.

### Is it safe to expose Docker containers through a tunnel?

For dev and testing purposes, yes -- fxTunnel encrypts all traffic with TLS. Just do not point a tunnel at production containers holding real data. Shut the tunnel down when you are done, and avoid exposing services with sensitive data without extra protection. More on the security side in [How to Expose Localhost to the Internet](/blog/expose-localhost-to-internet/).
