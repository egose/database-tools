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
	export MACHINE_HOST_IP=$$(hostname -I | awk '{print $$1}'); \
	docker-compose -f ./sandbox/docker-compose.yml up --build
