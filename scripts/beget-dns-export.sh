#!/bin/bash
# Export DNS records from Beget API as YAML config for fxtunnel DNS server.
# Docs: https://beget.com/ru/kb/api/funkczii-upravleniya-dns
#
# Usage:
#   Fill in BEGET_LOGIN / BEGET_PASSWORD below, then:
#   ./scripts/beget-dns-export.sh mfdev.ru > configs/dns-mfdev.yaml
set -euo pipefail

# Beget API credentials — fill in before running
BEGET_LOGIN="${BEGET_LOGIN:-}"
BEGET_PASSWORD="${BEGET_PASSWORD:-}"

if [ -z "$BEGET_LOGIN" ] || [ -z "$BEGET_PASSWORD" ]; then
  echo "ERROR: set BEGET_LOGIN and BEGET_PASSWORD env vars or edit the script" >&2
  exit 1
fi

DOMAIN="${1:-}"
if [ -z "$DOMAIN" ]; then
  echo "Usage: $0 <domain>" >&2
  exit 1
fi

API_URL="https://api.beget.com/api/dns/getData"

# Call API
RESPONSE=$(curl -sf -X POST "$API_URL" \
  --data-urlencode "login=$BEGET_LOGIN" \
  --data-urlencode "passwd=$BEGET_PASSWORD" \
  --data-urlencode "input_format=json" \
  --data-urlencode "output_format=json" \
  --data-urlencode "input_data={\"fqdn\":\"$DOMAIN\"}")

# Check status
STATUS=$(echo "$RESPONSE" | jq -r '.status')
if [ "$STATUS" != "success" ]; then
  echo "ERROR: API call failed:" >&2
  echo "$RESPONSE" | jq . >&2
  exit 1
fi

ANSWER_STATUS=$(echo "$RESPONSE" | jq -r '.answer.status')
if [ "$ANSWER_STATUS" != "success" ]; then
  echo "ERROR: Beget returned error:" >&2
  echo "$RESPONSE" | jq '.answer' >&2
  exit 1
fi

# Output raw JSON for debugging (stderr)
echo "# Raw response:" >&2
echo "$RESPONSE" | jq '.answer.result' >&2
echo "" >&2

# Generate YAML config
cat <<YAML
# DNS zone config for $DOMAIN — exported from Beget on $(date -Iseconds)
# Add dynamic tunnel subdomains via Redis tunnel registry (handled at runtime)
zones:
  - name: "$DOMAIN"
    tunnels_enabled: true
    ttl: 300
    records:
YAML

RECORDS=$(echo "$RESPONSE" | jq -r '.answer.result.records')

# Process each record type
for TYPE in A AAAA CNAME MX TXT NS CAA SRV; do
  COUNT=$(echo "$RECORDS" | jq -r ".$TYPE | length // 0")
  if [ "$COUNT" = "0" ] || [ "$COUNT" = "null" ]; then continue; fi

  echo "$RECORDS" | jq -r --arg type "$TYPE" '
    .[$type][] |
    if $type == "MX" then
      "      - {name: \"" + (.fqdn // "@") + "\", type: \"MX\", value: \"" + (.exchange // .value | tostring) + "\", priority: " + ((.preference // .priority // 10) | tostring) + "}"
    elif $type == "SRV" then
      "      - {name: \"" + (.fqdn // "@") + "\", type: \"SRV\", value: \"" + (.target // "" | tostring) + "\", priority: " + ((.priority // 0) | tostring) + ", weight: " + ((.weight // 0) | tostring) + ", port: " + ((.port // 0) | tostring) + "}"
    elif $type == "TXT" then
      "      - {name: \"" + (.fqdn // "@") + "\", type: \"TXT\", value: " + ((.txtdata // .value // .data // "" | tostring) | @json) + "}"
    elif $type == "CAA" then
      "      - {name: \"" + (.fqdn // "@") + "\", type: \"CAA\", value: \"" + ((.flags // 0) | tostring) + " " + (.tag // "issue") + " " + ((.value // "" | tostring) | @json) + "\"}"
    else
      "      - {name: \"" + (.fqdn // "@") + "\", type: \"" + $type + "\", value: \"" + ((.address // .value // .target // .data // "" | tostring)) + "\"}"
    end
  ' 2>/dev/null | sed "s/$DOMAIN\\.*\"/@\"/; s/\\.$DOMAIN\\.*/\"/"
done

echo ""
echo "# Total records exported: $(echo "$RECORDS" | jq '[.[] // [] | length] | add // 0')"
