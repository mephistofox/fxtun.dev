---
name: fxtunnel:export
description: Export captured HTTP traffic from fxTunnel inspector. Use when you need to save traffic data, create test fixtures, or share debug information.
---

# fxTunnel Export

Export captured HTTP exchanges for analysis, sharing, or creating test fixtures.

## Export All Exchanges

Save all captured traffic to a JSON file:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?limit=100&include_body=true' | jq > traffic_export.json
```

## Export Filtered Subset

Export only errors:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?status=5xx&include_body=true&limit=100' | jq > errors.json
```

Export specific endpoint:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?path=/api/webhook&include_body=true' | jq > webhook_traffic.json
```

## Export as cURL Commands

Convert exchanges to reproducible cURL commands:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?include_body=true&limit=10' | jq -r '.requests[] | "curl -X \(.method) http://localhost:PORT\(.path) -H \"Content-Type: \(.request_headers[\"Content-Type"] // "application/json")\" -d \"\(.request_body // "" | @base64d)\""'
```

## Export Summary Report

```bash
echo "=== Traffic Summary ===" && \
curl -s http://127.0.0.1:4040/api/requests/http/summary | jq && \
echo "=== Recent Errors ===" && \
curl -s 'http://127.0.0.1:4040/api/requests/http?status=5xx&limit=5' | jq '.requests[] | {method, path, status_code, duration_ms}'
```

## Create Test Fixtures

Extract request/response pairs for test data:
```bash
ID="EXCHANGE_ID"
curl -s "http://127.0.0.1:4040/api/requests/http/$ID" | jq '{
  request: {method, path, host, request_headers, request_body: (.request_body | @base64d | fromjson?)},
  response: {status_code, response_headers, response_body: (.response_body | @base64d | fromjson?)}
}' > fixture.json
```
