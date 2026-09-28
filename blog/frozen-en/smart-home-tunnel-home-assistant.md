---
title: "Home Assistant via Tunnel — No Static IP Needed"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Access Home Assistant remotely without a static IP or port forwarding. Set up a secure tunnel with fxTunnel for smart home control from anywhere."
tags: [tunneling, iot, home-assistant, smart-home, fxTunnel, localhost]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I access Home Assistant remotely without a static IP?"
    a: "Yes. Install fxTunnel on the machine running Home Assistant with curl -fsSL https://fxtun.ru/install.sh | bash, then run fxtunnel http 8123. You will get a public HTTPS URL like https://ha.fxtun.dev that forwards to your local Home Assistant instance. No static IP, no port forwarding, no DNS configuration required."
  - q: "Is it safe to expose Home Assistant through a tunnel?"
    a: "Yes, when configured properly. fxTunnel encrypts all traffic with TLS 1.3. Home Assistant has its own authentication system with username and password, multi-factor authentication, and IP ban lists. Combined with tunnel-level security, your smart home is protected by multiple layers of defense."
  - q: "Will a tunnel add noticeable latency to smart home control?"
    a: "For most operations, the added latency is negligible. Toggling a light or checking a sensor reading adds 20-50ms of round-trip time through the tunnel. Camera streams may require more bandwidth, but fxTunnel's direct TLS connection keeps overhead minimal compared to cloud-relay solutions."
---

## The Problem: Smart Homes Trapped Behind NAT

Home Assistant runs on a Raspberry Pi, an Intel NUC, or a dedicated server inside your home network, controlling lights, thermostats, cameras, locks, and hundreds of other devices through its 2,800+ integrations. The dashboard at `http://homeassistant.local:8123` works flawlessly from the couch. But the moment you step outside the front door, that dashboard becomes unreachable.

The reason is simple: home networks sit behind NAT. The router has a private IP on the local side and either a shared IP (via CGNAT) or a dynamic public IP on the internet side. Neither scenario gives Home Assistant a stable, publicly reachable address. Without one, there is no way to check the security camera feed from the office, turn off a forgotten appliance from a train, or let a voice assistant like Alexa trigger an automation through a webhook.

The problem extends beyond convenience. Many Home Assistant automations depend on external triggers: geolocation-based presence detection, webhook-driven routines from IFTTT or Zapier, and voice commands through Alexa or Google Home. All of these require the Home Assistant instance to be reachable from the internet. Without remote access, the smart home loses a significant portion of its automation potential.

This is not a niche inconvenience -- it is the central limitation of every self-hosted smart home. Cloud platforms like Google Home or Apple HomeKit route everything through vendor servers, which means giving up privacy, accepting lock-in, and paying subscription fees. If you chose Home Assistant, you probably wanted to avoid exactly that.

[Tunneling](/blog/what-is-tunneling/) offers a third path: keep the server at home, keep full control, and still access it from anywhere. A tunnel creates an outbound connection from Home Assistant to a public relay server and returns a URL that routes traffic back to the local instance. No static IP, no router changes, no cloud dependency.

{{< diagram >}}
Smart home behind NAT (no public IP)
┌──────────────────────────────────────────────────────┐
│  Home network                                        │
│                                                      │
│  ┌─────────────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Home Assistant  │  │  Zigbee  │  │  Camera  │    │
│  │ Raspberry Pi    │  │  Hub     │  │  RTSP    │    │
│  │ :8123 (HTTP)    │  │          │  │  :554    │    │
│  └────────┬────────┘  └──────────┘  └──────────┘    │
│           │                                          │
│   ┌───────▼────────┐                                 │
│   │  NAT / Router  │                                 │
│   │ (CGNAT / dynamic IP)                             │
│   └───────┬────────┘                                 │
└───────────┼──────────────────────────────────────────┘
            │ outbound TLS connection
            ▼
   ┌──────────────────┐
   │  fxTunnel Server │
   │  tunnel.fxtun.dev│
   └────────┬─────────┘
            │
            ▼
   Public URL:
   https://my-home.fxtun.dev → http://localhost:8123
{{< /diagram >}}

## Why Traditional Methods Fall Short

Before reaching for a tunnel, most Home Assistant users try one of four traditional approaches to remote access. Each has significant drawbacks that make it unsuitable for a 24/7 smart home setup.

### Port Forwarding

Port forwarding tells the router to redirect incoming traffic on a specific port to the Home Assistant machine. It is the most commonly suggested solution in forums, and the most problematic.

**Requirements**: admin access to the router, a public IP address (not CGNAT), and manual NAT rule configuration.

**Problems**:
- **CGNAT blocks it entirely.** Many ISPs — especially mobile and budget providers — put customers behind Carrier-Grade NAT. The router's "public" IP is actually a private IP in the ISP's network. Port forwarding simply does not work in this scenario.
- **Dynamic IP breaks it.** Even when port forwarding works today, the ISP may reassign the public IP tomorrow. Without a static IP, the Home Assistant URL changes unpredictably.
- **Security exposure.** An open port is visible to the entire internet. Bots scan for open ports continuously and will find Home Assistant within hours. Without TLS, credentials travel in plaintext.
- **Router complexity.** Every router has a different admin interface. Some ISP-provided routers restrict port forwarding entirely.

### Dynamic DNS (DDNS)

DDNS services (DuckDNS, No-IP) automatically update a hostname to point at the router's current public IP. This solves the dynamic IP problem but does nothing about CGNAT, and still requires port forwarding.

**Additional limitations**:
- DDNS updates are not instant — there is a propagation delay of seconds to minutes after an IP change, during which the smart home is unreachable.
- DDNS does not provide TLS. Without a certificate, traffic between the phone and Home Assistant is unencrypted.
- The combination of DDNS + Let's Encrypt + port forwarding is fragile and requires ongoing maintenance.

### VPN (WireGuard, OpenVPN)

A VPN creates an encrypted tunnel between a phone or laptop and the home network. It is the most secure traditional option but also the most complex.

**Problems**:
- **Setup complexity.** WireGuard is simpler than OpenVPN, but still requires generating key pairs, configuring the server, opening a UDP port on the router (which again requires a public IP), and installing a VPN client on every device.
- **CGNAT blocks it** just like port forwarding — the VPN server needs a reachable address.
- **Always-on overhead.** Running a VPN on a phone drains battery and routes all traffic through the home network, slowing down everything else.
- **No webhook support.** External services (Alexa, Google Home, IFTTT) cannot send webhooks to a VPN address. They need a public HTTPS endpoint.

### Home Assistant Cloud (Nabu Casa)

Nabu Casa is the official paid cloud service for Home Assistant. It provides remote access and voice assistant integration for $7.50 per month (or $75/year). It works reliably and supports the Home Assistant project financially.

**Limitations**:
- **Cost.** $90/year is significant for a platform that is otherwise free and self-hosted.
- **Vendor dependency.** Access depends on Nabu Casa's servers being online. If the service goes down, remote access stops.
- **Limited scope.** Nabu Casa tunnels only the Home Assistant web interface. It does not provide access to other services running on the same machine — Node-RED, Grafana, MQTT dashboards, or SSH.
- **No custom domain.** The URL is assigned by Nabu Casa and cannot be customized to a personal domain.

### Why a Tunnel Is Different

A [tunnel](/blog/expose-localhost-to-internet/) avoids every limitation listed above:

| Criteria | Port forwarding | DDNS | VPN | Nabu Casa | Tunnel (fxTunnel) |
|---|---|---|---|---|---|
| Works behind CGNAT | No | No | No | Yes | **Yes** |
| Requires router config | Yes | Yes | Yes | No | **No** |
| Built-in TLS | No | No | Yes | Yes | **Yes** |
| Webhook support | Partial | Partial | No | Yes | **Yes** |
| Custom domain | N/A | Yes | N/A | No | **Yes (Pro)** |
| Cost | Free | Free | Free (self-hosted) | $7.50/mo | **Free / $5/mo** |
| Multiple services | Manual | Manual | Yes | HA only | **Yes** |
| Setup time | 15-60 min | 15-30 min | 30-120 min | 5 min | **30 seconds** |

## How Tunneling Solves Remote Access

A tunnel works by reversing the connection direction. Instead of waiting for incoming connections (which NAT blocks), the Home Assistant machine initiates an outbound TLS connection to the fxTunnel relay server. This outbound connection passes through NAT and firewalls without any configuration because it looks like ordinary HTTPS traffic.

The relay server allocates a public URL — for example, `https://abc123.fxtun.dev` — and forwards incoming HTTP requests through the established connection to the local Home Assistant port (8123). From the user's perspective, opening `https://abc123.fxtun.dev` in a browser is indistinguishable from opening `http://homeassistant.local:8123` on the local network.

{{< diagram >}}
How the tunnel works (step by step):

1. fxTunnel client starts on the HA machine
   ┌───────────────┐
   │ Home Assistant │
   │ :8123         │──── outbound TLS ────▶ ┌─────────────────┐
   │ fxTunnel CLI  │                        │ fxTunnel Server  │
   └───────────────┘                        │ tunnel.fxtun.dev │
                                            └────────┬────────┘
                                                     │
2. Server allocates a public URL:                    │
   https://abc123.fxtun.dev                          │
                                                     │
3. User opens URL from anywhere:                     │
   ┌──────────┐                                      │
   │  Phone   │──── HTTPS request ──────────────────▶│
   └──────────┘                                      │
                                                     │
4. Server forwards request through the tunnel:       │
   ┌───────────────┐◀──── forwarded request ─────────┘
   │ Home Assistant │
   │ :8123         │
   └───────────────┘

5. Home Assistant responds, response travels back
   through the same tunnel to the user's phone.
{{< /diagram >}}

Key properties of this approach:

- **No inbound ports needed.** The connection is outbound, so NAT, firewalls, and CGNAT are irrelevant.
- **TLS everywhere.** Traffic between the user and the relay server is HTTPS. Traffic between the relay server and the Home Assistant machine is encrypted via the TLS tunnel. For details on how fxTunnel handles encryption, see [TLS 1.3 and Tunnel Security](/blog/tls-13-tunnel-security/).
- **Stable URL.** With fxTunnel SaaS, the public URL persists across tunnel restarts — even on the free tier.
- **WebSocket support.** Home Assistant heavily relies on WebSocket connections for its frontend. fxTunnel proxies WebSocket traffic transparently.

## Setting Up fxTunnel for Home Assistant

This section walks through the complete setup: installing fxTunnel, creating an HTTP tunnel to Home Assistant, configuring a custom domain, and setting up auto-start with systemd.

### Prerequisites

- Home Assistant running on a Raspberry Pi, Intel NUC, or any Linux machine, listening on port 8123.
- SSH access to the machine running Home Assistant (for Home Assistant OS, use the SSH add-on).
- Internet connectivity (outbound HTTPS — no special firewall rules).

### Step 1. Install fxTunnel

Connect to the Home Assistant machine via SSH and run the installer:

```bash
# One-line install (Linux / macOS — works on ARM and x86)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installation
fxtunnel --version
```

fxTunnel is written in Go and cross-compiled for ARM (armv6, armv7, arm64) and x86_64. The install script automatically detects the architecture. It works on every Raspberry Pi model, including Zero and Pi 5. The [fxTunnel Architecture](/blog/fxtunnel-architecture/) article covers the design in more depth.

**Home Assistant OS users:** Home Assistant OS is a managed operating system that does not provide a standard Linux shell. To install fxTunnel, first install the "Advanced SSH & Web Terminal" add-on from the Add-on Store, then run the install command from that terminal.

**Docker-based installations:** If Home Assistant runs in Docker, install fxTunnel on the host machine (not inside the container). The tunnel connects to `localhost:8123`, which Docker exposes on the host.

### Step 2. Create an HTTP Tunnel

Start the tunnel pointing at the Home Assistant port:

```bash
fxtunnel http 8123
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://d7f2a.fxtun.dev
Forwarding:  https://d7f2a.fxtun.dev -> http://localhost:8123
WebSocket:   supported

Press Ctrl+C to stop
```

Open `https://d7f2a.fxtun.dev` in a browser. The Home Assistant login page appears — the same interface as on the local network, but now accessible from anywhere in the world.

### Step 3. Update Home Assistant Configuration

Home Assistant blocks requests from unknown proxies by default. To allow traffic through the tunnel, update `configuration.yaml`:

```yaml
# configuration.yaml
http:
  use_x_forwarded_for: true
  trusted_proxies:
    - 127.0.0.1
    - ::1
```

This tells Home Assistant to trust the `X-Forwarded-For` header from the tunnel proxy running on localhost. Without this setting, Home Assistant may reject requests or misidentify user IP addresses.

After editing, restart Home Assistant:

```bash
# For Home Assistant OS / Supervised
ha core restart

# For Home Assistant Core (manual install)
sudo systemctl restart home-assistant@homeassistant
```

### Step 4. Configure a Custom Domain (Pro Feature)

The free tier provides a randomly generated subdomain like `d7f2a.fxtun.dev`. The Pro plan ($5/mo) allows custom domains, so the smart home can be accessed at a memorable address like `home.example.com`.

The [custom domain setup guide](/blog/custom-domain-tunnel-setup/) walks through this in detail.

**Step 4a.** Add a CNAME record in the domain's DNS settings:

```
home.example.com  CNAME  tunnel.fxtun.dev
```

**Step 4b.** Start the tunnel with the custom domain flag:

```bash
fxtunnel http 8123 --domain=home.example.com
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://home.example.com
Forwarding:  https://home.example.com -> http://localhost:8123
TLS:         automatic (Let's Encrypt)

Press Ctrl+C to stop
```

fxTunnel automatically provisions a TLS certificate via Let's Encrypt. The smart home is now accessible at `https://home.example.com` with a valid HTTPS certificate.

**Step 4c.** Update Home Assistant to accept the new domain:

```yaml
# configuration.yaml
homeassistant:
  external_url: "https://home.example.com"

http:
  use_x_forwarded_for: true
  trusted_proxies:
    - 127.0.0.1
    - ::1
```

### Step 5. Set Up as a systemd Service (Auto-Start)

A smart home tunnel must run 24/7 and survive reboots and crashes. A systemd service handles this automatically.

Create the service file:

```ini
# /etc/systemd/system/fxtunnel-ha.service
[Unit]
Description=fxTunnel — Home Assistant tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel http 8123
Restart=always
RestartSec=5
User=homeassistant
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable auto-start on boot
sudo systemctl enable fxtunnel-ha

# Start the tunnel now
sudo systemctl start fxtunnel-ha

# Verify it is running
sudo systemctl status fxtunnel-ha
```

Check the logs:

```bash
# View recent logs
sudo journalctl -u fxtunnel-ha -f

# Check for errors
sudo journalctl -u fxtunnel-ha --since "1 hour ago" --no-pager
```

The tunnel will now start automatically on every boot and restart within 5 seconds if it crashes or loses connection.

## Securing Your Smart Home Tunnel

A smart home controls physical devices — door locks, garage doors, alarm systems. Security is not optional. A layered approach combines tunnel-level encryption with Home Assistant's built-in security features.

### Layer 1: TLS Encryption

fxTunnel encrypts all traffic with [TLS 1.3](/blog/tls-13-tunnel-security/). Every connection between the user's device and the fxTunnel server uses HTTPS. The tunnel connection itself is also TLS-encrypted. There is no point in the traffic path where data travels in plaintext.

With a custom domain, fxTunnel automatically provisions and renews a Let's Encrypt certificate. No manual certificate management is needed.

### Layer 2: Home Assistant Authentication

Home Assistant has a robust authentication system. Every remote user must log in with a username and password. The following hardening steps are strongly recommended:

**Enable multi-factor authentication (MFA):**

1. Navigate to the user profile in Home Assistant (click the user icon in the sidebar).
2. Under "Multi-factor Authentication Modules," enable TOTP (Time-based One-Time Password).
3. Scan the QR code with an authenticator app (Google Authenticator, Authy).

Every login now requires both a password and a 6-digit code from the authenticator app.

**Configure IP ban:**

```yaml
# configuration.yaml
http:
  ip_ban_enabled: true
  login_attempts_threshold: 5
  use_x_forwarded_for: true
  trusted_proxies:
    - 127.0.0.1
    - ::1
```

After 5 failed login attempts from the same IP, Home Assistant bans that IP automatically. The bans are stored in `ip_bans.yaml` and can be managed manually.

### Layer 3: fxTunnel Authentication Tokens

Use authentication tokens to ensure that only authorized clients can create tunnels to the fxTunnel server:

```bash
# Pass the token at startup
fxtunnel http 8123 --token=your-secure-token

# Or use an environment variable (recommended for systemd)
export FXTUNNEL_TOKEN=your-secure-token
fxtunnel http 8123
```

### Layer 4: Network Segmentation

Place IoT devices on a separate VLAN or subnet from computers and phones. Most consumer routers support a "guest network" that provides basic isolation. More advanced setups use a managed switch and VLAN-aware router:

{{< diagram >}}
Recommended network segmentation:

┌──────────────────────────────────────────────────┐
│  Home network                                    │
│                                                  │
│  VLAN 1 (Main)        VLAN 2 (IoT)              │
│  ┌────────────┐       ┌──────────────┐           │
│  │ Laptop     │       │ Smart lights │           │
│  │ Phone      │       │ Thermostat   │           │
│  │ Home Asst. │◀─────▶│ Cameras      │           │
│  │ fxTunnel   │       │ Sensors      │           │
│  └────────────┘       └──────────────┘           │
│                                                  │
│  Home Assistant on VLAN 1 talks to IoT           │
│  devices on VLAN 2 via firewall rules.           │
│  IoT devices cannot reach the internet           │
│  or other VLANs directly.                        │
└──────────────────────────────────────────────────┘
{{< /diagram >}}

### Layer 5: Principle of Least Exposure

Only tunnel the services that genuinely need remote access. If the only requirement is controlling lights from a phone, there is no reason to also expose Node-RED or Grafana through the same tunnel. Open tunnels to additional services only when needed, and shut them down when done.

### Layer 6: Keep Software Updated

Regularly update Home Assistant, fxTunnel, and the underlying OS:

```bash
# Update fxTunnel to the latest version
curl -fsSL https://fxtun.ru/install.sh | bash

# Update Home Assistant Core
pip3 install --upgrade homeassistant

# Update the OS (Raspberry Pi OS / Debian)
sudo apt update && sudo apt upgrade -y
```

Outdated software is one of the most common attack vectors for smart home devices. Home Assistant releases security patches regularly, and running an old version behind a tunnel does not make it any safer — the tunnel protects the transport layer, but the application itself must also be secure.

## Advanced Configurations

A Home Assistant machine typically runs several companion services: Node-RED for visual automations, Grafana for metric dashboards, an MQTT broker for sensor data, and sometimes ESPHome or Zigbee2MQTT. fxTunnel can expose all of them simultaneously.

### Multiple HTTP Tunnels for Companion Services

Each web-based service gets its own tunnel:

```bash
# Home Assistant on port 8123
fxtunnel http 8123 &

# Node-RED on port 1880
fxtunnel http 1880 &

# Grafana on port 3000
fxtunnel http 3000 &

# Zigbee2MQTT frontend on port 8080
fxtunnel http 8080 &
```

Each tunnel gets its own public URL. With a Pro plan, assign custom subdomains:

```bash
fxtunnel http 8123 --domain=ha.example.com &
fxtunnel http 1880 --domain=nodered.example.com &
fxtunnel http 3000 --domain=grafana.example.com &
```

The [multiple tunnels guide](/blog/multiple-tunnels-microservices/) covers management strategies for production setups.

### TCP Tunnel for MQTT Broker

MQTT (Message Queuing Telemetry Transport) is the backbone protocol of many IoT setups. An MQTT broker (Mosquitto) listens on TCP port 1883 and acts as a message hub: sensors publish data to topics, and controllers subscribe to those topics to trigger actions. When the broker runs at home and remote IoT devices need to connect — sensors at a vacation home, weather stations on a rooftop, or ESP32 boards at a workshop — a TCP tunnel makes the broker globally reachable:

```bash
fxtunnel tcp 1883
# → tunnel.fxtun.dev:51883
```

Remote devices connect to the broker via the tunnel address:

```yaml
# mosquitto.conf — allow remote connections
listener 1883
allow_anonymous false
password_file /etc/mosquitto/passwd
```

```python
# Remote sensor (MicroPython / Python)
import paho.mqtt.client as mqtt

client = mqtt.Client()
client.username_pw_set("sensor01", "secure-password")
client.connect("tunnel.fxtun.dev", 51883)
client.publish("home/outdoor/temperature", "22.5")
client.disconnect()
```

More IoT tunneling scenarios are covered in [Tunnels for IoT](/blog/iot-tunnel-devices-cloud/).

### Webhook Integrations (Alexa, Google Home, IFTTT)

Voice assistants and automation platforms send HTTP webhooks to trigger Home Assistant automations. These webhooks require a publicly reachable HTTPS URL — exactly what a tunnel provides.

**Alexa Smart Home Skill:**

Home Assistant's Alexa integration requires a public URL for the smart home skill endpoint. With fxTunnel:

1. Start the tunnel: `fxtunnel http 8123`
2. In the Alexa Developer Console, set the skill endpoint to `https://d7f2a.fxtun.dev/api/alexa/smart_home`
3. Configure Home Assistant:

```yaml
# configuration.yaml
alexa:
  smart_home:
```

**Google Home:**

Google Home integration works similarly. The public URL serves as the fulfillment endpoint:

```yaml
# configuration.yaml
google_assistant:
  project_id: your-project-id
  report_state: true
  exposed_domains:
    - light
    - switch
    - climate
```

Register the tunnel URL as the fulfillment URL in the Google Actions Console. Home Assistant handles device discovery, state reporting, and command execution -- the tunnel simply provides the HTTPS endpoint that Google requires.

**IFTTT:**

IFTTT webhooks can trigger Home Assistant automations:

```yaml
# configuration.yaml
ifttt:

# automations.yaml
- alias: "IFTTT trigger: arriving home"
  trigger:
    platform: event
    event_type: ifttt_webhook_received
    event_data:
      action: arriving_home
  action:
    - service: light.turn_on
      target:
        entity_id: light.hallway
```

The IFTTT webhook URL is `https://d7f2a.fxtun.dev/api/webhook/ifttt_webhook_id`. The [webhook testing guide](/blog/webhook-testing-with-tunnel/) covers the general approach in more detail.

### systemd Services for Multiple Tunnels

When running multiple tunnels permanently, create a separate systemd service for each:

```ini
# /etc/systemd/system/fxtunnel-ha.service
[Unit]
Description=fxTunnel — Home Assistant
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel http 8123
Restart=always
RestartSec=5
User=homeassistant
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/fxtunnel-nodered.service
[Unit]
Description=fxTunnel — Node-RED
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel http 1880
Restart=always
RestartSec=5
User=homeassistant
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/fxtunnel-mqtt.service
[Unit]
Description=fxTunnel — MQTT broker
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 1883
Restart=always
RestartSec=5
User=homeassistant
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

Enable all services at once:

```bash
sudo systemctl daemon-reload
sudo systemctl enable fxtunnel-ha fxtunnel-nodered fxtunnel-mqtt
sudo systemctl start fxtunnel-ha fxtunnel-nodered fxtunnel-mqtt
```

## Performance Considerations

Smart home control has specific performance requirements: low latency for light switches and thermostats, moderate bandwidth for dashboard access, and high bandwidth for camera streams. Understanding these requirements helps set realistic expectations for a tunnel-based setup.

### Latency for Real-Time Control

Toggling a light through a tunnel adds one network hop: the request travels from the phone to the fxTunnel relay server, then through the tunnel to Home Assistant. In practice, this adds 20-50 ms of round-trip latency, depending on the geographic distance between the user and the relay server.

For comparison:
- Local network control: 5-15 ms
- Tunnel (same region): 20-50 ms
- Cloud relay (Nabu Casa): 50-150 ms
- VPN through a remote server: 30-100 ms

A 30 ms increase is imperceptible when toggling a light. Even for real-time dashboards that update every second, the added latency is not noticeable.

### Bandwidth for Camera Streams

Camera feeds are the most bandwidth-intensive smart home feature. A typical 1080p camera stream consumes 2-4 Mbps. fxTunnel supports this, but two factors affect the experience:

- **Upload bandwidth at home.** The tunnel uses the home network's upload speed. Most residential internet plans have asymmetric speeds — 100 Mbps download but only 10-20 Mbps upload. One camera stream at 4 Mbps is fine; four simultaneous streams at 4 Mbps each may saturate the upload link.
- **Relay server bandwidth.** fxTunnel's free tier has no traffic limits but shares relay server capacity. For dedicated camera streaming, the Pro plan provides prioritized bandwidth.

**Optimization tips:**
- Use Home Assistant's camera proxy to reduce stream quality for remote viewing.
- Set up motion-triggered snapshots instead of continuous streams.
- Use the MJPEG stream type rather than HLS for lower latency.

### Connection Stability and Reconnection

A smart home tunnel must stay connected 24/7. fxTunnel handles connection drops gracefully:

- **Automatic reconnection.** If the internet connection drops and recovers, fxTunnel reconnects automatically.
- **systemd restart.** If the fxTunnel process crashes, systemd restarts it within 5 seconds (as configured in the service file).
- **Stable URLs.** The public URL does not change after reconnection — bookmarks, voice assistant configurations, and webhook settings remain valid.

For internet connections that are genuinely unstable (frequent outages, flaky Wi-Fi), consider using a wired Ethernet connection for the Home Assistant machine and configuring a shorter `RestartSec` in the systemd service.

### Resource Usage

fxTunnel is a single statically linked Go binary. Its resource footprint is minimal:

- **Memory:** 10-20 MB per tunnel instance
- **CPU:** near zero at idle; negligible during active use
- **Disk:** ~15 MB for the binary

On a Raspberry Pi 4 running Home Assistant, fxTunnel's overhead is undetectable. Even on a Raspberry Pi Zero, it runs comfortably alongside a lightweight Home Assistant installation.

### Multiple Concurrent Connections

fxTunnel handles multiple simultaneous connections through a single tunnel. When several family members access the Home Assistant dashboard at the same time, or when the mobile app maintains a persistent connection while a webhook arrives from Alexa, the tunnel multiplexes all of these over a single outbound TLS connection. There is no need to create separate tunnels for separate users.

Connection limits are not a concern on any tier. The Pro ($5/mo) and Team ($10/mo) plans add custom domains and an Inspector for debugging, but the free tier already handles concurrent connections without artificial caps.

## Comparison: Tunnel vs VPN vs Cloud for Smart Home

Choosing the right remote access method depends on the specific smart home setup, technical skill level, and budget. The following comparison evaluates eight criteria that matter most for Home Assistant users.

| Criteria | Tunnel (fxTunnel) | VPN (WireGuard) | Cloud (Nabu Casa) | Port Forwarding + DDNS |
|---|---|---|---|---|
| **Works behind CGNAT** | Yes | No | Yes | No |
| **Setup time** | 30 seconds | 30-120 minutes | 5 minutes | 15-60 minutes |
| **TLS encryption** | Yes (automatic) | Yes | Yes | Manual (Let's Encrypt) |
| **Webhook support** | Yes | No | Yes | Yes (if port open) |
| **Custom domain** | Yes (Pro $5/mo) | N/A | No | Yes (with DDNS) |
| **Multiple services** | Yes (one tunnel each) | Yes (all on VPN) | HA only | Yes (one port each) |
| **Voice assistant integration** | Yes | No | Yes (native) | Yes (manual) |
| **Cost** | Free / $5/mo / $10/mo | Free (self-hosted) | $7.50/mo | Free |
| **Maintenance** | None | Key management, updates | None | Router config, cert renewal |
| **Open source** | Yes | Yes | No | N/A |
| **Phone battery impact** | None | Significant | None | None |
| **Survives IP changes** | Yes | Requires re-config | Yes | DDNS delay |
| **Works offline (local)** | Yes (local access unaffected) | Yes | No (cloud dependency) | Yes |

**Recommendation by scenario:**

- **Most Home Assistant users:** fxTunnel tunnel. Zero setup, works behind CGNAT, and the free tier covers the basics. Paid plans add custom domains, Inspector, and support for multiple tunnels across locations.
- **Advanced users who want full network access:** WireGuard VPN. Use it if port forwarding works on the network, all devices support VPN clients, and webhook support is not needed.
- **Users who prefer a managed solution:** Nabu Casa. Supports the Home Assistant project financially and provides native Alexa/Google integration. A good fit for non-technical users who only need access to the Home Assistant interface.

The [Tailscale vs fxTunnel vs ZeroTier](/blog/tailscale-vs-fxtunnel-vs-zerotier/) article compares tunneling and VPN approaches in more depth.

## Troubleshooting Common Issues

### Connection Drops and Reconnection Failures

**Symptom:** The tunnel connects initially but drops after a few hours or days.

**Causes and fixes:**

1. **ISP connection reset.** Some ISPs reset long-lived connections. fxTunnel reconnects automatically, but if the reconnection fails:
   ```bash
   # Check tunnel status
   sudo systemctl status fxtunnel-ha

   # Restart the tunnel
   sudo systemctl restart fxtunnel-ha

   # Check logs for errors
   sudo journalctl -u fxtunnel-ha --since "1 hour ago" --no-pager
   ```

2. **Wi-Fi instability.** Use a wired Ethernet connection for the Home Assistant machine. Wi-Fi drops cause tunnel disconnections.

3. **Router firewall.** Some routers block long-lived outbound connections. Try changing the keepalive interval or updating the router firmware.

### WebSocket Connection Failures

**Symptom:** The Home Assistant dashboard loads but shows "Unable to connect to Home Assistant" or the frontend does not update in real time.

**Cause:** Home Assistant uses WebSocket connections for real-time communication. Some network configurations or proxies do not properly support WebSocket upgrades.

**Fix:** Verify that the `http` section in `configuration.yaml` is correctly configured:

```yaml
# configuration.yaml
http:
  use_x_forwarded_for: true
  trusted_proxies:
    - 127.0.0.1
    - ::1
```

If the issue persists, check for a secondary reverse proxy (like Nginx) between fxTunnel and Home Assistant that might strip WebSocket headers.

### SSL Certificate Errors

**Symptom:** Browser shows "Your connection is not private" or "ERR_CERT_AUTHORITY_INVALID."

**Fixes:**

1. **Using the fxTunnel subdomain (e.g., d7f2a.fxtun.dev):** The certificate is managed automatically. Clear the browser cache or try a different browser. If the error persists, restart the tunnel.

2. **Using a custom domain:** Ensure the CNAME record points to `tunnel.fxtun.dev` and has had time to propagate (up to 24 hours for new records). Verify the DNS record:
   ```bash
   dig home.example.com CNAME
   # Expected: home.example.com. CNAME tunnel.fxtun.dev.
   ```

3. **Mixed content errors:** If Home Assistant serves some resources over HTTP while the tunnel uses HTTPS, the browser blocks the HTTP resources. Set the external URL in `configuration.yaml`:
   ```yaml
   homeassistant:
     external_url: "https://home.example.com"
   ```

### Home Assistant Shows "400 Bad Request"

**Symptom:** The tunnel URL returns a 400 error from Home Assistant.

**Cause:** Home Assistant rejects requests from untrusted proxies. The `trusted_proxies` setting is missing or incorrect.

**Fix:**

```yaml
# configuration.yaml
http:
  use_x_forwarded_for: true
  trusted_proxies:
    - 127.0.0.1
    - ::1
```

Restart Home Assistant after editing:

```bash
ha core restart
# or
sudo systemctl restart home-assistant@homeassistant
```

### Mobile App Cannot Connect

**Symptom:** The Home Assistant mobile app (iOS or Android) cannot connect to the tunnel URL.

**Fixes:**

1. **Set the external URL in the app.** Open the app, go to Settings > Companion App > Server Settings. Set the External URL to the tunnel address (e.g., `https://d7f2a.fxtun.dev`). Keep the Internal URL as `http://homeassistant.local:8123`.

2. **Check the Home Assistant configuration:**
   ```yaml
   homeassistant:
     internal_url: "http://homeassistant.local:8123"
     external_url: "https://d7f2a.fxtun.dev"
   ```

3. **Certificate trust.** The mobile app validates TLS certificates strictly. If using a custom domain, ensure the Let's Encrypt certificate is valid and the domain's DNS is correct.

### Tunnel Works Locally but Not Remotely

**Symptom:** The tunnel URL works from a browser on the same machine but returns timeouts when accessed from a phone on a cellular network.

**Causes and fixes:**

1. **Corporate or cellular network firewall.** Some networks block outbound connections to non-standard ports or domains. Try accessing the tunnel from a different network to confirm.

2. **DNS resolution issue.** Clear the DNS cache on the device or try using a different DNS resolver (e.g., `1.1.1.1` or `8.8.8.8`).

3. **fxTunnel server region.** If the relay server is geographically distant, latency can cause timeouts. This is rare but can occur with very slow cellular connections.

### High Latency for Camera Streams

**Symptom:** Camera feeds through the tunnel are choppy, delayed, or buffering frequently.

**Causes and fixes:**

1. **Upload bandwidth saturation.** Check the home network's upload speed with a speed test. A single 1080p camera stream requires 2-4 Mbps of upload. If the upload link is near capacity, reduce the camera's resolution or frame rate in Home Assistant's camera configuration.

2. **Stream type.** Switch from HLS to MJPEG for lower latency, or use WebRTC if the camera supports it. MJPEG has higher bandwidth usage but significantly lower delay.

3. **Multiple camera feeds.** Viewing multiple cameras simultaneously multiplies the bandwidth requirement. Use motion-triggered snapshots for cameras that do not need continuous monitoring.

## FAQ

### Can I access Home Assistant remotely without a static IP?

Yes. Install fxTunnel (`curl -fsSL https://fxtun.ru/install.sh | bash`) and run `fxtunnel http 8123`. You get a public HTTPS URL that forwards to your local instance -- no static IP, no port forwarding, no DNS setup. The tunnel works behind CGNAT, dynamic IPs, and restrictive firewalls because it only needs outbound HTTPS access. fxTunnel is open source at [https://github.com/mephistofox/fxtun.dev](https://github.com/mephistofox/fxtun.dev).

### Is it safe to expose Home Assistant through a tunnel?

With proper configuration, yes. All traffic is encrypted with TLS 1.3 -- nothing travels in plaintext. On the Home Assistant side, every user must log in with a password, and you can enable TOTP-based multi-factor authentication for an extra layer. The `ip_ban_enabled` setting blocks IPs after repeated failed logins. Since no inbound ports are open on your router, the attack surface is smaller than with port forwarding or DDNS. If your smart home controls door locks or an alarm system, MFA and IP banning are essential.

### Will a tunnel add noticeable latency to smart home control?

Toggling a light or reading a sensor adds 20-50 ms of round-trip time through the tunnel, compared to 5-15 ms on the local network. That difference is imperceptible in practice. Camera streams are more bandwidth-intensive, but the bottleneck is typically the home network's upload speed, not the tunnel itself. Real-time dashboards updating once per second show no visible delay.
