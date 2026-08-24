#!/usr/bin/env bash

set -euo pipefail

image_ref="${1:-database-tools:release-verify}"
postgres_image="postgres:16.10-bookworm@sha256:38471f330eb885e04de130b768d6db4e10469e2311879c7e5c699f6d2d8a1c74"
suffix="$(date +%s)-$$"
network="database-tools-container-roundtrip-$suffix"
postgres_container="database-tools-container-roundtrip-postgres-$suffix"
storage_volume="database-tools-container-roundtrip-storage-$suffix"
postgres_volume="database-tools-container-roundtrip-postgres-data-$suffix"

cleanup() {
  docker rm -f "$postgres_container" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  docker volume rm "$storage_volume" "$postgres_volume" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker network create "$network" >/dev/null
docker volume create "$storage_volume" >/dev/null
docker volume create "$postgres_volume" >/dev/null

docker run --rm --mount "type=volume,source=$storage_volume,target=/backup" alpine:3.23.5@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40 \
  sh -c 'chown 1000:1000 /backup && chmod 700 /backup'

docker run -d --name "$postgres_container" --network "$network" \
  --mount "type=volume,source=$postgres_volume,target=/var/lib/postgresql/data" \
  -e POSTGRES_DB=integration \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  "$postgres_image" >/dev/null

for _ in $(seq 1 60); do
  if docker exec "$postgres_container" pg_isready -U postgres -d integration -t 2 >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$postgres_container" pg_isready -U postgres -d integration -t 5 >/dev/null

docker exec -i "$postgres_container" psql -U postgres -d integration -v ON_ERROR_STOP=1 <<'SQL'
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
SQL

docker run --rm --network "$network" \
  --mount "type=volume,source=$storage_volume,target=/backup" \
  -e POSTGRES__PASSWORD=postgres \
  "$image_ref" postgres-archive \
    --host="$postgres_container" --port=5432 --user=postgres --database=integration --ssl-mode=disable --local-path=/backup

docker exec -i "$postgres_container" psql -U postgres -d postgres -v ON_ERROR_STOP=1 <<'SQL'
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'integration' AND pid <> pg_backend_pid();
DROP DATABASE integration;
CREATE DATABASE integration;
SQL

docker run --rm --network "$network" \
  --mount "type=volume,source=$storage_volume,target=/backup" \
  -e POSTGRES__PASSWORD=postgres \
  "$image_ref" postgres-unarchive \
    --host="$postgres_container" --port=5432 --user=postgres --database=integration --ssl-mode=disable --local-path=/backup

result="$(docker exec -i "$postgres_container" psql -U postgres -d integration --tuples-only --no-align -v ON_ERROR_STOP=1 <<'SQL'
SELECT CASE WHEN
  (SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname = 'integration' AND tablename IN ('accounts', 'notes')) = 2
  AND (SELECT count(*) FROM pg_catalog.pg_constraint c JOIN pg_catalog.pg_namespace n ON n.oid = c.connamespace WHERE n.nspname = 'integration' AND c.contype IN ('p', 'u', 'c', 'f')) >= 5
  AND to_regclass('integration.accounts_age_idx') IS NOT NULL
  AND to_regclass('integration.accounts_id_seq') IS NOT NULL
  AND (SELECT string_agg(email || ':' || age::text, ',' ORDER BY id) FROM integration.accounts) = 'ada@example.test:37,grace@example.test:42'
  AND (SELECT string_agg(body, ',' ORDER BY id) FROM integration.notes) = 'first backup fixture,second backup fixture'
THEN 'restored' ELSE 'mismatch' END;
SQL
)"

if [ "$result" != "restored" ]; then
  printf 'container PostgreSQL round trip failed: %s\n' "$result" >&2
  exit 1
fi

printf 'container PostgreSQL round trip passed using %s\n' "$image_ref"
