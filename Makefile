SHELL := /usr/bin/env bash
DIST_DIR := dist
TOOLS_DIR := tmp/bin
GOVULNCHECK := $(TOOLS_DIR)/govulncheck
GO_BUILD_FLAGS := -trimpath -buildvcs=false
SOURCE_DATE_EPOCH ?= 0
BUILD_JOBS ?= 2

PREFIX := database-tools
VERSION := localdev

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

.PHONY: build-all build-single build-archive check-toolchain check-reproducible-archives release-verify test-asdf

build-all:
	@set -euo pipefail; \
	printf '%s\n' $(OS_ARCH_PAIRS) | \
		xargs -P "$(BUILD_JOBS)" -I {} $(MAKE) --no-print-directory build-single OS_ARCH={} VERSION="$(VERSION)"; \
	for pair in $(OS_ARCH_PAIRS); do \
		os=$${pair%%:*}; \
		arch=$${pair##*:}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		test -s "$(DIST_DIR)/$$os-$$arch/mongo-archive$$ext"; \
		test -s "$(DIST_DIR)/$$os-$$arch/mongo-unarchive$$ext"; \
	done; \
	echo complete

build-single:
	@set -e; \
	OS_ARCH=$(OS_ARCH); \
	OS=$$(echo $$OS_ARCH | cut -d: -f1); \
	ARCH=$$(echo $$OS_ARCH | cut -d: -f2); \
	echo "Building for OS=$$OS and ARCH=$$ARCH" &&\
		DIR="$(DIST_DIR)/$$OS-$$ARCH"; \
		mkdir -p $$DIR; \
		EXT=$$(if [ "$$OS" = "windows" ]; then echo ".exe"; else echo ""; fi); \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build $(GO_BUILD_FLAGS) -ldflags "-buildid= -X \"main.version=$(VERSION) $${OS}-$${ARCH}\"" -o $$DIR/mongo-archive$$EXT ./mongoarchive/main/mongoarchive.go &&\
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build $(GO_BUILD_FLAGS) -ldflags "-buildid= -X \"main.version=$(VERSION) $${OS}-$${ARCH}\"" -o $$DIR/mongo-unarchive$$EXT ./mongounarchive/main/mongounarchive.go
	echo complete
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
	$(MAKE) --no-print-directory build-all VERSION=$(VERSION)
	$(MAKE) --no-print-directory build-archive VERSION=$(VERSION)
	$(MAKE) --no-print-directory check-reproducible-archives VERSION=$(VERSION) SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)

.PHONY: build
build:
	mkdir -p "$(DIST_DIR)"
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags "-buildid= -X main.version=$(VERSION)" -o "$(DIST_DIR)/mongo-archive" ./mongoarchive/main/mongoarchive.go
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags "-buildid= -X main.version=$(VERSION)" -o "$(DIST_DIR)/mongo-unarchive" ./mongounarchive/main/mongounarchive.go
	echo complete

build-archive:
	@set -euo pipefail; \
	for pair in $(OS_ARCH_PAIRS); do \
		os=$${pair%%:*}; \
		arch=$${pair##*:}; \
		dir="$(DIST_DIR)/$$os-$$arch"; \
		archive="$(DIST_DIR)/$(PREFIX)-$$os-$$arch.tar.gz"; \
		test -d "$$dir"; \
		tar --sort=name --mtime="@$(SOURCE_DATE_EPOCH)" --owner=0 --group=0 --numeric-owner -cf - -C "$$dir" . -C "$(CURDIR)" LICENSE | gzip -n > "$$archive"; \
		test -s "$$archive"; \
	done; \
	echo complete

check-reproducible-archives:
	@set -euo pipefail; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	compgen -G "$(DIST_DIR)/*.tar.gz" >/dev/null; \
	$(MAKE) --no-print-directory build-all build-archive DIST_DIR="$$tmp/rebuilt" VERSION=$(VERSION) SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH); \
	sha256sum "$(DIST_DIR)"/*.tar.gz | sed "s#$(DIST_DIR)/##" | sort -k2 > "$$tmp/release.sha256"; \
	sha256sum "$$tmp"/rebuilt/*.tar.gz | sed "s#$$tmp/rebuilt/##" | sort -k2 > "$$tmp/rebuilt.sha256"; \
	diff -u "$$tmp/release.sha256" "$$tmp/rebuilt.sha256"


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
