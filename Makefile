SHELL := /usr/bin/env bash

.PHONY: build
build:
	CGO_ENABLED=0 go build -o dist/mongoarchive ./mongoarchive/main/mongoarchive.go
	CGO_ENABLED=0 go build -o dist/mongounarchive ./mongounarchive/main/mongounarchive.go

.PHONY: format
format:
	gofmt -w -s .

.PHONY: db
db:
	mkdir -p ../_mongodb/mongo-tools-ext
	mongod --dbpath ../_mongodb/mongo-tools-ext
