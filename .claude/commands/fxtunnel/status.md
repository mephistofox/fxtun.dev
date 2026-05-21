---
name: fxtunnel:status
description: Check fxTunnel status, active tunnels, and inspector health. Use when verifying fxTunnel is running and tunnels are active.
---

# fxTunnel Status

Check the status of your fxTunnel client and inspector.

## Inspector Status

```bash
curl -s http://127.0.0.1:4040/api/status | jq
```

Response:
```json
{
  "version": "3.3.0",
  "uptime_seconds": 3600,
  "inspect_enabled": true,
  "total_exchanges": 42
}
```

## Active Tunnels

```bash
curl -s http://127.0.0.1:4040/api/tunnels | jq
```

Response:
```json
{
  "tunnels": [
    {
      "id": "tunnel-abc",
      "name": "web",
      "type": "http",
      "url": "https://myapp.mfdev.ru",
      "local_port": 3000
    }
  ]
}
```

## Traffic Summary

```bash
curl -s http://127.0.0.1:4040/api/requests/http/summary | jq
```

Response:
```json
{
  "total": 42,
  "by_status": {"2xx": 35, "3xx": 2, "4xx": 3, "5xx": 2},
  "by_method": {"GET": 30, "POST": 10, "PUT": 2},
  "error_rate": 0.119,
  "avg_duration_ms": 127,
  "last_request_at": "2026-02-12T10:30:00Z"
}
```

## Quick Health Check

Combined status check:
```bash
echo "=== Inspector ===" && \
curl -s http://127.0.0.1:4040/api/status | jq '{version, uptime_seconds, total_exchanges}' && \
echo "=== Tunnels ===" && \
curl -s http://127.0.0.1:4040/api/tunnels | jq '.tunnels[] | {name, type, url, local_port}' && \
echo "=== Traffic ===" && \
curl -s http://127.0.0.1:4040/api/requests/http/summary | jq '{total, error_rate, avg_duration_ms}'
```

## Troubleshooting

If `curl` fails to connect to `127.0.0.1:4040`:
- Check if fxTunnel client is running
- Inspector may be on a different port (4041-4049 if 4040 was busy)
- Inspector may be disabled (check `--no-inspect` flag or config `inspect.enabled: false`)
- Inspector requires a paid plan — free tier doesn't include inspector
