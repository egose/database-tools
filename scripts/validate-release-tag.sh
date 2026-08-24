#!/usr/bin/env bash

set -euo pipefail

tag=${1:-}
identifier='(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
semver="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-${identifier}(\\.${identifier})*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

if [[ ! "$tag" =~ $semver ]]; then
  printf 'release tag must be an anchored SemVer with a v prefix: %s\n' "$tag" >&2
  exit 1
fi
