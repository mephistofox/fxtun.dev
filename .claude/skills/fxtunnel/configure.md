---
name: fxtunnel:configure
description: Configure fxTunnel client settings. Use when setting up tunnels, adjusting inspector settings, or managing authentication.
---

# fxTunnel Configuration

Configure the fxTunnel client via YAML config file or CLI flags.

## Config File Location

fxTunnel looks for config in this order:
1. `fxtunnel.yaml` in current directory
2. `client.yaml` in current directory
3. `configs/client.yaml`
4. `~/.fxtunnel/client.yaml`

## Example Config

```yaml
server:
  address: mfdev.ru:4443
  token: sk_your_token_here
  compression: true

tunnels:
  - name: web
    type: http
    local_port: 3000
    subdomain: myapp

  - name: api
    type: http
    local_port: 8080

  - name: ssh
    type: tcp
    local_port: 22

inspect:
  enabled: true
  addr: "127.0.0.1:4040"
  max_body_size: 262144  # 256KB
  max_entries: 1000

reconnect:
  enabled: true
  interval: 5s
  max_attempts: 0  # 0 = infinite

logging:
  level: info
  format: console
```

## Inspector Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `inspect.enabled` | `true` | Enable/disable local inspector |
| `inspect.addr` | `127.0.0.1:4040` | Inspector listen address |
| `inspect.max_body_size` | `262144` (256KB) | Max body size to capture |
| `inspect.max_entries` | `1000` | Max entries in ring buffer |

## CLI Flags

```bash
fxtunnel http 3000                        # Quick HTTP tunnel
fxtunnel http 3000 --domain myapp         # With custom subdomain
fxtunnel tcp 22 --remote-port 2222        # TCP tunnel with specific port
fxtunnel --config custom.yaml             # Use specific config
fxtunnel http 3000 --no-inspect           # Disable inspector
fxtunnel http 3000 --inspect-addr 0.0.0.0:5050  # Custom inspector addr
```

## Environment Variables

All config can be overridden via `FXTUNNEL_` prefix:
```bash
FXTUNNEL_SERVER_ADDRESS=mfdev.ru:4443
FXTUNNEL_SERVER_TOKEN=sk_xxx
FXTUNNEL_INSPECT_ENABLED=true
FXTUNNEL_INSPECT_ADDR=127.0.0.1:4040
```
