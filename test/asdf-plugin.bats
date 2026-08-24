#!/usr/bin/env bats

setup() {
  REPO_ROOT=$(cd -- "$BATS_TEST_DIRNAME/.." && pwd)
  WORK=$(mktemp -d)
  MOCK_BIN="$WORK/bin"
  mkdir -p "$MOCK_BIN"
}

teardown() {
  rm -rf "$WORK"
}

make_release_archive() {
  local archive=$1
  mkdir -p "$WORK/source"
  printf '#!/bin/sh\nexit 0\n' > "$WORK/source/mongo-archive"
  printf '#!/bin/sh\nexit 0\n' > "$WORK/source/mongo-unarchive"
  printf 'Apache License\n' > "$WORK/source/LICENSE"
  tar -C "$WORK/source" -czf "$archive" mongo-archive mongo-unarchive LICENSE
}

make_uname() {
  cat > "$MOCK_BIN/uname" <<EOF
#!/usr/bin/env bash
if [ "\$1" = "-s" ]; then printf '%s\n' '${1}'; else printf '%s\n' '${2}'; fi
EOF
  chmod +x "$MOCK_BIN/uname"
}

make_curl() {
  cat > "$MOCK_BIN/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
url=
out=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out=$2; shift 2 ;;
    http*) url=$1; shift ;;
    *) shift ;;
  esac
done
printf '%s\n' "$url" >> "$CURL_LOG"
cp "$FIXTURES/${url##*/}" "$out"
EOF
  chmod +x "$MOCK_BIN/curl"
}

@test "download uses database-tools release assets and verifies their checksum" {
  make_uname Linux x86_64
  make_curl
  mkdir -p "$WORK/release"
  make_release_archive "$WORK/release/database-tools-linux-amd64.tar.gz"
  (cd "$WORK/release" && sha256sum ./database-tools-linux-amd64.tar.gz > database-tools-v1.2.3-sha256.txt)

  run env PATH="$MOCK_BIN:$PATH" CURL_LOG="$WORK/urls" FIXTURES="$WORK/release" ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$WORK/download" "$REPO_ROOT/bin/download"

  [ "$status" -eq 0 ]
  grep -Fx 'https://github.com/egose/database-tools/releases/download/v1.2.3/database-tools-linux-amd64.tar.gz' "$WORK/urls"
  grep -Fx 'https://github.com/egose/database-tools/releases/download/v1.2.3/database-tools-v1.2.3-sha256.txt' "$WORK/urls"
}

@test "download rejects a checksum mismatch" {
  make_uname Linux x86_64
  make_curl
  mkdir -p "$WORK/release"
  make_release_archive "$WORK/release/database-tools-linux-amd64.tar.gz"
  printf '%064d  ./database-tools-linux-amd64.tar.gz\n' 0 > "$WORK/release/database-tools-v1.2.3-sha256.txt"

  run env PATH="$MOCK_BIN:$PATH" CURL_LOG="$WORK/urls" FIXTURES="$WORK/release" ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$WORK/download" "$REPO_ROOT/bin/download"

  [ "$status" -ne 0 ]
  [[ $output == *"Checksum verification failed"* ]]
}

@test "install atomically installs both executables and the license" {
  mkdir -p "$WORK/download" "$WORK/installs"
  make_release_archive "$WORK/download/database-tools-linux-amd64.tar.gz"

  run env ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$WORK/download" ASDF_INSTALL_PATH="$WORK/installs/1.2.3" "$REPO_ROOT/bin/install"

  [ "$status" -eq 0 ]
  [ -x "$WORK/installs/1.2.3/bin/mongo-archive" ]
  [ -x "$WORK/installs/1.2.3/bin/mongo-unarchive" ]
  [ -f "$WORK/installs/1.2.3/LICENSE" ]
  [ -z "$(find "$WORK/installs" -maxdepth 1 -name '.database-tools.install.*' -print)" ]
}

@test "install accepts the checksummed legacy archive without a license" {
  mkdir -p "$WORK/download" "$WORK/source"
  printf '#!/bin/sh\nexit 0\n' > "$WORK/source/mongo-archive"
  printf '#!/bin/sh\nexit 0\n' > "$WORK/source/mongo-unarchive"
  tar -C "$WORK/source" -czf "$WORK/download/database-tools-linux-amd64.tar.gz" mongo-archive mongo-unarchive

  run env ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=0.16.0 ASDF_DOWNLOAD_PATH="$WORK/download" ASDF_INSTALL_PATH="$WORK/install" "$REPO_ROOT/bin/install"

  [ "$status" -eq 0 ]
  [ -x "$WORK/install/bin/mongo-archive" ]
  [ -x "$WORK/install/bin/mongo-unarchive" ]
}

@test "install rejects unexpected archive members" {
  mkdir -p "$WORK/download" "$WORK/source"
  printf 'payload\n' > "$WORK/source/unexpected"
  tar -C "$WORK/source" -czf "$WORK/download/database-tools-linux-amd64.tar.gz" unexpected

  run env ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$WORK/download" ASDF_INSTALL_PATH="$WORK/install" "$REPO_ROOT/bin/install"

  [ "$status" -ne 0 ]
  [ ! -e "$WORK/install" ]
}

@test "install rejects a symlink executable" {
  mkdir -p "$WORK/download" "$WORK/source"
  ln -s /bin/sh "$WORK/source/mongo-archive"
  printf '#!/bin/sh\nexit 0\n' > "$WORK/source/mongo-unarchive"
  printf 'Apache License\n' > "$WORK/source/LICENSE"
  tar -C "$WORK/source" -czf "$WORK/download/database-tools-linux-amd64.tar.gz" mongo-archive mongo-unarchive LICENSE

  run env ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$WORK/download" ASDF_INSTALL_PATH="$WORK/install" "$REPO_ROOT/bin/install"

  [ "$status" -ne 0 ]
  [ ! -e "$WORK/install" ]
}

@test "list-all returns only releases with the embedded plugin checksum contract" {
  cat > "$MOCK_BIN/curl" <<'EOF'
#!/usr/bin/env sh
cat <<'JSON'
[
  {
    "tag_name": "v1.2.3",
    "assets": [{"name": "database-tools-v1.2.3-sha256.txt"}]
  },
  {
    "tag_name": "v1.2.4",
    "assets": [{"name": "database-tools-linux-amd64.tar.gz"}]
  },
  {
    "tag_name": "v2.0.0-rc.1",
    "assets": [{"name": "database-tools-v2.0.0-rc.1-sha256.txt"}]
  },
  {
    "tag_name": "v2.0.0-rc.2",
    "assets": [{"name": "database-tools-v2.0.0-rc.2-sha256.txt"}]
  },
  {
    "tag_name": "v2.0.0",
    "assets": [{"name": "database-tools-v2.0.0-sha256.txt"}]
  }
]
JSON
EOF
  chmod +x "$MOCK_BIN/curl"

  run env PATH="$MOCK_BIN:$PATH" "$REPO_ROOT/bin/list-all"

  [ "$status" -eq 0 ]
  [ "$output" = "1.2.3 2.0.0-rc.1 2.0.0-rc.2 2.0.0" ]
}

@test "list-all fails closed on an API error" {
  cat > "$MOCK_BIN/curl" <<'EOF'
#!/usr/bin/env sh
exit 22
EOF
  chmod +x "$MOCK_BIN/curl"

  run env PATH="$MOCK_BIN:$PATH" "$REPO_ROOT/bin/list-all"

  [ "$status" -eq 22 ]
}
