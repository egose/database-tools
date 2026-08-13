SHELL := /usr/bin/env bash
DIST_DIR := dist
TOOLS_DIR := tmp/bin
GOVULNCHECK := $(TOOLS_DIR)/govulncheck

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

.PHONY: build-all build-single build-archive release-verify

build-all:
	@set -euo pipefail; \
	for pair in $(OS_ARCH_PAIRS); do \
		$(MAKE) --no-print-directory build-single OS_ARCH=$$pair VERSION=$(VERSION); \
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
	DIR="dist/$$OS-$$ARCH"; \
	mkdir -p $$DIR; \
	EXT=$$(if [ "$$OS" = "windows" ]; then echo ".exe"; else echo ""; fi); \
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build -ldflags "-X \"main.version=$(VERSION) $${OS}-$${ARCH}\"" -o $$DIR/mongo-archive$$EXT ./mongoarchive/main/mongoarchive.go &&\
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build -ldflags "-X \"main.version=$(VERSION) $${OS}-$${ARCH}\"" -o $$DIR/mongo-unarchive$$EXT ./mongounarchive/main/mongounarchive.go
	echo complete
$(GOVULNCHECK):
	@mkdir -p "$(TOOLS_DIR)"
	GOBIN="$(CURDIR)/$(TOOLS_DIR)" go install golang.org/x/vuln/cmd/govulncheck@v1.1.4

release-verify: $(GOVULNCHECK)
	pnpm install --frozen-lockfile
	go test -shuffle=on ./...
	go test -race -shuffle=on ./...
	go vet ./...
	go mod verify
	./scripts/release-govulncheck.sh "$(CURDIR)/$(GOVULNCHECK)" ./...
	docker buildx build --check .
	$(MAKE) --no-print-directory build-all VERSION=$(VERSION)
	$(MAKE) --no-print-directory build-archive VERSION=$(VERSION)

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION)" -o dist/mongo-archive ./mongoarchive/main/mongoarchive.go
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION)" -o dist/mongo-unarchive ./mongounarchive/main/mongounarchive.go
	echo complete

build-archive:
	@set -euo pipefail; \
	for pair in $(OS_ARCH_PAIRS); do \
		os=$${pair%%:*}; \
		arch=$${pair##*:}; \
		dir="$(DIST_DIR)/$$os-$$arch"; \
		archive="$(DIST_DIR)/$(PREFIX)-$$os-$$arch.tar.gz"; \
		test -d "$$dir"; \
		tar -czvf "$$archive" -C "$$dir" .; \
		test -s "$$archive"; \
	done; \
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

	export MACHINE_HOST_IP=$$(hostname -I | awk '{print $$1}'); \
	docker-compose --env-file .env.test -f ./sandbox/docker-compose.yml up --build

.PHONY: sandbox-down
sandbox-down:
	docker-compose --env-file .env.test -f ./sandbox/docker-compose.yml down
