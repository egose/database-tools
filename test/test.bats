#!/usr/bin/env bats

source .env.test
mkdir -p ./dist/backup

FAKE_GCP_URL="http://localhost:$FAKE_GCP_PORT"

@test "tools build" {
  result="$(make build | tail -n1)"
  [ "$result" == "complete" ]
}

@test "mongodb setup" {
  result="$(go run test/testdb-setup.go | tail -n1)"
  [ "$result" == "ready" ]
}

@test "data found before archive" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}

# Local Archive
@test "[local disk] archive" {
  result=$(./dist/mongo-archive --uri=$DATABASE_URL --db=$DATABASE_NAME --local-path=./dist/backup 2>&1 | tail -n1)
  [[ "$result" == *"Archive completed successfully"* ]]
}

@test "[local disk] drop" {
  result="$(go run test/testdb-drop.go | tail -n1)"
  [ "$result" == "dropped" ]
}

@test "[local disk] data notfound" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "notfound" ]
}

@test "[local disk] unarchive" {
  result=$(./dist/mongo-unarchive --uri=$DATABASE_URL --db=$DATABASE_NAME --local-path=./dist/backup 2>&1 | tail -n1)
  [[ "$result" == *"Unarchive completed successfull"* ]]
}

@test "[local disk] data found after unarchive" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}

# S3 Archive
@test "[S3 bucket] archive" {
  result=$(./dist/mongo-archive --uri=$DATABASE_URL --db=$DATABASE_NAME --aws-endpoint=$MINIO_URL --aws-access-key-id=$MINIO_ACCESS_KEY --aws-secret-access-key=$MINIO_SECRET_KEY --aws-bucket=$MINIO_BUCKET --aws-s3-force-path-style=true 2>&1 | tail -n1)
  [[ "$result" == *"Archive completed successfully"* ]]
}

@test "[S3 bucket] drop" {
  result="$(go run test/testdb-drop.go | tail -n1)"
  [ "$result" == "dropped" ]
}

@test "[S3 bucket] data notfound" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "notfound" ]
}

@test "[S3 bucket] unarchive" {
  result=$(./dist/mongo-unarchive --uri=$DATABASE_URL --db=$DATABASE_NAME --aws-endpoint=$MINIO_URL --aws-access-key-id=$MINIO_ACCESS_KEY --aws-secret-access-key=$MINIO_SECRET_KEY --aws-bucket=$MINIO_BUCKET --aws-s3-force-path-style=true 2>&1 | tail -n1)
  [[ "$result" == *"Unarchive completed successfull"* ]]
}

@test "[S3 bucket] data found after unarchive" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}

# Azure Storage Archive
@test "[Azure Storage] create container" {
  az storage container create -n testcontainer --connection-string "DefaultEndpointsProtocol=http;AccountName=$AZURITE_ACCOUNT_NAME;AccountKey=$AZURITE_ACCOUNT_KEY;BlobEndpoint=$AZURITE_URL/$AZURITE_ACCOUNT_NAME;"
}

@test "[Azure Storage] archive" {
  result=$(./dist/mongo-archive --uri=$DATABASE_URL --db=$DATABASE_NAME --az-endpoint=$AZURITE_URL --az-account-name=$AZURITE_ACCOUNT_NAME --az-account-key=$AZURITE_ACCOUNT_KEY --az-container-name=$AZURITE_CONTAINER 2>&1 | tail -n1)
  [[ "$result" == *"Archive completed successfully"* ]]
}

@test "[Azure Storage] drop" {
  result="$(go run test/testdb-drop.go | tail -n1)"
  [ "$result" == "dropped" ]
}

@test "[Azure Storage] data notfound" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "notfound" ]
}

@test "[Azure Storage] unarchive" {
  result=$(./dist/mongo-unarchive --uri=$DATABASE_URL --db=$DATABASE_NAME --az-endpoint=$AZURITE_URL --az-account-name=$AZURITE_ACCOUNT_NAME --az-account-key=$AZURITE_ACCOUNT_KEY --az-container-name=$AZURITE_CONTAINER 2>&1 | tail -n1)
  [[ "$result" == *"Unarchive completed successfull"* ]]
}

@test "[Azure Storage] data found after unarchive" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}

@test "[GCP Storage] archive" {
  result=$(STORAGE_EMULATOR_HOST="$FAKE_GCP_URL" ./dist/mongo-archive --uri=$DATABASE_URL --db=$DATABASE_NAME --gcp-endpoint=$FAKE_GCP_URL/storage/v1/ --gcp-bucket=$FAKE_GCP_BUCKET 2>&1 | tail -n1)
  [[ "$result" == *"Archive completed successfully"* ]]
}

@test "[GCP Storage] drop" {
  result="$(go run test/testdb-drop.go | tail -n1)"
  [ "$result" == "dropped" ]
}

@test "[GCP Storage] data notfound" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "notfound" ]
}

@test "[GCP Storage] unarchive" {
  result=$(STORAGE_EMULATOR_HOST="$FAKE_GCP_URL" ./dist/mongo-unarchive --uri=$DATABASE_URL --db=$DATABASE_NAME --gcp-endpoint=$FAKE_GCP_URL/storage/v1/ --gcp-bucket=$FAKE_GCP_BUCKET 2>&1 | tail -n1)
  [[ "$result" == *"Unarchive completed successfull"* ]]
}

@test "[GCP Storage] data found after unarchive" {
  result="$(go run test/testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}
