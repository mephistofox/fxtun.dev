---
name: fxtunnel:diff
description: Compare two HTTP exchanges from fxTunnel inspector. Use when comparing before/after a fix, original vs replay, or two different requests to the same endpoint.
---

# fxTunnel Diff

Compare two captured HTTP exchanges to spot differences.

## Compare Two Exchanges

Fetch both exchanges and diff key fields:
```bash
A=$(curl -s http://127.0.0.1:4040/api/requests/http/EXCHANGE_ID_A)
B=$(curl -s http://127.0.0.1:4040/api/requests/http/EXCHANGE_ID_B)

echo "=== Request Diff ==="
diff <(echo "$A" | jq '{method, path, host, request_headers}') \
     <(echo "$B" | jq '{method, path, host, request_headers}')

echo "=== Response Diff ==="
diff <(echo "$A" | jq '{status_code, response_headers}') \
     <(echo "$B" | jq '{status_code, response_headers}')

echo "=== Body Diff ==="
diff <(echo "$A" | jq -r '.request_body | @base64d' 2>/dev/null) \
     <(echo "$B" | jq -r '.request_body | @base64d' 2>/dev/null)
```

## Compare Original vs Replay

After replaying a request, compare original and replay:
```bash
# Replay
RESULT=$(curl -s -X POST http://127.0.0.1:4040/api/requests/http \
  -H 'Content-Type: application/json' \
  -d '{"id":"ORIGINAL_ID"}')

NEW_ID=$(echo "$RESULT" | jq -r '.exchange_id')

# Get both
ORIG=$(curl -s http://127.0.0.1:4040/api/requests/http/ORIGINAL_ID)
REPLAY=$(curl -s http://127.0.0.1:4040/api/requests/http/$NEW_ID)

# Compare response bodies
diff <(echo "$ORIG" | jq -r '.response_body | @base64d' | jq .) \
     <(echo "$REPLAY" | jq -r '.response_body | @base64d' | jq .)
```

## Workflow

1. Identify two exchanges to compare (by ID)
2. Run diff on request fields to check what changed
3. Run diff on response fields to see impact
4. Focus on body diff for content changes
