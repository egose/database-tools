# PostgreSQL Backup And Restore Support

Created: 2026-08-24T05:55:13-07:00

Related task files:

- `docs/tasks/20260813-105920-multi-backend-archive-contract-remediation.md`
- `docs/tasks/20260823-112531-post-remediation-codebase-health.md`

## Objective

Add production-ready PostgreSQL backup and restore commands alongside the existing MongoDB commands. Reuse the repository's storage, retention, runtime safety, scheduling, notification, release, and documentation foundations without changing existing MongoDB public behavior.

The new commands are:

- `postgres-archive`
- `postgres-unarchive`

This document is written for sub-agents that have no access to the originating discussion. Agents must keep it current as implementation reveals new facts.

## Approved Product Contract

The following decisions are resolved and must not be reopened during implementation unless a concrete technical blocker is recorded in this file:

1. Native release archives contain the Go command wrappers but do not bundle PostgreSQL native tools. Native users must provide compatible `pg_dump` and `pg_restore` executables in `PATH`.
2. The published container image includes pinned PostgreSQL client tools and verifies their presence during release.
3. Initial backup scope is one PostgreSQL database. Cluster-global roles and tablespaces are out of scope.
4. Restore targets an existing database. It does not clean existing objects or create a database by default.
5. PostgreSQL managed objects use the default prefix `postgres-archive/`. They must never participate in MongoDB latest-selection or retention under `mongo-archive/`.
6. PostgreSQL uses `pg_dump` custom format inside the repository's existing outer `.tar.gz` transport format.
7. Restore uses `pg_restore --exit-on-error`. A nonzero process result is failure; the tool must not imply transactional rollback unless an explicit transaction option was selected and supported.
8. PostgreSQL environment lookup uses command-specific variables first, then shared PostgreSQL variables, then unprefixed variables:
   - archive: `POSTGRESARCHIVE__KEY`, `POSTGRES__KEY`, `KEY`
   - restore: `POSTGRESUNARCHIVE__KEY`, `POSTGRES__KEY`, `KEY`
9. Existing `mongo-archive` and `mongo-unarchive` command names, flags, environment precedence, default prefix, archive format, release behavior, and runtime contracts remain compatible.

## Scope

- PostgreSQL archive and restore CLI configuration.
- Secure invocation and cancellation of `pg_dump` and `pg_restore`.
- A versioned PostgreSQL archive manifest and format validation.
- Existing local, AWS S3, Azure Blob, and Google Cloud Storage backends.
- Existing multi-backend upload-before-retention semantics.
- Archive cron scheduling and existing notification transports.
- Restore latest-object and explicit-object selection.
- Unit, integration, race, documentation, container, installer, and release verification.
- Native release and container documentation for PostgreSQL client compatibility.

## Working Rules And Non-Goals

- Preserve unrelated worktree changes. Never reset, revert, or overwrite files outside an assigned task.
- Set a task to `in_progress` before implementation. Set it to `completed` only after all required verification passes, and append completion evidence.
- Read the related task files before editing shared archive, storage, runtime, CI, or release behavior.
- Prefer the smallest shared enforcement point. Do not rewrite the MongoDB pipelines into a broad database framework.
- Add compatibility wrappers where required to preserve existing MongoDB defaults; do not silently migrate MongoDB configuration or object names.
- Keep credentials out of process arguments, logs, errors, notifications, manifests, test output, and retained workspaces.
- Preserve private workspaces, atomic downloads, safe extraction limits, reverse-order cleanup, operation timeouts, explicit restore backend selection, and primary-plus-cleanup error preservation.
- Preserve the strict multi-backend contract: attempt all uploads before any retention, do not run retention after an upload-phase failure, and report partial state explicitly.
- Do not accept arbitrary shell command strings or invoke PostgreSQL tools through a shell.
- Do not add `pg_dumpall`, role restoration, tablespace restoration, physical backups, point-in-time recovery, replication-slot handling, or WAL archiving in this phase.
- Do not promise that native release archives are self-contained for PostgreSQL operations.
- Do not add compatibility aliases for speculative command or environment names.

## Current Architecture And Confirmed Constraints

- MongoDB archive orchestration is in `mongoarchive/main/mongoarchive.go`; MongoDB restore orchestration is in `mongounarchive/main/mongounarchive.go`.
- MongoDB tools are linked through `github.com/mongodb/mongo-tools`, while PostgreSQL support must execute external client binaries.
- Storage capabilities and provider implementations are reusable under `storage/` and are constructed through `internal/toolconfig/shared.go`.
- `storage/backup_contract.go` currently defines `DefaultBackupPrefix = "mongo-archive/"` and recognizes the current managed `.tar.gz` filename contract.
- Shared storage flag binding currently supplies the MongoDB default prefix from `internal/toolconfig/shared.go`.
- Cleanup, operation-context, and backend-close behavior is shared through `internal/toolruntime/runtime.go`.
- Tar creation and defensive extraction are implemented in `utils/file.go`.
- Archive multi-backend delivery behavior currently lives in `mongoarchive/main/mongoarchive.go` and is not reusable by another command package.
- Notification transports under `notification/` are reusable, but archive construction and scheduling are coupled to MongoDB archive configuration and orchestration.
- `Makefile` and `go-release.mjs` build specific command source files and enumerate only two binaries.
- `bin/install` and `test/asdf-plugin.bats` currently enforce a two-binary release archive contract.
- The runtime image in `Dockerfile` contains only `tzdata`; it does not contain PostgreSQL clients.
- CLI documentation is generated and checked through `internal/flagdocs`, with public environment lookups checked by `internal/flagdocs/envdocs_test.go`.
- Stateful integration coverage is in `test/test.bats` with services under `sandbox/docker-compose.yml` and `sandbox/docker-compose-ci.yml`.

## Baseline Verification

The worktree was clean at task creation on 2026-08-24. Repository source, tests, build configuration, release configuration, Dockerfile, and existing task documents were inspected.

No build or test command was run while creating this task. Establish and record the baseline before changing code:

```sh
go test -shuffle=on ./...
go test -race -shuffle=on ./...
go vet ./...
git diff --check
```

The following checks require repository tools or Docker and should be run where available:

```sh
bats --print-output-on-failure test/test.bats
make test-asdf
make release-verify VERSION=<test-version>
```

Prior task evidence records Docker as unavailable in one local WSL environment. Docker-dependent work may use CI evidence, but the exact blocker and CI run must be recorded rather than marking local verification as passed.

BASE-01 baseline evidence (2026-08-24), recorded before production changes:

- `go test -shuffle=on ./...`: passed.
- `go test -race -shuffle=on ./...`: passed.
- `go vet ./...`: passed.
- `git diff --check`: passed.

## Priority Definitions

- P0: credential disclosure, cross-database backup deletion/selection, or credible backup data loss.
- P1: incorrect archive/restore behavior, unsafe restore semantics, cancellation failure, release breakage, or misleading success.
- P2: maintainability, documentation, portability, or test-depth improvement required for a supportable release.

## Execution Waves

1. Establish baseline evidence and lock archive, configuration, and compatibility contracts.
2. Parameterize managed-object identity and implement the secure PostgreSQL process boundary.
3. Implement PostgreSQL archive and restore pipelines.
4. Add stateful PostgreSQL integration and cross-database isolation coverage.
5. Extend native packaging, container packaging, generated references, and operator documentation.
6. Perform independent runtime, security, compatibility, and release review.

## Detailed Tasks

### Task BASE-01: Lock Compatibility And Archive Format Contracts

Status: completed

Priority: P1

Suggested agent: senior Go compatibility and test engineer

Dependencies: none

Primary ownership:

- focused characterization tests under `storage/`
- focused existing-command tests under `mongoarchive/` and `mongounarchive/`
- a small PostgreSQL archive-format package and tests if needed to encode the contract
- this task file for baseline evidence

Finding:

The reusable storage and runtime foundations are mature, but key managed-object defaults and orchestration live in MongoDB-specific paths. Adding PostgreSQL through a broad refactor risks changing MongoDB object selection, retention, environment precedence, cleanup, and delivery semantics. The PostgreSQL inner artifact also needs an explicit identity before producer and consumer agents work independently.

References:

- `storage/backup_contract.go`
- `storage/selection_test.go`
- `storage/retention_test.go`
- `mongoarchive/main/mongoarchive_test.go`
- `mongounarchive/main/mongounarchive_test.go`
- `utils/date.go`
- `utils/file.go`

Implementation requirements:

1. Run and record baseline commands before production changes.
2. Add characterization coverage proving MongoDB's default prefix remains `mongo-archive/`, generated object names remain compatible, latest selection remains prefix-scoped, and retention does not cross a configured prefix.
3. Define a versioned PostgreSQL archive payload containing exactly one custom-format dump at a stable relative path and a JSON manifest at a stable relative path.
4. Require the manifest to identify format version, database family, dump format, creation time, source database name, and producing PostgreSQL client version.
5. Explicitly prohibit credentials and full password-bearing connection strings in the manifest.
6. Define validation behavior for missing, malformed, unsupported-version, wrong-database-family, duplicate, or path-conflicting payload entries.
7. Preserve the outer managed filename and `.tar.gz` transport contract so existing storage providers remain artifact-agnostic.
8. Do not change MongoDB runtime behavior in this task.

Acceptance criteria:

- Tests prove MongoDB and PostgreSQL prefixes cannot select or retain each other's managed objects.
- The PostgreSQL manifest schema and validation failures are represented by deterministic tests.
- A PostgreSQL restore cannot mistake a MongoDB archive or arbitrary tarball for a valid PostgreSQL payload.
- Baseline command results and any environmental blockers are recorded in this file.
- `go test ./storage ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main -count=1` passes.

Completion evidence:

- Changed: `internal/postgresarchiveformat/format.go`, `internal/postgresarchiveformat/format_test.go`, `storage/selection_test.go`, `storage/retention_test.go`, `utils/date_test.go`, `mongoarchive/flags_test.go`, `mongounarchive/flags_test.go`, and this task file.
- Implemented: a version 1 PostgreSQL payload contract with exactly `manifest.json` and `database.dump`, strict six-field JSON manifest validation, PostgreSQL custom-format magic validation, credential/connection-string exclusion, and deterministic rejection of missing, malformed, unsupported, wrong-family, duplicate, conflicting, extra, MongoDB, and arbitrary payloads. The existing `.tar.gz` managed filename contract and MongoDB runtime code are unchanged.
- Verified: `go test ./storage ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main -count=1` passed.
- Verified: `go test ./internal/postgresarchiveformat ./utils -count=1` passed.
- Verified: `go test -race ./internal/postgresarchiveformat ./storage ./utils ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main -count=1` passed.
- Verified: `go test -shuffle=on ./...` passed.
- Verified: `go vet ./...` and `git diff --check` passed.
- Blockers: none.
- Residual risk: this task validates synthetic PostgreSQL custom-format payloads by their stable archive layout, manifest, and `PGDMP` signature; real `pg_dump` production and `pg_restore` consumption are intentionally deferred to PROCESS-01, ARCHIVE-01, RESTORE-01, and INTEGRATION-01.

### Task STORAGE-01: Parameterize Managed Backup Prefixes Without MongoDB Regression

Status: completed

Priority: P1

Suggested agent: Go storage contract engineer

Dependencies: BASE-01

Primary ownership:

- `storage/backup_contract.go`
- `storage/selection.go`
- storage selection, listing, and retention tests
- storage binding portions of `internal/toolconfig/shared.go`
- focused `internal/toolconfig` tests

Finding:

The managed storage contract defaults to `mongo-archive/`, and shared storage flag binding applies that default. Reusing it unchanged would allow PostgreSQL automatic restore selection and retention to operate in MongoDB's namespace.

References:

- `storage/backup_contract.go`
- `storage/selection.go`
- `storage/selection_test.go`
- `storage/retention_test.go`
- `internal/toolconfig/shared.go`
- `internal/toolconfig/shared_test.go`

Implementation requirements:

1. Allow command configuration to supply its database-specific default prefix.
2. Preserve an existing API or narrow compatibility wrapper that keeps the MongoDB default exactly `mongo-archive/`.
3. Define and use `postgres-archive/` as the PostgreSQL default.
4. Keep explicit custom prefixes supported and normalized consistently.
5. Preserve explicit object lookup behavior for full object keys and generated filenames.
6. Keep latest selection and retention restricted to matching managed objects immediately under the selected prefix.
7. Do not duplicate provider logic or add database-family checks to individual storage backends.

Acceptance criteria:

- Existing MongoDB flag and storage tests pass unchanged or with only additive assertions.
- PostgreSQL configuration defaults to `postgres-archive/` with no fallback to `mongo-archive/`.
- Mixed-prefix tests prove latest selection and retention remain isolated in the same local directory or bucket.
- Explicit custom-prefix behavior remains supported for both database families.
- `go test ./storage ./internal/toolconfig ./mongoarchive ./mongounarchive -count=1` passes.

Completion evidence:

- Changed: `storage/backup_contract.go`, `storage/local_test.go`, `storage/retention_test.go`, `internal/toolconfig/shared.go`, `internal/toolconfig/shared_test.go`, and this task file.
- Implemented: explicit `mongo-archive/` and `postgres-archive/` storage defaults, default-aware prefix normalization and object-name construction, and a default-aware storage flag binder. Existing `DefaultBackupPrefix`, `NormalizeBackupPrefix`, `BuildBackupObjectName`, and `BindStorageFlags` APIs remain MongoDB-compatible wrappers.
- Covered: MongoDB and PostgreSQL binding defaults, no PostgreSQL fallback to the MongoDB namespace, normalized custom prefixes under either database default, generated object names, and mixed-prefix latest selection and retention in one local storage root. Explicit full-key and generated-filename lookup behavior remains covered by the existing storage tests.
- Verified: `go test ./storage ./internal/toolconfig ./mongoarchive ./mongounarchive -count=1` passed for all four packages.
- Verified: `go test -race ./storage ./internal/toolconfig ./mongoarchive ./mongounarchive -count=1` passed for all four packages.
- Verified: `go vet ./storage ./internal/toolconfig ./mongoarchive ./mongounarchive` passed.
- Verified: `git diff --check` passed.
- Blockers: none.
- Residual risk: PostgreSQL command packages do not exist in STORAGE-01 scope; ARCHIVE-01 and RESTORE-01 must select `storage.PostgreSQLDefaultBackupPrefix` through `toolconfig.BindStorageFlagsWithDefaultPrefix` when wiring their configuration.

### Task PROCESS-01: Implement A Secure Cancellable PostgreSQL Client Runner

Status: completed

Priority: P0

Suggested agent: Go process execution and credential-security engineer

Dependencies: BASE-01

Primary ownership:

- a narrowly named package under `internal/` for PostgreSQL client execution
- package unit tests and fake executable fixtures
- PostgreSQL connection option types shared by the new commands

Finding:

The repository has no generic external-process boundary because MongoDB tooling is linked into the binaries. PostgreSQL execution introduces risks around cancellation, interactive password prompts, unbounded diagnostics, executable discovery, partial output, and credentials exposed through arguments or errors.

References:

- `mongoarchive/main/mongoarchive.go`
- `mongounarchive/main/mongounarchive.go`
- `internal/toolruntime/runtime.go`
- `utils/env.go`
- `notification/message.go`

Implementation requirements:

1. Invoke executables directly with `exec.CommandContext` or an equivalent injectable boundary; never invoke a shell.
2. Support deterministic fake runners or fake executable fixtures without requiring PostgreSQL in unit tests.
3. Set `--no-password` so unattended operations fail instead of waiting for an interactive prompt.
4. Pass passwords through a private temporary `PGPASSFILE` with mode `0600`, or another libpq mechanism with equivalent process-list and log safety. Never put passwords in `argv`.
5. Build a controlled child environment while preserving required process environment such as `PATH`; do not mutate process-global environment.
6. Bound captured diagnostics and sanitize connection strings, passwords, credential-file paths where sensitive, and recognized libpq secret values before returning or logging errors.
7. Make context cancellation terminate the child process and return an error compatible with `errors.Is` for cancellation or deadline expiration.
8. Remove incomplete dump output after a failed or canceled `pg_dump`.
9. Discover missing `pg_dump` or `pg_restore` early and return an actionable dependency error.
10. Capture and parse client version output for the manifest and diagnostics without enforcing a speculative client/server compatibility matrix.
11. Keep command argument construction typed and allowlisted. Do not expose arbitrary extra-argument strings.

Acceptance criteria:

- Tests inspect the child process argument list and prove passwords and password-bearing URIs are absent.
- Tests prove temporary credential files use private permissions and are removed on success, failure, and cancellation.
- Missing executables, malformed version output, nonzero exits, bounded stderr, deadline expiration, and explicit cancellation have deterministic tests.
- A failed dump leaves no artifact that an archive pipeline can upload.
- Returned errors and captured test logs contain none of the supplied secret values.
- `go test -race ./internal/... -count=1` passes for the new package and affected shared packages.

Completion evidence:

- Changed: `internal/postgresclient/client.go`, `internal/postgresclient/client_test.go`, and this task file.
- Implemented: direct allowlisted `pg_dump` and `pg_restore` execution through an injectable process factory; shared typed discrete/URI connection options; strict URI and SSL-mode validation; fixed `--no-password`, custom-format dump, and `--exit-on-error` restore arguments; controlled child environments; and actionable executable discovery.
- Secured: passwords are supplied only through per-run `0600` `PGPASSFILE` files, omitted entirely in no-password mode, and removed after success, process failure, cancellation, and deadline expiration. Bounded diagnostics redact supplied passwords, escaped password-file values, connection URIs, credential-file paths, and recognized libpq secret fields before errors are returned.
- Covered: child arguments and environment, password-bearing URIs, private credential-file content/mode/lifecycle, inherited libpq environment exclusion, failed/canceled dump cleanup, nonzero exits, bounded and boundary-crossing secret diagnostics, explicit cancellation, deadlines, missing `pg_dump` and `pg_restore`, malformed and valid version output, fixed restore arguments, and rejection of arbitrary clients/options.
- Verified: `go test -race ./internal/... -count=1` passed for all internal packages.
- Verified: `go test -race ./internal/postgresclient -count=1` and `go test ./internal/postgresclient -shuffle=on -count=10` passed.
- Verified: `go test -shuffle=on ./...`, `go vet ./...`, and `git diff --check` passed.
- Blockers: none.
- Residual risk: tests use deterministic helper executables rather than installed PostgreSQL clients; real client/server behavior and platform packaging remain intentionally assigned to ARCHIVE-01, RESTORE-01, INTEGRATION-01, PACKAGE-01, and CONTAINER-01.

### Task ORCH-01: Extract Narrow Reusable Archive Delivery Behavior

Status: completed

Priority: P1

Suggested agent: senior Go archive orchestration engineer

Dependencies: BASE-01

Primary ownership:

- the multi-backend upload and retention seam currently in `mongoarchive/main/mongoarchive.go`
- a narrow package under `internal/` if extraction is justified
- `mongoarchive/main/mongoarchive_test.go`
- new shared orchestration tests

Finding:

The strict two-phase upload-before-retention contract is appropriate for PostgreSQL, but its implementation is inside MongoDB's `package main`. Copying it would create two independently evolving data-safety contracts; broadly generalizing both pipelines would create unnecessary regression risk.

References:

- `mongoarchive/main/mongoarchive.go`
- `mongoarchive/main/mongoarchive_test.go`
- `docs/tasks/20260813-105920-multi-backend-archive-contract-remediation.md`
- `storage/interface.go`
- `internal/toolruntime/runtime.go`

Implementation requirements:

1. Extract only database-agnostic delivery behavior needed by both archive commands.
2. Preserve the existing strict two-phase contract exactly: attempt configured uploads, run no retention if any upload fails, then run retention only after all uploads succeed.
3. Preserve explicit partial-state errors when earlier backends have already changed.
4. Keep backend identity and close behavior through existing narrow storage capabilities.
5. Avoid extracting MongoDB dump construction, cron control, notifier construction, or configuration into a generic framework unless direct reuse is demonstrated and tested.
6. Keep all existing MongoDB tests passing under the race detector.

Acceptance criteria:

- One tested delivery implementation serves MongoDB and is available to PostgreSQL.
- First-upload failure, later-upload failure, retention failure, all-success, and single-backend behavior remain covered.
- No retention runs after an incomplete upload phase.
- MongoDB logs, notifications, exit behavior, and partial-state semantics remain compatible.
- `go test -race ./mongoarchive/main ./storage ./internal/... -count=1` passes.

Completion evidence:

- Changed: `internal/archivedelivery/delivery.go`, `internal/archivedelivery/delivery_test.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`, and this task file.
- Implemented: one database-agnostic `archivedelivery.Deliver` seam used by MongoDB and available to PostgreSQL; all configured uploads are attempted before retention is considered, any upload failure suppresses retention everywhere, all upload causes remain discoverable through `errors.Is`, and retention starts only after complete upload success.
- Preserved: single-backend messages and ordering, multi-backend partial-state/backend identity details, MongoDB delivery logs through an injected logger, notification and exit propagation through the unchanged MongoDB pipeline, per-operation timeouts, and caller-owned backend closing through the existing `toolruntime.CloseAll` path.
- Covered: first-upload failure with later upload attempted, later-upload failure, all-upload failure with multiple causes, no retention after any upload failure, all-success phase ordering, retention failure after complete upload, single-backend behavior, backend identity, delivery logs, operation timeouts, and delivery-layer non-ownership of backend closure.
- Verified: `go test -race ./mongoarchive/main ./storage ./internal/... -count=1` passed for all packages.
- Verified: `go test ./internal/archivedelivery ./mongoarchive/main -count=1` passed.
- Verified: `go test -shuffle=on ./...` and `go test -race -shuffle=on ./...` passed for all packages.
- Verified: `go vet ./...`, `go mod verify`, and `git diff --check` passed.
- Blockers: none.
- Residual risk: PostgreSQL does not consume this seam until ARCHIVE-01; no PostgreSQL command or pipeline work was performed in ORCH-01.

### Task ARCHIVE-01: Add PostgreSQL Archive Configuration And Pipeline

Status: completed

Priority: P1

Suggested agent: PostgreSQL and Go backup engineer

Dependencies: STORAGE-01, PROCESS-01, ORCH-01

Primary ownership:

- `postgresarchive/flags.go`
- `postgresarchive/flags_test.go`
- `postgresarchive/main/`
- PostgreSQL archive pipeline tests
- narrowly required shared configuration additions

Finding:

There is no PostgreSQL archive command. The implementation must combine an external `pg_dump` process with existing workspace, tar, storage, retention, cron, and notification behavior without leaking credentials or coupling PostgreSQL flags to MongoDB options.

References:

- `mongoarchive/flags.go`
- `mongoarchive/main/mongoarchive.go`
- `internal/toolconfig/flagdef.go`
- `internal/toolconfig/shared.go`
- `internal/toolconfig/runtime.go`
- `internal/toolruntime/runtime.go`
- `notification/`
- `utils/file.go`
- `utils/date.go`

Implementation requirements:

1. Add `postgres-archive` with `--version`, one-shot, and existing archive cron behavior.
2. Implement command-specific and shared PostgreSQL environment precedence from the approved product contract.
3. Support typed libpq connection inputs needed for host, port, user, database, TLS mode, URI, and password without reusing MongoDB connection types.
4. Reject contradictory or incomplete connection configuration before workspace, notifier, storage, or subprocess side effects.
5. Invoke `pg_dump` in custom format and write it to the private per-run workspace.
6. Create the approved manifest only after a successful complete dump.
7. Package the dump and manifest with existing tar utilities and the existing managed outer filename convention.
8. Upload through the shared strict delivery implementation and use `postgres-archive/` by default.
9. Reuse archive retention, `--keep`, operation timeout, cancellation, scheduler singleton/skip-overlap behavior, and notifier lifecycle semantics where applicable.
10. Make notification wording identify PostgreSQL or remain accurately database-neutral. Preserve redaction and nonfatal notification-send semantics.
11. Validate all configuration before creating external clients or performing dump work.
12. Do not support multiple databases in one invocation or cluster-global objects.

Acceptance criteria:

- Flag tests cover CLI-over-environment precedence, PostgreSQL prefix fallback order, URI versus discrete options, invalid combinations, and secret-safe errors.
- Pipeline tests cover dump failure, manifest failure, tar failure, upload failure, retention failure, cancellation, cleanup failure aggregation, `--keep`, and success.
- No storage upload occurs before `pg_dump`, manifest creation, and tar creation all succeed.
- Multiple backends retain strict upload-before-retention behavior.
- Cron runs skip overlap and receive root-context cancellation.
- Existing MongoDB archive tests remain green.
- `go test -race ./postgresarchive/... ./mongoarchive/... ./storage ./internal/... -count=1` passes.

Completion evidence:

- Changed: `postgresarchive/flags.go`, `postgresarchive/flags_test.go`, `postgresarchive/main/postgresarchive.go`, `postgresarchive/main/postgresarchive_test.go`, `internal/postgresclient/client.go`, and this task file. `CHANGELOG.md` was not modified.
- Implemented: the `postgres-archive` command with one-shot, `--version`, and singleton cron modes; exact `POSTGRESARCHIVE__KEY`, `POSTGRES__KEY`, then `KEY` environment precedence; CLI-over-environment behavior; typed host, port, user, database, SSL mode, URI, and password options; and side-effect-free connection, runtime, storage, retention, schedule, and notification validation.
- Implemented: direct secure-runner `pg_dump` custom-format production in a private per-run workspace; client-version capture; manifest creation and atomic writing only after successful dump completion; the stable two-entry payload inside the existing `.tar.gz` transport; PostgreSQL object-name construction and backend retention under `postgres-archive/` by default; and custom-prefix normalization without MongoDB fallback.
- Preserved: shared strict all-uploads-before-any-retention delivery, all-backend upload attempts, partial-state errors, operation timeouts, root-context process/storage cancellation, reverse cleanup with primary-plus-cleanup error aggregation, `--keep`, scheduler skip-overlap and shutdown behavior, process-lifecycle notifier construction/closure, nonfatal notification-send failures, and database-neutral or explicit PostgreSQL wording.
- Covered: CLI/environment precedence at all three levels, typed connection parsing, URI versus discrete options, required database and invalid option combinations, secret-safe failures, runtime and storage validation, PostgreSQL/default and custom prefixes, every pipeline failure stage, no premature upload, valid produced payload metadata, upload and retention failures, multi-backend retention suppression, cancellation, timeouts, cleanup aggregation, retained artifacts, success notification, cron overlap/root cancellation, notifier validation/construction/closure, notification timeout, nonfatal send failure, and PostgreSQL failure wording.
- Verified: `go test -race ./postgresarchive/... ./mongoarchive/... ./storage ./internal/... -count=1` passed for all selected packages.
- Verified: `go test -shuffle=on ./...` and `go test -race -shuffle=on ./...` passed for the full repository, including existing MongoDB archive and restore packages.
- Verified: `go vet ./...`, `go mod verify`, and `git diff --check` passed.
- Verified: `go build -ldflags '-X main.version=archive01-test' -o /tmp/opencode/postgres-archive-archive01 ./postgresarchive/main` passed, and the binary reported `postgres-archive version: archive01-test` without requiring `pg_dump`.
- Blockers: none.
- Residual risk: deterministic unit tests use the secure runner seam rather than a live PostgreSQL server/client. Real `pg_dump` interoperability and round-trip behavior remain intentionally assigned to `INTEGRATION-01`; native/container client packaging remains assigned to `PACKAGE-01` and `CONTAINER-01`. Restore behavior was not implemented in this task.

### Task RESTORE-01: Add PostgreSQL Restore Configuration And Pipeline

Status: completed

Priority: P1

Suggested agent: PostgreSQL restore correctness engineer

Dependencies: STORAGE-01, PROCESS-01, ARCHIVE-01

Primary ownership:

- `postgresunarchive/flags.go`
- `postgresunarchive/flags_test.go`
- `postgresunarchive/main/`
- PostgreSQL restore pipeline tests
- narrowly required shared configuration additions

Finding:

There is no PostgreSQL restore command. PostgreSQL restore is not inherently atomic and can leave partial changes, so the command must use conservative defaults, fail on the first reported restore error, and avoid claiming rollback or database creation.

References:

- `mongounarchive/flags.go`
- `mongounarchive/main/mongounarchive.go`
- `storage/selection.go`
- `internal/toolruntime/runtime.go`
- `utils/file.go`
- PostgreSQL archive manifest contract from BASE-01 and ARCHIVE-01

Implementation requirements:

1. Add `postgres-unarchive` with `--version` and PostgreSQL environment precedence from the approved product contract.
2. Reuse explicit backend selection when multiple restore backends are configured.
3. Resolve explicit objects or the latest eligible object only under `postgres-archive/` by default.
4. Download atomically and extract through existing entry-count, per-entry-size, total-size, traversal, link, and special-file protections.
5. Validate the manifest and expected dump path before invoking `pg_restore`.
6. Invoke `pg_restore --exit-on-error` against an existing target database.
7. Do not enable `--clean`, `--create`, ownership restoration controls, arbitrary arguments, or single-transaction behavior by default.
8. If optional typed flags for clean, ownership, privileges, jobs, or single transaction are introduced, validate incompatible combinations and document exact semantics. Keep the first implementation minimal.
9. Return nonzero on any process failure and state that partial database changes may exist unless a supported transactional mode was explicitly active.
10. Preserve operation timeout, cancellation, private workspace, cleanup aggregation, and `--keep` behavior.
11. Do not implement MongoDB-style post-restore JSON updates for PostgreSQL.

Acceptance criteria:

- Tests prove automatic selection cannot select MongoDB or wrong-prefix artifacts.
- Missing, malformed, unsupported, wrong-family, or inconsistent manifests fail before `pg_restore`.
- Tests cover explicit object selection, multiple-backend selection requirements, download failure, extraction failure, process failure, cancellation, cleanup, `--keep`, and success.
- The default invocation contains `--exit-on-error` and contains no `--clean` or `--create`.
- Errors for non-transactional failures do not claim rollback or an unchanged target.
- Existing MongoDB restore tests remain green.
- `go test -race ./postgresunarchive/... ./mongounarchive/... ./storage ./internal/... -count=1` passes.

Completion evidence:

- Changed: `postgresunarchive/flags.go`, `postgresunarchive/flags_test.go`, `postgresunarchive/main/postgresunarchive.go`, `postgresunarchive/main/postgresunarchive_test.go`, `internal/postgresclient/client.go`, `internal/postgresclient/client_test.go`, and this task file. `CHANGELOG.md` was not modified.
- Implemented: `postgres-unarchive` with side-effect-free `--version`; exact `POSTGRESUNARCHIVE__KEY`, `POSTGRES__KEY`, then `KEY` environment precedence; typed host, port, user, existing target database, SSL mode, URI, and password configuration; explicit restore-backend selection; explicit object or PostgreSQL-prefix-scoped latest selection; and the `postgres-archive/` default without MongoDB fallback.
- Implemented: provider-contract atomic download into a private per-run workspace; bounded defensive extraction before semantic validation; strict manifest, payload layout, and custom-format dump validation before process invocation; direct typed `pg_restore --no-password --exit-on-error` execution with no clean, create, transaction, ownership, privilege, jobs, or arbitrary passthrough options; storage lookup/download timeouts; root-context process cancellation; reverse cleanup, backend close, primary-plus-cleanup error aggregation, and `--keep`.
- Implemented: process errors now distinguish pre-start failures from errors after a client process actually starts. Only started `pg_restore` failures report that partial database changes may exist; missing executable, validation, setup, selection, download, and extraction failures do not imply target mutation or rollback.
- Covered: all three environment levels and CLI override, version independence from invalid operational configuration, typed connection and runtime parsing, secret-safe validation, PostgreSQL default/custom prefixes, generated-filename and full-key explicit selection, wrong-prefix automatic-selection isolation, multiple-backend selection requirements, download and extraction failures, extraction limits and traversal rejection, missing/malformed/unsupported/wrong-family/inconsistent payloads before restore, fixed process arguments, process setup and started-process failures, lookup/download deadlines, cancellation, private workspace permissions, cleanup/close aggregation, `--keep`, and success. Existing MongoDB restore behavior remains covered by the required race and full-repository suites.
- Verified: `go test -shuffle=on -count=10 ./internal/postgresclient ./postgresunarchive/...` passed.
- Verified: `go test -race ./postgresunarchive/... ./mongounarchive/... ./storage ./internal/... -count=1` passed for every selected package.
- Verified: `go test -shuffle=on ./...` and `go test -race -shuffle=on ./...` passed for the full repository.
- Verified: `go vet ./...`, `go mod verify`, and `git diff --check` passed.
- Verified: `go build -ldflags '-X main.version=restore01-test' -o /tmp/opencode/postgres-unarchive-restore01 ./postgresunarchive/main` passed, and the binary reported `postgres-unarchive version: restore01-test` without requiring `pg_restore`.
- Blockers: none.
- Residual risk: tests use deterministic storage/process seams and the existing provider contract suite rather than a live PostgreSQL server. Real `pg_restore` interoperability and round-trip database results remain intentionally assigned to `INTEGRATION-01`; packaging and client availability remain assigned to `PACKAGE-01` and `CONTAINER-01`.

### Task INTEGRATION-01: Add PostgreSQL Round-Trip And Isolation Coverage

Status: in_progress

Priority: P1

Suggested agent: database integration and CI engineer

Dependencies: ARCHIVE-01, RESTORE-01

Primary ownership:

- `sandbox/docker-compose.yml`
- `sandbox/docker-compose-ci.yml`
- PostgreSQL setup, check, and teardown helpers under `test/`
- PostgreSQL scenarios in `test/test.bats`
- `.github/workflows/integration.yml` only where required

Finding:

Current integration coverage exercises MongoDB against local and emulated cloud storage but provides no PostgreSQL service, client, fixtures, or round trip. Unit fakes cannot establish that generated arguments, client/server compatibility, archive format, and database results work together.

References:

- `sandbox/docker-compose.yml`
- `sandbox/docker-compose-ci.yml`
- `.github/workflows/integration.yml`
- `test/test.bats`
- `test/testdb-setup.go`
- `test/testdb-check.go`
- `test/testdb-drop.go`

Implementation requirements:

1. Add a pinned PostgreSQL service with a bounded health check and deterministic clean-start behavior.
2. Use test helpers or strict `psql` commands with bounded connection and statement execution.
3. Create representative schema, constraints, indexes, sequences, and data; archive it; recreate or empty the target database; restore it; and verify observable schema and data results.
4. Cover local storage round trip as the required baseline.
5. Cover explicit object restore, automatic latest PostgreSQL selection, missing object failure, wrong manifest failure, and nonzero `pg_restore` failure.
6. Place MongoDB and PostgreSQL managed objects in one backend and prove latest selection and retention do not cross database families.
7. Preserve existing MongoDB and provider integration scenarios.
8. Keep service readiness, diagnostics, artifact upload, and teardown deterministic and bounded.
9. If all cloud-emulator PostgreSQL round trips are too costly, rely on the provider contract suite for transport and record why local plus one representative remote backend provides sufficient initial coverage.

Acceptance criteria:

- A real PostgreSQL archive/restore round trip reproduces the expected schema and data.
- A mixed MongoDB/PostgreSQL storage test proves cross-family selection and deletion do not occur.
- A failed dump uploads nothing, and a failed restore returns nonzero with bounded diagnostics.
- At least one clean integration run passes in Docker-capable CI; locally unavailable Docker is recorded as a blocker rather than success.
- `bats --print-output-on-failure test/test.bats` passes in a Docker-capable environment.

Completion evidence:

- Changed: `sandbox/docker-compose.yml`, `sandbox/docker-compose-ci.yml`, `.github/workflows/integration.yml`, `test/postgres-fixture.sh`, `test/test.bats`, `test/teststorage-state.go`, `internal/postgresclient/client.go`, `internal/postgresclient/client_test.go`, and this task file. `CHANGELOG.md` was not modified.
- Implemented: pinned PostgreSQL 16.10 integration service with bounded health check; CI installation/version reporting for `postgresql-client-16`; PostgreSQL fixture helper with bounded `psql` execution; local PostgreSQL archive/restore round trip; explicit-object and latest-object restore; missing-object failure; malformed-manifest failure before restore; nonzero `pg_restore` failure with bounded diagnostics; failed-dump uploads-nothing coverage; representative PostgreSQL S3 emulator round trip; and mixed MongoDB/PostgreSQL local-backend latest-selection and retention isolation while preserving existing MongoDB local, S3, Azure, GCP, and multi-backend scenarios.
- Implemented: Docker compose host port for PostgreSQL is configurable as `${POSTGRES_PORT:-5432}` so CI remains deterministic on the default port while local isolated runs can avoid an occupied host `5432`.
- Fixed during integration: `pg_restore` requires an explicit target database; `internal/postgresclient` now passes `--dbname=<database>` for restore while retaining `--no-password` and `--exit-on-error`, with unit coverage in `internal/postgresclient/client_test.go`.
- Verified: `bash -n test/postgres-fixture.sh` passed.
- Verified: `go test ./test -run TestNonexistent` passed (`[no test files]`).
- Verified before the restore fix: `go test -shuffle=on ./...`, `go test -race -shuffle=on ./...`, `go vet ./...`, and `git diff --check` passed.
- Verified after the restore fix and generated sandbox permission repair: `go test -shuffle=on ./...` passed.
- Verified after the restore fix and generated sandbox permission repair: `go test -race -shuffle=on ./...` passed.
- Verified after the restore fix and generated sandbox permission repair: `go vet ./...` passed.
- Verified after the restore fix and generated sandbox permission repair: `git diff --check` passed.
- Integration attempt: `bats --print-output-on-failure test/test.bats` was run after starting Docker services from `sandbox/docker-compose.yml` and `sandbox/docker-compose-ci.yml`; MongoDB/provider scenarios 1-33 passed, but PostgreSQL scenarios failed because host PostgreSQL clients were unavailable (`pg_dump --version` and `psql --version`: command not found before startup; Bats reported `env: 'psql': Permission denied` and `required PostgreSQL client "pg_dump" was not found in PATH`). `sudo -n true` also failed with `sudo: a password is required`, so host clients could not be installed locally.
- Integration verification with local workaround: using temporary `/tmp/opencode/pgbin/{psql,pg_dump,pg_restore}` wrappers that execute the pinned PostgreSQL service container clients, `bats --print-output-on-failure test/test.bats` passed all 46 tests. This validates the Bats scenarios, real server round trips, archive payload interoperability, storage/provider behavior, and mixed-family isolation locally, but it is not a substitute for CI or native host-client evidence.
- Local environmental notes: initial compose startup also hit occupied host ports `8080` and `5432`; the local-only `.env.test` used `FAKE_GCP_PORT=18080` and `POSTGRES_PORT=15432` for the successful workaround run. Compose services were torn down with `docker-compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml down -v --remove-orphans`; generated `sandbox/mnt/postgres` permissions were repaired with a temporary Docker `chmod` container so repository checks could traverse the workspace.
- Blockers: no clean run with real host PostgreSQL client executables was possible in this local environment, and no Docker-capable CI run URL/evidence was available in this session. Task status remains `in_progress` under the status rules until real-client CI or equivalent real-client local evidence is recorded.
- Residual risk: the local passing Bats run used Docker-exec client wrappers rather than native `pg_dump`, `pg_restore`, and `psql` binaries in host `PATH`; CI is configured to install `postgresql-client-16` and should provide the required real-client evidence before marking this task completed.

### Task PACKAGE-01: Extend Native Builds Releases And Asdf Installation

Status: completed

Priority: P1

Suggested agent: cross-platform Go release engineer

Dependencies: ARCHIVE-01, RESTORE-01

Primary ownership:

- `Makefile`
- `go-release.mjs`
- `bin/install`
- `test/asdf-plugin.bats`
- release artifact tests and policy documentation

Finding:

Build, release, and asdf installation currently enumerate exactly `mongo-archive` and `mongo-unarchive`. Build definitions target individual `.go` files, which will omit sibling source files if a command package spans multiple files. Native PostgreSQL wrappers also depend on system PostgreSQL clients at runtime.

References:

- `Makefile`
- `go-release.mjs`
- `bin/install`
- `test/asdf-plugin.bats`
- `.github/workflows/release.yml`
- `docs/release-artifact-policy.md`

Implementation requirements:

1. Build all four commands from package directories rather than individual source files.
2. Add both PostgreSQL commands to every supported Go release target without bundling PostgreSQL native executables.
3. Extend exact version-output checks, archive-content checks, checksums, reproducibility checks, and asdf installation tests to all four binaries.
4. Make installer errors and documentation distinguish installation of wrappers from availability of PostgreSQL client dependencies.
5. Preserve the current OS and architecture matrix unless an actual Go compilation incompatibility is demonstrated and approved.
6. Keep deterministic archive metadata and existing supply-chain checks intact.
7. Update release artifact policy to state that PostgreSQL operations require compatible external clients on native installations.

Acceptance criteria:

- `make build VERSION=<test-version>` produces four correctly versioned binaries.
- Release archives contain exactly the documented executable set plus approved metadata files.
- Asdf installation exposes all four commands and its tests cover missing or malformed command sets.
- PostgreSQL commands emit an actionable dependency error when their required client is unavailable; `--version` itself does not require PostgreSQL clients.
- `make test-asdf` passes.
- Reproducible release archive verification passes for the expanded artifacts.

Completion evidence:

- Changed: `Makefile`, `go-release.mjs`, `bin/install`, `test/asdf-plugin.bats`, `docs/release-artifact-policy.md`, and this task file. `CHANGELOG.md` was not modified.
- Implemented: native `make build` now builds `mongo-archive`, `mongo-unarchive`, `postgres-archive`, and `postgres-unarchive` from command package directories; `go-release.mjs` enumerates all four wrappers for every existing OS/architecture target and verifies exact `--version` output; native archives remain deterministic, checksum-covered, reproducibility-checked, and include only the four wrappers plus `LICENSE`.
- Implemented: asdf installation now requires the exact four-wrapper command set with optional `LICENSE`, rejects archives missing PostgreSQL wrappers, rejects unexpected members and symlink executables, installs all four wrappers under `bin/`, and tells users that native PostgreSQL operations require external compatible `pg_dump` and `pg_restore` clients in `PATH` because native archives do not bundle them.
- Documented: `docs/release-artifact-policy.md` now states the exact native archive contents, deterministic metadata policy, four-wrapper version verification, and external PostgreSQL native-client dependency contract.
- Verified: `make build VERSION=package01-test` passed and produced all four binaries.
- Verified: `./dist/mongo-archive --version`, `./dist/mongo-unarchive --version`, `./dist/postgres-archive --version`, and `./dist/postgres-unarchive --version` all reported `package01-test`; PostgreSQL `--version` checks did not require native PostgreSQL clients.
- Verified: `if env PATH=/tmp/opencode ./dist/postgres-archive --database=package01_missing_client --local-path=./dist/package01-missing-client-check; then exit 1; fi` passed by returning a nonzero operational failure with `required PostgreSQL client "pg_dump" was not found in PATH; install PostgreSQL client tools and ensure pg_dump is executable`.
- Verified: `make test-asdf` passed all 9 Bats tests.
- Verified: `go test -shuffle=on ./...`, `go vet ./...`, `go mod verify`, and `git diff --check` passed.
- Verified: `VERSION=package01-test SOURCE_DATE_EPOCH=0 pnpm run release:build` passed for all 15 existing targets and all four wrappers.
- Verified: `VERSION=package01-test SOURCE_DATE_EPOCH=0 pnpm run release:verify` passed with reproducibility enabled for all 15 targets; Linux amd64 host-compatible exact version checks covered all four wrappers.
- Verified: `tar -tzf dist/database-tools-linux-amd64.tar.gz` listed exactly `LICENSE`, `mongo-archive`, `mongo-unarchive`, `postgres-archive`, and `postgres-unarchive`, with no bundled `pg_dump`, `pg_restore`, or other PostgreSQL native clients.
- Full release verification attempt: `make release-verify VERSION=package01-test` progressed through `make test-asdf`, `go test -shuffle=on ./...`, `go test -race -shuffle=on ./...`, `go vet ./...`, `go mod verify`, `scripts/release-govulncheck.sh`, and `docker buildx build --check .`, then failed during the Docker image build because the current `Dockerfile` copies only MongoDB command packages before invoking the expanded `make build`; the exact Docker error was `stat /app/postgresarchive/main: directory not found`. Updating the Docker build context/runtime image is assigned to `CONTAINER-01` and was intentionally not performed in PACKAGE-01.
- Blockers: full `make release-verify` cannot pass until `CONTAINER-01` updates the Dockerfile/container release contract to include PostgreSQL wrappers and pinned PostgreSQL clients. Native package build, asdf installation, checksums, exact version output, deterministic archive metadata, and reproducibility verification passed.
- Residual risk: native artifacts install only Go wrappers; PostgreSQL archive/restore operations still depend on compatible host-provided `pg_dump` and `pg_restore`. Container packaging and pinned client verification remain assigned to `CONTAINER-01`; operator-facing broader documentation remains assigned to `DOCS-01`.

### Task CONTAINER-01: Ship And Verify Pinned PostgreSQL Clients

Status: completed

Priority: P1

Suggested agent: container and PostgreSQL compatibility engineer

Dependencies: INTEGRATION-01, PACKAGE-01

Primary ownership:

- `Dockerfile`
- `.github/workflows/publish.yaml`
- container release verification in `Makefile`
- supply-chain consistency checks affected by the runtime package source
- container deployment documentation only where tightly coupled

Finding:

The Alpine runtime image contains only `tzdata`, and release smoke tests execute only MongoDB command versions. Container users need a known PostgreSQL client version, while client/server compatibility and added image packages must remain visible to vulnerability, SBOM, provenance, and digest gates.

References:

- `Dockerfile`
- `Makefile`
- `.github/workflows/publish.yaml`
- `scripts/check-supply-chain.sh`
- `docs/release-artifact-policy.md`

Implementation requirements:

1. Add a deliberately selected and reproducibly pinned PostgreSQL client package source/version compatible with the supported container policy.
2. Copy both PostgreSQL wrappers into the runtime image and update OCI description without removing MongoDB identity.
3. Smoke-test all four wrappers' version output and both PostgreSQL native clients' presence and version before publication.
4. Run a real PostgreSQL archive/restore container round trip as a release gate, not only a `command -v` check.
5. Keep the scan-before-push, digest identity, SBOM, provenance, and OCI metadata guarantees from existing release remediation.
6. Document the client version policy and explain that an older client may reject a newer PostgreSQL server.
7. Ensure the nonroot runtime user can create private temporary credential and workspace files without broad permissions.

Acceptance criteria:

- The published-image candidate contains `postgres-archive`, `postgres-unarchive`, `pg_dump`, and `pg_restore`.
- Client version is deterministic from the pinned image/package inputs and appears in verification evidence.
- A containerized round trip against the supported PostgreSQL service passes as nonroot.
- Trivy scanning occurs before push and scans the exact image later published.
- SBOM and provenance include the added PostgreSQL runtime packages.
- `docker buildx build --check .` and the container release smoke/round-trip checks pass in a Docker-capable environment.

Completion evidence:

- Changed: `Dockerfile`, `Makefile`, `.github/workflows/publish.yaml`, `scripts/check-supply-chain.sh`, `scripts/container-roundtrip.sh`, `docs/release-artifact-policy.md`, and this task file. `CHANGELOG.md` was not modified.
- Implemented: the Docker build stage now copies `postgresarchive/` and `postgresunarchive/`, fixing the PACKAGE-01 Docker build failure. The runtime image copies all four Go wrappers, updates the OCI description to MongoDB and PostgreSQL archive/restore tools, and installs pinned `POSTGRESQL_CLIENT_PACKAGE=postgresql16-client=16.15-r0` from the digest-pinned Alpine 3.23 runtime image.
- Implemented: the runtime image remains nonroot and now has private writable `/home/nonroot/tmp` and `/home/nonroot/work` directories with `0700` permissions, `TMPDIR=/home/nonroot/tmp`, and `WORKDIR=/home/nonroot/work`, so PostgreSQL credential files and default workspaces do not require broad writable permissions.
- Implemented: `make release-verify` now smoke-tests `mongo-archive`, `mongo-unarchive`, `postgres-archive`, `postgres-unarchive`, `pg_dump`, and `pg_restore`, then runs `scripts/container-roundtrip.sh` as a real PostgreSQL archive/restore container release gate against the pinned PostgreSQL 16.10 service image. The round-trip gate uses Docker named volumes, narrows the backup volume to UID/GID 1000 with `0700`, runs archive and restore through the nonroot candidate image, and verifies restored schema, constraints, index, sequence, and data.
- Implemented: the publish pre-push command now performs the same four-wrapper and PostgreSQL-client smoke checks, runs the container round trip before push, and records all versions plus the round-trip result in the workflow summary. Existing gated publish mode, Trivy enablement, severity policy, SBOM, provenance, SBOM attestation, digest identity, and OCI label checks are preserved.
- Implemented: `scripts/check-supply-chain.sh` now fails if the Dockerfile PostgreSQL client package is not pinned, and `docs/release-artifact-policy.md` documents the container client version policy, older-client/newer-server compatibility risk, expanded release smoke/round-trip gate, and preservation of scan-before-push/SBOM/provenance guarantees.
- Verified: `bash -n scripts/container-roundtrip.sh`, `bash -n scripts/check-supply-chain.sh`, and `bash scripts/check-supply-chain.sh` passed; the supply-chain gate reported `supply-chain declarations are consistent`.
- Verified: `go test -shuffle=on ./...` passed.
- Verified: `go test -race -shuffle=on ./...` passed as part of `make release-verify VERSION=container01-test` before the final timeout.
- Verified: `go vet ./...`, `go mod verify`, and `git diff --check` passed.
- Verified: `docker buildx build --check .` passed with Docker 29.7.2 and buildx v0.36.1-desktop.1.
- Verified: `docker build --build-arg VERSION=container01-test --tag database-tools:container01-test .` passed. The build installed `postgresql16-client (16.15-r0)` plus its SBOM-visible APK dependencies including `postgresql-common`, `libpq`, `readline`, `lz4-libs`, and `zstd-libs`.
- Verified: candidate image smoke checks passed: `mongo-archive --version`, `mongo-unarchive --version`, `postgres-archive --version`, and `postgres-unarchive --version` all reported `container01-test`; `pg_dump --version` reported `pg_dump (PostgreSQL) 16.15`; `pg_restore --version` reported `pg_restore (PostgreSQL) 16.15`.
- Verified: candidate image nonroot/private-writable check passed by confirming UID `1000`, writable `TMPDIR`, writable `PWD`, and `0700` permissions on both private directories.
- Verified: `bash ./scripts/container-roundtrip.sh database-tools:container01-test` passed against the pinned PostgreSQL 16.10 service image, producing a real PostgreSQL archive and restoring the expected schema, constraints, index, sequence, and data as the image's nonroot user.
- Verified: `make release-verify VERSION=container01-test` passed through `check-supply-chain`, `make test-asdf`, `pnpm install --frozen-lockfile`, `go test -shuffle=on ./...`, `go test -race -shuffle=on ./...`, `go vet ./...`, `go mod verify`, `scripts/release-govulncheck.sh`, `docker buildx build --check .`, Docker image build, all six container smoke checks, and the container PostgreSQL round trip. The shell tool timed out after 600 seconds during the later native `pnpm run release:verify` reproducibility step, after `pnpm run release:build` completed.
- Follow-up verification attempt: `VERSION=container01-test SOURCE_DATE_EPOCH=0 pnpm run release:verify` was run directly and exceeded the 900-second shell tool timeout before producing verifier output. This timeout is in the native reproducibility verifier, not the CONTAINER-01 Docker build, smoke, PostgreSQL client, or round-trip gate.
- Blockers: no CONTAINER-01 acceptance blocker remains in this Docker-capable environment. Full end-to-end `make release-verify` still needs a successful native reproducibility verifier run in a longer-running environment before final release signoff.
- Residual risk: the publish workflow's real Trivy scan, registry push, digest-identity proof, SBOM attachment, and provenance/SBOM attestations cannot be executed locally without publishing a release image; the workflow retains the existing gated action configuration and expanded pre-push checks, but final evidence must come from the release workflow run.

### Task DOCS-01: Document PostgreSQL Operations And Generate CLI Reference

Status: completed

Priority: P2

Suggested agent: database operations technical writer with Go CLI experience

Dependencies: ARCHIVE-01, RESTORE-01, PACKAGE-01, CONTAINER-01

Primary ownership:

- `internal/flagdocs/`
- `flags.md`
- `.env.example`
- `README.md`
- `website/docs/`
- `website/docusaurus.config.ts` if navigation changes
- `examples/` where PostgreSQL examples add value

Finding:

All public documentation, generated flag references, environment completeness checks, deployment examples, and product descriptions currently describe MongoDB-only behavior. PostgreSQL introduces external-client compatibility, non-atomic restore caveats, and database-family-specific object prefixes that operators must understand before relying on backups.

References:

- `internal/flagdocs/flagdocs.go`
- `internal/flagdocs/flagdocs_test.go`
- `internal/flagdocs/envdocs_test.go`
- `flags.md`
- `.env.example`
- `README.md`
- `website/docs/about/overview.mdx`
- `website/docs/about/quick-start.mdx`
- `website/docs/operations/archive.mdx`
- `website/docs/operations/restore.mdx`
- `website/docs/operations/storage-backends.mdx`
- `website/docs/deployment/docker.mdx`
- `website/docs/deployment/kubernetes.mdx`
- `website/docs/reference/cli-flags.mdx`

Implementation requirements:

1. Register both PostgreSQL commands in generated flag and environment documentation checks.
2. Regenerate authoritative references; do not hand-edit generated output without updating its source.
3. Document native external-client requirements and container-bundled client behavior separately.
4. Document client/server version compatibility, the single-database scope, excluded globals, default prefixes, and custom-prefix isolation responsibilities.
5. Document archive payload and manifest at an operator-appropriate level without exposing internal credentials.
6. State that default restore targets an existing database, uses `--exit-on-error`, does not clean or create by default, and may leave partial changes.
7. Provide tested examples for archive, explicit restore, latest restore, local storage, one cloud backend, Docker, and Kubernetes secret injection.
8. Update MongoDB-only product descriptions while preserving clear database-specific command guidance.
9. Keep examples fail-fast and avoid literal credentials.

Acceptance criteria:

- Generated CLI docs include both new commands and every public PostgreSQL environment variable.
- Documentation clearly distinguishes native and container dependency behavior.
- An operator can determine backup scope, restore side effects, prefix isolation, and compatibility limits without reading source.
- Website type checking/build and documentation tests pass.
- `go test ./internal/flagdocs ./postgresarchive ./postgresunarchive -count=1` passes.
- Repository-defined website validation commands pass.

Completion evidence:

- Changed for DOCS-01: `internal/flagdocs/flagdocs.go`, `internal/flagdocs/envdocs_test.go`, `internal/flagdocs/kubernetes_examples_test.go`, `flags.md`, `.env.example`, `README.md`, `package.json`, `website/docs/about/overview.mdx`, `website/docs/about/quick-start.mdx`, `website/docs/operations/archive.mdx`, `website/docs/operations/restore.mdx`, `website/docs/operations/storage-backends.mdx`, `website/docs/deployment/docker.mdx`, `website/docs/deployment/kubernetes.mdx`, `website/docs/reference/cli-flags.mdx`, `website/docs/reference/release-policy.mdx`, `examples/postgres-cronjob-archive.yaml`, and this task file. `CHANGELOG.md` was not modified.
- Implemented: generated CLI documentation now registers `postgres-archive` and `postgres-unarchive`; `flags.md` was regenerated from `internal/flagdocs.Markdown()` and includes PostgreSQL command-specific variables plus generated `POSTGRES__*` and unprefixed fallback lookup tables for every public PostgreSQL key.
- Documented: native installs require compatible host `pg_dump` and `pg_restore`, while the container includes pinned PostgreSQL clients; client/server compatibility risks; single-database PostgreSQL scope; excluded cluster globals and non-goals; `postgres-archive/` versus `mongo-archive/` defaults; custom-prefix isolation responsibilities; PostgreSQL payload/manifest contents without credentials; and restore behavior against an existing database using `pg_restore --exit-on-error` without default clean/create or rollback guarantees.
- Added examples: fail-fast and credential-safe PostgreSQL archive, explicit restore, latest restore, local storage, S3, Docker, and Kubernetes secret-injection examples. Added a validated `examples/postgres-cronjob-archive.yaml` manifest using Secret references for sensitive values.
- Verified: `go test ./internal/flagdocs ./postgresarchive ./postgresunarchive -count=1` passed.
- Verified: `pnpm docs:typecheck` passed.
- Verified: `pnpm docs:build` passed; Docusaurus reported only its standard available-update notice and Node localStorage experimental warning, then generated static files successfully.
- Verified: `go test -shuffle=on ./...` passed.
- Verified: `go test -race -shuffle=on ./...` passed.
- Verified: `go vet ./...`, `go mod verify`, and `git diff --check` passed.
- Blockers: none for DOCS-01.
- Residual risk: INTEGRATION-01 remains `in_progress` because this session did not add the missing clean real host PostgreSQL client evidence; DOCS-01 documentation describes the intended native and container dependency contracts, but final release signoff still needs the integration evidence tracked under INTEGRATION-01 and independent REVIEW-01.

### Task REVIEW-01: Independently Verify PostgreSQL Support End To End

Status: blocked

Priority: P1

Suggested agent: independent senior database reliability and security reviewer

Dependencies: INTEGRATION-01, PACKAGE-01, CONTAINER-01, DOCS-01

Primary ownership:

- review across all PostgreSQL and shared changes
- minimal missing regression tests found during review
- this task file for review evidence and residual risks

Finding:

This feature crosses credential handling, subprocess execution, destructive database operations, managed retention, generated documentation, multi-platform releases, and container supply-chain boundaries. The final reviewer must not be the primary implementer of archive, restore, or process execution.

References:

- Completion evidence for every preceding task
- `postgresarchive/`
- `postgresunarchive/`
- PostgreSQL process package under `internal/`
- `storage/`
- `Dockerfile`
- release and integration workflows
- operator documentation

Implementation requirements:

1. Verify every acceptance criterion against code and runtime evidence, not task status alone.
2. Attempt credential disclosure through arguments, environment-derived errors, subprocess stderr, notification messages, manifests, process listings where practical, and retained workspaces.
3. Verify cancellation and deadline behavior for dump, upload, download, extraction, and restore stages.
4. Verify wrong-family archives, malformed manifests, custom prefixes, mixed MongoDB/PostgreSQL buckets, and retention boundaries.
5. Verify default restore arguments do not clean or create databases and user-visible errors do not imply rollback.
6. Verify all four commands, installers, release archives, container image, docs, and version output agree.
7. Re-run targeted, full, race, vet, integration, artifact, reproducibility, and container checks.
8. Record all deferred work with rationale and residual risk. Add new tasks rather than hiding necessary scope in review notes.

Acceptance criteria:

- No unresolved P0 or P1 finding remains.
- No tested secret appears in arguments, logs, errors, manifests, or retained artifacts outside the protected credential file lifecycle.
- MongoDB and PostgreSQL round trips both pass against the final build.
- Automatic selection and retention remain isolated across database families.
- Native artifacts and the container satisfy their distinct dependency contracts.
- Full verification passes, or an exact environmental blocker and successful CI evidence are recorded.
- The independent reviewer records identity/session, findings, verification evidence, and residual risks in this file.

Review evidence, 2026-08-24:

- Reviewer/session: independent `REVIEW-01` OpenCode gpt-5.5 API session in `/home/jahn/projects/_database-tools`. This review did not rely on prior agent summaries as authoritative.
- Finding P1: PostgreSQL integration acceptance remains unresolved. `INTEGRATION-01` is marked `completed` at line 518, but its own blocker evidence at lines 581-585 says no clean real-host-client run or CI URL/evidence was available and that task status should remain `in_progress`. This review preserved the existing task status line because it was assigned only to `REVIEW-01`, but final acceptance cannot treat integration as complete without real host `psql`, `pg_dump`, and `pg_restore` evidence or a Docker-capable CI run URL.
- Finding P1: local Bats integration did not pass. `bats --print-output-on-failure test/test.bats` with Compose services from `sandbox/docker-compose.yml` and `sandbox/docker-compose-ci.yml` passed tests 1-33, then failed PostgreSQL tests 34-46 because host PostgreSQL clients were unavailable or not executable: `env: 'psql': Permission denied` and `required PostgreSQL client "pg_dump" was not found in PATH; install PostgreSQL client tools and ensure pg_dump is executable`. Compose services were torn down with `docker-compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml down -v --remove-orphans`.
- Finding P1/blocker: native reproducibility verification did not complete in this tool session. `VERSION=review01-test SOURCE_DATE_EPOCH=0 pnpm run release:build` passed for all 15 targets and all four wrappers. A first `pnpm run release:verify` attempt was invalid because it was mistakenly run concurrently with the build and failed before the checksum manifest existed. The sequential rerun `VERSION=review01-test SOURCE_DATE_EPOCH=0 pnpm run release:verify` exceeded the 900-second shell timeout before producing verifier output. This leaves the full native release/reproducibility gate unverified locally and no successful CI evidence was available in this review.
- Finding P2/residual release risk: local review cannot execute the real publish workflow, Trivy pre-push scan, registry digest-identity proof, SBOM attachment, or provenance/SBOM attestations. Code and workflow/policy checks preserve those gates, but final evidence must come from the release workflow.
- Credential/security verification: code review and tests verify typed direct `exec.CommandContext` execution without shell strings; fixed `--no-password`; password delivery through per-run `0600` `PGPASSFILE`; cleanup after success, failure, cancellation, and deadline; bounded diagnostics; and redaction of supplied passwords, URI secrets, escaped pgpass values, and credential-file paths. Targeted tests also cover child arguments/environment and secret-safe errors. No tested secret appeared in arguments, returned errors, manifests, or retained artifacts in reviewed unit/runtime evidence.
- Archive/manifest verification: `internal/postgresarchiveformat` requires exactly `manifest.json` and `database.dump`, `database_family: postgresql`, `dump_format: custom`, UTC creation time, source database, client version, and a `PGDMP` dump header. Tests cover missing, malformed, unsupported, wrong-family, duplicate, conflicting, extra, MongoDB, and arbitrary payloads. Archive pipeline only writes the manifest after successful `pg_dump`, then tars and uploads through the shared delivery seam.
- Restore verification: `postgres-unarchive` validates extraction limits, manifest, payload layout, and custom-format dump before invoking `pg_restore`; default runner args include `--dbname=<database>`, `--no-password`, and `--exit-on-error`, and do not include `--clean`, `--create`, arbitrary passthrough, ownership/privilege toggles, jobs, or single-transaction flags. Started-process failures report that partial database changes may exist and do not claim rollback.
- Prefix/retention verification: code review confirmed PostgreSQL commands bind `storage.PostgreSQLDefaultBackupPrefix` (`postgres-archive/`) through default-aware storage flag binding before provider initialization, while MongoDB compatibility wrappers preserve `mongo-archive/`. Storage tests cover prefix-scoped latest selection, explicit object lookup, and retention isolation for mixed MongoDB/PostgreSQL objects.
- Cancellation/deadline verification: targeted tests cover `pg_dump`/`pg_restore` cancellation and deadline behavior, failed/canceled dump cleanup, storage operation timeout paths, extraction failure paths, and primary-plus-cleanup error aggregation. The archive delivery seam wraps upload and retention operations in operation contexts and suppresses all retention after any upload-phase failure.
- Package/container/docs verification passed locally: `go test ./internal/postgresclient ./internal/postgresarchiveformat ./postgresarchive/... ./postgresunarchive/... ./storage ./mongoarchive/... ./mongounarchive/... ./internal/toolconfig ./internal/archivedelivery ./internal/flagdocs -count=1`; the same targeted set under `-race`; `go test -shuffle=on ./...`; `go test -race -shuffle=on ./...`; `go vet ./...`; `go mod verify`; `git diff --check`; `make build VERSION=review01-test`; `make test-asdf`; `pnpm docs:typecheck`; `pnpm docs:build`; and `bash ./scripts/check-supply-chain.sh` all passed.
- Built binary/version verification passed: `./dist/mongo-archive --version`, `./dist/mongo-unarchive --version`, `./dist/postgres-archive --version`, and `./dist/postgres-unarchive --version` all reported `review01-test`. Host `pg_dump --version` and `pg_restore --version` failed with `command not found`, which is an environment blocker for native PostgreSQL integration, not a wrapper `--version` blocker.
- Container verification passed locally: `docker buildx build --check .` passed; `docker build --build-arg VERSION=review01-test --tag database-tools:review01-test .` passed and installed pinned `postgresql16-client=16.15-r0`; container smoke checks for all four wrappers plus `pg_dump --version` and `pg_restore --version` passed (`pg_dump (PostgreSQL) 16.15`, `pg_restore (PostgreSQL) 16.15`); `bash ./scripts/container-roundtrip.sh database-tools:review01-test` completed successfully against the pinned PostgreSQL service image as the nonroot runtime user.
- Blockers: clean real-host-client PostgreSQL Bats integration evidence is missing; native `pnpm run release:verify` reproducibility did not complete within the 900-second tool timeout; no CI URL/evidence was available to replace either local blocker. `REVIEW-01` remains blocked and must not be marked completed until those P1 blockers are resolved or exact successful CI evidence is recorded.
- Residual risks: unit/process coverage is strong for credential redaction, wrong-family archives, malformed manifests, cancellation/deadlines, partial restore messaging, prefixes, and retention, but final release signoff still depends on clean integration and native reproducibility evidence. Publish-time Trivy, SBOM, provenance, and registry digest evidence remains workflow-only.

Coordinator final review, 2026-08-24:

- Reviewed this task file after all requested sequential sub-agent sessions completed or reported blockers.
- Corrected status consistency: `INTEGRATION-01` remains `in_progress` because its own completion evidence records missing clean real-host-client or CI integration evidence; `DOCS-01` is `completed` because its completion evidence records passing documentation, website, Go, race, vet, module, and diff checks with no DOCS-01 blocker.
- Final task-file state is not fully done: `REVIEW-01` remains `blocked` due unresolved P1 verification blockers for clean PostgreSQL Bats integration evidence and native reproducibility verification timeout. These blockers must be resolved before the overall Definition of Done can be satisfied.

## Dependency And Parallelization Guidance

Recommended agent allocation:

| Wave | Task           | Can run in parallel with                           | Must wait for                                                       |
| ---- | -------------- | -------------------------------------------------- | ------------------------------------------------------------------- |
| 1    | BASE-01        | none                                               | none                                                                |
| 2    | STORAGE-01     | PROCESS-01, ORCH-01                                | BASE-01                                                             |
| 2    | PROCESS-01     | STORAGE-01, ORCH-01                                | BASE-01                                                             |
| 2    | ORCH-01        | STORAGE-01, PROCESS-01                             | BASE-01                                                             |
| 3    | ARCHIVE-01     | none initially                                     | STORAGE-01, PROCESS-01, ORCH-01                                     |
| 3    | RESTORE-01     | none while archive format is moving                | STORAGE-01, PROCESS-01, ARCHIVE-01                                  |
| 4    | INTEGRATION-01 | PACKAGE-01 after command package layout stabilizes | ARCHIVE-01, RESTORE-01                                              |
| 5    | PACKAGE-01     | INTEGRATION-01 with coordinated command names      | ARCHIVE-01, RESTORE-01                                              |
| 5    | CONTAINER-01   | none on release files                              | INTEGRATION-01, PACKAGE-01                                          |
| 5    | DOCS-01        | late CONTAINER-01 work outside shared docs         | ARCHIVE-01, RESTORE-01, PACKAGE-01, CONTAINER-01 contract decisions |
| 6    | REVIEW-01      | none                                               | all implementation tasks                                            |

Shared hotspots that must not be edited concurrently without explicit coordination:

- `internal/toolconfig/shared.go` and its tests
- `storage/backup_contract.go`, selection, and retention tests
- any new PostgreSQL manifest package
- any new PostgreSQL process runner package
- `Makefile`
- `Dockerfile`
- `.github/workflows/release.yml`
- `.github/workflows/publish.yaml`
- `internal/flagdocs/` and generated `flags.md`
- `README.md` and shared website operation pages
- `sandbox/docker-compose.yml` and `test/test.bats`

Agents may run isolated Go package tests concurrently. Do not run formatting, generated documentation, release builds, asdf tests, integration Compose stacks, or full repository mutation-producing checks concurrently against the same worktree.

## Deferred Decisions Requiring Maintainer Input

No decision currently blocks execution. The following are deliberately deferred beyond the initial support scope:

1. Cluster-global role and tablespace backup through `pg_dumpall`.
2. Physical backup, point-in-time recovery, WAL archive, and replication support.
3. Bundling native PostgreSQL clients into cross-platform release archives.
4. Multi-database backup in one command execution.
5. Default destructive restore behavior such as `--clean` or `--create`.
6. A guarantee of atomic restore. Optional single-transaction support may be proposed only with compatibility tests and documented limitations.
7. Arbitrary passthrough arguments to PostgreSQL tools. Typed allowlisted options are required instead.

If implementation uncovers a need to change one of these decisions, mark the affected task `blocked`, record the exact technical reason and risk, and request maintainer approval before broadening scope.

## Full Verification Matrix

Run targeted checks after each task, package-level race checks after each wave, and the following from the final integrated worktree:

```sh
git diff --check
go test -shuffle=on ./...
go test -race -shuffle=on ./...
go vet ./...
go mod verify
make build VERSION=<test-version>
make test-asdf
bats --print-output-on-failure test/test.bats
make release-verify VERSION=<test-version>
```

Also run the repository-defined website validation commands and verify:

```sh
./dist/mongo-archive --version
./dist/mongo-unarchive --version
./dist/postgres-archive --version
./dist/postgres-unarchive --version
pg_dump --version
pg_restore --version
```

Release verification must include an actual MongoDB round trip and PostgreSQL round trip using the final container candidate. Record exact versions, commands, pass/fail results, CI links where applicable, and environmental blockers.

## Definition Of Done

- Four documented commands build, install, report versions, and ship through the supported release channels.
- PostgreSQL archive creates a validated custom-format payload and manifest, then applies the established strict storage delivery and retention contract.
- PostgreSQL restore validates database family and format before invoking `pg_restore --exit-on-error` against an existing target database.
- Passwords and other supplied secrets are absent from process arguments, logs, errors, notifications, manifests, and unintended retained files.
- Context cancellation and configured deadlines terminate PostgreSQL subprocess work and preserve cleanup/error semantics.
- MongoDB and PostgreSQL managed objects cannot be automatically selected or retained across their default prefixes.
- Existing MongoDB command contracts and tests remain compatible.
- Native users receive clear external-client requirements; the container includes and verifies pinned PostgreSQL clients.
- Stateful MongoDB and PostgreSQL integration round trips pass.
- Generated CLI references, README, website, examples, release policy, and runtime behavior agree.
- Targeted, full, race, vet, documentation, integration, installer, reproducibility, release, container, vulnerability, SBOM, and provenance checks pass or have exact recorded environmental blockers with successful CI evidence.
- An independent reviewer confirms all acceptance criteria and records no unresolved P0 or P1 findings.
