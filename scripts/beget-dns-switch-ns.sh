#!/bin/bash
# Switch mfdev.ru DNS delegation to our own NS servers via Beget API.
#
# SAFETY:
#   - Creates a backup of current DNS records before any change
#   - Verifies our NS servers are responding BEFORE switching
#   - Supports rollback via --restore
#
# Usage:
#   export BEGET_LOGIN="your_login"
#   export BEGET_PASSWORD="your_password"
#
#   ./scripts/beget-dns-switch-ns.sh backup mfdev.ru
#     → saves current zone to backups/dns-<domain>-<ts>.json
#
#   ./scripts/beget-dns-switch-ns.sh check 155.212.137.199 159.194.203.103
#     → verifies our NS servers work before switching
#
#   ./scripts/beget-dns-switch-ns.sh switch mfdev.ru ns1.mfdev.ru ns2.mfdev.ru
#     → sets NS records for mfdev.ru to our custom NS (after backup + check)
#
#   ./scripts/beget-dns-switch-ns.sh restore mfdev.ru backups/dns-mfdev.ru-20260414.json
#     → restores from backup
set -euo pipefail

BEGET_LOGIN="${BEGET_LOGIN:-}"
BEGET_PASSWORD="${BEGET_PASSWORD:-}"
API_URL="https://api.beget.com/api"
BACKUP_DIR="$(cd "$(dirname "$0")/.." && pwd)/backups/dns"

if [ -z "$BEGET_LOGIN" ] || [ -z "$BEGET_PASSWORD" ]; then
  echo "ERROR: set BEGET_LOGIN and BEGET_PASSWORD env vars" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"

# --- API helpers ---

beget_call() {
  local method="$1"
  local input_data="$2"
  curl -sf -X POST "$API_URL/$method" \
    --data-urlencode "login=$BEGET_LOGIN" \
    --data-urlencode "passwd=$BEGET_PASSWORD" \
    --data-urlencode "input_format=json" \
    --data-urlencode "output_format=json" \
    --data-urlencode "input_data=$input_data"
}

check_status() {
  local response="$1"
  local status
  status=$(echo "$response" | jq -r '.status')
  if [ "$status" != "success" ]; then
    echo "ERROR: API returned status=$status" >&2
    echo "$response" | jq . >&2
    return 1
  fi
  local answer_status
  answer_status=$(echo "$response" | jq -r '.answer.status')
  if [ "$answer_status" != "success" ]; then
    echo "ERROR: Beget returned error:" >&2
    echo "$response" | jq '.answer' >&2
    return 1
  fi
  return 0
}

# --- Commands ---

cmd_backup() {
  local domain="$1"
  echo "→ Fetching current DNS records for $domain..."
  local response
  response=$(beget_call "dns/getData" "{\"fqdn\":\"$domain\"}")
  check_status "$response"

  local ts
  ts=$(date +%Y%m%d-%H%M%S)
  local file="$BACKUP_DIR/dns-${domain}-${ts}.json"
  echo "$response" | jq '.answer.result' > "$file"
  echo "✓ Backup saved: $file"
  echo ""
  echo "Current NS servers:"
  echo "$response" | jq -r '.answer.result.records.NS[]? | .fqdn // .value'
}

cmd_check() {
  local ns1_ip="$1"
  local ns2_ip="$2"
  local domain="${3:-mfdev.ru}"

  echo "→ Verifying NS servers respond to DNS queries..."
  local fail=0
  for ip in "$ns1_ip" "$ns2_ip"; do
    echo -n "  Testing $ip... "
    if dig +short +time=3 +tries=1 "@$ip" "$domain" SOA >/dev/null 2>&1; then
      if dig +short +time=3 +tries=1 "@$ip" "$domain" SOA | grep -q .; then
        echo "✓ OK"
      else
        echo "✗ FAIL (no SOA returned)"
        fail=1
      fi
    else
      echo "✗ FAIL (no response)"
      fail=1
    fi
  done

  if [ "$fail" -eq 1 ]; then
    echo "" >&2
    echo "✗ One or more NS servers are not responding correctly." >&2
    echo "  Do NOT switch NS until fxtunnel DNS server is working on both." >&2
    exit 1
  fi
  echo ""
  echo "✓ Both NS servers are authoritative for $domain"
}

cmd_switch() {
  local domain="$1"
  local ns1="$2"
  local ns2="$3"

  # Auto-backup first
  echo "→ Backing up current state before switch..."
  cmd_backup "$domain"
  echo ""

  # Fetch current records (we need to preserve A/MX/TXT)
  echo "→ Fetching current records..."
  local response
  response=$(beget_call "dns/getData" "{\"fqdn\":\"$domain\"}")
  check_status "$response"

  local current
  current=$(echo "$response" | jq '.answer.result.records')

  # Build new records: keep everything except NS, replace NS with ours
  local new_records
  new_records=$(echo "$current" | jq --arg ns1 "$ns1." --arg ns2 "$ns2." '
    . as $orig
    | {
        A: ($orig.A // []),
        AAAA: ($orig.AAAA // []),
        MX: ($orig.MX // []),
        TXT: ($orig.TXT // []),
        CAA: ($orig.CAA // []),
        CNAME: ($orig.CNAME // []),
        SRV: ($orig.SRV // []),
        NS: [
          {"value": $ns1},
          {"value": $ns2}
        ]
      }
  ')

  # Show diff
  echo "→ New NS records:"
  echo "  ns1: $ns1."
  echo "  ns2: $ns2."
  echo ""
  echo "Other records preserved:"
  echo "$new_records" | jq -r 'to_entries[] | select(.key != "NS") | "  \(.key): \(.value | length) records"'
  echo ""
  read -p "Proceed with NS change? This will affect the domain globally. [yes/NO]: " confirm
  if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 0
  fi

  # Build changeRecords payload
  local payload
  payload=$(jq -n --arg fqdn "$domain" --argjson records "$new_records" \
    '{fqdn: $fqdn, records: $records}')

  echo "→ Applying NS change..."
  response=$(beget_call "dns/changeRecords" "$payload")
  check_status "$response"

  echo "✓ NS records updated."
  echo ""
  echo "Note: parent-zone NS propagation (for .ru registry) takes up to a few hours."
  echo "Monitor with: dig NS $domain"
}

cmd_restore() {
  local domain="$1"
  local backup_file="$2"

  if [ ! -f "$backup_file" ]; then
    echo "ERROR: backup file not found: $backup_file" >&2
    exit 1
  fi

  echo "→ Restoring DNS records for $domain from $backup_file..."
  local records
  records=$(jq '.records' "$backup_file")

  local payload
  payload=$(jq -n --arg fqdn "$domain" --argjson records "$records" \
    '{fqdn: $fqdn, records: $records}')

  read -p "Restore will OVERWRITE current DNS records. Proceed? [yes/NO]: " confirm
  if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 0
  fi

  local response
  response=$(beget_call "dns/changeRecords" "$payload")
  check_status "$response"

  echo "✓ DNS records restored from backup."
}

# --- Dispatch ---

CMD="${1:-}"
shift || true

case "$CMD" in
  backup)
    if [ $# -lt 1 ]; then echo "Usage: $0 backup <domain>" >&2; exit 1; fi
    cmd_backup "$1"
    ;;
  check)
    if [ $# -lt 2 ]; then echo "Usage: $0 check <ns1_ip> <ns2_ip> [domain]" >&2; exit 1; fi
    cmd_check "$@"
    ;;
  switch)
    if [ $# -lt 3 ]; then echo "Usage: $0 switch <domain> <ns1_fqdn> <ns2_fqdn>" >&2; exit 1; fi
    cmd_switch "$1" "$2" "$3"
    ;;
  restore)
    if [ $# -lt 2 ]; then echo "Usage: $0 restore <domain> <backup_file>" >&2; exit 1; fi
    cmd_restore "$1" "$2"
    ;;
  *)
    cat <<EOF
Usage: $0 <command> [args]

Commands:
  backup <domain>
      Save current DNS records to backups/dns/<domain>-<ts>.json

  check <ns1_ip> <ns2_ip> [domain]
      Verify both NS servers respond with SOA for the domain.
      Run BEFORE switch to avoid breaking DNS.

  switch <domain> <ns1_fqdn> <ns2_fqdn>
      Replace NS records with our custom NS (auto-backup first).
      Example:
        $0 switch mfdev.ru ns1.mfdev.ru ns2.mfdev.ru

  restore <domain> <backup_file>
      Restore DNS records from a backup.
      Example:
        $0 restore mfdev.ru backups/dns/dns-mfdev.ru-20260414-120000.json

Env vars required: BEGET_LOGIN, BEGET_PASSWORD
EOF
    exit 1
    ;;
esac
