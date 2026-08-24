SHELL := /usr/bin/env bash
DIST_DIR := dist
TOOLS_DIR := tmp/bin
GOVULNCHECK := $(TOOLS_DIR)/govulncheck
GO_BUILD_FLAGS := -trimpath -buildvcs=false
SOURCE_DATE_EPOCH ?= 0
VERSION := localdev

.PHONY: check-toolchain release-verify test-asdf
$(GOVULNCHECK):
	@mkdir -p "$(TOOLS_DIR)"
	GOBIN="$(CURDIR)/$(TOOLS_DIR)" go install golang.org/x/vuln/cmd/govulncheck@v1.1.4

check-toolchain:
	bash ./scripts/check-supply-chain.sh

test-asdf:
	bats test/asdf-plugin.bats

release-verify: check-toolchain test-asdf $(GOVULNCHECK)
	pnpm install --frozen-lockfile
	go test -shuffle=on ./...
	go test -race -shuffle=on ./...
	go vet ./...
	go mod verify
	bash ./scripts/release-govulncheck.sh "$(CURDIR)/$(GOVULNCHECK)" ./...
	docker buildx build --check .
	@set -euo pipefail; \
	image="database-tools:release-verify"; \
	trap 'docker image rm -f "$$image" >/dev/null 2>&1 || true' EXIT; \
	docker build --build-arg VERSION="$(VERSION)" --tag "$$image" .; \
	docker run --rm "$$image" mongo-archive --version; \
	docker run --rm "$$image" mongo-unarchive --version
	VERSION="$(VERSION)" SOURCE_DATE_EPOCH="$(SOURCE_DATE_EPOCH)" pnpm run release:build
	VERSION="$(VERSION)" SOURCE_DATE_EPOCH="$(SOURCE_DATE_EPOCH)" pnpm run release:verify

.PHONY: build
build:
	mkdir -p "$(DIST_DIR)"
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags "-buildid= -X main.version=$(VERSION)" -o "$(DIST_DIR)/mongo-archive" ./mongoarchive/main/mongoarchive.go
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags "-buildid= -X main.version=$(VERSION)" -o "$(DIST_DIR)/mongo-unarchive" ./mongounarchive/main/mongounarchive.go
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
	mkdir -p ./sandbox/mnt/{mongodb,minio,azurite,fake-gcs-server}

	docker-compose --env-file .env.test -f ./sandbox/docker-compose.yml up --build

.PHONY: sandbox-down
sandbox-down:
	docker-compose --env-file .env.test -f ./sandbox/docker-compose.yml down
