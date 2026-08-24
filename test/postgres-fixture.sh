#!/usr/bin/env bash

set -euo pipefail

readonly postgres_host="${POSTGRES_HOST:-127.0.0.1}"
readonly postgres_port="${POSTGRES_PORT:-5432}"
readonly postgres_user="${POSTGRES_USER:-postgres}"
readonly postgres_database="${POSTGRES_DATABASE:-integration}"
readonly postgres_password="${POSTGRES_PASSWORD:-postgres}"
readonly command_timeout="${POSTGRES_HELPER_TIMEOUT:-20s}"

if [[ ! "$postgres_database" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
  printf 'invalid PostgreSQL test database name: %s\n' "$postgres_database" >&2
  exit 2
fi

run_psql() {
  local database="$1"
  shift
  timeout --foreground "$command_timeout" env \
    PGPASSWORD="$postgres_password" \
    PGCONNECT_TIMEOUT=5 \
    PGOPTIONS='-c statement_timeout=10000 -c lock_timeout=5000' \
    psql --no-psqlrc -X --quiet --set=ON_ERROR_STOP=1 \
      --host="$postgres_host" --port="$postgres_port" --username="$postgres_user" \
      --dbname="$database" "$@"
}

reset_database() {
  run_psql postgres --command="SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$postgres_database' AND pid <> pg_backend_pid();" >/dev/null
  run_psql postgres --command="DROP DATABASE IF EXISTS \"$postgres_database\";" >/dev/null
  run_psql postgres --command="CREATE DATABASE \"$postgres_database\";" >/dev/null
}

setup_database() {
  reset_database
  run_psql "$postgres_database" <<'SQL'
BEGIN;
CREATE SCHEMA integration;
CREATE TABLE integration.accounts (
  id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email text NOT NULL UNIQUE,
  age integer NOT NULL CHECK (age > 0),
  created_at timestamptz NOT NULL
);
CREATE TABLE integration.notes (
  id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  account_id integer NOT NULL REFERENCES integration.accounts(id),
  body text NOT NULL
);
CREATE INDEX accounts_age_idx ON integration.accounts(age);
INSERT INTO integration.accounts (email, age, created_at) VALUES
  ('ada@example.test', 37, '2026-08-24T12:00:00Z'),
  ('grace@example.test', 42, '2026-08-24T13:00:00Z');
INSERT INTO integration.notes (account_id, body) VALUES
  (1, 'first backup fixture'),
  (2, 'second backup fixture');
COMMIT;
SQL
  printf 'ready\n'
}

check_database() {
  local result
  result="$(run_psql "$postgres_database" --tuples-only --no-align <<'SQL'
SELECT CASE WHEN
  (SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname = 'integration' AND tablename IN ('accounts', 'notes')) = 2
  AND (SELECT count(*) FROM pg_catalog.pg_constraint c JOIN pg_catalog.pg_namespace n ON n.oid = c.connamespace WHERE n.nspname = 'integration' AND c.contype IN ('p', 'u', 'c', 'f')) >= 5
  AND to_regclass('integration.accounts_age_idx') IS NOT NULL
  AND to_regclass('integration.accounts_id_seq') IS NOT NULL
  AND (SELECT string_agg(email || ':' || age::text, ',' ORDER BY id) FROM integration.accounts) = 'ada@example.test:37,grace@example.test:42'
  AND (SELECT string_agg(body, ',' ORDER BY id) FROM integration.notes) = 'first backup fixture,second backup fixture'
  AND (SELECT last_value FROM integration.accounts_id_seq) = 2
THEN 'restored' ELSE 'mismatch' END;
SQL
)"
  printf '%s\n' "$result"
  [ "$result" = "restored" ]
}

check_empty() {
  local result
  result="$(run_psql "$postgres_database" --tuples-only --no-align --command="SELECT CASE WHEN to_regnamespace('integration') IS NULL THEN 'empty' ELSE 'notempty' END;")"
  printf '%s\n' "$result"
  [ "$result" = "empty" ]
}

create_conflict() {
  reset_database
  run_psql "$postgres_database" <<'SQL'
CREATE SCHEMA integration;
CREATE TABLE integration.accounts (id integer PRIMARY KEY);
SQL
  printf 'conflict-ready\n'
}

create_malformed_archive() {
  local source_archive="$1"
  local destination_archive="$2"
  local workspace
  workspace="$(mktemp -d)"
  trap 'rm -rf "$workspace"' RETURN
  timeout --foreground "$command_timeout" tar -xzf "$source_archive" -C "$workspace"
  printf '{malformed manifest\n' >"$workspace/manifest.json"
  mkdir -p "$(dirname "$destination_archive")"
  timeout --foreground "$command_timeout" tar -czf "$destination_archive" -C "$workspace" manifest.json database.dump
}

case "${1:-}" in
  setup)
    setup_database
    ;;
  reset)
    reset_database
    printf 'reset\n'
    ;;
  check)
    check_database
    ;;
  empty)
    check_empty
    ;;
  conflict)
    create_conflict
    ;;
  malformed-archive)
    if [ "$#" -ne 3 ]; then
      printf 'usage: %s malformed-archive SOURCE DESTINATION\n' "$0" >&2
      exit 2
    fi
    create_malformed_archive "$2" "$3"
    ;;
  *)
    printf 'usage: %s {setup|reset|check|empty|conflict|malformed-archive SOURCE DESTINATION}\n' "$0" >&2
    exit 2
    ;;
esac
