#!/usr/bin/env bash
# IndexNow submission script for fxTunnel
# Submits all sitemap URLs to Yandex and Bing IndexNow API
set -euo pipefail

# Domain configs: domain → key
declare -A KEYS=(
  ["fxtun.ru"]="c7679285937613cc3f1f188a761e464d"
  ["fxtun.dev"]="1c290e9b7ace4a0ca39d173a08028ae4"
)

ENGINES=("https://yandex.com/indexnow" "https://api.indexnow.org/indexnow")

# Parse args
DOMAIN="${1:-}"
if [[ -z "$DOMAIN" ]]; then
  echo "Usage: $0 <domain> [url1 url2 ...]"
  echo "  $0 fxtun.ru                  # Submit all sitemap URLs"
  echo "  $0 fxtun.ru /pricing /about  # Submit specific paths"
  exit 1
fi

KEY="${KEYS[$DOMAIN]:-}"
if [[ -z "$KEY" ]]; then
  echo "Error: Unknown domain '$DOMAIN'. Supported: ${!KEYS[*]}"
  exit 1
fi

shift
SPECIFIC_PATHS=("$@")

# Collect URLs
if [[ ${#SPECIFIC_PATHS[@]} -gt 0 ]]; then
  # Specific paths provided
  URL_LIST=()
  for path in "${SPECIFIC_PATHS[@]}"; do
    # Normalize: ensure leading slash, no trailing slash
    path="/${path#/}"
    path="${path%/}"
    [[ "$path" == "/" ]] && path=""
    URL_LIST+=("https://${DOMAIN}${path}")
  done
else
  # Fetch all URLs from sitemap
  echo "Fetching sitemap from https://${DOMAIN}/sitemap.xml..."
  SITEMAP_XML=$(curl -sf "https://${DOMAIN}/sitemap.xml" 2>/dev/null) || {
    echo "Error: Failed to fetch sitemap from https://${DOMAIN}/sitemap.xml"
    exit 1
  }
  # Extract <loc> URLs
  mapfile -t URL_LIST < <(echo "$SITEMAP_XML" | grep -oP '<loc>\K[^<]+')
  if [[ ${#URL_LIST[@]} -eq 0 ]]; then
    echo "Error: No URLs found in sitemap"
    exit 1
  fi
fi

echo "Domain: ${DOMAIN}"
echo "Key: ${KEY}"
echo "URLs to submit (${#URL_LIST[@]}):"
printf "  %s\n" "${URL_LIST[@]}"
echo ""

# Build JSON payload
URL_JSON=$(printf '%s\n' "${URL_LIST[@]}" | jq -R . | jq -s .)
PAYLOAD=$(jq -n \
  --arg host "$DOMAIN" \
  --arg key "$KEY" \
  --arg keyLocation "https://${DOMAIN}/${KEY}.txt" \
  --argjson urlList "$URL_JSON" \
  '{host: $host, key: $key, keyLocation: $keyLocation, urlList: $urlList}')

# Submit to each engine
for ENGINE in "${ENGINES[@]}"; do
  ENGINE_NAME=$(echo "$ENGINE" | grep -oP '//\K[^/]+')
  echo "Submitting to ${ENGINE_NAME}..."

  HTTP_CODE=$(curl -sf -o /dev/null -w "%{http_code}" \
    -X POST "$ENGINE" \
    -H "Content-Type: application/json; charset=utf-8" \
    -d "$PAYLOAD" 2>/dev/null) || HTTP_CODE="000"

  case "$HTTP_CODE" in
    200) echo "  OK (200) — URLs submitted successfully" ;;
    202) echo "  Accepted (202) — URLs queued for processing" ;;
    400) echo "  Bad Request (400) — check payload format"; exit 1 ;;
    403) echo "  Forbidden (403) — key validation failed"; exit 1 ;;
    422) echo "  Unprocessable (422) — URLs don't match host"; exit 1 ;;
    429) echo "  Too Many Requests (429) — rate limited, try later" ;;
    *)   echo "  Unexpected response: HTTP ${HTTP_CODE}" ;;
  esac
done

echo ""
echo "Done! IndexNow submission complete."
