#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 1 ]; then
  printf 'usage: %s <govulncheck> [args...]\n' "$0" >&2
  exit 2
fi

govulncheck=$1
shift

stdout_file=$(mktemp)
stderr_file=$(mktemp)
trap 'rm -f "$stdout_file" "$stderr_file"' EXIT

set +e
"$govulncheck" -json "$@" >"$stdout_file" 2>"$stderr_file"
status=$?
set -e

cat "$stdout_file"
if [ -s "$stderr_file" ]; then
  cat "$stderr_file" >&2
fi

go run ./internal/releasegovulncheck/cmd/release-govulncheck \
  -scanner-status "$status" \
  -stderr-output "$stderr_file" \
  "$stdout_file"
