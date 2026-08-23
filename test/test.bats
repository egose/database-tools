#!/usr/bin/env bats

set -a
source .env.test
set +a

mkdir -p ./dist/backup

FAKE_GCP_URL="http://localhost:$FAKE_GCP_PORT"

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

@test "tools build" {
  run bash -lc 'make build | tail -n1'
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
