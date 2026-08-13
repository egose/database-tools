# Release Artifact Policy

Tagged releases publish only after the exact tagged commit passes the release verification gate.

## Verification Gate

The release gate for `v*.*.*` tags runs the following checks against the tagged SHA before either GitHub Releases or GHCR publication is allowed:

- `pnpm install --frozen-lockfile`
- `go test -shuffle=on ./...`
- `go test -race -shuffle=on ./...`
- `go vet ./...`
- `go mod verify`
- `govulncheck ./...`
- `docker buildx build --check .`
- `make build-all VERSION=<tag>`
- `make build-archive VERSION=<tag>`

## Published Release Assets

Binary releases publish the following alongside the archive artifacts:

- SHA-256 checksums for every release archive
- An SPDX JSON SBOM generated from the built `dist/` tree
- GitHub artifact provenance attestations for the published release files

## Vulnerability Policy

`govulncheck` is a hard gate for every reachable vulnerability with an upstream fix, and for every newly introduced reachable vulnerability. The only temporary exceptions are `GO-2025-4020`, `GO-2025-3605`, and `GO-2024-2698`, which currently have no upstream fix and are already tracked for removal by `ARCHIVE-01`.

Container images are scanned after the verified image is pushed. The image scan fails the workflow for fixable `HIGH` or `CRITICAL` findings. Findings without an upstream fix remain visible in the SARIF upload and must be tracked before the next release.

## Toolchain Contract

Release builds use Go `1.26.5` across `go.mod`, `.tool-versions`, and the Docker build stage so the local, CI, and containerized release paths use the same toolchain baseline.
