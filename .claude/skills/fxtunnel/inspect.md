---
name: fxtunnel:inspect
description: Inspect recent HTTP traffic through fxTunnel tunnels. Use when debugging API requests, checking webhook payloads, or reviewing HTTP traffic patterns.
---

# fxTunnel Inspector

Inspect HTTP traffic flowing through fxTunnel tunnels.

## Quick Start

Fetch recent requests:

```bash
curl -s http://127.0.0.1:4040/api/requests/http?limit=20 | jq
```

## Filtering

Filter by method:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?method=POST&limit=10' | jq
```

Filter by status code range (2xx, 3xx, 4xx, 5xx) or exact code:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?status=5xx' | jq
curl -s 'http://127.0.0.1:4040/api/requests/http?status=404' | jq
```

Filter by path pattern (glob):
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?path=/api/*' | jq
```

Search in request/response bodies:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?search=error' | jq
```

Filter by time window:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?since=5m' | jq
```

Include request/response bodies (base64 encoded):
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?include_body=true&limit=5' | jq
```

## Detail View

Get full details of a specific exchange:
```bash
curl -s http://127.0.0.1:4040/api/requests/http/{id} | jq
```

## Response Format

List response:
```json
{
  "requests": [
    {
      "id": "c-abc123",
      "tunnel_id": "...",
      "method": "POST",
      "path": "/api/webhook",
      "host": "myapp.example.com",
      "status_code": 200,
      "duration_ms": 45,
      "timestamp": "2026-02-12T10:30:00Z",
      "request_body_size": 1024,
      "response_body_size": 256
    }
  ],
  "total": 42
}
```

## Workflow

1. First check if inspector is running: `curl -s http://127.0.0.1:4040/api/status | jq`
2. List recent requests to understand traffic patterns
3. Filter by method/status/path to narrow down
4. Get full details of specific exchange by ID
5. Use `include_body=true` to inspect payloads when needed
