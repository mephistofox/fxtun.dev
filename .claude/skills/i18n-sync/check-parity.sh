#!/usr/bin/env bash
# Report vue-i18n locale key parity for fxTunnel frontends.
# Lists keys present in one locale of a pair but missing in the other.
# Exit 1 if any mismatch is found.
set -uo pipefail

pairs=(
  "web/src/i18n/en.json:web/src/i18n/ru.json"
  "gui/frontend/src/i18n/en.json:gui/frontend/src/i18n/ru.json"
)

status=0
for pair in "${pairs[@]}"; do
  a="${pair%%:*}"
  b="${pair##*:}"
  [ -f "$a" ] && [ -f "$b" ] || { echo "skip (missing): $a / $b"; continue; }

  only_a="$(comm -23 <(jq -r 'paths(scalars)|join(".")' "$a" | sort) <(jq -r 'paths(scalars)|join(".")' "$b" | sort))"
  only_b="$(comm -13 <(jq -r 'paths(scalars)|join(".")' "$a" | sort) <(jq -r 'paths(scalars)|join(".")' "$b" | sort))"

  if [ -z "$only_a" ] && [ -z "$only_b" ]; then
    echo "OK  $a ↔ $b  ($(jq '[paths(scalars)]|length' "$a") keys)"
  else
    status=1
    echo "MISMATCH  $a ↔ $b"
    [ -n "$only_a" ] && { echo "  only in $(basename "$a"):"; echo "$only_a" | sed 's/^/    /'; }
    [ -n "$only_b" ] && { echo "  only in $(basename "$b"):"; echo "$only_b" | sed 's/^/    /'; }
  fi
done

exit $status
