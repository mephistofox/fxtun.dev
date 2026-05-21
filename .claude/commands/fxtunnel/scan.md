---
name: fxtunnel:scan
description: Use when analyzing HTTP traffic for security threats - prompt injection attacks, data exfiltration attempts, anomalous request patterns, suspicious WebSocket frames, or when auditing AI agent traffic safety.
---

# fxTunnel Security Scanner

AI-powered analysis of HTTP traffic through fxTunnel inspector to detect prompt injection, data exfiltration, and anomalous patterns targeting AI agents (OpenClaw, etc.).

## Step 1: Collect Traffic

Fetch recent traffic with full bodies:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?include_body=true&limit=50' | jq > /tmp/fxtunnel-scan-data.json
```

If user specified a time window or filter, apply it:
```bash
curl -s 'http://127.0.0.1:4040/api/requests/http?include_body=true&since=1h&limit=100' | jq > /tmp/fxtunnel-scan-data.json
```

## Step 2: Analyze for Threats

Read the exported file and analyze each exchange for these threat categories:

### A. Prompt Injection Detection

Search request bodies for patterns:
- Direct injection: `ignore previous`, `disregard instructions`, `you are now`, `system prompt`, `new instructions`
- Role hijacking: `pretend you are`, `act as`, `you are a`, `switch to`, `override`
- Delimiter attacks: `###`, `---`, `<|endoftext|>`, `[INST]`, `<<SYS>>`, `</s>`
- Encoded payloads: base64-encoded strings containing injection patterns
- Markdown/HTML injection: script tags, event handlers in user-controlled fields
- Indirect injection: URLs pointing to attacker-controlled pages with injected content

```bash
# Quick grep for common injection patterns
curl -s 'http://127.0.0.1:4040/api/requests/http?search=ignore+previous&include_body=true' | jq '.requests[] | {id, method, path, timestamp}'
curl -s 'http://127.0.0.1:4040/api/requests/http?search=system+prompt&include_body=true' | jq '.requests[] | {id, method, path, timestamp}'
```

### B. Data Exfiltration Detection

Check response bodies for:
- API keys / tokens: patterns like `sk-`, `sk_`, `Bearer `, `api_key`, `AKIA` (AWS)
- Private keys: `-----BEGIN`, `PRIVATE KEY`
- Credentials: `password`, `secret`, `credential` in response bodies
- Large response bodies: unusually big responses that may contain extracted data
- Base64-encoded blobs in responses (hidden data extraction)
- File paths: `/etc/passwd`, `/home/`, `.ssh/`, `.env`, `id_rsa`

```bash
# Check for potential key leaks in responses
curl -s 'http://127.0.0.1:4040/api/requests/http?search=sk-&include_body=true' | jq '.requests[] | {id, path, status_code, response_body_size}'
curl -s 'http://127.0.0.1:4040/api/requests/http?search=BEGIN+PRIVATE&include_body=true' | jq '.requests[] | {id, path, status_code}'
```

### C. Anomalous Request Patterns

Flag:
- Unusual HTTP methods: `DELETE`, `PATCH`, `OPTIONS` to unexpected endpoints
- Rapid request bursts: many requests from same origin in short time
- Path traversal: `../`, `..%2F`, `%2e%2e` in paths
- Command injection in headers: `; curl`, `| wget`, backticks in User-Agent or other headers
- Unexpected content types: `multipart/form-data` to JSON API endpoints
- WebSocket upgrade requests to unexpected paths

```bash
# Check for path traversal
curl -s 'http://127.0.0.1:4040/api/requests/http?search=..%2F&include_body=true' | jq '.requests[] | {id, method, path}'

# Large responses (potential data dump)
curl -s 'http://127.0.0.1:4040/api/requests/http?limit=100' | jq '[.requests[] | select(.response_body_size > 50000)] | sort_by(-.response_body_size) | .[] | {id, path, response_body_size, status_code}'
```

### D. WebSocket Frame Analysis

If WebSocket traffic is captured:
- Unusually large frames (data exfiltration)
- Rapid frame bursts (command flooding)
- Frames containing shell commands or code execution patterns
- Frames with encoded payloads that decode to injection attempts

## Step 3: Generate Report

For each finding, output:

```
## Security Scan Report

### Summary
- Scanned: N exchanges
- Time range: [first] — [last]
- Threats found: N

### Findings

#### [CRITICAL/HIGH/MEDIUM/LOW] — Category
- Exchange ID: {id}
- Timestamp: {timestamp}
- Method: {method} Path: {path}
- Evidence: [specific pattern found]
- Recommendation: [what to do]
```

### Severity Levels

| Severity | Criteria |
|----------|----------|
| CRITICAL | Active prompt injection with success indicators, confirmed key leak |
| HIGH | Prompt injection attempt, path traversal, potential exfiltration |
| MEDIUM | Anomalous patterns, unusual methods, suspicious payloads |
| LOW | Minor anomalies, information disclosure |

## Step 4: Recommendations

Based on findings, suggest:
- Specific IP blocks (if attacks from identifiable sources)
- Endpoint restrictions (block sensitive paths at tunnel level)
- Auth enforcement (add `--auth` to tunnel)
- Rate limiting
- OpenClaw configuration changes (disable tools, restrict access)
- Key rotation if any leak detected

## Quick Scan (One-liner)

For a fast assessment:
```bash
echo "=== Prompt Injection ===" && \
curl -s 'http://127.0.0.1:4040/api/requests/http?search=ignore+previous&limit=5' | jq '.total' && \
curl -s 'http://127.0.0.1:4040/api/requests/http?search=system+prompt&limit=5' | jq '.total' && \
echo "=== Key Leaks ===" && \
curl -s 'http://127.0.0.1:4040/api/requests/http?search=sk-&limit=5' | jq '.total' && \
curl -s 'http://127.0.0.1:4040/api/requests/http?search=PRIVATE+KEY&limit=5' | jq '.total' && \
echo "=== Path Traversal ===" && \
curl -s 'http://127.0.0.1:4040/api/requests/http?search=..%2F&limit=5' | jq '.total' && \
echo "=== Error Spike ===" && \
curl -s 'http://127.0.0.1:4040/api/requests/http/summary' | jq '{error_rate, total, by_status}'
```
