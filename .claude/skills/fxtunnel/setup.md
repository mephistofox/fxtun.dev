---
name: fxtunnel:setup
description: Set up fxTunnel for a project. Use when the user wants to start using fxTunnel tunnels in their development workflow.
---

# fxTunnel Quick Setup

Set up fxTunnel for your project in under a minute.

## Step 1: Install

Check if fxTunnel is installed:
```bash
which fxtunnel && fxtunnel version
```

If not installed, download from the server:
```bash
# Linux amd64
curl -Lo fxtunnel https://mfdev.ru/downloads/fxtunnel-linux-amd64 && chmod +x fxtunnel && sudo mv fxtunnel /usr/local/bin/

# macOS arm64
curl -Lo fxtunnel https://mfdev.ru/downloads/fxtunnel-darwin-arm64 && chmod +x fxtunnel && sudo mv fxtunnel /usr/local/bin/

# macOS amd64
curl -Lo fxtunnel https://mfdev.ru/downloads/fxtunnel-darwin-amd64 && chmod +x fxtunnel && sudo mv fxtunnel /usr/local/bin/
```

## Step 2: Authenticate

```bash
fxtunnel login
```

This opens a browser for authentication, or use:
```bash
fxtunnel login -t sk_your_api_token
```

## Step 3: Create Config

Create `fxtunnel.yaml` in your project root:

```yaml
server:
  address: mfdev.ru:4443

tunnels:
  - name: web
    type: http
    local_port: 3000  # Your local dev server port

inspect:
  enabled: true
```

## Step 4: Start Tunnel

```bash
fxtunnel
```

Or for quick one-off:
```bash
fxtunnel http 3000
```

## Step 5: Verify

Check tunnel status:
```bash
curl -s http://127.0.0.1:4040/api/status | jq
```

List active tunnels:
```bash
curl -s http://127.0.0.1:4040/api/tunnels | jq
```

## Add to .gitignore

The `fxtunnel.yaml` may contain tokens. Add to `.gitignore`:
```
fxtunnel.yaml
```

Or better: use `fxtunnel login` to store token in system keyring and keep config token-free.
