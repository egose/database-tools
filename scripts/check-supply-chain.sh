#!/usr/bin/env bash

set -euo pipefail

failures=0

require_equal() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  if [ "$expected" != "$actual" ]; then
    printf '%s drift: expected %s, got %s\n' "$label" "$expected" "$actual" >&2
    failures=$((failures + 1))
  fi
}

require_match() {
  local label="$1"
  local value="$2"
  local pattern="$3"
  if [[ ! "$value" =~ $pattern ]]; then
    printf '%s is not pinned as required: %s\n' "$label" "$value" >&2
    failures=$((failures + 1))
  fi
}

go_mod_version=$(awk '$1 == "go" { print $2; exit }' go.mod)
tool_versions_go=$(awk '$1 == "golang" { print $2; exit }' .tool-versions)
docker_go_version=$(awk '/^FROM golang:/ { sub(/^FROM golang:/, ""); sub(/@.*/, ""); print; exit }' Dockerfile)
policy_go_version=$(awk '/Go `/ { value=$0; sub(/^.*Go `/, "", value); sub(/`.*/, "", value); print value; exit }' docs/release-artifact-policy.md)
postgres_client_package=$(awk '/^ARG POSTGRESQL_CLIENT_PACKAGE=/ { sub(/^ARG POSTGRESQL_CLIENT_PACKAGE=/, ""); print; exit }' Dockerfile)

require_equal "Go version in .tool-versions" "$go_mod_version" "$tool_versions_go"
require_equal "Go version in Dockerfile" "$go_mod_version" "$docker_go_version"
require_equal "Go version in release policy" "$go_mod_version" "$policy_go_version"
require_match "Dockerfile PostgreSQL client package" "$postgres_client_package" '^postgresql[0-9]+-client=[0-9]+\.[0-9]+-r[0-9]+$'

package_manager=$(python3 - <<'PY'
import json
from pathlib import Path
package = json.loads(Path("package.json").read_text())
print(package.get("packageManager", ""))
if package.get("private") is not True:
    raise SystemExit("package.json must set private=true")
PY
)
package_pnpm_version=${package_manager#pnpm@}
tool_versions_pnpm=$(awk '$1 == "pnpm" { print $2; exit }' .tool-versions)
require_equal "pnpm version in package.json" "$tool_versions_pnpm" "$package_pnpm_version"

while IFS= read -r from_line; do
  require_match "Dockerfile base image" "$from_line" '^FROM [^ ]+@sha256:[0-9a-f]{64}([[:space:]]|$)'
done < <(awk '/^FROM / { print }' Dockerfile)

while IFS= read -r image_ref; do
  require_match "release workflow container image" "$image_ref" '^[^[:space:]]+@sha256:[0-9a-f]{64}$'
done < <(awk '/anchore\/syft:/ { print $1 }' .github/workflows/release.yml)

while IFS= read -r plugin_pin; do
  require_match "asdf plugin pin" "$plugin_pin" '^[a-z0-9_-]+:[0-9a-f]{40}$'
done < <(awk '/^add_pinned_plugin / { print $2 ":" $4 }' scripts/setup-asdf-tools.sh)

declared_plugins=$(awk '!/^($|#)/ { print $1 }' .tool-versions | sort)
pinned_plugins=$(awk '/^add_pinned_plugin / { print $2 }' scripts/setup-asdf-tools.sh | sort)
require_equal "declared and pinned asdf plugins" "$declared_plugins" "$pinned_plugins"

if ! python3 - <<'PY'
from pathlib import Path

entries = []
current = []
for raw_line in Path("requirements-lock.txt").read_text().splitlines():
    line = raw_line.strip()
    if not line or line.startswith("#"):
        continue
    if not raw_line[:1].isspace():
        if current:
            entries.append(current)
        current = [line]
    elif current:
        current.append(line)
if current:
    entries.append(current)

unhashed = [entry[0] for entry in entries if not any(part.startswith("--hash=sha256:") for part in entry[1:])]
if unhashed:
    raise SystemExit("unhashed Python lock entries: " + ", ".join(unhashed))
PY
then
  printf 'requirements-lock.txt is not fully hashed\n' >&2
  failures=$((failures + 1))
fi

if [ "$failures" -ne 0 ]; then
  exit 1
fi

printf 'supply-chain declarations are consistent\n'
