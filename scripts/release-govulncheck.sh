#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 1 ]; then
  printf 'usage: %s <govulncheck> [args...]\n' "$0" >&2
  exit 2
fi

allowed_ids=(
  GO-2025-4020
  GO-2025-3605
  GO-2024-2698
)

output_file=$(mktemp)
trap 'rm -f "$output_file"' EXIT

if "$@" >"$output_file" 2>&1; then
  cat "$output_file"
  exit 0
fi

mapfile -t ids < <(grep -oE 'GO-[0-9]{4}-[0-9]+' "$output_file" | sort -u || true)

if [ "${#ids[@]}" -eq 0 ]; then
  cat "$output_file"
  exit 1
fi

for id in "${ids[@]}"; do
  allowed=0
  for allowed_id in "${allowed_ids[@]}"; do
    if [ "$id" = "$allowed_id" ]; then
      allowed=1
      break
    fi
  done

  if [ "$allowed" -eq 0 ]; then
    cat "$output_file"
    exit 1
  fi
done

cat "$output_file"
printf '\nOnly documented no-fix archive vulnerabilities remain; tracked by ARCHIVE-01.\n'
