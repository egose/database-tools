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
- a real Docker image build followed by `mongo-archive --version`, `mongo-unarchive --version`, `postgres-archive --version`, `postgres-unarchive --version`, `pg_dump --version`, and `pg_restore --version` smoke tests, plus a containerized PostgreSQL archive/restore round trip
- `pnpm run release:build` with `VERSION=<tag>` and `SOURCE_DATE_EPOCH=<epoch>`
- `pnpm run release:verify` with the same environment, including independent reproducibility builds

## Published Release Assets

Binary releases publish the following alongside the archive artifacts:

- SHA-256 checksums for every release archive
- An SPDX JSON SBOM generated from the built `dist/` tree
- GitHub artifact provenance attestations for the published release files

## Vulnerability Policy

`govulncheck` is a hard gate for every reachable vulnerability with an upstream fix, and for every newly introduced reachable vulnerability. The release gate consumes `govulncheck -json` output and fails closed on scanner/runtime errors, malformed JSON, expired exception metadata, findings outside a documented exception's scan mode/module/package/symbol tuple, and any newly reachable finding.

Temporary exceptions must include a rationale, owner, creation date, review/expiry date, and live tracking reference in `internal/releasegovulncheck`. The current no-fix exceptions are limited to reachable `source`-mode findings in `github.com/mholt/archiver` and `github.com/nwaples/rardecode` symbols retained through the legacy archive dependency. They are owned by `release-security-maintainers`, were created on `2026-08-23`, expire for review on `2026-11-30`, and are tracked by `docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01`.

Container publication is scan-before-push. The publish workflow uses the digest-pinned shared `egose/actions/docker-build-push` action in gated mode to build a local image for the exact tag, while repository-specific pre-push checks smoke-test all four wrappers' `--version` output, verify `pg_dump` and `pg_restore` version output, run a containerized PostgreSQL archive/restore round trip, and verify the OCI `version` and `revision` labels. The action receives the release tag and tagged commit SHA as Docker build arguments, scans the local image before authenticating to GHCR, and pushes the immutable version tag and mutable major/minor tags only after the scan succeeds. Workflow summary evidence records the scanned local image config digest and verifies every published tag points at a manifest whose config digest is identical to the scanned image. The workflow requires `jq` so this verification fails closed. The repository `trivy.yaml` preserves the `HIGH` and `CRITICAL` severity threshold when Trivy generates SARIF, which otherwise removes the action-provided severity filter. The image scan fails the workflow for fixable `HIGH` or `CRITICAL` findings, so those findings create or mutate no GHCR tags. Findings without an upstream fix are excluded from this fixable-vulnerability gate and require separate review and tracking before the next release.

Published container images include OCI labels for title, description, source, version, revision, and license. The runtime image includes the exact PostgreSQL client package pinned by `POSTGRESQL_CLIENT_PACKAGE` in `Dockerfile`; the package currently selects PostgreSQL 18 clients from the digest-pinned Alpine 3.23 package source. Container PostgreSQL operations use the bundled `pg_dump` and `pg_restore`; operators should run a client major version compatible with the target server because an older PostgreSQL client may reject or incompletely support a newer PostgreSQL server. The shared action generates the image SBOM and attaches provenance and SBOM attestations to the verified published manifest digest after digest identity has been proven. GitHub artifact metadata storage records are disabled because registry attestations do not require them.

## Toolchain Contract

Release builds use Go `1.26.6` across `go.mod`, `.tool-versions`, the Docker build stage, and this policy. JavaScript tooling uses pnpm `11.24.0` across `package.json` and `.tool-versions`. `package.json` is marked private because npm publication is not a release product for this repository.

`scripts/check-supply-chain.sh` is the automated consistency gate. It fails release verification when Go or pnpm declarations drift, when Dockerfile bases or release-workflow container invocations are not digest-pinned, when the Dockerfile PostgreSQL client package is not version-pinned, when any declared asdf plugin is not pinned to a full commit SHA, when any Python lock entry lacks a hash, or when `package.json` stops being private.

Release-critical container inputs are content-addressed. The Docker build uses tagged digest references for the Go build image and Alpine runtime image, and the release SBOM generation image is invoked by tag plus digest. Renovate is configured to update Docker and Compose digests and the workflow `anchore/syft` digest. All asdf plugin repositories are checked out to explicit commit SHAs before tool installation, and Renovate is configured to propose updates for those git refs.

Python CI/release tools are installed from `requirements-lock.txt` with `pip --require-hashes`; `requirements.txt` remains the short input constraint file for regenerating that lock with `uv pip compile requirements.txt --generate-hashes --prerelease allow -o requirements-lock.txt`.

`@repo-toolkit/go-release` builds the 15-target matrix with two targets in flight. Builds use `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, and an empty Go build ID to remove local path, VCS dirty-state, and linker build-ID variance. Each binary embeds the exact release tag and target as `<version> <os>-<arch>`. CI caches compiled Go packages by runner, job, toolchain, and `go.sum`. Native release archives contain exactly `mongo-archive`, `mongo-unarchive`, `postgres-archive`, `postgres-unarchive`, and `LICENSE`, with sorted entries, owner/group `0`, numeric owners, deterministic gzip metadata, and all archive entry mtimes set from `SOURCE_DATE_EPOCH`, which defaults to `0` if not supplied. Native archives intentionally do not bundle PostgreSQL native clients; PostgreSQL archive and restore operations require compatible external `pg_dump` and `pg_restore` executables in `PATH`, while each wrapper's `--version` command remains independent of those clients. The build writes `database-tools-<version>-sha256.txt`, preserving the release tag exactly. With identical source and epoch, `pnpm run release:verify` validates every archive, checksum, exact host-compatible `--version` output for all four wrappers, and two independent release builds.
