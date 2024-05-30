SHELL := /usr/bin/env bash

.PHONY: build
build:
	CGO_ENABLED=0 go build -o dist/mongo-archive ./mongoarchive/main/mongoarchive.go
	CGO_ENABLED=0 go build -o dist/mongo-unarchive ./mongounarchive/main/mongounarchive.go
	echo complete

.PHONY: format
format:
	gofmt -w -s .

.PHONY: db
db:
	mkdir -p ../_mongodb/database-tools
	mongod --dbpath ../_mongodb/database-tools

.PHONY: sandbox
sandbox:
	mkdir -p ./sandbox/mnt/mongodb
	mkdir -p ./sandbox/mnt/minio
	mkdir -p ./sandbox/mnt/azurite
	mkdir -p ./sandbox/mnt/fake-gcs-server

	export MACHINE_HOST_IP=$$(hostname -I | awk '{print $$1}'); \
	docker-compose --env-file .env.test -f ./sandbox/docker-compose.yml up --build

.PHONY: sandbox-down
sandbox-down:
	docker-compose --env-file .env.test -f ./sandbox/docker-compose.yml down
