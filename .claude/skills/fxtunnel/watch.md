---
name: fxtunnel:watch
description: Watch live HTTP traffic through fxTunnel in real-time using SSE stream. Use when monitoring ongoing requests or waiting for specific events.
---

# fxTunnel Watch

Monitor HTTP traffic in real-time using Server-Sent Events.

## Live Stream

Watch all incoming exchanges:
```bash
curl -s -N http://127.0.0.1:4040/api/requests/http/stream
```

Events arrive as:
```
event: exchange
data: {"id":"c-abc123","method":"POST","path":"/api/webhook","status_code":200,"duration_ns":45000000}
```

## Filtered Watching with jq

Watch only errors:
```bash
curl -s -N http://127.0.0.1:4040/api/requests/http/stream | \
  grep --line-buffered '^data:' | sed 's/^data: //' | \
  jq --unbuffered 'select(.status_code >= 400) | {method, path, status_code}'
```

Watch specific path:
```bash
curl -s -N http://127.0.0.1:4040/api/requests/http/stream | \
  grep --line-buffered '^data:' | sed 's/^data: //' | \
  jq --unbuffered 'select(.path | startswith("/api/")) | {method, path, status_code}'
```

## Workflow

1. Start watching in background
2. Trigger the action you want to observe
3. Review captured exchanges
4. Use exchange IDs to get full details: `curl -s http://127.0.0.1:4040/api/requests/http/{id} | jq`

## Notes

- SSE stream sends `: ping` every 30 seconds to keep connection alive
- Stream includes all new exchanges across all tunnels
- Stop with Ctrl+C
