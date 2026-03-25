#!/bin/bash
# Fetch current plans from API and save as static JSON for SSG build.
# Falls back to existing cache if API is unreachable.

API_URL="${PLANS_API_URL:-https://fxtun.dev/api/plans/public}"
OUT_FILE="$(dirname "$0")/../src/data/plans-cache.json"

mkdir -p "$(dirname "$OUT_FILE")"

echo "Fetching plans from $API_URL..."
response=$(curl -sf --connect-timeout 5 --max-time 10 "$API_URL" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$response" ]; then
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
