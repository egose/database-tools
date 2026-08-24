#!/usr/bin/env bash

set -euo pipefail

add_pinned_plugin() {
  local name="$1"
  local url="$2"
  local ref="$3"
  local plugin_dir="${ASDF_DATA_DIR:-$HOME/.asdf}/plugins/$name"

  asdf plugin add "$name" "$url" || true
  git -C "$plugin_dir" remote set-url origin "$url"
  git -C "$plugin_dir" fetch --depth 1 origin "$ref"
  git -C "$plugin_dir" checkout --detach "$ref"
  test "$(git -C "$plugin_dir" rev-parse HEAD)" = "$ref"
}

add_pinned_plugin nodejs https://github.com/asdf-vm/asdf-nodejs.git 779c8dc84b3bdab38c2c80622d315c2c3267f74b
add_pinned_plugin python https://github.com/danhper/asdf-python.git abc2a03863e4d569b4f9de0d0efc1a88d96c2c12
add_pinned_plugin golang https://github.com/asdf-community/asdf-golang.git a75b761963d8e6eda1a185c73476da8a75b8d300
add_pinned_plugin pnpm https://github.com/jonathanmorley/asdf-pnpm.git eb065a91568590a00a22c823430b2f32f09e83e7
add_pinned_plugin bats https://github.com/timgluz/asdf-bats.git 299551f1668b2ba11804bdca709da8933b647bb5
add_pinned_plugin act https://github.com/gr1m0h/asdf-act.git 1fca84f81bb033afc79d1f6a1787305cdcbcda18
add_pinned_plugin docker-compose https://github.com/egose/asdf-docker-compose.git 477ce07b7e17666cc1a3f06c486e0bde2d92db08
add_pinned_plugin mongosh https://github.com/itspngu/asdf-mongosh.git 1b249027e87d9c7f5b4b8850182bf84e1e79fc95
add_pinned_plugin actionlint https://github.com/crazy-matt/asdf-actionlint.git f8df3cfa05a1c875415b015808548bc8bac3b3a4
add_pinned_plugin shellcheck https://github.com/luizm/asdf-shellcheck.git 8b954a95d44b8d8a6c6cd5a5a52ed643b7bb52e3

asdf plugin list --urls --refs
asdf install
asdf reshim
