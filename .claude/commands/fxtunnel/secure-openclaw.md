---
name: fxtunnel:secure-openclaw
description: Use when hardening an OpenClaw AI agent deployment - binding to localhost, configuring firewall, setting up fxTunnel secure tunnel, Docker isolation, or following the security checklist from the OpenClaw vulnerability report.
---

# Secure OpenClaw with fxTunnel

Step-by-step hardening guide for OpenClaw AI agent deployments. Based on SecurityScorecard STRIKE research (135K+ exposed instances, 3 CVEs, 15,200 without auth).

**Core principle:** Close port 18789, firewall everything, tunnel through fxTunnel.

## Pre-flight Check

Verify current exposure:
```bash
ss -tlnp | grep 18789
# 0.0.0.0:18789 = EXPOSED RIGHT NOW
# 127.0.0.1:18789 = localhost only (good)
```

## Step 1: Bind to localhost

Edit OpenClaw config:
```json
{
  "gateway": {
    "bind": "127.0.0.1",
    "port": 18789,
    "auth": {
      "mode": "token",
      "token": "RANDOM-TOKEN-MIN-32-CHARS"
    }
  }
}
```

Generate a strong token:
```bash
openssl rand -hex 32
```

Restart OpenClaw. Verify:
```bash
ss -tlnp | grep 18789
# Must show 127.0.0.1:18789, NOT 0.0.0.0
```

## Step 2: Firewall

### Ubuntu / Debian (UFW)
```bash
sudo apt update && sudo apt install ufw -y
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw limit 22/tcp
sudo ufw deny 18789/tcp
sudo ufw enable
```

### CentOS / RHEL / Fedora (firewalld)
```bash
sudo dnf install firewalld -y
sudo systemctl enable --now firewalld
sudo firewall-cmd --zone=public --add-service=ssh --permanent
sudo firewall-cmd --zone=public \
  --add-rich-rule='rule port port="18789" protocol="tcp" reject' \
  --permanent
sudo firewall-cmd --reload
```

### iptables (any distro)
```bash
sudo iptables -A INPUT -i lo -p tcp --dport 18789 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 18789 -j DROP
sudo iptables-save > /etc/iptables/rules.v4
```

Verify port is blocked from outside:
```bash
# From another machine:
nmap -p 18789 <server-ip>
# Should show: filtered or closed
```

## Step 3: fxTunnel — Secure Remote Access

Install:
```bash
curl -fsSL https://fxtun.dev/install.sh | sh
```

Authenticate:
```bash
fxtunnel auth login
```

Create tunnel:
```bash
fxtunnel http 18789 --subdomain my-ai-agent
# OpenClaw accessible at: https://my-ai-agent.fxtun.dev
# Port 18789 stays CLOSED — all traffic through encrypted tunnel
```

### Systemd auto-start
```bash
sudo tee /etc/systemd/system/fxtunnel-openclaw.service << 'EOF'
[Unit]
Description=fxTunnel for OpenClaw
After=network-online.target

[Service]
Type=simple
User=openclaw
ExecStart=/usr/local/bin/fxtunnel http 18789 --subdomain my-ai-agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now fxtunnel-openclaw
```

## Step 4: Docker Isolation

```yaml
version: '3.8'
services:
  openclaw:
    image: openclaw/agent:latest
    security_opt:
      - no-new-privileges:true
    read_only: true
    user: "1000:1000"
    cap_drop:
      - ALL
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=64M
    ports:
      - "127.0.0.1:18789:18789"  # localhost ONLY
    restart: unless-stopped
    # NEVER mount Docker socket
```

## Step 5: Hardening

```bash
# Dedicated user
useradd -r -m -d /home/openclaw -s /bin/bash openclaw
chmod 700 /home/openclaw/.openclaw
chmod 600 /home/openclaw/.openclaw/openclaw.json

# Fail2Ban + auto-updates
sudo apt install fail2ban unattended-upgrades -y
sudo systemctl enable fail2ban

# Rotate ALL API keys after securing
```

## Verification Checklist

Run each check — ALL must pass:

```bash
echo "1. Bind check:" && ss -tlnp | grep 18789

echo "2. Firewall active:" && sudo ufw status | head -3

echo "3. Port blocked externally:" && \
  curl -s --connect-timeout 3 http://$(curl -s ifconfig.me):18789 && \
  echo "FAIL: port reachable!" || echo "OK: port blocked"

echo "4. fxTunnel running:" && \
  curl -s http://127.0.0.1:4040/api/status | jq '{version, uptime_seconds}'

echo "5. Tunnel active:" && \
  curl -s http://127.0.0.1:4040/api/tunnels | jq '.tunnels[] | {name, url}'

echo "6. OpenClaw auth enabled:" && \
  curl -s http://127.0.0.1:18789 -o /dev/null -w '%{http_code}' && echo ""
```

## 10-Minute Checklist

| # | Action | Command |
|---|--------|---------|
| 1 | Bind = 127.0.0.1 | Edit `openclaw.json` |
| 2 | Auth token set | `openssl rand -hex 32` |
| 3 | Firewall deny 18789 | `ufw deny 18789/tcp && ufw enable` |
| 4 | Install fxTunnel | `curl -fsSL https://fxtun.dev/install.sh \| sh` |
| 5 | Create tunnel | `fxtunnel http 18789 --subdomain <name>` |
| 6 | IP allowlist | Configure in fxTun.dev panel |
| 7 | Docker hardening | `cap_drop: ALL`, `no-new-privileges` |
| 8 | Update OpenClaw | Version 2026.2.1+ |
| 9 | Rotate all keys | API keys, tokens, SSH keys |
| 10 | Verify | `ss -tlnp \| grep 18789` → 127.0.0.1 only |

## CVE Reference

- **CVE-2026-25253** (CVSS 8.8) — 1-click RCE via `gatewayUrl` parameter, token leak via CSWSH
- **CVE-2026-25157** (CVSS 7.8) — Command injection via SSH in macOS
- **CVE-2026-24763** (CVSS 8.8) — Docker sandbox escape via PATH manipulation

All three have public exploit code. Update immediately.
