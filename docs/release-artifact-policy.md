# Release Artifact Policy

Tagged releases publish only after the exact tagged commit passes the release verification gate.

## Verification Gate

The release gate for `v*.*.*` tags runs the following checks against the tagged SHA before either GitHub Releases or GHCR publication is allowed:

- `bash ./scripts/check-supply-chain.sh`
- `pnpm install --frozen-lockfile`
- `go test -shuffle=on ./...`
- `go test -race -shuffle=on ./...`
- `go vet ./...`
- `go mod verify`
- `govulncheck -json ./...` through `scripts/release-govulncheck.sh`
- `docker buildx build --check .`
- a real Docker image build followed by `mongo-archive --version` and `mongo-unarchive --version` smoke tests
- `make build-all VERSION=<tag>`
- `make build-archive VERSION=<tag>` (each archive includes `LICENSE`)
- `make check-reproducible-archives VERSION=<tag> SOURCE_DATE_EPOCH=<epoch>`

## Published Release Assets

Binary releases publish the following alongside the archive artifacts:

- SHA-256 checksums for every release archive
- An SPDX JSON SBOM generated from the built `dist/` tree
- GitHub artifact provenance attestations for the published release files

## Vulnerability Policy

`govulncheck` is a hard gate for every reachable vulnerability with an upstream fix, and for every newly introduced reachable vulnerability. The release gate consumes `govulncheck -json` output and fails closed on scanner/runtime errors, malformed JSON, expired exception metadata, findings outside a documented exception's scan mode/module/package/symbol tuple, and any newly reachable finding.

Temporary exceptions must include a rationale, owner, creation date, review/expiry date, and live tracking reference in `internal/releasegovulncheck`. The current no-fix exceptions are limited to reachable `source`-mode findings in `github.com/mholt/archiver` and `github.com/nwaples/rardecode` symbols retained through the legacy archive dependency. They are owned by `release-security-maintainers`, were created on `2026-08-23`, expire for review on `2026-11-30`, and are tracked by `docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01`.

Container publication is scan-before-push. The publish workflow builds a local image for the exact tag, passes the release tag and tagged commit SHA into the Docker build, smoke-tests both binaries' `--version` output, verifies the OCI `version` and `revision` labels, and scans that local image before authenticating to GHCR. Only after the scan succeeds does the workflow push the immutable version tag and mutable major/minor tags. Workflow summary evidence records the scanned local image config digest and verifies every published tag points at a manifest whose config digest is identical to the scanned image. The image scan fails the workflow for fixable `HIGH` or `CRITICAL` findings, so those findings create or mutate no GHCR tags. Findings without an upstream fix remain visible in the SARIF upload and must be tracked before the next release.

Published container images include OCI labels for title, description, source, version, revision, and license. Image SBOM and provenance attestations are generated for the verified published manifest digest after digest identity has been proven.

## Toolchain Contract

Release builds use Go `1.26.6` across `go.mod`, `.tool-versions`, the Docker build stage, and this policy. JavaScript tooling uses pnpm `11.17.0` across `package.json` and `.tool-versions`. `package.json` is marked private because npm publication is not a release product for this repository.

`scripts/check-supply-chain.sh` is the automated consistency gate. It fails release verification when Go or pnpm declarations drift, when Dockerfile bases or release-workflow container invocations are not digest-pinned, when any declared asdf plugin is not pinned to a full commit SHA, when any Python lock entry lacks a hash, or when `package.json` stops being private.

Release-critical container inputs are content-addressed. The Docker build uses tagged digest references for the Go build image and Alpine runtime image, and the release SBOM generation image is invoked by tag plus digest. Renovate is configured to update Docker and Compose digests and the workflow `anchore/syft` digest. All asdf plugin repositories are checked out to explicit commit SHAs before tool installation, and Renovate is configured to propose updates for those git refs.

Python CI/release tools are installed from `requirements-lock.txt` with `pip --require-hashes`; `requirements.txt` remains the short input constraint file for regenerating that lock with `uv pip compile requirements.txt --generate-hashes --prerelease allow -o requirements-lock.txt`.

Builds use `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, and an empty Go build ID to remove local path, VCS dirty-state, and linker build-ID variance. Release archives are written with sorted entries, owner/group `0`, numeric owners, gzip `-n`, and all archive entry mtimes set from `SOURCE_DATE_EPOCH`, which defaults to `0` if not supplied. With identical source and epoch, `make check-reproducible-archives VERSION=<tag> SOURCE_DATE_EPOCH=<epoch>` builds two independent trees and fails if any release archive SHA-256 differs.
