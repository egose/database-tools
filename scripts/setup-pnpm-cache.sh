#!/usr/bin/env bash

set -euo pipefail

printf 'STORE_PATH=%s\n' "$(pnpm store path)" >> "$GITHUB_ENV"
