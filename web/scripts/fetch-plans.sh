#!/bin/bash
# Fetch current plans from API and save as static JSON for SSG build.
# Falls back to existing cache if API is unreachable.

API_URL="${PLANS_API_URL:-https://fxtun.ru/api/plans/public}"
OUT_FILE="$(dirname "$0")/../src/data/plans-cache.json"

mkdir -p "$(dirname "$OUT_FILE")"

echo "Fetching plans from $API_URL..."
response=$(curl -sfL --connect-timeout 5 --max-time 10 "$API_URL" 2>/dev/null)

# Only overwrite the cache with something that actually parses. A redirect
# body or an error page is a successful curl as far as the exit code goes,
# and writing it here breaks the type-check for everyone.
if [ -n "$response" ] && RESP="$response" node -e 'if (!Array.isArray(JSON.parse(process.env.RESP).plans)) process.exit(1)' 2>/dev/null; then
  echo "$response" > "$OUT_FILE"
  echo "Plans cached to $OUT_FILE"
else
  if [ -f "$OUT_FILE" ]; then
    echo "API unreachable, using existing cache"
  else
    echo "API unreachable and no cache exists, creating empty fallback"
    echo '{"plans":[]}' > "$OUT_FILE"
  fi
fi
