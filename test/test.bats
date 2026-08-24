#!/usr/bin/env bats

set -a
source .env.test
set +a

mkdir -p ./dist/backup

FAKE_GCP_URL="http://localhost:$FAKE_GCP_PORT"
POSTGRES_HOST="${POSTGRES_HOST:-127.0.0.1}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DATABASE="${POSTGRES_DATABASE:-integration}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
export POSTGRES_HOST POSTGRES_PORT POSTGRES_USER POSTGRES_DATABASE POSTGRES_PASSWORD

postgres_connection_args=(
  "--host=$POSTGRES_HOST"
  "--port=$POSTGRES_PORT"
  "--user=$POSTGRES_USER"
  "--database=$POSTGRES_DATABASE"
  "--ssl-mode=disable"
)

setup_file() {
  rm -rf ./dist/backup /tmp/datadump /tmp/datarestore
  mkdir -p ./dist/backup
  curl -fsS -X POST -H "Content-Type: application/json" "${FAKE_GCP_URL}/storage/v1/b?project=fake-gcs" -d "{\"name\":\"${FAKE_GCP_BUCKET}\"}" >/dev/null || true
}

assert_success() {
  if [ "$status" -ne 0 ]; then
    echo "$output" >&2
  fi
  [ "$status" -eq 0 ]
}

assert_failure() {
  if [ "$status" -eq 0 ]; then
    echo "$output" >&2
  fi
  [ "$status" -ne 0 ]
}

assert_output_contains() {
  [[ "$output" == *"$1"* ]]
}

storage_count() {
  go run ./test/teststorage-state.go "$@"
}

postgres_storage_count() {
  storage_count "$@" --backup-prefix=postgres-archive/
}

latest_local_object() {
  local prefix="$1"
  local object
  object="$(printf '%s\n' "./dist/backup/$prefix"/*.tar.gz | sort | head -n 1)"
  [ -f "$object" ]
  printf '%s\n' "$object"
}

@test "tools build" {
  run bash -lc 'make build && CGO_ENABLED=0 go build -trimpath -buildvcs=false -o ./dist/postgres-archive ./postgresarchive/main && CGO_ENABLED=0 go build -trimpath -buildvcs=false -o ./dist/postgres-unarchive ./postgresunarchive/main && printf "complete\n"'
  assert_success
  assert_output_contains "complete"
}

@test "mongodb setup" {
  run bash -lc 'go run ./test/testdb-setup.go | tail -n1'
  assert_success
  [ "$output" = "ready" ]
}

@test "data found before archive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}

@test "run captures nonzero exit status despite success output" {
  run bash -lc 'printf "Unarchive completed successfully\n"; exit 1'
  assert_failure
  assert_output_contains "Unarchive completed successfully"
}

# Local Archive
@test "[local disk] archive" {
  before_count="$(storage_count --provider=local --local-path=./dist/backup)"

  run ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup
  assert_success
  assert_output_contains "Successfully uploaded backup to *storage.LocalStorage"

  after_count="$(storage_count --provider=local --local-path=./dist/backup)"
  [ "$after_count" -eq $((before_count + 1)) ]
}

@test "[local disk] drop" {
  run bash -lc 'go run ./test/testdb-drop.go | tail -n1'
  assert_success
  [ "$output" = "dropped" ]
}

@test "[local disk] data notfound" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "notfound" ]
}

@test "[local disk] unarchive missing object fails" {
  run ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup --object-name=missing.tar.gz
  assert_failure
  assert_output_contains "missing.tar.gz"
}

@test "[local disk] data remains missing after failed unarchive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "notfound" ]
}

@test "[local disk] unarchive" {
  run ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup
  assert_success
  assert_output_contains "Unarchive completed successfully"
}

@test "[local disk] data found after unarchive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}

# AWS S3 Archive
@test "[S3 bucket] archive" {
  before_count="$(storage_count --provider=s3)"

  run ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$MINIO_BUCKET" --aws-s3-force-path-style=true
  assert_success
  assert_output_contains "Successfully uploaded backup to *storage.AwsS3:"

  after_count="$(storage_count --provider=s3)"
  [ "$after_count" -eq $((before_count + 1)) ]
}

@test "[S3 bucket] drop" {
  run bash -lc 'go run ./test/testdb-drop.go | tail -n1'
  assert_success
  [ "$output" = "dropped" ]
}

@test "[S3 bucket] data notfound" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "notfound" ]
}

@test "[S3 bucket] unarchive" {
  run ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$MINIO_BUCKET" --aws-s3-force-path-style=true
  assert_success
  assert_output_contains "Unarchive completed successfully"
}

@test "[S3 bucket] data found after unarchive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}

# Azure Storage Archive
@test "[Azure Storage] create container" {
  run az storage container create -n "$AZURITE_CONTAINER" --connection-string "DefaultEndpointsProtocol=http;AccountName=$AZURITE_ACCOUNT_NAME;AccountKey=$AZURITE_ACCOUNT_KEY;BlobEndpoint=$AZURITE_URL/$AZURITE_ACCOUNT_NAME;"
  assert_success
}

@test "[Azure Storage] archive" {
  before_count="$(storage_count --provider=azure)"

  run ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --az-endpoint="$AZURITE_URL" --az-account-name="$AZURITE_ACCOUNT_NAME" --az-account-key="$AZURITE_ACCOUNT_KEY" --az-container-name="$AZURITE_CONTAINER"
  assert_success
  assert_output_contains "Successfully uploaded backup to *storage.AzBlob:"

  after_count="$(storage_count --provider=azure)"
  [ "$after_count" -eq $((before_count + 1)) ]
}

@test "[Azure Storage] drop" {
  run bash -lc 'go run ./test/testdb-drop.go | tail -n1'
  assert_success
  [ "$output" = "dropped" ]
}

@test "[Azure Storage] data notfound" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "notfound" ]
}

@test "[Azure Storage] unarchive" {
  run ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --az-endpoint="$AZURITE_URL" --az-account-name="$AZURITE_ACCOUNT_NAME" --az-account-key="$AZURITE_ACCOUNT_KEY" --az-container-name="$AZURITE_CONTAINER"
  assert_success
  assert_output_contains "Unarchive completed successfully"
}

@test "[Azure Storage] data found after unarchive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}

# GCP Storage Archive
@test "[GCP Storage] archive" {
  before_count="$(storage_count --provider=gcp)"

  run env STORAGE_EMULATOR_HOST="$FAKE_GCP_URL" ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --gcp-endpoint="$FAKE_GCP_URL/storage/v1/" --gcp-bucket="$FAKE_GCP_BUCKET"
  assert_success
  assert_output_contains "Successfully uploaded backup to *storage.GcpStorage:"

  after_count="$(storage_count --provider=gcp)"
  [ "$after_count" -eq $((before_count + 1)) ]
}

@test "[GCP Storage] drop" {
  run bash -lc 'go run ./test/testdb-drop.go | tail -n1'
  assert_success
  [ "$output" = "dropped" ]
}

@test "[GCP Storage] data notfound" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "notfound" ]
}

@test "[GCP Storage] unarchive" {
  run env STORAGE_EMULATOR_HOST="$FAKE_GCP_URL" ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --gcp-endpoint="$FAKE_GCP_URL/storage/v1/" --gcp-bucket="$FAKE_GCP_BUCKET"
  assert_success
  assert_output_contains "Unarchive completed successfully"
}

@test "[GCP Storage] data found after unarchive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}

# Local + AWS S3 Storage Archives
@test "[local disk + S3 bucket] archive" {
  local_before_count="$(storage_count --provider=local --local-path=./dist/backup)"
  s3_before_count="$(storage_count --provider=s3)"

  run ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$MINIO_BUCKET" --aws-s3-force-path-style=true
  assert_success
  assert_output_contains "Successfully uploaded backup to backend #1 (local)"
  assert_output_contains "Successfully uploaded backup to backend #2 (aws)"

  local_after_count="$(storage_count --provider=local --local-path=./dist/backup)"
  s3_after_count="$(storage_count --provider=s3)"
  [ "$local_after_count" -eq $((local_before_count + 1)) ]
  [ "$s3_after_count" -eq $((s3_before_count + 1)) ]
}

@test "[local disk + S3 bucket] drop" {
  run bash -lc 'go run ./test/testdb-drop.go | tail -n1'
  assert_success
  [ "$output" = "dropped" ]
}

@test "[local disk + S3 bucket] data notfound" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "notfound" ]
}

@test "[local disk + S3 bucket] unarchive" {
  run ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --storage-backend=local --local-path=./dist/backup --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$MINIO_BUCKET" --aws-s3-force-path-style=true
  assert_success
  assert_output_contains "Unarchive completed successfully"
}

@test "[local disk + S3 bucket] data found after unarchive" {
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}

@test "[local disk + S3 bucket] second upload failure reports partial state and skips retention" {
  prefix="mixed-failure"
  old_object="$prefix/1000000000000-2026-08-01T010203.456Z.tar.gz"
  old_path="./dist/backup/$old_object"
  mkdir -p "$(dirname "$old_path")"
  printf 'old backup sentinel' >"$old_path"
  touch -d '1970-01-01 UTC' "$old_path"

  missing_bucket="missing-${MINIO_BUCKET}"
  run ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$missing_bucket" --aws-s3-force-path-style=true --backup-prefix="$prefix" --expiry-days=1
  assert_failure
  assert_output_contains "archive upload failed after successful uploads to backend #1 (local)"
  assert_output_contains "retention was not run on any backend"
  [ -f "$old_path" ]
}

# PostgreSQL local archive and restore
@test "[PostgreSQL] setup representative schema and data" {
  run ./test/postgres-fixture.sh setup
  assert_success
  [ "$output" = "ready" ]
}

@test "[PostgreSQL] schema and data found before archive" {
  run ./test/postgres-fixture.sh check
  assert_success
  [ "$output" = "restored" ]
}

@test "[PostgreSQL local disk] failed dump uploads nothing" {
  before_count="$(postgres_storage_count --provider=local --local-path=./dist/backup)"

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-archive "${postgres_connection_args[@]}" --port=1 --local-path=./dist/backup
  assert_failure
  assert_output_contains "PostgreSQL archive failed"
  [ "${#output}" -le 70000 ]

  after_count="$(postgres_storage_count --provider=local --local-path=./dist/backup)"
  [ "$after_count" -eq "$before_count" ]
}

@test "[PostgreSQL local disk] archive" {
  before_count="$(postgres_storage_count --provider=local --local-path=./dist/backup)"

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-archive "${postgres_connection_args[@]}" --local-path=./dist/backup
  assert_success
  assert_output_contains "Successfully uploaded backup"

  after_count="$(postgres_storage_count --provider=local --local-path=./dist/backup)"
  [ "$after_count" -eq $((before_count + 1)) ]
}

@test "[PostgreSQL local disk] reset target database" {
  run ./test/postgres-fixture.sh reset
  assert_success
  [ "$output" = "reset" ]

  run ./test/postgres-fixture.sh empty
  assert_success
  [ "$output" = "empty" ]
}

@test "[PostgreSQL local disk] missing object fails without changing target" {
  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --local-path=./dist/backup --object-name=missing.tar.gz
  assert_failure
  assert_output_contains "missing.tar.gz"

  run ./test/postgres-fixture.sh empty
  assert_success
  [ "$output" = "empty" ]
}

@test "[PostgreSQL local disk] explicit object restore round trip" {
  archive_path="$(latest_local_object postgres-archive)"
  object_name="${archive_path##*/}"

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --local-path=./dist/backup --object-name="$object_name"
  assert_success
  assert_output_contains "PostgreSQL restore completed successfully"

  run ./test/postgres-fixture.sh check
  assert_success
  [ "$output" = "restored" ]
}

@test "[PostgreSQL local disk] automatic latest restore round trip" {
  run ./test/postgres-fixture.sh reset
  assert_success

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --local-path=./dist/backup
  assert_success
  assert_output_contains "PostgreSQL restore completed successfully"

  run ./test/postgres-fixture.sh check
  assert_success
  [ "$output" = "restored" ]
}

@test "[PostgreSQL local disk] malformed manifest fails before restore" {
  archive_path="$(latest_local_object postgres-archive)"
  malformed_path="./dist/backup/postgres-archive/0000000000000-2099-01-01T000000.000Z.tar.gz"
  ./test/postgres-fixture.sh malformed-archive "$archive_path" "$malformed_path"
  touch -d '2100-01-01 UTC' "$malformed_path"

  run ./test/postgres-fixture.sh reset
  assert_success
  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --local-path=./dist/backup
  assert_failure
  assert_output_contains "manifest"
  [ "${#output}" -le 70000 ]
  rm -f "$malformed_path"

  run ./test/postgres-fixture.sh empty
  assert_success
  [ "$output" = "empty" ]
}

@test "[PostgreSQL local disk] nonzero pg_restore is surfaced with bounded diagnostics" {
  archive_path="$(latest_local_object postgres-archive)"
  object_name="${archive_path##*/}"
  run ./test/postgres-fixture.sh conflict
  assert_success

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --local-path=./dist/backup --object-name="$object_name"
  assert_failure
  assert_output_contains "partial database changes may exist"
  [ "${#output}" -le 70000 ]
}

# PostgreSQL representative remote-provider round trip
@test "[PostgreSQL S3 bucket] archive" {
  run ./test/postgres-fixture.sh setup
  assert_success
  before_count="$(postgres_storage_count --provider=s3)"

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-archive "${postgres_connection_args[@]}" --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$MINIO_BUCKET" --aws-s3-force-path-style=true
  assert_success

  after_count="$(postgres_storage_count --provider=s3)"
  [ "$after_count" -eq $((before_count + 1)) ]
}

@test "[PostgreSQL S3 bucket] automatic restore round trip" {
  run ./test/postgres-fixture.sh reset
  assert_success

  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --aws-endpoint="$MINIO_URL" --aws-access-key-id="$MINIO_ACCESS_KEY" --aws-secret-access-key="$MINIO_SECRET_KEY" --aws-bucket="$MINIO_BUCKET" --aws-s3-force-path-style=true
  assert_success

  run ./test/postgres-fixture.sh check
  assert_success
  [ "$output" = "restored" ]
}

@test "[mixed database families] latest selection and retention stay isolated" {
  mongo_archive="$(latest_local_object mongo-archive)"
  postgres_archive="$(latest_local_object postgres-archive)"
  old_mongo="./dist/backup/mongo-archive/1000000000000-2020-01-01T000000.000Z.tar.gz"
  old_postgres="./dist/backup/postgres-archive/1000000000000-2020-01-01T000000.000Z.tar.gz"

  cp "$mongo_archive" "$old_mongo"
  touch -d '2020-01-01 UTC' "$old_mongo"
  run ./test/postgres-fixture.sh setup
  assert_success
  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-archive "${postgres_connection_args[@]}" --local-path=./dist/backup --expiry-days=1
  assert_success
  [ -f "$old_mongo" ]

  cp "$postgres_archive" "$old_postgres"
  touch -d '2020-01-01 UTC' "$old_postgres"
  run ./dist/mongo-archive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup --expiry-days=1
  assert_success
  [ -f "$old_postgres" ]

  touch -d '2100-01-01 UTC' "$mongo_archive"
  run ./test/postgres-fixture.sh reset
  assert_success
  run env POSTGRES__PASSWORD="$POSTGRES_PASSWORD" ./dist/postgres-unarchive "${postgres_connection_args[@]}" --local-path=./dist/backup
  assert_success
  run ./test/postgres-fixture.sh check
  assert_success
  [ "$output" = "restored" ]

  touch -d '2100-01-02 UTC' "$postgres_archive"
  run bash -lc 'go run ./test/testdb-drop.go | tail -n1'
  assert_success
  run ./dist/mongo-unarchive --uri="$DATABASE_URL" --db="$DATABASE_NAME" --local-path=./dist/backup
  assert_success
  run bash -lc 'go run ./test/testdb-check.go | tail -n1'
  assert_success
  [ "$output" = "found" ]
}
