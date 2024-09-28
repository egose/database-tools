SHELL := /usr/bin/env bash

DIRS := $(wildcard dist/*/)
ARCHIVES := $(patsubst dist/%/,dist/%.tar.gz,$(DIRS))

OS_ARCH_PAIRS := \
    linux:amd64 \
    linux:arm64 \
    linux:386 \
    linux:arm \
    linux:mips \
    linux:mips64 \
    windows:amd64 \
    windows:386 \
    darwin:amd64 \
    darwin:arm64 \
    freebsd:amd64 \
    freebsd:arm64 \
    openbsd:amd64 \
    openbsd:arm64 \
    netbsd:amd64

# See https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04#step-4-building-executables-for-different-architectures
build-all:
	@$(foreach pair, $(OS_ARCH_PAIRS), $(MAKE) build-single OS_ARCH=$(pair);)

build-single:
	@set -e; \
	OS_ARCH=$(OS_ARCH); \
	OS=$$(echo $$OS_ARCH | cut -d: -f1); \
	ARCH=$$(echo $$OS_ARCH | cut -d: -f2); \
	echo "Building for OS=$$OS and ARCH=$$ARCH" &&\
	DIR="dist/$$OS-$$ARCH"; \
	mkdir -p $$DIR; \
	EXT=$$(if [ "$$OS" = "windows" ]; then echo ".exe"; else echo ""; fi); \
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build -o $$DIR/mongo-archive$$EXT ./mongoarchive/main/mongoarchive.go &&\
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build -o $$DIR/mongo-unarchive$$EXT ./mongounarchive/main/mongounarchive.go
	echo complete

.PHONY: build
build:
	CGO_ENABLED=0 go build -o dist/mongo-archive ./mongoarchive/main/mongoarchive.go
	CGO_ENABLED=0 go build -o dist/mongo-unarchive ./mongounarchive/main/mongounarchive.go
	echo complete

build-archive: $(ARCHIVES)
dist/%.tar.gz: dist/%
	tar -czvf $@ -C dist/$* .

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
