---
name: fxtunnel:replay
description: Replay HTTP requests through fxTunnel inspector. Use when you need to re-send a request to test fixes, reproduce bugs, or verify API changes.
---

# fxTunnel Replay

Re-send captured HTTP requests to your local service with optional modifications.

## Basic Replay

Replay an existing request by ID:
```bash
curl -s -X POST http://127.0.0.1:4040/api/requests/http \
  -H 'Content-Type: application/json' \
  -d '{"id":"EXCHANGE_ID"}' | jq
```

## Modified Replay

Change method, path, headers, or body:
```bash
curl -s -X POST http://127.0.0.1:4040/api/requests/http \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "EXCHANGE_ID",
    "method": "PUT",
    "path": "/api/v2/webhook",
    "headers": {"Authorization": "Bearer new-token"},
    "body": "BASE64_ENCODED_BODY"
  }' | jq
```

## Response Format

```json
{
  "status_code": 200,
  "response_headers": {"Content-Type": "application/json"},
  "response_body": "BASE64_ENCODED_RESPONSE",
  "exchange_id": "NEW_EXCHANGE_ID"
}
```

The replayed request creates a new exchange in the inspector with `replay_ref` pointing to the original.

## Workflow

1. Find the request to replay: `curl -s 'http://127.0.0.1:4040/api/requests/http?limit=10' | jq '.requests[] | {id, method, path, status_code}'`
2. Get full details: `curl -s http://127.0.0.1:4040/api/requests/http/{id} | jq`
3. Replay with modifications if needed
4. Compare original and replayed responses
5. Decode base64 response: `echo 'BASE64' | base64 -d | jq`
