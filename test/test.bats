#!/usr/bin/env bats

source .env.test
mkdir -p ./dist/backup

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
  result="$(az storage container create -n testcontainer --connection-string 'DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;' | tail -n1)"
  [ "$result" == "}" ]
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
