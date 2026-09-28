---
title: "Tunnels for IoT: Connecting Local Devices to Cloud Services"
date: 2026-03-07T14:00:00+03:00
draft: false
description: "Connect Raspberry Pi, ESP32, and IoT devices to the cloud via a tunnel: remote SSH, MQTT broker, web UI, and monitoring — no static IP."
tags: [iot, raspberry-pi, mqtt, tunneling, fxTunnel, esp32, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "How do I get remote access to a Raspberry Pi without a static IP?"
    a: "Install fxTunnel on the Raspberry Pi with curl -fsSL https://fxtun.ru/install.sh | bash, then run fxtunnel tcp 22. You will get a public address like tunnel.fxtun.dev:41234 that you can SSH into from anywhere in the world. No static IP and no port forwarding required."
  - q: "Does fxTunnel support ARM architecture for Raspberry Pi?"
    a: "Yes. fxTunnel is written in Go and cross-compiled for ARM (armv6, armv7, arm64). The install script automatically detects your architecture and downloads the correct binary. It works on every Raspberry Pi model, including Zero and Pi 5."
  - q: "Can I tunnel an MQTT broker?"
    a: "Yes. MQTT runs over TCP, so fxtunnel tcp 1883 creates a tunnel to your local Mosquitto broker. IoT devices on any network can then publish and subscribe through the tunnel's public address."
  - q: "How do I auto-start the tunnel on Raspberry Pi boot?"
    a: "Create a systemd service file at /etc/systemd/system/fxtunnel.service with ExecStart=/usr/local/bin/fxtunnel tcp 22. Then run systemctl enable fxtunnel and systemctl start fxtunnel. The tunnel will start automatically on every boot and restart on failure."
  - q: "Is it safe to expose IoT devices through a tunnel?"
    a: "Yes, when configured properly. fxTunnel encrypts all traffic with TLS. On top of that, use SSH keys instead of passwords, configure a firewall (ufw) on the device, and shut down tunnels to services you are not actively using. Never expose devices with sensitive data without authentication."
---

## The IoT Challenge: Devices Behind NAT With No Public Access

Your Raspberry Pi sits on a shelf at home. Your ESP32 sensors are scattered across a warehouse. A smart home controller runs behind your ISP's CGNAT. None of these devices have a static IP or direct internet access. When you need to SSH into the Pi from the office, push sensor data to the cloud, or show an ESP32 web interface to a colleague, the network blocks you.

This is the fundamental IoT connectivity problem: devices need to be reachable from the outside, but the network infrastructure actively prevents it. [Tunneling](/blog/what-is-tunneling/) solves this -- the device establishes an outbound connection to a public server and receives an address that is reachable from anywhere.

{{< diagram >}}
IoT devices behind NAT (no public IP)
┌──────────────────────────────────────────────┐
│  Home / office network                       │
│                                              │
│  ┌─────────────┐  ┌─────────────┐            │
│  │ Raspberry Pi│  │   ESP32     │            │
│  │ SSH :22     │  │ Web :80     │            │
│  │ Grafana:3000│  │ MQTT :1883  │            │
│  └──────┬──────┘  └──────┬──────┘            │
│         │                │                   │
│         └───────┬────────┘                   │
│                 │                            │
│         ┌───────▼────────┐                   │
│         │  NAT / Router  │                   │
│         │ (no forwarding)│                   │
│         └───────┬────────┘                   │
└─────────────────┼────────────────────────────┘
                  │ outbound connection (TLS)
                  ▼
        ┌──────────────────┐
        │  fxTunnel Server │
        │  tunnel.fxtun.dev│
        └────────┬─────────┘
                 │
                 ▼
        Public addresses:
        ssh  → tunnel.fxtun.dev:41234
        web  → https://rpi.fxtun.dev
        mqtt → tunnel.fxtun.dev:51883
{{< /diagram >}}

## Traditional Solutions and Their Problems

Without a tunnel, you have three classic options: port forwarding on the router, a VPN, or renting a static IP. All three are complex to set up and introduce additional issues. We cover the tradeoffs in more depth in [How to Expose Localhost to the Internet](/blog/expose-localhost-to-internet/).

**Port forwarding on the router** requires admin access to the router, a public IP (not CGNAT), and manual configuration for every device. The port is open to the entire internet without encryption — a direct security threat for IoT devices that often have weak defenses.

**VPN (WireGuard, OpenVPN)** creates an encrypted channel but requires a server with a public IP, per-device setup, and ongoing maintenance. For a single Raspberry Pi, it is overkill. For an ESP32, it is often impossible due to memory and CPU constraints.

**Static IP** costs money, ties you to a specific ISP, and still requires port forwarding and firewall configuration. Many ISPs do not offer static IPs for residential plans at all.

**A tunnel** is one command on the device. No router settings, no VPN server, no static IP. The device makes an outbound TLS connection and [fxTunnel](/blog/ssh-tunnel-vs-modern-tools/) returns a public address.

## How Tunnels Solve IoT Connectivity

The client on your IoT device opens an outbound TLS connection to the fxTunnel server. The server allocates a public address (a URL for HTTP or a host:port for TCP/UDP) and routes incoming traffic through that connection to the device's local port. NAT and firewalls do not interfere because the connection is initiated from inside the network.

What makes fxTunnel particularly useful for IoT is its support for [both TCP and UDP](/blog/tcp-udp-tunneling-explained/). Most IoT protocols (MQTT, CoAP, Modbus TCP) run over TCP, while some (CoAP, mDNS) run over UDP. fxTunnel covers both, unlike ngrok (TCP/HTTP only) or Cloudflare Tunnel (HTTP only on the free tier).

## Scenario 1: Remote Access to Raspberry Pi

Need to SSH into a Raspberry Pi that sits at home while you are at the office? This is probably the most common IoT tunnel use case.

### SSH via TCP Tunnel

```bash
# On the Raspberry Pi: expose the SSH port
fxtunnel tcp 22
# → tunnel.fxtun.dev:41234
```

```bash
# From any computer: connect to the Raspberry Pi
ssh -p 41234 pi@tunnel.fxtun.dev
```

That is it. You now have SSH access to the Raspberry Pi without a static IP, without port forwarding, and without a VPN.

### Web Interface via HTTP Tunnel

If the Raspberry Pi runs a web application (Home Assistant, OctoPrint, Pi-hole), open an HTTP tunnel:

```bash
# Home Assistant web UI on port 8123
fxtunnel http 8123
# → https://rpi-home.fxtun.dev
```

The public HTTPS URL opens in any browser on any device — phone, work laptop, tablet.

## Scenario 2: MQTT Broker via TCP Tunnel

MQTT is the primary protocol for IoT sensors and actuators. The broker (Mosquitto, EMQX) typically runs on a Raspberry Pi or a local server. To let IoT devices on other networks publish and subscribe to messages, the broker needs a public address.

```bash
# On the machine running the MQTT broker
# Mosquitto listens on port 1883 (TCP)
fxtunnel tcp 1883
# → tunnel.fxtun.dev:51883
```

{{< diagram >}}
┌─────────────┐     ┌─────────────────────┐     ┌─────────────┐
│ ESP32 #1    │     │   fxTunnel Server   │     │ Raspberry Pi│
│ (temperature│────▶│ tunnel.fxtun.dev    │◀────│ Mosquitto   │
│  sensor)    │     │ :51883              │     │ :1883       │
└─────────────┘     └─────────────────────┘     └─────────────┘
                            ▲
┌─────────────┐             │
│ ESP32 #2    │─────────────┘
│ (humidity   │
│  sensor)    │
└─────────────┘
{{< /diagram >}}

Now an ESP32 on any network can connect to the broker:

```cpp
// Arduino / ESP32 — connect to MQTT through the tunnel
#include <WiFi.h>
#include <PubSubClient.h>

const char* mqtt_server = "tunnel.fxtun.dev";
const int mqtt_port = 51883;

WiFiClient espClient;
PubSubClient client(espClient);

void setup() {
  Serial.begin(115200);
  WiFi.begin("SSID", "password");
  while (WiFi.status() != WL_CONNECTED) delay(500);

  client.setServer(mqtt_server, mqtt_port);
  while (!client.connected()) {
    client.connect("esp32-sensor-01");
    delay(1000);
  }
}

void loop() {
  float temp = analogRead(34) * 0.1; // example sensor reading
  char payload[16];
  snprintf(payload, sizeof(payload), "%.1f", temp);
  client.publish("sensors/temperature", payload);
  client.loop();
  delay(5000);
}
```

## Scenario 3: ESP32 Web Server

The ESP32 is commonly used with a built-in web server for configuration and monitoring. The problem is that the web interface is only reachable on the local network. A tunnel makes it accessible remotely.

Since the ESP32 cannot run fxTunnel directly (resource constraints), use an intermediate host — a Raspberry Pi or any computer on the same network:

```bash
# On a Raspberry Pi on the same network as the ESP32
# The ESP32 has IP 192.168.1.50 and a web server on port 80
fxtunnel http 192.168.1.50:80
# → https://esp32-web.fxtun.dev
```

The ESP32 web interface is now available at a public HTTPS address. This is useful for remote debugging, device configuration, and demonstrating prototypes.

## Scenario 4: Remote Monitoring Dashboard (Grafana / Node-RED)

Raspberry Pi setups often include dashboards for monitoring IoT data: Grafana (metric visualization), Node-RED (flow automation), or custom web applications. To give your team access, open an HTTP tunnel:

```bash
# Grafana on port 3000
fxtunnel http 3000
# → https://iot-dashboard.fxtun.dev

# Node-RED on port 1880
fxtunnel http 1880
# → https://nodered.fxtun.dev
```

When you have multiple services on one Raspberry Pi, start a separate tunnel for each:

```bash
# Start all tunnels at once
fxtunnel tcp 22 &       # SSH access
fxtunnel http 3000 &    # Grafana
fxtunnel http 1880 &    # Node-RED
fxtunnel tcp 1883 &     # MQTT broker
```

## Setting Up fxTunnel on Raspberry Pi

fxTunnel is written in Go and ships ARM binaries out of the box. The one-line installer works on every Raspberry Pi model, from Zero to Pi 5.

### Installation

```bash
# Install fxTunnel (automatically detects ARM architecture)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installation
fxtunnel --version
```

### Auto-Start With systemd

For IoT devices it is critical that the tunnel starts automatically on boot and restarts on failure. Create a systemd service:

```ini
# /etc/systemd/system/fxtunnel.service
[Unit]
Description=fxTunnel - IoT tunnel service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 22
Restart=always
RestartSec=5
User=pi
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
# Reload systemd configuration
sudo systemctl daemon-reload

# Enable auto-start on boot
sudo systemctl enable fxtunnel

# Start now
sudo systemctl start fxtunnel

# Check status
sudo systemctl status fxtunnel
```

### Multiple Tunnels With systemd

If you need to expose several ports, create a separate service for each:

```ini
# /etc/systemd/system/fxtunnel-ssh.service
[Unit]
Description=fxTunnel SSH tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 22
Restart=always
RestartSec=5
User=pi
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/fxtunnel-mqtt.service
[Unit]
Description=fxTunnel MQTT tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 1883
Restart=always
RestartSec=5
User=pi
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable fxtunnel-ssh fxtunnel-mqtt
sudo systemctl start fxtunnel-ssh fxtunnel-mqtt
```

## Security for IoT Tunnels

IoT devices are frequent attack targets because of weak passwords, outdated firmware, and lack of monitoring. A tunnel encrypts traffic with TLS, but device security is your responsibility.

### Authentication Tokens

Use fxTunnel auth tokens so that only authorized clients can create tunnels:

```bash
# Pass the token at startup
fxtunnel tcp 22 --token=your-secure-token

# Or use an environment variable (recommended for systemd)
export FXTUNNEL_TOKEN=your-secure-token
fxtunnel tcp 22
```

### SSH Keys Instead of Passwords

For SSH access to a Raspberry Pi, always use key-based authentication:

```bash
# On the Raspberry Pi: disable password login
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo systemctl restart sshd
```

### Device Firewall

Configure ufw to restrict port access:

```bash
# Install ufw
sudo apt install ufw

# Allow only SSH and necessary ports
sudo ufw default deny incoming
sudo ufw allow ssh
sudo ufw allow 1883/tcp  # MQTT
sudo ufw enable
```

### Principle of Least Privilege

Only tunnel the ports you actually need. Do not expose every port on the device "just in case." Shut down tunnels to services you are not actively using.

## FAQ

### How do I get remote access to a Raspberry Pi without a static IP?

Run `curl -fsSL https://fxtun.ru/install.sh | bash` on the Pi, then `fxtunnel tcp 22`. You get a public address like `tunnel.fxtun.dev:41234` and can SSH in from anywhere. No static IP, no port forwarding on the router.

### Does fxTunnel support ARM architecture for Raspberry Pi?

It does. The Go binary is cross-compiled for armv6, armv7, and arm64. The install script detects your architecture automatically and downloads the matching binary. Every Pi model works, from Zero to Pi 5.

### Can I tunnel an MQTT broker?

Yes -- MQTT runs over TCP, so `fxtunnel tcp 1883` exposes your local Mosquitto broker. After that, IoT devices on any network can publish and subscribe through the tunnel's public address.

### How do I auto-start the tunnel on Raspberry Pi boot?

Set up a systemd service at `/etc/systemd/system/fxtunnel.service` with `ExecStart=/usr/local/bin/fxtunnel tcp 22`, then enable it with `systemctl enable fxtunnel`. The tunnel will come up on every boot and restart automatically if it crashes.

### Is it safe to expose IoT devices through a tunnel?

With proper precautions, yes. fxTunnel encrypts traffic with TLS, but you should also use SSH keys instead of passwords, set up a firewall (ufw), and close tunnels to services you are not actively using. Never expose a device with real data unless authentication is in place.
