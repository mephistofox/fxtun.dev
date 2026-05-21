---
name: fxtunnel
description: Use when the user asks anything about fxTunnel operations - inspecting traffic, debugging, replaying requests, checking status, configuring tunnels, exporting data, or any tunnel-related task. Universal dispatcher for all fxTunnel operations.
---

# fxTunnel Agent

Universal dispatcher for all fxTunnel operations. Analyze the user's request from `$ARGUMENTS` and execute the appropriate action.

## Routing

Match user intent to the correct operation:

| Intent | Action |
|--------|--------|
| Check status, health, "is it running?" | → Status check |
| View/list/show traffic, requests | → Inspect traffic |
| Watch live, monitor, stream | → Watch SSE stream |
| Debug errors, slow requests, investigate | → Debug workflow |
| Replay, re-send, retry request | → Replay request |
| Compare, diff two requests | → Diff exchanges |
| Export, save, download traffic | → Export data |
| Configure, set up, change settings | → Configure client |
| Install, first time, quick start | → Setup guide |
| Security scan, prompt injection, anomalies | → Security scan |
| Harden OpenClaw, secure server | → Secure OpenClaw |

## Execution

### Status Check
```bash
echo "=== Inspector ===" && \
curl -s http://127.0.0.1:4040/api/status | jq '{version, uptime_seconds, total_exchanges}' && \
echo "=== Tunnels ===" && \
curl -s http://127.0.0.1:4040/api/tunnels | jq '.tunnels[] | {name, type, url, local_port}' && \
echo "=== Traffic ===" && \
curl -s http://127.0.0.1:4040/api/requests/http/summary | jq '{total, error_rate, avg_duration_ms}'
```

### Inspect Traffic
Fetch and filter requests. Apply filters from user's request:
- Method: `?method=POST`
- Status: `?status=5xx` or `?status=404`
- Path: `?path=/api/*`
- Search body: `?search=keyword`
- Time window: `?since=5m`
- Include bodies: `?include_body=true`
- Limit: `?limit=N`

```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?limit=20' | jq
```

Detail by ID:
```bash
curl -s http://127.0.0.1:4040/api/requests/http/{id} | jq
```

### Watch Live
```bash
curl -s -N http://127.0.0.1:4040/api/requests/http/stream | \
  grep --line-buffered '^data:' | sed 's/^data: //' | jq --unbuffered '{method, path, status_code, duration_ms}'
```

### Debug Workflow
1. Get summary: `GET /api/requests/http/summary`
2. Find errors: `?status=5xx&limit=10` and `?status=4xx&limit=10`
3. Inspect each error by ID for full exchange
4. Check for slow requests: filter `duration_ms > 1000`
5. Suggest fix, replay to verify

### Replay Request
```bash
curl -s -X POST http://127.0.0.1:4040/api/requests/http \
  -H 'Content-Type: application/json' \
  -d '{"id":"EXCHANGE_ID"}' | jq
```

### Diff Exchanges
Fetch both exchanges, diff request fields, response fields, and bodies.

### Export Data
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?limit=100&include_body=true' | jq > traffic_export.json
```

### Configure / Setup
Refer to config structure in `configs/client.yaml`. Inspector on `127.0.0.1:4040`.

### Security Scan
Invoke skill `fxtunnel:scan` with the user's arguments.

### Secure OpenClaw
Invoke skill `fxtunnel:secure-openclaw` with the user's arguments.

## Rules

1. Always check inspector is reachable first: `curl -s http://127.0.0.1:4040/api/status`
2. If inspector is down, tell user to start fxTunnel client
3. Use `jq` for all JSON output formatting
4. When user intent is ambiguous, ask for clarification
5. For security-related requests, prefer `fxtunnel:scan` or `fxtunnel:secure-openclaw` sub-skills
