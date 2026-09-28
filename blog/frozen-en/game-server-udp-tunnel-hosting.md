---
title: "Hosting a Game Server Behind NAT with a UDP Tunnel"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Host Minecraft, Valheim, CS2, and more behind NAT with a UDP tunnel: port setup, latency tips, and player connection guide using fxTunnel."
tags: [game-server, udp, tunneling, fxTunnel, minecraft, valheim, cs2, networking, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I host a game server behind NAT without port forwarding?"
    a: "Yes. fxTunnel creates a UDP tunnel from your local game server to a public address. Players connect to the public address, and traffic is forwarded to your machine through the tunnel. No router port forwarding, no static IP, and no CGNAT workarounds are needed."
  - q: "How much latency does a UDP tunnel add to a game server?"
    a: "fxTunnel adds +1–5 ms on top of the base network latency. For most games (Minecraft, Valheim, Terraria) this is imperceptible. For competitive shooters like CS2, the overhead is noticeable only if you are already near the latency threshold. The tunnel server location relative to your players matters more than the tunnel overhead itself."
  - q: "Which games work with a UDP tunnel?"
    a: "Any game that uses UDP for its server protocol works with fxTunnel. This includes Minecraft (Bedrock Edition), Valheim, Counter-Strike 2, Terraria, Factorio, ARK, Rust, and many others. Games using TCP (like Minecraft Java Edition) work with fxtunnel tcp instead."
  - q: "Do other tunneling tools support UDP for game servers?"
    a: "No. ngrok only supports TCP and HTTP tunnels. Cloudflare Tunnel's free tier is limited to HTTP. Tailscale and ZeroTier create VPN meshes that require installation on every player's machine. fxTunnel is the only tunneling tool that offers native UDP tunnel support where players connect directly without installing anything."
  - q: "How many players can connect through a UDP tunnel?"
    a: "fxTunnel does not impose a player limit on the tunnel itself. The limiting factor is your game server hardware and your internet connection's upload bandwidth. A typical home connection (10+ Mbps upload) can handle 10–20 players on most games without issues."
---

## The Problem: Game Servers Behind NAT

You have a powerful machine at home, a game server installed and configured, and friends who want to join. There is just one problem: your machine is behind NAT. Your ISP assigns a shared IP through CGNAT, or your router does not allow port forwarding, or you simply do not want to open ports on your home network. Players cannot reach your server because the network blocks incoming connections.

This is the fundamental challenge of self-hosted game servers: the server needs to accept incoming UDP connections from players, but NAT and firewalls prevent any inbound traffic from reaching it. Traditional solutions — port forwarding, static IPs, VPNs — are either impossible, expensive, or require every player to install software. A [tunnel](/blog/what-is-tunneling/) solves this with a single command.

{{< diagram >}}
The problem: players cannot reach your game server

Player A ──UDP──▶ ╳  ┌─────────────────────┐
                     │  NAT / Router       │
Player B ──UDP──▶ ╳  │  (no port forwarding)│
                     │  CGNAT / dynamic IP  │
Player C ──UDP──▶ ╳  └──────────┬──────────┘
                                │
                         ┌──────▼──────┐
                         │ Game Server │
                         │ 192.168.1.10│
                         │ :25565      │
                         └─────────────┘
{{< /diagram >}}

## Why Traditional Solutions Fail for Game Servers

Before reaching for a tunnel, you have probably tried one of these approaches. Each has serious drawbacks for game server hosting.

**Port forwarding on the router** requires admin access to the router and a public IP (not CGNAT). Many residential ISPs use CGNAT, which means dozens of customers share a single public IP. Port forwarding is simply impossible. Even if you have a public IP, opening a UDP port permanently exposes your home network. We cover the risks in more depth in [How to Expose Localhost to the Internet](/blog/expose-localhost-to-internet/).

**Static IP from the ISP** costs money (often $5–15/month), ties you to a specific provider, and still requires firewall configuration. It does not solve the CGNAT problem — some ISPs do not offer static IPs on residential plans at all.

**VPN mesh (Tailscale, ZeroTier)** creates an encrypted network between machines but requires every player to install the VPN client, create an account, and join your network. For a casual gaming session with friends, asking everyone to install a VPN is a dealbreaker.

**A UDP tunnel** is one command on the server machine. Players connect to a public address without installing anything. No router settings, no static IP, no VPN accounts.

## How a UDP Tunnel Works for Game Servers

A UDP tunnel creates a public address that forwards all incoming UDP datagrams to your local game server. The fxTunnel client on your machine establishes an outbound TLS connection to the fxTunnel server and registers a UDP tunnel for a specific local port. The server allocates a public UDP port and begins accepting player connections.

The [TCP and UDP Tunneling](/blog/tcp-udp-tunneling-explained/) article covers the protocol mechanics in more depth.

{{< diagram >}}
The solution: UDP tunnel makes the server reachable

Player A ──UDP──▶ ┌──────────────────────────┐
                  │  fxTunnel Server          │
Player B ──UDP──▶ │  tunnel.fxtun.dev:52001   │
                  │                           │
Player C ──UDP──▶ │  ┌──────────────────────┐ │
                  │  │ Virtual sessions      │ │
                  │  │ Player A → Sess #1    │ │
                  │  │ Player B → Sess #2    │ │
                  │  │ Player C → Sess #3    │ │
                  │  └──────────┬───────────┘ │
                  └─────────────┼─────────────┘
                                │ TLS connection
                                │ (outbound from your machine)
                  ┌─────────────┼─────────────┐
                  │  NAT / Router             │
                  └─────────────┼─────────────┘
                                │
                  ┌─────────────▼─────────────┐
                  │  Your machine              │
                  │  fxTunnel Client           │
                  │                            │
                  │  ┌──────────────────────┐  │
                  │  │ Decapsulation        │  │
                  │  │ Sess #1 → UDP :25565 │  │
                  │  │ Sess #2 → UDP :25565 │  │
                  │  │ Sess #3 → UDP :25565 │  │
                  │  └──────────────────────┘  │
                  │            │               │
                  │     ┌──────▼──────┐        │
                  │     │ Game Server │        │
                  │     │ :25565      │        │
                  │     └─────────────┘        │
                  └────────────────────────────┘
{{< /diagram >}}

Each player's UDP datagrams arrive at the fxTunnel server, which groups them by source address:port into virtual sessions. The datagrams are encapsulated inside frames and forwarded through the TLS connection to your machine. The fxTunnel client decapsulates the frames and sends raw UDP datagrams to the local game server port. Response datagrams from the server travel back the same way.

The key point: **players do not install anything**. They connect to `tunnel.fxtun.dev:52001` as if it were any other game server. The tunnel is transparent.

## Why fxTunnel Is the Only Option for UDP Game Servers

Most tunneling tools do not support UDP at all. This is a critical limitation for game servers because the vast majority of multiplayer games use UDP for real-time communication.

| Tool | HTTP | TCP | UDP | Players need to install? |
|---|---|---|---|---|
| **fxTunnel** | Yes | Yes | Yes | No |
| ngrok | Yes | Yes | No | N/A |
| Cloudflare Tunnel (free) | Yes | No | No | N/A |
| Tailscale | N/A | Yes (VPN) | Yes (VPN) | Yes (VPN client) |
| ZeroTier | N/A | Yes (VPN) | Yes (VPN) | Yes (VPN client) |
| [bore](/blog/bore-vs-fxtunnel-vs-localtunnel/) | No | Yes | No | N/A |

fxTunnel is the only tunneling tool that supports native UDP tunnels where players connect directly without installing any software. The [ngrok vs Cloudflare Tunnel vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) comparison breaks this down further.

## Quick Start: One Command to Host a Game Server

### Installation

```bash
# Install fxTunnel (Linux/macOS/Windows)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify
fxtunnel --version
```

### Create a UDP Tunnel

```bash
# Forward your game server's UDP port to the internet
fxtunnel udp 25565
```

fxTunnel prints the public address:

```
fxTunnel v1.x — tunnel is active
Protocol:    UDP
Public addr: tunnel.fxtun.dev:52001
Forwarding:  tunnel.fxtun.dev:52001 → localhost:25565

Press Ctrl+C to stop
```

Tell your players to connect to `tunnel.fxtun.dev:52001`. Done.

## Game-Specific Setup Guides

### Minecraft Bedrock Edition (UDP)

Minecraft Bedrock Edition uses UDP on port 19132 by default. This is the version that runs on consoles, mobile devices, and Windows 10/11.

**1. Configure the server** — edit `server.properties`:

```properties
# server.properties (Minecraft Bedrock)
server-name=My Tunneled Server
server-port=19132
server-portv6=19133
max-players=10
view-distance=16
tick-distance=4
online-mode=true
```

**2. Start the game server:**

```bash
# Start Bedrock Dedicated Server
./bedrock_server
```

**3. Create the tunnel:**

```bash
# Forward Bedrock's UDP port
fxtunnel udp 19132
# → tunnel.fxtun.dev:52001
```

**4. Players connect** by adding the server in Minecraft:
- Address: `tunnel.fxtun.dev`
- Port: `52001`

> **Note:** Minecraft Java Edition uses TCP, not UDP. For Java Edition, use `fxtunnel tcp 25565` instead.

### Valheim (UDP)

Valheim uses UDP on ports 2456–2458. Port 2456 is the game port, 2457 is the query port, and 2458 is reserved.

**1. Configure the server** — launch script:

```bash
#!/bin/bash
# start_valheim.sh
./valheim_server.x86_64 \
  -name "My Valheim Server" \
  -port 2456 \
  -world "MyWorld" \
  -password "secretpassword" \
  -public 0
```

Set `-public 0` to prevent the server from registering on the public server list (the tunnel address would not match).

**2. Create the tunnel:**

```bash
# Forward Valheim's game port (UDP)
fxtunnel udp 2456
# → tunnel.fxtun.dev:52001

# Forward the query port separately (if needed for server browser)
fxtunnel udp 2457
# → tunnel.fxtun.dev:52002
```

**3. Players connect** by entering the address in Steam:
- Open Steam → View → Game Servers → Favorites
- Add Server: `tunnel.fxtun.dev:52001`
- Or connect directly in-game: `tunnel.fxtun.dev:52001`

### Counter-Strike 2 (UDP)

CS2 uses UDP on port 27015 by default.

**1. Start the server:**

```bash
# CS2 dedicated server
./cs2 -dedicated \
  -port 27015 \
  +game_type 0 \
  +game_mode 1 \
  +map de_dust2 \
  +sv_setsteamaccount YOUR_GSLT_TOKEN
```

**2. Create the tunnel:**

```bash
# Forward CS2's UDP port
fxtunnel udp 27015
# → tunnel.fxtun.dev:52001
```

**3. Players connect** via the console:
```
connect tunnel.fxtun.dev:52001
```

### Terraria (TCP)

Terraria uses TCP on port 7777. Use a TCP tunnel:

```bash
# Forward Terraria's TCP port
fxtunnel tcp 7777
# → tunnel.fxtun.dev:41234
```

Players join by entering `tunnel.fxtun.dev` and port `41234` in the multiplayer menu.

### Factorio (UDP)

Factorio uses UDP on port 34197 by default.

```bash
# Forward Factorio's UDP port
fxtunnel udp 34197
# → tunnel.fxtun.dev:52001
```

Players connect by entering `tunnel.fxtun.dev:52001` in the multiplayer menu.

## Quick Reference: Game Ports

| Game | Protocol | Default Port | fxTunnel Command |
|---|---|---|---|
| Minecraft Bedrock | UDP | 19132 | `fxtunnel udp 19132` |
| Minecraft Java | TCP | 25565 | `fxtunnel tcp 25565` |
| Valheim | UDP | 2456 | `fxtunnel udp 2456` |
| Counter-Strike 2 | UDP | 27015 | `fxtunnel udp 27015` |
| Terraria | TCP | 7777 | `fxtunnel tcp 7777` |
| Factorio | UDP | 34197 | `fxtunnel udp 34197` |
| ARK: Survival Evolved | UDP | 7777 | `fxtunnel udp 7777` |
| Rust | UDP | 28015 | `fxtunnel udp 28015` |
| Palworld | UDP | 8211 | `fxtunnel udp 8211` |
| Satisfactory | UDP | 7777 | `fxtunnel udp 7777` |

## Latency Considerations for Gaming

How much lag does a tunnel actually add? Latency is the most important factor for online gaming, especially in competitive titles. Here is what you need to know.

### What the Tunnel Adds

fxTunnel adds +1–5 ms of overhead on top of the base network latency between the player and the tunnel server. This comes from:

- **Encapsulation/decapsulation** — wrapping UDP datagrams into frames adds a small processing delay.
- **Extra network hop** — traffic goes Player → fxTunnel Server → Your machine instead of Player → Your machine directly. However, if your machine is behind CGNAT, direct connection is impossible anyway.

### Latency Breakdown

{{< diagram >}}
Player ──[30ms]──▶ fxTunnel Server ──[2ms overhead]──▶ Your machine
                                                        (game server)

Total latency: 30ms base + 2ms tunnel = 32ms
Without tunnel (if direct were possible): 30ms

Tunnel overhead: ~2ms (+6.7%)
{{< /diagram >}}

For most games, 2–5 ms of additional latency is imperceptible:

| Game type | Acceptable latency | Tunnel impact |
|---|---|---|
| Survival/sandbox (Minecraft, Valheim, Factorio) | < 150 ms | Negligible |
| Co-op/RPG (Terraria, ARK) | < 100 ms | Negligible |
| Competitive shooters (CS2, Valorant) | < 50 ms | Noticeable if already near threshold |

### How to Minimize Latency

1. **Choose a tunnel server close to your players.** If your friends are in Europe, a European tunnel server is ideal. Tunnel server location matters more than tunnel overhead.
2. **Use a wired connection** on the machine running the game server. Wi-Fi adds 2–10 ms of jitter.
3. **Close unnecessary tunnels and bandwidth-heavy applications** while hosting the game.
4. **Monitor latency** with the fxTunnel [traffic inspector](/blog/traffic-inspector-debug-requests/) to identify bottlenecks.

## Running the Tunnel as a Background Service

For a game server that should be available 24/7 (or at least whenever the machine is on), set up the tunnel to start automatically.

### systemd Service (Linux)

```ini
# /etc/systemd/system/fxtunnel-game.service
[Unit]
Description=fxTunnel - Game server UDP tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel udp 25565
Restart=always
RestartSec=5
User=gameserver
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable fxtunnel-game
sudo systemctl start fxtunnel-game

# Check status
sudo systemctl status fxtunnel-game
```

### Running Alongside Docker

If your game server runs in Docker, you can tunnel it from the host:

```bash
# Start the game server in Docker
docker run -d -p 25565:25565/udp --name minecraft-bedrock itzg/minecraft-bedrock-server

# Create a tunnel to the mapped port
fxtunnel udp 25565
```

You can read more about this workflow in [Docker + Tunnel: Access Containers from the Internet](/blog/docker-tunnel-expose-containers/).

## Performance Tips for Game Server Hosting

### Upload Bandwidth

The most common bottleneck for home-hosted game servers is upload bandwidth. Each connected player consumes upload bandwidth from your connection. A typical multiplayer game uses 30–100 KB/s per player.

| Players | Estimated upload needed |
|---|---|
| 4 players | 0.5–1 Mbps |
| 10 players | 1–3 Mbps |
| 20 players | 3–6 Mbps |

Check your upload speed with `speedtest-cli` or any online speed test. If your upload is below 5 Mbps, limit the player count accordingly.

### Server Resources

The tunnel itself uses minimal CPU and memory. The game server is the resource bottleneck. General recommendations:

- **CPU:** 2+ cores for most game servers. Minecraft and Factorio are particularly CPU-hungry.
- **RAM:** 4 GB minimum for Minecraft, 2 GB for Valheim, 1 GB for Terraria.
- **Storage:** Use an SSD for the game world files. Disk I/O affects tick rate.

### Multiple Game Servers

If you host several game servers, create a separate tunnel for each:

```bash
# Minecraft Bedrock on port 19132
fxtunnel udp 19132 &

# Valheim on port 2456
fxtunnel udp 2456 &

# Factorio on port 34197
fxtunnel udp 34197 &
```

Each tunnel gets its own public address. The free plan supports multiple simultaneous tunnels. Custom domains, the traffic inspector, and request replay start from $5/month, and dedicated support with 10+ tunnels from $10/month.

## Security Considerations

Game servers are frequent targets for DDoS attacks and unauthorized access. A tunnel adds a layer of protection because your real IP is never exposed to players.

### Your IP Stays Hidden

Players connect to `tunnel.fxtun.dev`, not to your home IP. Even if someone decides to launch a DDoS attack, it hits the tunnel server, not your home network. The fxTunnel server is built to handle this.

### Game Server Authentication

Always enable authentication on your game server:

- **Minecraft:** Set `online-mode=true` in `server.properties` to require Mojang authentication.
- **Valheim:** Use the `-password` flag to require a password.
- **CS2:** Use a Game Server Login Token (GSLT) for Steam authentication.

### Firewall on Your Machine

Even with a tunnel, configure a firewall to restrict direct access to the game server port. Only the fxTunnel client should be able to reach it:

```bash
# Allow only localhost to access the game port
sudo ufw default deny incoming
sudo ufw allow ssh
sudo ufw enable
```

The game server listens on `localhost`, and the fxTunnel client connects to it locally. No external traffic reaches the port directly.

## FAQ

### Can I host a game server behind NAT without port forwarding?

Absolutely. fxTunnel sets up a UDP tunnel that gives your local game server a public address. Your friends connect to that address, and traffic flows through the tunnel to your machine. No router configuration, no static IP, and no CGNAT workarounds involved.

### How much latency does a UDP tunnel add to a game server?

Expect an additional 1-5 ms on top of whatever the base network latency is. For sandbox and survival games like Minecraft, Valheim, or Terraria, that is invisible. In competitive shooters like CS2, you will only feel it if your ping is already borderline. Picking a tunnel server close to your players matters more than the tunnel overhead.

### Which games work with a UDP tunnel?

Any game with a UDP-based server protocol. That covers Minecraft Bedrock, Valheim, CS2, Terraria, Factorio, ARK, Rust, and many more. For TCP-based games like Minecraft Java Edition, use `fxtunnel tcp` instead.

### Do other tunneling tools support UDP for game servers?

Not really. ngrok handles TCP and HTTP only. Cloudflare Tunnel's free tier is HTTP-only. Tailscale and ZeroTier use VPN meshes that require every player to install a client. fxTunnel is currently the only tunneling tool offering native UDP support where players connect directly with no software to install.

### How many players can connect through a UDP tunnel?

There is no player cap on the tunnel side. The real bottleneck is your game server hardware and upload bandwidth. A typical home connection with 10+ Mbps upload handles 10-20 players comfortably on most games.
