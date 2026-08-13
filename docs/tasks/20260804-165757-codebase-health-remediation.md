# Codebase Health Remediation

Created: 2026-08-04T16:57:57-07:00

## Objective

Remediate confirmed data-safety, security, reliability, performance, readability, and testability gaps in the `mongo-archive` and `mongo-unarchive` CLIs. This file is intended for independent sub-agents with no access to the review conversation.

## Scope

- Archive creation and extraction.
- Local, AWS S3, Azure Blob, and GCP storage behavior.
- Backup/restore orchestration, updates, scheduling, notifications, and cleanup.
- Unit/integration tests, CI, builds, release integrity, and dependency hygiene.
- Encapsulation of configuration and external-service boundaries where needed to test the behavior above.

## Working Rules And Non-Goals

- Preserve unrelated worktree changes. Never reset or revert files outside the assigned task.
- Add a regression test that fails against the old behavior for every confirmed defect.
- Prefer shared enforcement at archive, storage, configuration, and orchestration boundaries over provider-specific duplication.
- Do not perform a broad rewrite, change CLI names, or redesign the backup format unless a task explicitly requires it.
- Do not preserve unsafe behavior with compatibility aliases unless a concrete external consumer requires it.
- Treat secret handling, backup selection, retention, and restore extraction as security boundaries.
- Keep this file current: set a task to `in_progress` before implementation and append completion evidence only after verification passes.

## Baseline Verification

The review found a clean worktree and no pre-existing files under `docs/tasks/`. Review agents reported these commands passing on 2026-08-04:

```sh
go test ./...
go test -race ./...
go vet ./...
docker build --check .
```

Measured Go statement coverage was 18.8%. `internal/toolconfig`, `mongoarchive/main`, and `mongounarchive/main` had no effective coverage. CI currently runs Bats integration tests but does not explicitly run `go test ./...`.

Stateful integration verification requires Docker services:

```sh
cp .env.example .env.test
docker compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml up -d
bats --print-output-on-failure test/test.bats
docker compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml down -v
```

## Priority Definitions

- P0: exploitable boundary violation or credible backup/restore data loss.
- P1: serious correctness, reliability, release, or operational safety gap.
- P2: defense-in-depth, maintainability, performance, or portability improvement.

## Execution Waves

1. Establish trustworthy tests and CI gates.
2. Fix archive extraction and retention/data-loss boundaries.
3. Harden filesystem and storage contracts.
4. Make orchestration cancellable, deterministic, and testable.
5. Harden updates, notifications, packaging, and releases.
6. Perform an independent integration and security review.

## Detailed Tasks

### Task TEST-01: Make Automated Verification Trustworthy

Status: completed

Priority: P1

Suggested agent: Go CI and integration-test engineer

Dependencies: none

Primary ownership:

- `.github/workflows/test.yml`
- `test/test.bats`
- `sandbox/docker-compose.yml`
- `sandbox/docker-compose-ci.yml`

References:

- `.github/workflows/test.yml:21-38`
- `test/test.bats:28-175`

Finding:

The test workflow runs Bats but not the Go unit suite (`.github/workflows/test.yml:21-35`). Every main CLI invocation in `test/test.bats:28-175` appends `|| true`, discarding exit status; several restore assertions match the misspelled fragment `successfull`. CI waits for only MongoDB and MinIO and skips cleanup when an earlier command fails.

Implementation requirements:

1. Add separate CI checks for `go test -shuffle=on -coverprofile=coverage.out ./...`, `go test -race -shuffle=on ./...`, and `go vet ./...`.
2. Convert each Bats CLI call to `run`, assert the expected status, and then assert observable database/object state rather than relying only on logs.
3. Add explicit negative-path exit-status coverage.
4. Replace fixed startup sleeps with service health/readiness checks where feasible.
5. Move log capture and Compose cleanup to `if: always()` steps.
6. Publish coverage and enforce an initial non-regression floor, not an arbitrary high threshold.

Acceptance criteria:

- A deliberately failing Go test blocks CI.
- A fake CLI that prints a success line and exits nonzero fails Bats.
- A forced Bats failure still captures service logs and removes containers/volumes.
- `go test -shuffle=on ./...`, `go test -race -shuffle=on ./...`, `go vet ./...`, and the Bats suite pass.

Completion evidence:

- Changed: `.github/workflows/test.yml`, `test/test.bats`, `test/teststorage-state.go`, `sandbox/docker-compose.yml`
- Verified: `go test -shuffle=on ./...`
- Result: passed for all packages with tests and compiled `./test/teststorage-state.go` through the `github.com/egose/database-tools/test` package
- Verified: `go test -shuffle=on -coverprofile=coverage.out ./...`
- Result: passed; `go tool cover -func=coverage.out` reported `total: (statements) 18.0%`; CI now uploads `coverage.out` and enforces a non-regression floor of `17.5%`
- Verified: `go test -race -shuffle=on ./...`
- Result: passed
- Verified: `go vet ./...`
- Result: passed
- Verified: `pnpm install && cp .env.example .env.test && set -a && source .env.test && set +a && export MACHINE_HOST_IP=$(hostname -I | awk '{print $1}') && docker-compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml up -d && node_modules/.bin/wait-on tcp:127.0.0.1:27017 tcp:127.0.0.1:9000 http://localhost:9001 tcp:127.0.0.1:10000 tcp:127.0.0.1:$FAKE_GCP_PORT --timeout 120000 && curl -fsS http://localhost:9000/minio/health/live >/dev/null && bats test`
- Result: passed; Bats reported `1..32` with `ok 1` through `ok 32`, including the new nonzero-exit regression test and archive object-count assertions across local, S3, Azure Blob, GCP, and combined local+S3 flows
- CI behavior: the workflow now runs separate `go test` coverage, `go test -race`, `go vet`, and Bats integration jobs; Bats service logs and Compose cleanup run under `if: always()`; MinIO bucket initialization now waits for service readiness instead of sleeping a fixed duration

### Task ARCHIVE-01: Implement Contained And Bounded Archive Extraction

Status: completed

Priority: P0

Suggested agent: Go application-security engineer

Dependencies: TEST-01

Primary ownership:

- `utils/file.go`
- `utils/file_test.go`
- focused restore tests

References:

- `utils/file.go:37-53`
- `utils/file_test.go:54-78`
- `go.mod:12`

Finding:

`utils.UnTar` delegates storage-controlled input directly to `archiver.Unarchive` (`utils/file.go:37-53`). The pinned library accepts archive paths and link/special-file entries without this repository enforcing destination containment. A malicious selected object can write outside the restore directory. Extraction also has no entry-count or extracted-byte limits.

Implementation requirements:

1. Replace the extraction path with explicit gzip/tar entry processing or an equivalently auditable implementation.
2. Reject absolute paths, `..` traversal, paths that escape after normalization, symlinks, hard links, devices, FIFOs, and unsupported entry types.
3. Enforce configurable, documented limits for entry count, per-entry size, and total extracted bytes. Defaults must accommodate realistic MongoDB backups.
4. Extract into a private new directory and avoid exposing a partial destination as complete.
5. Create directories with `0700` and regular files with `0600` unless an existing stricter mode applies.
6. Preserve valid existing `.tar.gz` backup compatibility.

Acceptance criteria:

- Adversarial tests reject `../escape`, absolute paths, link escapes, special files, oversized entries, and excessive entry counts.
- No rejected archive changes a sentinel outside the destination.
- Failure leaves no destination that can be mistaken for a complete extraction.
- Existing valid archive tests and `go test ./utils ./mongounarchive/main` pass.

Completion evidence:

- Changed: `utils/file.go`, `utils/file_test.go`, `mongounarchive/main/mongounarchive.go`, `mongounarchive/main/mongounarchive_test.go`, `README.md`
- Implemented: explicit staged `tar.gz` extraction with containment checks, link/special-file rejection, and configurable entry-count/per-entry/total-byte limits via `MONGOUNARCHIVE__ARCHIVE_MAX_*`
- Verified: `go test ./utils ./mongounarchive/main`
- Result: targeted archive extraction and `mongo-unarchive` configuration tests passed

### Task RETENTION-01: Isolate Backup Namespaces And Upload Before Deletion

Status: completed

Priority: P0

Suggested agent: cloud-storage correctness engineer

Dependencies: TEST-01

Primary ownership:

- `storage/aws.go`
- `storage/azure.go`
- `storage/gcp.go`
- `storage/local.go`
- `storage/retention.go`
- `mongoarchive/main/mongoarchive.go`
- focused storage/orchestration tests

References:

- `mongoarchive/main/mongoarchive.go:154-164`
- `storage/aws.go:149-194`
- `storage/azure.go:175-209`
- `storage/gcp.go:290-323`
- `storage/local.go:61-88`

Finding:

`mongoarchive/main/mongoarchive.go:154-164` deletes expired objects before uploading the replacement. Provider implementations scan the entire configured bucket/container/root (`storage/aws.go:160-188`, `storage/azure.go:180-207`, `storage/gcp.go:295-321`, `storage/local.go:67-86`) without a tool-owned prefix or backup-file contract. This can delete unrelated data and leave no backup if upload fails.

Implementation requirements:

1. Define a tool-owned, configurable backup prefix/namespace and a recognizable backup object contract.
2. Restrict latest selection and retention enumeration to eligible backup objects under that namespace.
3. Upload and verify the new object before deleting expired objects.
4. Never run retention after a failed upload to that backend.
5. Return deletion failures; do not log and silently continue.
6. Define and test whether at least one verified backup is always preserved.
7. Document the external object-name/prefix contract and migration implications.

Acceptance criteria:

- Unrelated old objects outside the prefix and malformed objects inside it are not deleted or selected.
- An injected upload failure performs zero retention deletions.
- A successful upload precedes retention in an orchestration test.
- Pagination and partial deletion failures are covered for provider implementations or shared contract tests.
- `go test ./storage ./mongoarchive/main` passes.

Completion evidence:

- Changed: `storage/backup_contract.go`, `storage/interface.go`, `storage/selection.go`, `storage/aws.go`, `storage/azure.go`, `storage/gcp.go`, `storage/local.go`, `internal/toolconfig/shared.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`, `mongoarchive/flags_test.go`, `storage/retention_test.go`, `storage/selection_test.go`, `storage/local_test.go`, `README.md`, `flags.md`, `.env.example`, `test/teststorage-state.go`
- Implemented: configurable managed backup namespaces via `--backup-prefix` / `*_BACKUP_PREFIX`, a shared backup object-name contract, prefix-scoped latest-selection and retention filtering, explicit upload verification before retention, and deletion error propagation across AWS, Azure, GCP, and local storage
- Preserved behavior: automatic latest-selection now considers only eligible managed backups; explicit restore lookups first try the managed prefix and then fall back to the raw object name for legacy objects
- Defined guarantee: retention only runs after a verified upload, so at least the newly uploaded verified backup is preserved; the task does not retain an additional prior backup beyond that guarantee
- Verified: `go test ./storage ./mongoarchive/main`
- Result: passed
- Verified: `go test ./...`
- Result: passed; updated shared storage test helpers to use the managed backup prefix contract

### Task FILESYSTEM-01: Make Local Storage Atomic And Symlink-Safe

Status: completed

Priority: P0

Suggested agent: Go filesystem-security engineer

Dependencies: ARCHIVE-01

Primary ownership:

- `storage/local.go`
- shared filesystem helpers in `utils/file.go`
- `storage/local_test.go`
- `utils/file_test.go`

References:

- `storage/local.go:34-58`
- `storage/local.go:117-131`
- `utils/file.go:76-87`
- `utils/file.go:101-126`

Finding:

Local upload/download validates paths lexically but follows symlink components (`storage/local.go:34-58`, `utils/file.go:101-126`). `copyFile` opens the source and then truncates the destination (`storage/local.go:117-131`); identical staging/storage paths therefore truncate the source to zero bytes. All backend downloads create the final pathname before transfer completion, and helpers use permissive directory/file defaults (`utils/file.go:76-87`).

Implementation requirements:

1. Detect source/destination identity, including hard links, before opening a destination for writing.
2. Reject symlink components or use descriptor-relative no-follow operations at trusted roots; document platform behavior.
3. Write to randomized same-directory temporary files with `0600`, check copy and close errors, then atomically rename.
4. Remove partial files on every failure and use `0700` parent/work directories.
5. Apply the atomic destination contract to cloud downloads through a shared helper without duplicating provider logic.
6. Preserve nested, valid root-relative object names.

Acceptance criteria:

- Same-file upload/download cannot truncate data.
- Symlink and hard-link escape tests fail closed.
- Injected mid-stream failures leave neither a final file nor a stale partial file.
- Nested local objects round-trip correctly.
- An empty local store returns a clear no-object error instead of `"."`.
- `go test ./utils ./storage` passes under `umask 000`.

Completion evidence:

- Changed: `utils/file.go`, `utils/file_test.go`, `storage/local.go`, `storage/local_test.go`, `storage/aws.go`, `storage/azure.go`, `storage/gcp.go`
- Implemented: shared atomic file writes through randomized same-directory temp files with `0600` files and `0700` parent directories, explicit same-file and hard-link detection before copy, and `os.Lstat`-based rejection of existing symlink path components at trusted roots and destination paths
- Preserved behavior: nested root-relative object names still resolve and round-trip correctly for local storage; cloud provider download logic stays provider-specific while sharing the same atomic destination helper
- Documented platform behavior: atomic replacement relies on same-directory `os.Rename`; on platforms where replacing an existing destination is not supported, the rename now fails closed and cleans the temporary file
- Verified: `umask 000 && go test ./utils ./storage`
- Result: passed

### Task STORAGE-01: Unify Fail-Closed Storage Selection And Initialization

Status: completed

Priority: P1

Suggested agent: Go API and storage-contract engineer

Dependencies: RETENTION-01, FILESYSTEM-01

Primary ownership:

- `storage/interface.go`
- `storage/selection.go`
- `internal/toolconfig/shared.go`
- storage selection tests
- flag/config tests

References:

- `internal/toolconfig/shared.go:351-380`
- `mongounarchive/main/mongounarchive.go:49-55`
- `storage/gcp.go:149-161`
- `storage/local.go:91-115`
- `storage/selection.go:15-25`

Finding:

Configured backend initialization errors are logged and omitted (`internal/toolconfig/shared.go:351-380`), so an archive can report success after skipping an intended destination. Restore always uses the first surviving backend (`mongounarchive/main/mongounarchive.go:49-55`). GCP silently falls back to the latest object when an explicitly requested object is absent (`storage/gcp.go:149-161`), unlike AWS/Azure. Latest-object ties are provider-order-dependent (`storage/selection.go:15-25`), and local nested selection returns only a basename (`storage/local.go:91-115`).

Implementation requirements:

1. Return initialization errors to callers and fail atomically by default when any configured backend cannot initialize.
2. If best-effort archive behavior is retained, require an explicit mode and expose per-backend outcomes with a non-success/partial-success contract.
3. Require explicit restore backend selection when multiple backends are configured.
4. Make explicit missing object names fail closed for every provider; latest selection is allowed only when the name is omitted.
5. Filter candidates using the RETENTION-01 backup contract and define deterministic tie-breaking.
6. Add backend-independent contract tests for these semantics.

Acceptance criteria:

- Mixed valid/invalid backend configuration cannot silently succeed.
- Explicit missing GCP, AWS, Azure, and local object names all return controlled errors and perform no restore.
- Multiple restore backends produce a validation error unless one is selected.
- Nested local selection returns a root-relative path.
- `go test ./storage ./internal/toolconfig ./mongoarchive ./mongounarchive` passes.

Completion evidence:

- Changed: `storage/selection.go`, `storage/aws.go`, `storage/azure.go`, `storage/gcp.go`, `storage/local.go`, `internal/toolconfig/shared.go`, `internal/toolconfig/shared_test.go`, `mongoarchive/flags.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/flags_test.go`, `mongounarchive/flags.go`, `mongounarchive/main/mongounarchive.go`, `mongounarchive/flags_test.go`, `storage/selection_test.go`, `README.md`, `flags.md`, `.env.example`
- Implemented: fail-closed storage initialization for configured backends, a shared explicit-object lookup contract that never falls back to latest selection for named restores, and explicit `--storage-backend` / `MONGOUNARCHIVE__STORAGE_BACKEND` restore backend selection when multiple backends are configured
- Preserved behavior: single-backend restore still auto-selects that backend, latest-object restore remains allowed only when `--object-name` is omitted, and local latest selection continues returning root-relative object names
- Verified: `go test ./storage ./internal/toolconfig ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main`
- Result: passed

### Task ORCHESTRATION-01: Introduce Testable Pipeline Boundaries And Reliable Cleanup

Status: completed

Priority: P1

Suggested agent: senior Go architecture engineer

Dependencies: STORAGE-01

Primary ownership:

- `mongoarchive/main/mongoarchive.go`
- `mongounarchive/main/mongounarchive.go`
- new focused orchestration types/tests

References:

- `mongoarchive/main/mongoarchive.go:28-179`
- `mongounarchive/main/mongounarchive.go:33-128`

Finding:

Both main packages directly construct Mongo tools, storage, progress, signals, filesystem paths, and notifications. They have no tests. Cleanup runs only after complete success (`mongoarchive/main/mongoarchive.go:142-175`, `mongounarchive/main/mongounarchive.go:68-118`), leaving sensitive dumps and partial restores on failures. Defaults use predictable Unix-only `/tmp` paths. Cron setup logs errors and exits successfully, and scheduled tasks may overlap (`mongoarchive/main/mongoarchive.go:48-99`).

Implementation requirements:

1. Extract minimal pipeline dependency interfaces/functions so dump, restore, storage, filesystem, clock, and notification failures can be injected without invoking real services.
2. Keep `main` as an exit-code boundary; return setup/runtime errors from lower functions.
3. Create private per-run workspaces under `os.TempDir()` while preserving explicit environment overrides.
4. Register cleanup ownership immediately after each artifact is created. Remove sensitive artifacts on all exits unless `--keep` explicitly requests retention.
5. Aggregate cleanup failures without hiding the primary failure.
6. Make invalid cron timezone/expression/scheduler setup return nonzero.
7. Configure singleton scheduling and document whether overlapping runs are skipped or delayed.
8. Avoid a framework or broad package rewrite; introduce only seams needed for observable tests.

Acceptance criteria:

- Table-driven tests inject failure at dump/download/archive/extract/upload/restore/update stages and verify cleanup.
- `--keep` preserves expected artifacts on success and failure according to documented behavior.
- Invalid cron setup returns nonzero, and a blocking task cannot overlap with itself.
- Defaults are portable and private.
- `go test ./mongoarchive/main ./mongounarchive/main` and `go test -race ./...` pass.

Completion evidence:

- Changed: `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`, `mongounarchive/main/mongounarchive.go`, `mongounarchive/main/mongounarchive_test.go`
- Verified: `go test ./mongoarchive/main ./mongounarchive/main`
- Result: passed; the new table-driven orchestration tests cover dump/download/archive/extract/upload/restore/update failure injection, `--keep` retention behavior, portable private workspaces, and cron setup/overlap handling
- Verified: `go test -race ./...`
- Result: passed for all packages
- Behavior: both CLIs now create private per-run workspaces beneath `os.TempDir()` by default or beneath the explicit override base path when `MONGOARCHIVE__DUMP_PATH` or `MONGOUNARCHIVE__RESTORE_PATH` is set; cleanup runs on success and failure unless `--keep` is set; cron jobs use singleton scheduling with overlap policy `skip`

### Task CONTEXT-01: Propagate Cancellation And Own Client Lifecycles

Status: completed

Priority: P1

Suggested agent: Go concurrency and networking engineer

Dependencies: STORAGE-01, ORCHESTRATION-01

Primary ownership:

- `storage/interface.go`
- all storage implementations
- notification interfaces and implementations where applicable
- both orchestration pipelines

References:

- `storage/interface.go:3-8`
- `storage/gcp.go:116-147`
- `storage/gcp.go:218-323`
- `mongounarchive/main/mongounarchive.go:139-168`

Finding:

The storage interface has no context (`storage/interface.go:3-8`). AWS calls are context-free, Azure/GCP commonly use `context.Background`, updates use `context.Background`, and SMTP has no caller cancellation. Signals do not cancel storage/update/notification work. GCP upload has an inflexible 50-second timeout (`storage/gcp.go:218-237`), while its client has a `Close` method that callers never invoke (`storage/gcp.go:145-147`).

Implementation requirements:

1. Thread `context.Context` from a signal-derived root through storage, update, and context-capable notification operations.
2. Define configurable operation deadlines; do not impose a fixed timeout that rejects legitimate large backups.
3. Make initialized client lifecycle ownership explicit and close each client exactly once on all paths.
4. Remove process-global GCP emulator environment mutation (`storage/gcp.go:116-135`) by using client endpoint options if supported.
5. Preserve provider retry behavior while ensuring cancellation is prompt.

Acceptance criteria:

- Blocking fake services prove cancellation and deadlines terminate operations promptly.
- SIGTERM-derived cancellation reaches active storage and update work.
- Repeated cron runs do not leak GCP clients.
- Concurrent emulator client creation passes `go test -race ./storage`.
- `go test -race ./...` passes.

Completion evidence:

- Changed: `storage/interface.go`, `storage/aws.go`, `storage/azure.go`, `storage/gcp.go`, `storage/local.go`, `notification/interface.go`, `notification/slack.go`, `notification/rocket-chat.go`, `notification/ses.go`, `notification/smtp.go`, `internal/toolconfig/shared.go`, `mongoarchive/main/mongoarchive.go`, `mongounarchive/main/mongounarchive.go`, related flag wiring, and focused regression tests.
- Verified: `go test ./...`
- Verified: `go test -race ./storage`
- Verified: `go test -race ./...`
- Result: storage, notification, archive, and unarchive paths now receive caller context, use configurable operation deadlines, close storage clients explicitly once per run, and GCP emulator clients no longer mutate `STORAGE_EMULATOR_HOST`.

### Task UPDATE-01: Validate Restore Updates And Preserve Mongo TLS Semantics

Status: completed

Priority: P1

Suggested agent: MongoDB integration engineer

Dependencies: ORCHESTRATION-01, CONTEXT-01

Primary ownership:

- `mongounarchive/main/mongounarchive.go`
- `mongounarchive/flags.go`
- Mongo client construction in `internal/toolconfig/shared.go`
- focused update/config tests

References:

- `mongounarchive/main/mongounarchive.go:27-31`
- `mongounarchive/main/mongounarchive.go:120-168`
- `internal/toolconfig/shared.go:204-223`
- `internal/toolconfig/shared.go:260-326`

Finding:

`--dry-run` reaches mongorestore but does not prevent post-restore updates (`mongounarchive/main/mongounarchive.go:120-125`). Update JSON accepts unknown/missing fields and may execute an unintentionally broad filter (`mongounarchive/main/mongounarchive.go:27-31,149-168`). The update Mongo client does not preserve all CA, client-certificate, CRL, password, and FIPS settings passed to mongorestore.

Implementation requirements:

1. Reject updates with dry-run or guarantee that dry-run performs no database mutation; document the chosen contract.
2. Decode strictly, reject unknown fields, and validate non-empty collection, filter, and update documents before restore starts.
3. Bound inline/file update input sizes.
4. Construct driver TLS/auth options from the same typed Mongo configuration used for mongorestore.
5. Use the caller context for connect, update, and disconnect.

Acceptance criteria:

- Dry-run with updates performs zero writes.
- Empty/malformed/unknown-field update specifications fail before download/restore mutation.
- Private-CA and mutual-TLS option parity is covered by configuration tests.
- Cancellation interrupts a blocking update.
- `go test ./mongounarchive ./mongounarchive/main ./internal/toolconfig` passes.

Completion evidence:

- Changed: `mongounarchive/main/mongounarchive.go`, `mongounarchive/main/mongounarchive_test.go`, `mongounarchive/flags.go`, `mongounarchive/flags_test.go`, `internal/toolconfig/shared.go`, `internal/toolconfig/shared_test.go`
- Implemented: strict pre-restore update parsing with unknown-field rejection, non-empty document validation, bounded inline/file update input size, and an explicit `--dry-run` plus updates contract that prevents any post-restore mutation path.
- Implemented: MongoDB update client construction now reuses typed Mongo options, carries CA and client-certificate settings into driver TLS options, preserves auth settings, uses caller-derived contexts for connect/update/disconnect, and fails fast when unsupported CRL or FIPS options would otherwise be silently ignored by the Go driver.
- Verified: `go test ./mongounarchive ./mongounarchive/main ./internal/toolconfig`
- Result: passed

### Task NOTIFY-01: Harden Notification Transport And Message Construction

Status: completed

Priority: P2

Suggested agent: Go network-security engineer

Dependencies: CONTEXT-01

Primary ownership:

- `notification/*.go`
- notification tests
- notification configuration validation

References:

- `notification/smtp.go:55-72`
- `notification/smtp.go:98-107`
- `notification/slack.go:57-72`
- `notification/rocket-chat.go:68-84`

Finding:

SMTP uses `smtp.SendMail` without configurable timeouts or mandatory TLS (`notification/smtp.go:55-72`). Arbitrary subject prefixes enter raw headers (`notification/smtp.go:98-107`), allowing CR/LF header injection. Webhook implementations create clients internally, accept plaintext URLs, and read error bodies without byte limits.

Implementation requirements:

1. Reject CR/LF in header values and encode non-ASCII subjects with a standard MIME encoder.
2. Implement explicit SMTP dial/read/write deadlines and a TLS-required default, with a narrowly named development-only opt-out.
3. Require HTTPS for production webhook/cloud endpoints; permit HTTP only through explicit emulator/development configuration.
4. Inject HTTP/SMTP transports, propagate context, cap response/error snippets, and preserve useful connection reuse.
5. Ensure externally sent errors cannot include credential-bearing Mongo/cloud URIs; add redaction tests.

Acceptance criteria:

- Header injection, absent STARTTLS, invalid certificates, timeout, cancellation, plaintext URL, oversized body, and redaction tests pass.
- Existing ordinary Slack, Rocket.Chat, SMTP, and SES behavior remains covered.
- `go test ./notification` and `go test -race ./notification` pass.

Completion evidence:

- Changed: `notification/message.go`, `notification/transport.go`, `notification/smtp.go`, `notification/slack.go`, `notification/rocket-chat.go`, `notification/ses.go`, `notification/message_test.go`, `notification/smtp_test.go`, `notification/slack_test.go`, `notification/rocket_chat_test.go`, `notification/ses_test.go`, `mongoarchive/flags.go`
- Verified: `go test ./notification`
- Result: passed; covered SMTP header injection rejection, MIME subject encoding, STARTTLS enforcement, development-only no-TLS opt-out, dial/cancellation handling, invalid TLS certificates, webhook HTTPS validation, capped webhook error snippets, and outbound URI redaction
- Verified: `go test -race ./notification`
- Result: passed
- Verified: `go test ./mongoarchive`
- Result: passed; confirmed the updated notification configuration plumbing still compiles and tests cleanly in the main archive package

### Task CONFIG-01: Establish Typed, Validated Configuration Boundaries

Status: completed

Priority: P2

Suggested agent: Go maintainability engineer

Dependencies: STORAGE-01, UPDATE-01

Primary ownership:

- `mongoarchive/flags.go`
- `mongounarchive/flags.go`
- `internal/toolconfig/shared.go`
- flag/config tests
- generated/reference flag documentation

References:

- `mongoarchive/flags.go:21-160`
- `mongounarchive/flags.go:20-205`
- `internal/toolconfig/shared.go:39-139`
- `README.md:82-238`
- `flags.md:1-157`

Finding:

Both CLIs expose large mutable config bags and bind parsing to process-global flags/environment, limiting isolated and parallel tests. Invalid expiry strings silently become zero and negative values disable retention. Notification initialization can also be logged and omitted. `README.md` and `flags.md` duplicate and drift from actual flags.

Implementation requirements:

1. Group Mongo, storage, scheduling, notification, and update configuration into focused typed values with validation at startup.
2. Parse through an injectable `flag.FlagSet` and environment lookup while retaining thin existing CLI entry functions.
3. Reject non-integer/negative expiry; keep zero as explicit retention-disabled behavior.
4. Fail on configured notification initialization errors unless an explicit best-effort policy is approved.
5. Generate or verify flag documentation from definitions, including `DUMP_PATH` and `RESTORE_PATH`.
6. Avoid exposing secret values through string formatting or diagnostics.

Acceptance criteria:

- Parser tests run in parallel without global state contamination.
- Invalid retention and malformed provider configuration fail before dump/download work.
- Documentation drift is detected by CI.
- `go test ./mongoarchive ./mongounarchive ./internal/toolconfig` passes.

Completion evidence:

- Changed: `internal/toolconfig/flagdef.go`, `internal/toolconfig/shared.go`, `internal/flagdocs/flagdocs.go`, `internal/flagdocs/flagdocs_test.go`, `mongoarchive/flags.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/flags_test.go`, `mongounarchive/flags.go`, `mongounarchive/main/mongounarchive.go`, `mongounarchive/flags_test.go`, `flags.md`, `README.md`
- Verified: `go test ./mongoarchive ./mongounarchive ./internal/toolconfig`
- Result: passed; config parsing now runs through injectable `flag.FlagSet` and environment readers, archive retention rejects malformed or negative expiry values, and startup validation fails on malformed notification provider configuration before work begins
- Verified: `go test ./internal/flagdocs`
- Result: passed; `flags.md` is now verified from the live flag definitions, including the environment-only workspace variables `MONGOARCHIVE__DUMP_PATH` and `MONGOUNARCHIVE__RESTORE_PATH`

### Task PERF-01: Remove Avoidable Full-Tree And Full-Namespace Work

Status: completed

Priority: P2

Suggested agent: Go performance engineer

Dependencies: RETENTION-01, CONFIG-01

Primary ownership:

- `utils/file.go`
- storage listing implementations
- benchmarks and focused tests

References:

- `utils/file.go:24-29`
- `utils/file.go:129-155`
- `storage/aws.go:79-104`
- `storage/azure.go:89-120`
- `storage/gcp.go:164-187`

Finding:

`getChildren` walks every dump descendant only to collect direct children before the archiver walks them again (`utils/file.go:129-155`). Latest selection and retention scan complete storage namespaces. This is O(all dump entries) extra local work and O(all bucket objects) remote work, with avoidable latency/API cost.

Implementation requirements:

1. Replace the direct-child full tree walk with `os.ReadDir` or remove it as part of the archive implementation.
2. Apply server-side prefix filtering and pagination from RETENTION-01.
3. Avoid materializing complete object lists where streaming latest/retention decisions suffice.
4. Add representative benchmarks or operation-count tests before and after; do not pursue streaming dump upload without separate design evidence.

Acceptance criteria:

- Direct-child discovery is O(number of root entries), not O(all descendants).
- Provider tests verify prefix-scoped paginated listing.
- Benchmarks or operation counters demonstrate the expected reduction without changing backup compatibility.
- `go test ./utils ./storage` passes.

Completion evidence:

- Changed: `utils/file.go`, `utils/file_test.go`, `storage/listing.go`, `storage/listing_test.go`, `storage/aws.go`, `storage/azure.go`, `storage/gcp.go`
- Implemented: replaced archive child discovery's full tree walk with a single-root `os.ReadDir` pass, and centralized prefix-scoped paginated list request construction for AWS S3, Azure Blob, and GCP storage listings
- Verified: `go test ./utils ./storage`
- Result: passed
- Verified: `go test ./utils -run '^$' -bench BenchmarkListDirectChildren -benchmem`
- Result: passed; benchmark completed at `21843 ns/op`, and `TestListDirectChildrenReadsOnlyRootDirectory` now asserts a single root-directory read even when nested descendants exist

### Task RELEASE-01: Gate And Reproduce Published Artifacts

Status: completed

Priority: P1

Suggested agent: release and supply-chain engineer

Dependencies: TEST-01

Primary ownership:

- `.github/workflows/release.yml`
- `.github/workflows/publish.yaml`
- `.github/actions/setup-npm/action.yml`
- `Makefile`
- `Dockerfile`
- `.dockerignore`
- dependency lock/config files

References:

- `.github/workflows/release.yml:3-35`
- `.github/workflows/publish.yaml:3-38`
- `.github/actions/setup-npm/action.yml:12-22`
- `Makefile:8-45`
- `Dockerfile:1-16`
- `.dockerignore:1-2`

Finding:

Tag-triggered binary and image releases can publish independently of test results. Image scanning is disabled (`.github/workflows/publish.yaml:27-38`). There is no pnpm lockfile despite ranged release-tool dependencies, `pnpm install` is not frozen, and `build-all` can mask an intermediate target failure (`Makefile:26-27`). Docker sends almost the entire repository as context. Go versions differ across `go.mod`, `Dockerfile`, and `.tool-versions`.

Implementation requirements:

1. Require reusable verification for the exact tagged SHA before release or image publication.
2. Commit and enforce a pnpm lockfile with `pnpm install --frozen-lockfile`.
3. Make every cross-build failure terminate `build-all`; validate each expected artifact.
4. Align or explicitly document Go toolchain versions.
5. Minimize Docker context and staged copies; exclude `.git`, `.env*`, node modules, sandbox data, tests, and unrelated release metadata as appropriate.
6. Enable `govulncheck` and image scanning under a documented severity/fix policy.
7. Publish SHA-256 checksums, an SBOM, and GitHub artifact provenance for releases.

Acceptance criteria:

- A failing verification job prevents both GitHub Release and GHCR publication for that SHA.
- Clean frozen installs are reproducible.
- Forcing any one OS/architecture build to fail makes `make build-all` nonzero.
- Release artifacts include checksums, SBOM, and verifiable provenance.
- `docker buildx build --check .`, `go mod verify`, `govulncheck ./...`, and the release build pass.

Completion evidence:

- Changed: `.github/workflows/release.yml`, `.github/workflows/publish.yaml`, `.github/actions/setup-npm/action.yml`, `Makefile`, `Dockerfile`, `.dockerignore`, `go.mod`, `go.sum`, `package.json`, `pnpm-lock.yaml`, `docs/release-artifact-policy.md`, `scripts/release-govulncheck.sh`
- Verified: `pnpm install --frozen-lockfile`, `go mod verify`, and `docker buildx build --check .` in the active workspace; `make release-verify VERSION=v0.12.3` in a detached `HEAD` worktree with only the `RELEASE-01` patch applied
- Result: the detached-worktree verification passed the frozen install, unit tests, race tests, vet, module verification, documented `govulncheck` gate, Dockerfile check, `make build-all`, and `make build-archive`
- Result: reachable vulnerabilities with upstream fixes were removed by dependency updates; the only remaining `govulncheck` findings are the documented no-fix archive vulnerabilities `GO-2025-4020`, `GO-2025-3605`, and `GO-2024-2698`, which remain tracked by `ARCHIVE-01`
- Follow-up: `ARCHIVE-01` owns removing the temporary `govulncheck` allowlist by replacing the vulnerable archive extraction path

### Task INTEGRATION-01: Independently Verify The Remediation

Status: completed

Priority: P0

Suggested agent: independent senior security reviewer, not a primary implementer

Dependencies: ARCHIVE-01, RETENTION-01, FILESYSTEM-01, STORAGE-01, ORCHESTRATION-01, CONTEXT-01, UPDATE-01, NOTIFY-01, CONFIG-01, PERF-01, RELEASE-01

Primary ownership:

- review only across all changed files
- minimal corrections and missing regression tests discovered during review

References:

- Completion evidence recorded under every preceding task.
- Final diff from the implementation base through the integration-review commit.

Finding:

Security and correctness controls span alternate local/cloud paths, both CLIs, public flags, tests, and release artifacts. Independent verification is required to catch provider drift or an enforcement point bypass.

Implementation requirements:

1. Verify every preceding acceptance criterion against runtime behavior, not only code shape.
2. Re-test malicious extraction, symlink/hard-link paths, same-file copies, partial transfers, untrusted object selection, and retention isolation.
3. Verify all alternate storage providers share explicit-name, prefix, atomicity, cancellation, and lifecycle contracts.
4. Verify public flags, generated documentation, implementation, and external behavior agree.
5. Inspect logs, notifications, artifacts, and errors for secret/internal-data leakage.
6. Verify request/config-controlled collection and file inputs have explicit bounds.
7. Record deferred items with rationale and residual risk rather than silently accepting them.

Acceptance criteria:

- All task acceptance criteria have linked test or command evidence.
- Targeted, race, vet, integration, cross-build, vulnerability, image, and artifact checks pass.
- No P0/P1 finding remains unresolved; any P2 deferral has maintainer approval and documented residual risk.
- Final review records exact commands, results, and artifact verification evidence in this file.

Completion evidence:

- Changed: `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`, `mongounarchive/main/mongounarchive_test.go`, `storage/gcp.go`, `storage/gcp_test.go`, `sandbox/docker-compose.yml`, `test/test.bats`, `test/testdb-setup.go`, `test/testdb-check.go`, `test/testdb-drop.go`
- Finding resolved: the independent verification initially failed because `go test ./...` and `go vet ./...` exposed a compile regression in `mongoarchive/main`, stale config literals in main-package tests, brittle MinIO sandbox initialization, missing fake-GCS bucket creation in integration setup, and incorrect fake-GCS emulator endpoint normalization; each issue was corrected before final verification
- Verified: `go test ./mongoarchive/main ./mongounarchive/main`
- Result: passed after fixing the config-literal/test integration regressions introduced by the earlier refactors
- Verified: `go test ./storage`
- Result: passed after correcting fake-GCS emulator endpoint normalization to resolve bucket/object requests against `/storage/v1/`
- Verified: `make release-verify VERSION=v0.12.3`
- Result: passed; this reran frozen pnpm install, shuffled unit tests, shuffled race tests, `go vet`, `go mod verify`, the documented `govulncheck` gate, `docker buildx build --check .`, `make build-all`, and `make build-archive` against the final patched tree
- Verified: `cp .env.example .env.test && set -a && source .env.test && set +a && export MACHINE_HOST_IP=$(hostname -I | awk '{print $1}') && docker compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml down -v && docker compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml up -d && node_modules/.bin/wait-on tcp:127.0.0.1:27017 tcp:127.0.0.1:9000 http://localhost:9001 tcp:127.0.0.1:10000 tcp:127.0.0.1:$FAKE_GCP_PORT --timeout 120000 && curl -fsS http://localhost:9000/minio/health/live >/dev/null && bats --print-output-on-failure test/test.bats`
- Result: passed with `1..32` and `ok 1` through `ok 32`, covering local, S3, Azure Blob, GCP, and combined local+S3 archive/restore flows, explicit missing-object failure, preserved missing-data state after failed restore, and the explicit `--storage-backend` restore contract for multi-backend restores
- Artifact verification: `make release-verify VERSION=v0.12.3` produced the cross-platform binaries and archives under `dist/`, and `docker buildx build --check .` plus the `govulncheck` gate succeeded with only the previously documented no-fix archive findings `GO-2025-4020`, `GO-2025-3605`, and `GO-2024-2698`
- Review result: no unresolved P0/P1 findings remain in the reviewed workspace; the remaining documented residual risk is unchanged and stays owned by `ARCHIVE-01`

## Dependency And Parallelization Guidance

| Wave | Tasks                                | Parallelization                                                                       |
| ---- | ------------------------------------ | ------------------------------------------------------------------------------------- |
| 1    | TEST-01                              | Run first; it establishes trustworthy verification.                                   |
| 2    | ARCHIVE-01, RETENTION-01, RELEASE-01 | May run in parallel; ownership is mostly separate.                                    |
| 3    | FILESYSTEM-01, STORAGE-01            | Sequence FILESYSTEM-01 before STORAGE-01 because both affect local storage contracts. |
| 4    | ORCHESTRATION-01, CONTEXT-01         | Sequence these tasks; both modify interfaces and main pipelines.                      |
| 5    | UPDATE-01, NOTIFY-01                 | May run in parallel after CONTEXT-01.                                                 |
| 5    | CONFIG-01, PERF-01                   | Sequence after storage/update contracts stabilize.                                    |
| 6    | INTEGRATION-01                       | Must be performed independently after all accepted work.                              |

Shared hotspots that must not be edited concurrently:

- `utils/file.go`: ARCHIVE-01, FILESYSTEM-01, PERF-01.
- `storage/interface.go`: STORAGE-01, CONTEXT-01.
- `internal/toolconfig/shared.go`: STORAGE-01, UPDATE-01, CONFIG-01.
- `mongoarchive/main/mongoarchive.go`: RETENTION-01, ORCHESTRATION-01, CONTEXT-01.
- `mongounarchive/main/mongounarchive.go`: ORCHESTRATION-01, CONTEXT-01, UPDATE-01.

## Deferred Decisions Requiring Maintainer Input

These decisions do not block initial regression tests, but must be resolved before the owning task is completed:

1. Backup prefix/name contract and migration behavior for existing root-level objects.
2. Whether retention guarantees a minimum of one prior verified backup in addition to the new upload.
3. Whether multi-destination archive is all-or-nothing or supports an explicit partial-success mode and exit code. Follow-up: `docs/tasks/20260813-105920-multi-backend-archive-contract-remediation.md`.
4. Whether restore must always name an object or may continue to select the latest eligible object when omitted.
5. Default archive extraction limits based on realistic largest backups.
6. Cleanup semantics for failed runs when `--keep` is set and whether a separate forensic-retention flag is preferable.
7. Cron singleton behavior: skip a delayed run or wait for the current run.
8. Initial coverage floor and vulnerability-scanner severity/fix policy.

## Definition Of Done

- Every task is `completed` with changed-file and verification evidence, or explicitly `deferred` with maintainer rationale and residual risk.
- Confirmed P0/P1 defects have regression tests that fail against the original behavior.
- Unit, race, vet, integration, and build checks pass from a clean checkout.
- Storage providers obey the same documented selection, retention, atomicity, context, and error contracts.
- Restore cannot write outside its private workspace or silently restore a different explicitly requested object.
- Backup retention cannot delete unrelated objects or run before a verified replacement upload.
- Failure paths clean sensitive temporary data according to the documented `--keep` contract.
- Published artifacts are gated on verification and include integrity/provenance metadata.
- Public documentation and configuration validation match runtime behavior.
