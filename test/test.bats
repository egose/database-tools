#!/usr/bin/env bats

mkdir -p ./dist/backup

@test "tools build" {
  result="$(make build | tail -n1)"
  [ "$result" == "complete" ]
}

@test "mongodb setup" {
  result="$(go run testdb-setup.go | tail -n1)"
  [ "$result" == "ready" ]
}

@test "data found before archive" {
  result="$(go run testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}

@test "archive" {
  result=$(./dist/mongo-archive --uri=mongodb://localhost:27017 --db=testdb --local-path=./dist/backup 2>&1 | tail -n1)
  [[ "$result" == *"Archive completed successfully"* ]]
}

@test "drop" {
  result="$(go run testdb-drop.go | tail -n1)"
  [ "$result" == "dropped" ]
}

@test "data notfound" {
  result="$(go run testdb-check.go | tail -n1)"
  [ "$result" == "notfound" ]
}

@test "unarchive" {
  result=$(./dist/mongo-unarchive --uri=mongodb://localhost:27017 --db=testdb --local-path=./dist/backup 2>&1 | tail -n1)
  [[ "$result" == *"Unarchive completed successfull"* ]]
}

@test "data found after unarchive" {
  result="$(go run testdb-check.go | tail -n1)"
  [ "$result" == "found" ]
}
