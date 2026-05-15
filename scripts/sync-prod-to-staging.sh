#!/bin/bash
# Sync production SQLite → staging PostgreSQL (upsert)
# Reads SQLite from prod via SSH, writes to PG via mfdev (internal access)
set -euo pipefail

PROD_HOST="root@217.114.5.207"
PROD_DB="/root/fxtunnel/data/fxtunnel.db"
STAGING_HOST="root@mfdev.ru"
PG_HOST="10.16.0.3"
PG_USER="mephisto"
PG_PASS="2&6OJ&ROheAi"
PG_DB="fxtunnel"

PSQL_CMD="PGPASSWORD='$PG_PASS' psql -h $PG_HOST -U $PG_USER -d $PG_DB -q --no-psqlrc"

# table:pk_column (most use 'id', some use composite)
TABLES=(
  "plans:id"
  "users:id"
  "api_tokens:id"
  "sessions:id"
  "reserved_domains:id"
  "custom_domains:id"
  "tls_certificates:id"
  "totp_secrets:id"
  "invite_codes:id"
  "subscriptions:id"
  "payments:id"
  "audit_logs:id"
  "user_bundles:id"
  "user_history:id"
  "user_settings:user_id,key"
)

echo "=== fxTunnel Prod (SQLite) → Staging (PostgreSQL) ==="
echo ""

for entry in "${TABLES[@]}"; do
  table="${entry%%:*}"
  pk="${entry##*:}"

  # Export CSV from prod
  ssh "$PROD_HOST" "sqlite3 -header -csv '$PROD_DB' 'SELECT * FROM $table;'" 2>/dev/null \
    | ssh "$STAGING_HOST" "cat > /tmp/_sync_${table}.csv"

  rows=$(ssh "$STAGING_HOST" "wc -l < /tmp/_sync_${table}.csv" | tr -d ' ')
  rows=$((rows - 1))
  if [ "$rows" -le 0 ]; then
    echo "  $table: empty, skip"
    continue
  fi

  # Get columns from CSV header
  cols=$(ssh "$STAGING_HOST" "head -1 /tmp/_sync_${table}.csv" | tr -d '\r\n')

  # Build SET clause: all columns except PK columns
  IFS=',' read -ra pk_arr <<< "$pk"
  set_parts=""
  for col in $(echo "$cols" | tr ',' '\n'); do
    is_pk=false
    for p in "${pk_arr[@]}"; do
      if [ "$col" = "$p" ]; then is_pk=true; break; fi
    done
    if [ "$is_pk" = false ]; then
      if [ -n "$set_parts" ]; then set_parts="$set_parts,"; fi
      set_parts="$set_parts $col = EXCLUDED.$col"
    fi
  done

  # Build sequence reset (only for tables with 'id' as single PK)
  seq_reset=""
  if [ "$pk" = "id" ]; then
    seq_reset="SELECT setval(pg_get_serial_sequence('${table}', 'id'), COALESCE(MAX(id), 1)) FROM ${table};"
  fi

  # Execute upsert with FK constraints disabled
  ssh "$STAGING_HOST" "$PSQL_CMD" <<SQL 2>&1 | grep -v "^$\|SETVAL\|setval\| *[0-9]"
SET session_replication_role = 'replica';
CREATE TEMP TABLE _tmp (LIKE ${table} INCLUDING ALL);
\COPY _tmp(${cols}) FROM '/tmp/_sync_${table}.csv' WITH (FORMAT csv, HEADER true, NULL '');
DELETE FROM ${table} WHERE ${pk%%,*} IN (SELECT ${pk%%,*} FROM _tmp);
INSERT INTO ${table} SELECT * FROM _tmp ON CONFLICT DO NOTHING;
${seq_reset}
DROP TABLE _tmp;
SET session_replication_role = 'origin';
SQL

  echo "  $table: $rows rows ✓"
done

# Cleanup
ssh "$STAGING_HOST" "rm -f /tmp/_sync_*.csv"

echo ""
echo "=== Verification ==="
for entry in "${TABLES[@]}"; do
  table="${entry%%:*}"
  count=$(ssh "$STAGING_HOST" "$PSQL_CMD -t -c \"SELECT count(*) FROM $table;\"" | tr -d ' ')
  echo "  $table: $count"
done

echo ""
echo "✓ Sync complete!"
