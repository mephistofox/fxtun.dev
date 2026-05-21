#!/usr/bin/env bash
# PostToolUse hook: gofmt the file that was just edited/written.
# Keeps commits clean without a manual `make fmt` pass.
set -euo pipefail

input="$(cat)"
file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')"

case "$file" in
  *.go)
    if [ -f "$file" ] && command -v gofmt >/dev/null 2>&1; then
      gofmt -w "$file" 2>/dev/null || true
    fi
    ;;
esac

exit 0
