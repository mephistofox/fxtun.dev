---
title: "Database Tunneling: PostgreSQL, MySQL, Redis via TCP"
date: 2026-03-17T14:00:00+03:00
draft: false
description: "Access PostgreSQL, MySQL, and Redis through a TCP tunnel with fxTunnel: connection examples, security best practices, and SSH comparison."
tags: [database, tunneling, postgresql, mysql, redis, tcp, fxTunnel, developer-tools, security]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I tunnel PostgreSQL through fxTunnel?"
    a: "Yes. Run fxtunnel tcp 5432 to create a TCP tunnel to your local PostgreSQL instance. You will receive a public address like tunnel.fxtun.dev:41234 that any psql client, GUI tool, or application can connect to. The connection uses the standard PostgreSQL wire protocol — no configuration changes are needed on the database side."
  - q: "Is it safe to expose a database through a tunnel?"
    a: "For development and testing, yes. fxTunnel encrypts all traffic between the client and server via TLS 1.3. However, the tunnel endpoint itself is publicly reachable, so always use strong database passwords, restrict user privileges, and close the tunnel when you are done. Never expose a production database with real customer data through a tunnel."
  - q: "How much latency does a database tunnel add?"
    a: "fxTunnel adds +1-5 ms on top of the base network latency. For interactive queries (psql, mysql CLI, GUI tools) this is imperceptible. For bulk operations like migrations or large data imports, the overhead is minimal compared to the time spent on disk I/O and query execution."
  - q: "What is the difference between an SSH tunnel and fxTunnel for database access?"
    a: "An SSH tunnel requires your own server with SSH access, manual setup with ssh -L flags, and no automatic reconnection. fxTunnel works without your own server, sets up in one command, reconnects automatically, and supports a traffic inspector. Both encrypt traffic — SSH via the SSH protocol, fxTunnel via TLS 1.3."
  - q: "Can I tunnel multiple databases at the same time?"
    a: "Yes. Run a separate fxtunnel tcp command for each database port: 5432 for PostgreSQL, 3306 for MySQL, 6379 for Redis. Each tunnel gets its own public address. fxTunnel supports multiple simultaneous tunnels on the free tier."
---

## Why Tunnel a Database?

Your PostgreSQL runs on localhost, your teammate needs to query it, and your CI server needs to run migrations against it. Sound familiar? The traditional answers -- VPN, SSH tunnels, opening ports on the router -- are either slow to set up, insecure, or flat-out impossible behind CGNAT and corporate firewalls.

A [TCP tunnel](/blog/tcp-udp-tunneling-explained/) solves this in one command. It creates a public address that forwards TCP connections directly to your local database port. The database itself requires no configuration changes — it still listens on localhost, and the tunnel handles the rest.

This guide covers three databases that developers tunnel most often — PostgreSQL, MySQL, and Redis — with step-by-step fxTunnel commands, security best practices, and a comparison with SSH tunneling.

## How Database Tunneling Works

Database tunneling is simply TCP tunneling applied to a database port. Every major database -- PostgreSQL, MySQL, Redis, MongoDB -- communicates over TCP. The tunnel creates a virtual channel between a public port on a relay server and your local database port. External clients connect to the public address, and every byte is forwarded to your local database and back.

The flow looks like this:

```
Remote client (psql, mysql, redis-cli, DBeaver, your app)
        |
        | TCP connection
        v
+------------------------------+
|  fxTunnel Server             |
|  Public port: 41234          |
|                              |
|  Multiplexer: stream per     |
|  client connection           |
+--------------+---------------+
               | single TLS connection
               | (all streams multiplexed)
               v
+------------------------------+
|  fxTunnel Client             |
|                              |
|  Demultiplexer:              |
|  stream -> localhost:5432    |
+------------------------------+
               |
               v
        localhost:5432
        (PostgreSQL)
```

Key properties of a database tunnel:

- **Protocol-agnostic** — the tunnel forwards raw TCP bytes. It does not care whether the payload is the PostgreSQL wire protocol, MySQL's COM protocol, or Redis RESP. Any TCP-based database works.
- **Multiplexed** — multiple concurrent connections from different clients travel through a single TLS connection between the fxTunnel client and server. Each connection gets its own logical stream.
- **Encrypted** — all data between the fxTunnel client and server is encrypted with TLS 1.3. For more on the security model, see [TLS 1.3 in Tunnel Security](/blog/tls-13-tunnel-security/).
- **Low latency** — fxTunnel adds +1-5 ms on top of the base network latency. For database queries, this is negligible.

## Installation

Install fxTunnel on the machine where the database is running:

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify
fxtunnel --version
```

Docker-based setups and other installation options are covered in [Docker + Tunnel](/blog/docker-tunnel-expose-containers/).

## PostgreSQL Tunneling

PostgreSQL is the database developers tunnel most often -- for collaborative debugging, sharing a dev database with teammates, running migrations from CI, or connecting GUI tools like DBeaver and pgAdmin.

### Create the Tunnel

```bash
# Forward local PostgreSQL (default port 5432)
fxtunnel tcp 5432
```

fxTunnel prints the public address:

```
fxTunnel v1.x — tunnel is active
Protocol:    TCP
Public addr: tunnel.fxtun.dev:41234
Forwarding:  tunnel.fxtun.dev:41234 -> localhost:5432

Press Ctrl+C to stop
```

### Connect with psql

From any machine with network access:

```bash
psql -h tunnel.fxtun.dev -p 41234 -U myuser -d mydb
```

The connection behaves exactly as if you were connecting to a local PostgreSQL instance. All psql features work: `\dt`, `\d table_name`, `COPY`, transactions, and prepared statements.

### Connect with a GUI Tool (DBeaver, pgAdmin)

In your GUI tool, create a new PostgreSQL connection with:

| Field | Value |
|---|---|
| **Host** | `tunnel.fxtun.dev` |
| **Port** | `41234` (from fxTunnel output) |
| **Database** | `mydb` |
| **Username** | `myuser` |
| **Password** | your database password |

No special configuration is needed. The GUI tool sees a standard PostgreSQL server.

### Connect from an Application

Update the connection string in your application to point at the tunnel:

```bash
# Standard PostgreSQL connection string
DATABASE_URL=postgres://myuser:mypass@tunnel.fxtun.dev:41234/mydb
```

```python
# Python (psycopg2)
import psycopg2
conn = psycopg2.connect(
    host="tunnel.fxtun.dev",
    port=41234,
    user="myuser",
    password="mypass",
    dbname="mydb"
)
```

```javascript
// Node.js (pg)
const { Client } = require('pg');
const client = new Client({
  host: 'tunnel.fxtun.dev',
  port: 41234,
  user: 'myuser',
  password: 'mypass',
  database: 'mydb',
});
await client.connect();
```

### PostgreSQL on a Non-Standard Port

If PostgreSQL listens on a custom port, specify it:

```bash
fxtunnel tcp 15432
```

## MySQL Tunneling

MySQL tunneling works identically to PostgreSQL. The tunnel forwards raw TCP — the MySQL wire protocol passes through without modification.

### Create the Tunnel

```bash
# Forward local MySQL (default port 3306)
fxtunnel tcp 3306
```

Output:

```
fxTunnel v1.x — tunnel is active
Protocol:    TCP
Public addr: tunnel.fxtun.dev:38901
Forwarding:  tunnel.fxtun.dev:38901 -> localhost:3306

Press Ctrl+C to stop
```

### Connect with the mysql CLI

```bash
mysql -h tunnel.fxtun.dev -P 38901 -u root -p
```

Note the capital `-P` for the port (lowercase `-p` is the password prompt in the MySQL client).

### Connect from an Application

```bash
# MySQL connection string
DATABASE_URL=mysql://root:mypass@tunnel.fxtun.dev:38901/mydb
```

```python
# Python (mysql-connector)
import mysql.connector
conn = mysql.connector.connect(
    host="tunnel.fxtun.dev",
    port=38901,
    user="root",
    password="mypass",
    database="mydb"
)
```

```javascript
// Node.js (mysql2)
const mysql = require('mysql2/promise');
const conn = await mysql.createConnection({
  host: 'tunnel.fxtun.dev',
  port: 38901,
  user: 'root',
  password: 'mypass',
  database: 'mydb',
});
```

### MySQL SSL Mode

Some MySQL clients default to SSL when connecting to a non-localhost host. If your local MySQL instance does not have SSL configured, you may need to disable SSL on the client side:

```bash
mysql -h tunnel.fxtun.dev -P 38901 -u root -p --ssl-mode=DISABLED
```

This is safe because the tunnel itself is encrypted with TLS 1.3 between the fxTunnel client and server. The traffic between your local MySQL and the fxTunnel client stays on localhost and does not leave the machine.

## Redis Tunneling

Redis is the most lightweight database to tunnel. It uses a simple text-based protocol (RESP) over TCP, and connections are typically short-lived or use a connection pool.

### Create the Tunnel

```bash
# Forward local Redis (default port 6379)
fxtunnel tcp 6379
```

Output:

```
fxTunnel v1.x — tunnel is active
Protocol:    TCP
Public addr: tunnel.fxtun.dev:44567
Forwarding:  tunnel.fxtun.dev:44567 -> localhost:6379

Press Ctrl+C to stop
```

### Connect with redis-cli

```bash
redis-cli -h tunnel.fxtun.dev -p 44567
```

All redis-cli commands work normally:

```
tunnel.fxtun.dev:44567> SET mykey "hello"
OK
tunnel.fxtun.dev:44567> GET mykey
"hello"
tunnel.fxtun.dev:44567> KEYS *
1) "mykey"
```

### Connect from an Application

```python
# Python (redis-py)
import redis
r = redis.Redis(
    host="tunnel.fxtun.dev",
    port=44567,
    decode_responses=True
)
r.set("key", "value")
print(r.get("key"))
```

```javascript
// Node.js (ioredis)
const Redis = require('ioredis');
const client = new Redis({
  host: 'tunnel.fxtun.dev',
  port: 44567,
});
await client.set('key', 'value');
const val = await client.get('key');
```

### Redis AUTH

If your Redis instance requires authentication, the tunnel does not interfere. Use the standard AUTH command or pass the password in the connection:

```bash
redis-cli -h tunnel.fxtun.dev -p 44567 -a yourpassword
```

## Tunneling Multiple Databases Simultaneously

A common development setup includes PostgreSQL, Redis, and possibly MySQL running on the same machine. You can tunnel all of them at once:

```bash
# Terminal 1: PostgreSQL
fxtunnel tcp 5432

# Terminal 2: MySQL
fxtunnel tcp 3306

# Terminal 3: Redis
fxtunnel tcp 6379
```

Each tunnel gets an independent public address. fxTunnel supports multiple simultaneous tunnels on the free tier.

### Docker Compose Example

For a complete development stack with tunnels, add fxTunnel as a service in docker-compose:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  tunnel-postgres:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["tcp", "postgres:5432"]
    depends_on:
      - postgres

  tunnel-redis:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["tcp", "redis:6379"]
    depends_on:
      - redis
```

```bash
docker compose up
# tunnel-postgres_1 | tcp://pg-xyz.fxtun.dev:18432 -> tcp://postgres:5432
# tunnel-redis_1    | tcp://rd-xyz.fxtun.dev:19637 -> tcp://redis:6379
```

More Docker-specific scenarios are covered in [Docker + Tunnel: Access Containers from the Internet](/blog/docker-tunnel-expose-containers/).

## Database Tunnel vs SSH Tunnel

SSH tunneling (`ssh -L`) has been the go-to method for remote database access for decades. It still works, but fxTunnel addresses several pain points that SSH does not. A broader comparison is available in [SSH Tunnel vs Modern Tools](/blog/ssh-tunnel-vs-modern-tools/).

### Command Comparison

| Task | SSH Tunnel | fxTunnel |
|---|---|---|
| **Tunnel PostgreSQL** | `ssh -L 5432:localhost:5432 user@server` | `fxtunnel tcp 5432` |
| **Tunnel MySQL** | `ssh -L 3306:localhost:3306 user@server` | `fxtunnel tcp 3306` |
| **Tunnel Redis** | `ssh -L 6379:localhost:6379 user@server` | `fxtunnel tcp 6379` |
| **Multiple databases** | `ssh -L 5432:localhost:5432 -L 6379:localhost:6379 user@server` | `fxtunnel tcp 5432` and `fxtunnel tcp 6379` |
| **Background mode** | `ssh -f -N -L 5432:localhost:5432 user@server` | `fxtunnel tcp 5432` (runs in foreground, Ctrl+C to stop) |

### Feature Comparison

| Feature | SSH Tunnel | fxTunnel |
|---|---|---|
| **Own server required** | Yes (with public IP) | No |
| **Setup time** | 5-30 min (server + keys) | 30 seconds |
| **Direction** | Remote -> local (ssh -L) | Local -> public (reverse) |
| **Auto-reconnect** | No (requires autossh) | Yes |
| **Multiple clients** | Requires sshd config changes | Built-in multiplexing |
| **Encryption** | SSH protocol | TLS 1.3 |
| **Traffic inspector** | No | Yes (from $5/mo) |
| **UDP support** | No | Yes |
| **Vendor dependency** | None | None (open source) |

### When SSH Is Better

SSH tunnels remain the right choice when:

- You already have a server with SSH access and the database runs on that server.
- You need to access a database on a **remote** server from your **local** machine (ssh -L direction).
- Your corporate policy prohibits installing third-party tools.

### When fxTunnel Is Better

fxTunnel is the better choice when:

- You need to give **others** access to your **local** database (reverse direction).
- You do not have a server with a public IP.
- You need auto-reconnect, multiplexing, and simple setup.
- You are running databases in Docker and want a declarative docker-compose setup.

## Security Best Practices

Exposing a database -- even for development -- requires care. A TCP tunnel makes your database reachable from the internet, and anyone who discovers the public address and port can attempt to connect. A few straightforward precautions go a long way.

### 1. Use Strong Database Credentials

Never use default passwords or passwordless access when tunneling:

```bash
# PostgreSQL: set a strong password
ALTER USER myuser WITH PASSWORD 'a-strong-random-password';

# MySQL: set a strong password
ALTER USER 'root'@'%' IDENTIFIED BY 'a-strong-random-password';

# Redis: enable AUTH
# In redis.conf:
requirepass a-strong-random-password
```

### 2. Restrict Database User Privileges

Create a dedicated user with minimal privileges for tunnel access:

```sql
-- PostgreSQL: read-only user for tunnel access
CREATE USER tunnel_user WITH PASSWORD 'strong-pass';
GRANT CONNECT ON DATABASE mydb TO tunnel_user;
GRANT USAGE ON SCHEMA public TO tunnel_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO tunnel_user;

-- MySQL: read-only user
CREATE USER 'tunnel_user'@'%' IDENTIFIED BY 'strong-pass';
GRANT SELECT ON mydb.* TO 'tunnel_user'@'%';
FLUSH PRIVILEGES;
```

### 3. Close the Tunnel When Done

A tunnel is not a permanent service. Close it as soon as you finish your work:

```bash
# Press Ctrl+C in the terminal running fxTunnel
# The tunnel shuts down immediately, the public address stops working
```

### 4. Bind to Localhost

Ensure your database only listens on localhost, not on all interfaces. This prevents direct access even if someone scans your machine:

```bash
# PostgreSQL (postgresql.conf)
listen_addresses = 'localhost'

# MySQL (my.cnf)
bind-address = 127.0.0.1

# Redis (redis.conf)
bind 127.0.0.1
```

The tunnel connects to localhost, so this configuration works perfectly with tunneling.

### 5. Never Tunnel Production Databases

A TCP tunnel is designed for development and testing. Never expose a production database with real customer data through a tunnel. For production access, use VPNs, private networks, or database-specific access management tools.

## Comparison: Database Access Methods

| Method | Setup | Security | Cost | Best For |
|---|---|---|---|---|
| **fxTunnel TCP** | 30 seconds, one command | TLS 1.3 + db auth | Free | Dev/test, sharing, quick access |
| **SSH Tunnel** | 5-30 min, own server | SSH encryption + db auth | VPS from $5/mo | Accessing remote server databases |
| **VPN** | 30-60 min, infrastructure | Full network encryption | Varies | Permanent team access |
| **Direct port opening** | 5 min, router access | Minimal (no encryption) | Free | Never recommended |
| **Cloud DB public IP** | 5 min, provider console | Provider firewall + db auth | Provider pricing | Cloud-hosted databases |

## Common Use Cases

### Collaborative Debugging

Your teammate reports a bug that only reproduces with specific data in the database. Instead of creating a dump and sending it over, tunnel the database and let them query it directly:

```bash
fxtunnel tcp 5432
# Share the public address with your teammate
# They connect with: psql -h tunnel.fxtun.dev -p 41234 -U debug_user mydb
```

### CI/CD Pipeline Testing

Run integration tests in CI against your local database:

```bash
# On your machine: start the tunnel
fxtunnel tcp 5432

# In CI config: point the test database URL at the tunnel
DATABASE_URL=postgres://ci_user:pass@tunnel.fxtun.dev:41234/mydb
npm test
```

### Mobile App Development

A mobile app running on a physical device cannot reach localhost. Tunnel the API server and the database:

```bash
# API server
fxtunnel http 3000

# Database (if the mobile app needs direct DB access for debugging)
fxtunnel tcp 5432
```

### Remote DBA Work

A database administrator needs to inspect and optimize your development database. Instead of granting SSH access to your machine, tunnel the database port:

```bash
fxtunnel tcp 5432
# Share the address and a read-only user with the DBA
```

## Pricing

The cloud service at [fxtun.dev](https://fxtun.dev) lets you start tunneling in 30 seconds with no server setup.

| Plan | Price | Features |
|---|---|---|
| **Free** | $0 | HTTP/TCP/UDP tunnels, no connection limits |
| **Pro** | from $5/mo | Custom domains, request inspector, replay |
| **Team** | from $10/mo | 10+ concurrent tunnels, priority support |

Database tunneling works on the free tier. The source code is fully open on [GitHub](https://github.com/mephistofox/fxtun.dev).

## FAQ

### Can I tunnel PostgreSQL through fxTunnel?

Yes -- `fxtunnel tcp 5432` creates a TCP tunnel to your local PostgreSQL. You get a public address like `tunnel.fxtun.dev:41234` that any psql client, GUI tool, or application can connect to. PostgreSQL's wire protocol passes through unchanged, so no database-side configuration is needed.

### Is it safe to expose a database through a tunnel?

For development and testing -- yes. All traffic between the client and the relay server is encrypted with TLS 1.3. That said, the tunnel endpoint is publicly reachable, so you should always use strong database passwords, limit user privileges to what is needed, and shut the tunnel down when the session is over. Never expose a production database with real customer data this way.

### How much latency does a database tunnel add?

Expect +1-5 ms on top of the base network latency. For interactive queries in psql, mysql CLI, or a GUI tool, that is imperceptible. For bulk operations like migrations or large data imports, the tunnel overhead is small compared to the time spent on disk I/O and query execution.

### What is the difference between an SSH tunnel and fxTunnel for database access?

SSH tunnels require your own server with SSH access, manual `ssh -L` setup, and do not auto-reconnect (unless you use autossh). fxTunnel needs no server, sets up in one command, reconnects on its own, and includes a traffic inspector. Both encrypt traffic -- SSH via its own protocol, fxTunnel via TLS 1.3. The [SSH Tunnel vs Modern Tools](/blog/ssh-tunnel-vs-modern-tools/) article goes deeper into the trade-offs.

### Can I tunnel multiple databases at the same time?

Yes. Run a separate `fxtunnel tcp` command for each port -- `5432` for PostgreSQL, `3306` for MySQL, `6379` for Redis. Each tunnel gets its own public address, and you can run multiple tunnels simultaneously even on the free tier.
