---
name: fxtunnel:debug
description: Systematic debugging workflow using fxTunnel inspector. Use when investigating API errors, slow requests, or unexpected behavior in tunneled services.
---

# fxTunnel Debug Workflow

Systematic approach to debugging HTTP issues using the fxTunnel inspector.

## Step 1: Assess the Situation

Get traffic summary:
```bash
curl -s http://127.0.0.1:4040/api/requests/http/summary | jq
```

This tells you:
- `total` — how many requests captured
- `by_status` — distribution of 2xx/3xx/4xx/5xx
- `error_rate` — percentage of 4xx+5xx
- `avg_duration_ms` — average response time

## Step 2: Find Errors

Get recent errors:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?status=5xx&limit=10' | jq '.requests[] | {id, method, path, status_code, duration_ms}'
```

Get client errors:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?status=4xx&limit=10' | jq '.requests[] | {id, method, path, status_code, duration_ms}'
```

## Step 3: Inspect Error Details

For each error, get full exchange:
```bash
curl -s http://127.0.0.1:4040/api/requests/http/{id} | jq
```

Check:
- Request headers (auth, content-type)
- Request body (malformed JSON, missing fields)
- Response body (error messages, stack traces)
- Duration (timeout issues)

## Step 4: Find Slow Requests

List all requests sorted by time and check durations:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?limit=50' | jq '[.requests[] | select(.duration_ms > 1000)] | sort_by(-.duration_ms) | .[] | {id, method, path, duration_ms}'
```

## Step 5: Replay and Verify Fix

After fixing the issue, replay the failing request:
```bash
curl -s -X POST http://127.0.0.1:4040/api/requests/http \
  -H 'Content-Type: application/json' \
  -d '{"id":"FAILING_EXCHANGE_ID"}' | jq
```

## Step 6: Clear and Re-test

Clear buffer for a clean slate:
```bash
curl -s -X DELETE http://127.0.0.1:4040/api/requests/http
```

Then trigger the flow again and check results.

## Common Patterns

**Auth issues:** Filter for 401/403 and check Authorization headers
**Payload issues:** Get exchange with full body and validate JSON structure
**Timeout issues:** Check duration_ms for requests near your timeout limit
**CORS issues:** Look for OPTIONS preflight requests and check response headers
