#!/usr/bin/env bash
# PreToolUse hook: refuse edits to sqlc-generated code.
# CLAUDE.md mandates that internal/server/database/sqlc/ is generated and must
# not be hand-edited — change queries/ or migrations/ and run `sqlc generate`.
set -euo pipefail

input="$(cat)"
file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')"

[ -z "$file" ] && exit 0

case "$file" in
  */internal/server/database/sqlc/*)
    echo "BLOCKED: $file is sqlc-generated and read-only." >&2
    echo "Edit internal/server/database/queries/ or migrations/, then run 'sqlc generate'." >&2
    exit 2
    ;;
esac

exit 0
