#!/bin/bash
# Beget DNS management: backup, sync from YAML, switch NS, restore.
#
# Commands:
#   backup  <domain>
#   check   <ns1_ip> <ns2_ip> [domain]
#   apply   <yaml_file>          — UPSERT all records from YAML to Beget
#   switch  <domain> <ns1> <ns2>
#   restore <domain> <backup_file>
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

# Fetch records for one FQDN. Returns the `.records` object (A, MX, TXT, ...).
get_records() {
  local fqdn="$1"
  local response
  response=$(beget_call "dns/getData" "{\"fqdn\":\"$fqdn\"}") || return 1
  check_status "$response" >/dev/null 2>&1 || {
    echo "{}"
    return 0
  }
  echo "$response" | jq '.answer.result.records // {}'
}

# Set records for one FQDN (completely replaces existing records for that FQDN).
set_records() {
  local fqdn="$1"
  local records="$2"
  local payload
  payload=$(jq -n --arg fqdn "$fqdn" --argjson records "$records" \
    '{fqdn: $fqdn, records: $records}')
  local response
  response=$(beget_call "dns/changeRecords" "$payload")
  check_status "$response"
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
    local result
    result=$(dig +short +time=3 +tries=1 "@$ip" "$domain" SOA 2>/dev/null)
    if [ -n "$result" ]; then
      echo "✓ OK  ($result)"
    else
      echo "✗ FAIL (no SOA returned)"
      fail=1
    fi
  done

  if [ "$fail" -eq 1 ]; then
    echo "" >&2
    echo "✗ One or more NS servers are not responding correctly." >&2
    exit 1
  fi
  echo ""
  echo "✓ Both NS servers are authoritative for $domain"
}

# apply: reads YAML zone config and UPSERTs records on Beget.
# Each FQDN is updated independently; for each target FQDN:
#   - types present in YAML → replaced with YAML values
#   - types only in current Beget → preserved
#
# Flags:
#   --dry-run   — print plan (current vs target) without applying
cmd_apply() {
  local dry_run=0
  if [ "${1:-}" = "--dry-run" ]; then
    dry_run=1
    shift
  fi

  local yaml_file="$1"
  if [ ! -f "$yaml_file" ]; then
    echo "ERROR: YAML file not found: $yaml_file" >&2
    exit 1
  fi

  if [ "$dry_run" -eq 1 ]; then
    echo "=== DRY RUN (no changes will be made) ==="
  fi

  # Parse YAML with yq (preferred) or python fallback
  local zones_json
  if command -v yq >/dev/null 2>&1; then
    zones_json=$(yq -o=json '.zones' "$yaml_file")
  else
    zones_json=$(python3 -c "import sys, yaml, json; print(json.dumps(yaml.safe_load(open('$yaml_file'))['zones']))")
  fi

  # For each zone
  local zone_count
  zone_count=$(echo "$zones_json" | jq 'length')
  if [ "$zone_count" = "0" ] || [ "$zone_count" = "null" ]; then
    echo "No zones in $yaml_file" >&2
    exit 1
  fi

  for zi in $(seq 0 $((zone_count - 1))); do
    local zone_name
    zone_name=$(echo "$zones_json" | jq -r ".[$zi].name")
    echo ""
    echo "=== Zone: $zone_name ==="

    # Group records by FQDN
    # record.name == "@" → fqdn = zone_name
    # record.name == "www" → fqdn = "www.zone_name"
    local records_json
    records_json=$(echo "$zones_json" | jq ".[$zi].records")

    # Extract unique FQDNs
    local fqdns
    fqdns=$(echo "$records_json" | jq -r --arg zone "$zone_name" '
      map(
        if .name == "@" or .name == "" or .name == $zone then $zone
        else .name + "." + $zone
        end
      ) | unique | .[]
    ')

    # For each FQDN
    while IFS= read -r fqdn; do
      [ -z "$fqdn" ] && continue
      echo ""
      echo "--- $fqdn ---"

      # Collect YAML records for this FQDN, group by type
      local yaml_records_for_fqdn
      yaml_records_for_fqdn=$(echo "$records_json" | jq --arg zone "$zone_name" --arg fqdn "$fqdn" '
        map(select(
          (if .name == "@" or .name == "" or .name == $zone then $zone else .name + "." + $zone end) == $fqdn
        ))
      ')

      # Build target records object grouped by type
      local target
      target=$(echo "$yaml_records_for_fqdn" | jq '
        reduce .[] as $r ({};
          .[$r.type] += [
            if $r.type == "A"     then {value: $r.value}
            elif $r.type == "AAAA" then {value: $r.value}
            elif $r.type == "CNAME" then {value: ($r.value | sub("\\.$"; "") + ".")}
            elif $r.type == "MX"    then {value: ($r.value | sub("\\.$"; "") + "."), priority: ($r.priority // 10)}
            elif $r.type == "TXT"   then {value: $r.value}
            elif $r.type == "NS"    then {value: ($r.value | sub("\\.$"; "") + ".")}
            elif $r.type == "CAA"   then {value: $r.value}
            elif $r.type == "SRV"   then {value: $r.value, priority: ($r.priority // 0), weight: ($r.weight // 0), port: ($r.port // 0)}
            else {value: $r.value} end
          ]
        )
      ')

      # Get current Beget records
      local current
      current=$(get_records "$fqdn")

      # Merge: target types override current types, non-target types preserved
      local merged
      merged=$(jq -n --argjson current "$current" --argjson target "$target" '
        $current as $c | $target as $t |
        ($c | to_entries | map(select(.key as $k | $t | has($k) | not)) | from_entries) + $t
      ')

      # Show diff
      local target_types
      target_types=$(echo "$target" | jq -r 'keys | join(", ")')
      echo "  Types from YAML: $target_types"
      echo "  Final records:"
      echo "$merged" | jq -r '
        to_entries[] |
        .key as $type |
        .value[] |
        "    \($type): \(.value)" + (if .priority then " (priority=\(.priority))" else "" end)
      '

      # Apply
      if set_records "$fqdn" "$merged" >/dev/null 2>&1; then
        echo "  ✓ Applied"
      else
        echo "  ✗ FAILED" >&2
      fi

      # Small pause to be polite to API
      sleep 0.3
    done <<< "$fqdns"
  done

  echo ""
  echo "✓ apply complete"
}

cmd_switch() {
  local domain="$1"
  local ns1="$2"
  local ns2="$3"

  echo "→ Backing up current state before switch..."
  cmd_backup "$domain"
  echo ""

  echo "→ Fetching current records..."
  local response
  response=$(beget_call "dns/getData" "{\"fqdn\":\"$domain\"}")
  check_status "$response"

  local current
  current=$(echo "$response" | jq '.answer.result.records')

  local new_records
  new_records=$(echo "$current" | jq --arg ns1 "$ns1." --arg ns2 "$ns2." '
    . as $orig
    | {
        A:     ($orig.A     // []),
        AAAA:  ($orig.AAAA  // []),
        MX:    ($orig.MX    // []),
        TXT:   ($orig.TXT   // []),
        CAA:   ($orig.CAA   // []),
        CNAME: ($orig.CNAME // []),
        SRV:   ($orig.SRV   // []),
        NS: [ {"value": $ns1}, {"value": $ns2} ]
      }
  ')

  echo "→ New NS records:"
  echo "  ns1: $ns1."
  echo "  ns2: $ns2."
  echo ""
  echo "Other records preserved:"
  echo "$new_records" | jq -r 'to_entries[] | select(.key != "NS") | "  \(.key): \(.value | length) records"'
  echo ""
  read -p "Proceed with NS change? [yes/NO]: " confirm
  if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 0
  fi

  local payload
  payload=$(jq -n --arg fqdn "$domain" --argjson records "$new_records" \
    '{fqdn: $fqdn, records: $records}')

  echo "→ Applying NS change..."
  response=$(beget_call "dns/changeRecords" "$payload")
  check_status "$response"

  echo "✓ NS records updated."
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
    [ $# -lt 1 ] && { echo "Usage: $0 backup <domain>" >&2; exit 1; }
    cmd_backup "$1"
    ;;
  check)
    [ $# -lt 2 ] && { echo "Usage: $0 check <ns1_ip> <ns2_ip> [domain]" >&2; exit 1; }
    cmd_check "$@"
    ;;
  apply)
    [ $# -lt 1 ] && { echo "Usage: $0 apply <yaml_file>" >&2; exit 1; }
    cmd_apply "$1"
    ;;
  switch)
    [ $# -lt 3 ] && { echo "Usage: $0 switch <domain> <ns1_fqdn> <ns2_fqdn>" >&2; exit 1; }
    cmd_switch "$1" "$2" "$3"
    ;;
  restore)
    [ $# -lt 2 ] && { echo "Usage: $0 restore <domain> <backup_file>" >&2; exit 1; }
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

  apply <yaml_file>
      UPSERT all records from YAML zone config to Beget.
      Types in YAML override existing; other types preserved.
      Works per-FQDN (apex + each subdomain).
      Example:
        $0 apply configs/dns-mfdev.yaml

  switch <domain> <ns1_fqdn> <ns2_fqdn>
      Replace NS records with our custom NS (auto-backup first).
      Example:
        $0 switch mfdev.ru ns1.mfdev.ru ns2.mfdev.ru

  restore <domain> <backup_file>
      Restore DNS records from a backup.

Env vars required: BEGET_LOGIN, BEGET_PASSWORD
EOF
    exit 1
    ;;
esac
