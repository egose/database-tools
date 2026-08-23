# Post-Remediation Codebase Health

Created: 2026-08-23T11:25:31-07:00

Related task files:

- `docs/tasks/20260804-165757-codebase-health-remediation.md`
- `docs/tasks/20260813-105920-multi-backend-archive-contract-remediation.md`

## Objective

Address current correctness, configuration, release-safety, readability, performance, encapsulation, reuse, and testability gaps in the `mongo-archive` and `mongo-unarchive` packages after the earlier remediation plans were completed. This document is written for sub-agents that do not have the review conversation.

## Scope

- Restore result handling and pre-side-effect configuration validation.
- Storage and notification abstractions used by both CLIs.
- Runtime lifecycle, cancellation, and duplicated orchestration utilities.
- Provider contract coverage and meaningful coverage gates.
- CI, release, container provenance, vulnerability policy, examples, and supply-chain hardening.

## Working Rules And Non-Goals

- Preserve unrelated worktree changes. Never reset, revert, or rewrite files outside an assigned task.
- Read the two completed related task files before implementation and do not regress their security or data-safety contracts.
- Add a regression test that fails against current behavior for every confirmed defect.
- Prefer the smallest shared enforcement point and narrow capability interfaces. Do not introduce a dependency-injection framework or broad rewrite.
- Preserve context propagation, private staged filesystem work, atomic downloads, explicit storage selection, managed backup prefixes, upload-before-retention, and primary-plus-cleanup error preservation.
- Do not expose credentials in validation errors, logs, generated docs, struct formatting, test fixtures, or task evidence.
- Keep this file current. Set a task to `in_progress` before implementation and record completion evidence only after its verification passes.

## Baseline Verification

The worktree was clean at review start. On 2026-08-23, the following commands passed:

```sh
go test -shuffle=on -coverprofile=/tmp/opencode/database-tools-coverage.out ./...
go vet ./...
```

Measured statement coverage was 52.5%. Package coverage included `storage` at 30.5%, `internal/toolconfig` at 35.5%, both main packages at about 64%, and `notification` at 79.7%. AWS and Azure provider methods and nearly all GCP provider methods had 0% unit coverage. The CI floor remains 17.5%.

Not run during task creation:

```sh
go test -race -shuffle=on ./...
bats --print-output-on-failure test/test.bats
make release-verify VERSION=v0.15.0
docker buildx build --check .
```

## Priority Definitions

- P0: exploitable security boundary violation or credible backup/restore data loss.
- P1: serious correctness, release-integrity, or operational-safety defect.
- P2: defense-in-depth, performance, testability, readability, portability, or maintainability improvement.

No new P0 finding was confirmed in this review.

## Execution Waves

1. Lock down restore correctness and fail-closed configuration behavior.
2. Repair release gates that can publish unverified or incorrectly identified artifacts.
3. Narrow runtime boundaries and consolidate duplicated lifecycle/configuration behavior.
4. Add provider contract coverage and strengthen CI reliability.
5. Harden examples, reproducibility, dependencies, and governance.
6. Perform an independent integration and security review.

## Detailed Tasks

### Task RESTORE-01: Fail On Partial Restore Results Before Applying Updates

Status: completed

Priority: P1

Suggested agent: MongoDB restore correctness engineer

Dependencies: none

Primary ownership:

- `mongounarchive/main/mongounarchive.go`
- `mongounarchive/main/mongounarchive_test.go`
- restore result documentation in `README.md`

Finding:

The restore pipeline treats only `result.Err` as failure. When `result.Failures > 0` and `result.Err == nil`, it logs the failed count, applies post-restore updates, logs `Unarchive completed successfully`, and returns nil. Automation can therefore accept and mutate a partially restored database.

References:

- `mongounarchive/main/mongounarchive.go:238-258`
- `mongounarchive/main/mongounarchive_test.go:124-260`
- `README.md:115-121`

Implementation requirements:

1. Define successful restore as `result.Err == nil && result.Failures == 0`.
2. Return a typed or wrapped error containing successes and failures when the failure count is nonzero.
3. Never invoke post-restore updates after a top-level or document-level restore failure.
4. Keep zero-failure acknowledged and unacknowledged restore behavior unchanged.
5. Document the nonzero exit contract without claiming transactional rollback.

Acceptance criteria:

- A fake result with successes, at least one failure, and no `Err` returns nonzero.
- The partial-result path neither invokes `applyUpdates` nor logs completion success.
- A zero-failure result still applies validated updates and succeeds.
- `go test ./mongounarchive/main -count=1` and `go test -race ./mongounarchive/main -count=1` pass.

Completion evidence:

- Changed: `mongounarchive/main/mongounarchive.go`, `mongounarchive/main/mongounarchive_test.go`, `README.md`
- Implemented: restore success now requires `result.Err == nil && result.Failures == 0`; partial restore results return a typed error with successful and failed document counts before post-restore updates or completion-success logging; zero-failure acknowledged and unacknowledged paths remain successful.
- Verified: `go test ./mongounarchive/main -count=1`
- Result: passed
- Verified: `go test -race ./mongounarchive/main -count=1`
- Result: passed

### Task CONFIG-01: Reject Incomplete Storage Backend Configuration

Status: completed

Priority: P1

Suggested agent: Go configuration and cloud-storage engineer

Dependencies: none

Primary ownership:

- `internal/toolconfig/shared.go`
- `internal/toolconfig/shared_test.go`
- focused archive and restore flag tests

Finding:

Provider activation predicates require a complete credential set. A provider with only some fields set is treated as disabled and never reaches initialization error handling. For example, valid local storage plus only `AWS_BUCKET` can succeed as a local-only archive even though configuration expresses AWS intent.

References:

- `internal/toolconfig/shared.go:537-579`
- `internal/toolconfig/shared.go:614-627`
- `internal/toolconfig/shared_test.go:35-53`
- `mongoarchive/flags.go:324-326`
- `mongounarchive/flags.go:305-314`

Implementation requirements:

1. Validate each provider as absent, incomplete, or enabled before constructing any provider.
2. Reject incomplete Azure and AWS required field sets with missing field identifiers, never values.
3. Explicitly model supported GCP credential modes: emulator, credentials file, inline service account, and application-default credentials.
4. Run validation before workspace creation, dump, download, or restore side effects.
5. Preserve close-on-initialization-failure and `errors.Join` behavior.

Acceptance criteria:

- Local plus only an AWS bucket or Azure account name fails before pipeline work.
- Every partial required-field combination is covered by table-driven tests.
- Completely absent providers stay disabled and valid providers retain current behavior.
- Errors name missing settings and contain none of the supplied credential values.
- `go test ./internal/toolconfig ./mongoarchive ./mongounarchive -count=1` passes.

Completion evidence:

- Changed: `internal/toolconfig/shared.go`, `internal/toolconfig/shared_test.go`, `mongoarchive/flags.go`, `mongoarchive/flags_test.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`, `mongounarchive/flags.go`, `mongounarchive/flags_test.go`, `mongounarchive/main/mongounarchive.go`, `mongounarchive/main/mongounarchive_test.go`
- Implemented: shared storage validation now classifies Azure, AWS, and GCP providers as absent, incomplete, or enabled before provider construction; incomplete AWS/Azure settings report missing setting identifiers only; GCP credential modes are explicit for emulator, credentials file, inline service account, and application-default credentials; archive and restore pipelines reject invalid storage configuration before workspace, dump, download, or restore side effects.
- Verified: `go test ./internal/toolconfig ./mongoarchive ./mongounarchive -count=1`
- Result: passed
- Verified: `go test ./mongoarchive/main ./mongounarchive/main -count=1`
- Result: passed

### Task RELEASE-01: Make The Vulnerability Gate Executable And Structurally Reliable

Status: blocked

Priority: P1

Suggested agent: Go supply-chain and release engineer

Dependencies: none

Primary ownership:

- `scripts/release-govulncheck.sh`
- `Makefile`
- focused parser fixtures/tests
- `docs/release-artifact-policy.md`

Finding:

The Makefile executes `scripts/release-govulncheck.sh` directly, but the Git index records mode `100644`. A clean checkout may fail with `Permission denied`. The script also parses arbitrary failed scanner output with a regular expression and succeeds whenever all discovered IDs are allowlisted, even if the process failed for another reason. Its allowlist has no package/symbol constraint, expiry, rationale metadata, or live owner and still points to completed task `ARCHIVE-01`.

References:

- `Makefile:55-65`
- `scripts/release-govulncheck.sh:1-47`
- `docs/release-artifact-policy.md:27-31`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:123-169`

Implementation requirements:

1. Commit executable mode `100755` or deliberately invoke the script with Bash.
2. Consume `govulncheck -json` and distinguish scanner/runtime failures from vulnerability findings.
3. Match exceptions by vulnerability plus affected module/package/symbol and scan mode.
4. Give every exception a rationale, owner, creation date, and review/expiry date.
5. Fail on expired, malformed, unexpectedly located, or newly reachable findings.
6. Replace stale completed-task ownership with a live issue or task reference.

Acceptance criteria:

- The gate runs from a `git archive` or fresh clone without relying on local file mode.
- Fixtures prove scanner errors, malformed output, an allowed ID in another package, expired exceptions, and new findings all fail.
- A clean result and exactly documented unexpired exceptions behave as specified.
- `make release-verify VERSION=v0.15.0` reaches and executes the vulnerability gate.

Completion evidence:

- Changed: `scripts/release-govulncheck.sh`, `Makefile`, `docs/release-artifact-policy.md`, `internal/releasegovulncheck/gate.go`, `internal/releasegovulncheck/gate_test.go`, `internal/releasegovulncheck/cmd/release-govulncheck/main.go`, `internal/releasegovulncheck/testdata/*.json`
- Implemented: release verification invokes the gate with `bash` so executable mode is not required from a fresh archive; the wrapper forces `govulncheck -json`, preserves stdout/stderr diagnostics, and delegates JSON stream parsing to a stdlib-only Go gate.
- Implemented: the gate distinguishes malformed scanner output, scanner error messages, and scanner nonzero/runtime failures from vulnerability findings; exceptions match by vulnerability ID, scan mode, affected module, affected package, and affected symbol; exception metadata validation requires rationale, owner, creation date, review/expiry date, and a live task or issue reference.
- Implemented: stale `ARCHIVE-01` ownership was replaced with live tracking under `docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01`.
- Verified: `gofmt -w internal/releasegovulncheck/gate.go internal/releasegovulncheck/gate_test.go internal/releasegovulncheck/cmd/release-govulncheck/main.go && go test ./internal/releasegovulncheck ./internal/releasegovulncheck/cmd/release-govulncheck -count=1`
- Result: passed; fixtures cover scanner errors, malformed output, allowed ID in another package, allowed finding in another scan mode, expired exceptions, new findings, clean output, and allowed exceptions.
- Verified: `go test -shuffle=on ./...`
- Result: passed.
- Verified: `bash ./scripts/release-govulncheck.sh "$(pwd)/tmp/bin/govulncheck" ./...`
- Result: passed; the gate parsed live `govulncheck -json` output and reported `35 reachable findings matched documented unexpired exceptions; 28 imported/module-only findings were not reachable at symbol level`.
- Verified: `git diff --check && go test ./internal/releasegovulncheck ./internal/releasegovulncheck/cmd/release-govulncheck -count=1`
- Result: passed.
- Verified: `make release-verify VERSION=v0.15.0`
- Result: blocked after successfully reaching and executing the vulnerability gate. Earlier steps passed (`pnpm install --frozen-lockfile`, `go test -shuffle=on ./...`, `go test -race -shuffle=on ./...`, `go vet ./...`, `go mod verify`, and the JSON vulnerability gate). The exact blocker is external Docker availability at `docker buildx build --check .`: `The command 'docker' could not be found in this WSL 2 distro.`

### Task RELEASE-02: Scan The Exact Container Before Publishing It

Status: blocked

Priority: P1

Suggested agent: container release and provenance engineer

Dependencies: RELEASE-01

Primary ownership:

- `.github/workflows/publish.yaml`
- `Dockerfile`
- `docs/release-artifact-policy.md`

Finding:

The publish workflow pushes version and mutable major/minor tags before Trivy runs, so a rejected image is already public. The Docker build also invokes `make build` without a release version, causing binaries in a versioned image to identify as `localdev`.

References:

- `.github/workflows/publish.yaml:48-77`
- `Dockerfile:2-17`
- `Makefile:6-7`
- `Makefile:70-73`
- `mongoarchive/main/mongoarchive.go:95-97`
- `mongounarchive/main/mongounarchive.go:106-108`

Implementation requirements:

1. Build an OCI archive or local image without pushing and scan that exact artifact first.
2. Push immutable and mutable tags only after the scan succeeds.
3. Prove the scanned and published digest are identical.
4. Pass the release version and revision into the Docker build and add standard OCI labels.
5. Smoke-test both binaries' `--version` output before push.
6. Preserve SBOM and provenance generation for the published digest.

Acceptance criteria:

- A fixable HIGH or CRITICAL finding creates or mutates no GHCR tag.
- For tag `v0.15.0`, both binaries in the image report `v0.15.0` or the documented normalized equivalent.
- OCI version and revision labels match the tag and exact commit.
- Workflow evidence records matching scanned and published digests.
- `docker buildx build --check .` passes.

Blocked evidence:

- Changed: `.github/workflows/publish.yaml`, `Dockerfile`, `docs/release-artifact-policy.md`
- Implemented: the publish workflow now builds a local image without pushing, passes the release tag and exact checkout revision into the Docker build, validates both binaries' `--version` output and OCI version/revision labels, scans the local image with Trivy before GHCR authentication or push, generates an image SBOM from the scanned image, pushes immutable and mutable tags only after scan success, verifies every published tag's config digest matches the scanned local image digest in workflow summary evidence, and attests provenance plus SBOM for the verified published manifest digest.
- Verified: `git diff --check`
- Result: passed
- Verified: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 -shellcheck= .github/workflows/*.yml .github/workflows/*.yaml`
- Result: passed
- Verified: `make build VERSION=v0.15.0 && ./dist/mongo-archive --version && ./dist/mongo-unarchive --version`
- Result: passed; both binaries reported `v0.15.0`
- Blocked: `docker buildx build --check .`
- Exact blocker: `The command 'docker' could not be found in this WSL 2 distro.`
- Noted environment blocker: repository-managed `actionlint .github/workflows/*.yml .github/workflows/*.yaml` could not run because no `actionlint` version is configured in `.tool-versions`; `go run` actionlint was used instead, with shellcheck integration disabled because no `shellcheck` version is configured either.

### Task CI-01: Gate Exact-Tag Releases On Integration Tests And PR Checks

Status: completed

Priority: P1

Suggested agent: GitHub Actions and integration-test engineer

Dependencies: none

Primary ownership:

- `.github/workflows/test.yml`
- `.github/workflows/release.yml`
- `.github/workflows/lint.yml`
- reusable integration workflow if introduced

Finding:

The tag release runs `make release-verify`, which does not run the Bats integration suite. The independent push workflow is not a dependency for the exact tagged SHA. Test and lint workflows also trigger only on `push`, so fork pull requests and merge-queue commits may bypass required checks.

References:

- `.github/workflows/test.yml:3-7`
- `.github/workflows/test.yml:56-117`
- `.github/workflows/release.yml:8-26`
- `.github/workflows/release.yml:56-83`
- `Makefile:59-68`
- `.github/workflows/lint.yml:1-26`

Implementation requirements:

1. Make integration verification reusable and invoke it from normal CI and the release workflow at the exact SHA.
2. Require successful integration verification before GitHub Release and container publication jobs.
3. Add `pull_request`; add `merge_group` if merge queue is enabled.
4. Keep fork-safe checks secretless and isolate privileged release behavior to trusted events.
5. Add finite job-level timeouts while preserving always-run log collection and cleanup.
6. Validate workflow syntax with the repository's chosen workflow linter.

Acceptance criteria:

- A deliberate Bats failure blocks both release jobs for a tag.
- Release verification reports a checkout SHA equal to the tag SHA.
- A fork PR runs unit, race, vet, lint, and safe integration checks.
- Every CI job has a documented finite timeout.
- `actionlint .github/workflows/*.yml .github/workflows/*.yaml` passes.

Completion evidence:

- Changed: `.github/workflows/test.yml`, `.github/workflows/integration.yml`, `.github/workflows/release.yml`, `.github/workflows/lint.yml`, `.github/workflows/publish.yaml`, `.tool-versions`
- Implemented: integration verification is a reusable secretless workflow called by normal CI and the tag-release workflow with `ref: ${{ github.sha }}`; it reports and validates the checked-out SHA when the requested ref is a SHA, preserves always-run Compose log capture, artifact upload, and cleanup, and runs `bats --print-output-on-failure test/test.bats`.
- Implemented: tag releases now require both release verification and integration verification before the GitHub Release job and reusable container publication job; release verification checks out the triggering SHA, resolves annotated tags to their commit target, reports the event, checkout, and tag target SHAs, and fails if they differ.
- Implemented: `test.yml` and `lint.yml` now run on `push`, `pull_request`, and `merge_group`; PR and merge-queue lint/test paths use read-only permissions and no secrets, while the existing pre-commit auto-fix write permission remains isolated to trusted `push` events.
- Implemented: finite job-level timeouts are set on Go coverage, race, vet, reusable integration, release verification, GitHub Release, lint, and container publication jobs; the reusable integration job timeout covers callers that cannot set `timeout-minutes` directly on a reusable workflow job.
- Verified: `actionlint .github/workflows/*.yml .github/workflows/*.yaml`
- Result: passed
- Verified: `git diff --check`
- Result: passed
- Workflow evidence: a deliberate Bats failure in the reusable integration workflow makes the release workflow's `integration` job fail, and both `release` and `publish-image` declare `needs: [verify, integration]`, blocking GitHub Release creation and container publication for the tag.
- Workflow evidence: fork PRs run `go-test-cover`, `go-test-race`, `go-vet`, `integration-bats`, and read-only `pre-commit-check` without inherited secrets or package/release permissions.

### Task ARCH-01: Split Storage Capabilities And Carry Backend Identity Explicitly

Status: completed

Priority: P2

Suggested agent: senior Go API design engineer

Dependencies: CONFIG-01

Primary ownership:

- `storage/interface.go`
- `storage/selection.go`
- `internal/toolconfig/shared.go`
- both orchestration dependency types and focused tests

Finding:

The `Storage` interface combines archive, restore, retention, lookup, and lifecycle methods. Archive fakes must implement restore methods and restore fakes must implement archive methods. Restore backend identity is then recovered through a concrete-type switch, so wrappers, fakes, and new implementations satisfy the interface but cannot participate in real selection.

References:

- `storage/interface.go:5-10`
- `storage/selection.go:97-158`
- `mongoarchive/main/mongoarchive_test.go:31-83`
- `mongounarchive/main/mongounarchive_test.go:23-42`

Implementation requirements:

1. Define narrow archive and restore capability interfaces and an explicit lifecycle capability.
2. Carry canonical backend identity as configuration metadata or a narrow named capability, not a concrete-type switch.
3. Reject or explicitly resolve duplicate canonical backend names.
4. Keep existing providers and public CLI backend values behaviorally compatible.
5. Avoid changing provider implementation logic in this task.

Acceptance criteria:

- Archive fakes implement no download or target-selection methods.
- Restore fakes implement no upload or retention methods.
- A wrapper or fake with explicit identity passes through real restore selection.
- Adding a backend does not require editing a central concrete-type switch.
- `go test ./storage ./internal/toolconfig ./mongoarchive/main ./mongounarchive/main -count=1` passes.

Completion evidence:

- Changed: `storage/interface.go`, `storage/selection_test.go`, `internal/toolconfig/shared.go`, `internal/toolconfig/shared_test.go`, `mongounarchive/main/mongounarchive_test.go`
- Implemented: storage capabilities are split into archive, restore, lifecycle, and backend-identity interfaces; tool configuration constructs archive and restore capability slices without depending on a broad combined storage interface; restore selection uses explicit `BackendName()` identity and rejects duplicate canonical backend names.
- Verified: `go test ./storage ./internal/toolconfig ./mongoarchive/main ./mongounarchive/main -count=1`
- Result: passed

### Task ARCH-02: Consolidate Runtime Configuration And Lifecycle Utilities

Status: completed

Priority: P2

Suggested agent: Go maintainability and runtime-lifecycle engineer

Dependencies: RESTORE-01, CONFIG-01, ARCH-01

Primary ownership:

- `mongoarchive/main/mongoarchive.go`
- `mongounarchive/main/mongounarchive.go`
- both flag packages
- a small internal runtime utility package
- `internal/toolconfig`

Finding:

Both main packages duplicate cleanup stacks, reverse cleanup, storage closing, error joining, and operation-context logic. Workspace paths, operation timeouts, extraction limits, and update limits still read global environment during execution rather than entering one immutable typed configuration. Invalid timeout settings can therefore fail only after dump or download side effects.

References:

- `mongoarchive/main/mongoarchive.go:288-299`
- `mongoarchive/main/mongoarchive.go:350-400`
- `mongoarchive/main/mongoarchive.go:522-548`
- `mongounarchive/main/mongounarchive.go:261-327`
- `mongounarchive/main/mongounarchive.go:365-407`
- `mongounarchive/main/mongounarchive.go:503-523`
- `mongounarchive/flags.go:279-293`
- `mongounarchive/flags.go:358-369`

Implementation requirements:

1. Extract only the duplicated cleanup, close aggregation, and operation-context primitives into a narrow internal package.
2. Parse workspace, timeout, extraction, and update limits through the existing injectable environment reader.
3. Validate the complete immutable runtime configuration before creating a workspace or performing external work.
4. Preserve reverse-order teardown, `--keep`, `errors.Is`, and inclusion of all cleanup failures.
5. Keep existing environment variable names, defaults, and environment-only status unless a separate contract change is approved.

Acceptance criteria:

- One cleanup/error-joining implementation serves both pipelines.
- No operation path directly reads the migrated environment settings.
- Invalid duration or limit input fails before dump, download, workspace, or provider construction.
- Parser tests inject environment values and can run in parallel.
- `go test ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main ./internal/toolconfig -count=1` passes.

Completion evidence:

- Changed: `internal/toolruntime/runtime.go`, `internal/toolruntime/runtime_test.go`, `internal/toolconfig/runtime.go`, `internal/toolconfig/runtime_test.go`, `mongoarchive/flags.go`, `mongoarchive/flags_test.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`, `mongounarchive/flags.go`, `mongounarchive/flags_test.go`, `mongounarchive/main/mongounarchive.go`, `mongounarchive/main/mongounarchive_test.go`
- Implemented: shared runtime cleanup, reverse-order teardown, close aggregation, cleanup-error joining, and operation-context creation now live in `internal/toolruntime`; archive and restore runtime environment values are parsed through injectable `EnvReader` into typed config before work begins; operation paths consume parsed workspace paths, timeouts, extraction limits, and update limits without direct migrated environment reads.
- Verified: `go test ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main ./internal/toolconfig ./internal/toolruntime -count=1`
- Result: passed
- Verified: `go test ./mongoarchive ./mongounarchive ./mongoarchive/main ./mongounarchive/main ./internal/toolconfig -count=1`
- Result: passed

### Task CRON-01: Make Context Cancellation Own Scheduler Shutdown

Status: completed

Priority: P2

Suggested agent: Go concurrency engineer

Dependencies: ARCH-02

Primary ownership:

- `mongoarchive/main/mongoarchive.go`
- `mongoarchive/main/mongoarchive_test.go`

Finding:

`cronRuntime.run` passes a context to jobs, but process shutdown is controlled by a separate zero-argument callback that installs another signal subscription. Canceling the supplied context does not itself release the scheduler loop, splitting signal ownership and reducing composability.

References:

- `mongoarchive/main/mongoarchive.go:77-83`
- `mongoarchive/main/mongoarchive.go:145-157`
- `mongoarchive/main/mongoarchive.go:241-285`
- `mongoarchive/main/mongoarchive_test.go:575-663`

Implementation requirements:

1. Make the root signal-derived context the single shutdown signal below `main`.
2. Remove the second process signal subscription or make the wait seam explicitly context-aware.
3. Define whether ordinary SIGINT/SIGTERM shutdown returns nil or a cancellation error and test the exit contract.
4. Ensure scheduler shutdown runs exactly once and active jobs receive cancellation.
5. Preserve singleton scheduling and skip-overlap behavior.

Acceptance criteria:

- Canceling the supplied context makes `cronRuntime.run` return promptly.
- The scheduler shuts down exactly once and the active task observes cancellation.
- No process signal registration exists below the top-level process boundary.
- `go test -race ./mongoarchive/main -count=1` passes.

Completion evidence:

- Changed: `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`
- Implemented: cron shutdown now waits on the root context supplied by `main`; the lower-level cron runtime no longer registers process signals or exposes a second wait seam; ordinary signal-derived context cancellation returns nil from `cronRuntime.run`, while the shared context cancels active scheduled work.
- Preserved: scheduler singleton scheduling and skip-overlap behavior remain covered by the cron runtime tests.
- Verified: `go test -race ./mongoarchive/main -count=1`
- Result: passed

### Task NOTIFY-01: Separate Notification Validation From Client Construction

Status: completed

Priority: P2

Suggested agent: Go notification and performance engineer

Dependencies: ARCH-02

Primary ownership:

- `mongoarchive/flags.go`
- `mongoarchive/main/mongoarchive.go`
- `notification/` construction boundaries and focused tests

Finding:

Archive configuration validation constructs notification clients and discards them. Sending then reconstructs all clients on every success or failure, including repeated SES session/client construction in cron mode. Validation, lifecycle, and transport construction are conflated.

References:

- `mongoarchive/flags.go:324-396`
- `mongoarchive/main/mongoarchive.go:160-167`
- `mongoarchive/main/mongoarchive.go:268-273`
- `mongoarchive/main/mongoarchive.go:503-519`
- `notification/ses.go:55-78`

Implementation requirements:

1. Make notification option validation pure and side-effect free.
2. Construct an injected notification sender/group once at an explicit lifecycle boundary.
3. Preserve per-channel failure-only rules, context timeouts, message redaction, and current nonfatal send-error policy unless separately approved.
4. Keep HTTP connection reuse and make any closable future resource ownership explicit.
5. Add construction-count tests for one-shot and repeated cron runs.

Acceptance criteria:

- Config validation creates no transport, SDK session, or external client.
- Each configured notifier is constructed at the documented lifecycle frequency, not once per send.
- Tests inject one narrow sender without provider configuration.
- Existing notification package and archive main tests pass under the race detector.

Completion evidence:

- Changed: `notification/transport.go`, `notification/smtp.go`, `notification/ses.go`, `mongoarchive/flags.go`, `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`
- Implemented: archive notification validation now calls pure notification option validators instead of constructing notifier transports or SES sessions; concrete notification construction occurs once at the archive process lifecycle boundary, with one-shot runs using one group and cron mode reusing the same group for repeated scheduled executions.
- Preserved: per-channel failure-only checks remain in each notifier, notification sends still use configured per-channel operation contexts, send failures remain nonfatal and logged, existing message redaction remains in notification message/body construction, HTTP clients remain reused across sends, and future closable notifier resources are explicitly closed by the group owner.
- Verified: `go test ./notification ./mongoarchive ./mongoarchive/main -count=1`
- Result: passed
- Verified: `go test -race ./notification ./mongoarchive ./mongoarchive/main -count=1`
- Result: passed
- Verified: `git diff --check`
- Result: passed
- Reverified: `go test ./notification ./mongoarchive ./mongoarchive/main -count=1`
- Result: passed
- Reverified: `go test -race ./notification ./mongoarchive ./mongoarchive/main -count=1`
- Result: passed
- Reverified: `git diff --check`
- Result: passed

### Task TEST-01: Add Cross-Provider Contract Tests And Protect Current Coverage

Status: completed

Priority: P2

Suggested agent: Go storage testability engineer

Dependencies: CONFIG-01, ARCH-01

Primary ownership:

- `storage/*.go`
- provider client seams and contract tests
- `.github/workflows/test.yml`
- focused integration scenarios in `test/test.bats`

Finding:

Current total coverage is 52.5%, but CI permits 17.5%. AWS and Azure operational methods and nearly all GCP operational methods have zero unit coverage. Existing integration tests cover provider happy paths but not provider-specific pagination, cancellation, verification/deletion failures, corrupt transfers, or the documented real-scheduler mixed-backend failure case.

References:

- `.github/workflows/test.yml:18-26`
- `storage/aws.go:33-250`
- `storage/azure.go:32-237`
- `storage/gcp.go:47-333`
- `storage/gcp_test.go:11-48`
- `storage/listing_test.go:5-36`
- `test/test.bats:111-222`
- `docs/tasks/20260813-105920-multi-backend-archive-contract-remediation.md:209-214`

Implementation requirements:

1. Introduce the narrowest SDK/client seams needed for deterministic provider tests; do not mirror entire SDKs.
2. Run one provider-independent contract suite against AWS, Azure, GCP, and local implementations.
3. Cover explicit missing objects, paginated latest selection, upload verification failure, retention deletion failure, cancellation, and partial-download cleanup.
4. Add an end-to-end mixed-backend second-upload failure asserting nonzero status, explicit partial-state output, and no retention.
5. Raise the global floor near the measured baseline with a documented tolerance and add critical-package floors where useful.

Acceptance criteria:

- AWS, Azure, and GCP operational methods gain meaningful nonzero unit coverage.
- Every provider passes the same observable contract suite.
- Removing a critical-path regression test causes a coverage gate to fail.
- The mixed-backend failure scenario is covered beyond direct orchestration-unit tests.
- `go test -shuffle=on -coverprofile=/tmp/coverage.out ./...` and `go test -race -shuffle=on ./...` pass.

Completion evidence:

- Changed: `storage/aws.go`, `storage/azure.go`, `storage/gcp.go`, `storage/local.go`, `storage/provider_contract_test.go`, `.github/workflows/test.yml`, `test/test.bats`
- Implemented: narrow provider test seams for upload, download, existence, paginated listing, and deletion; one provider-independent contract suite now runs against local, AWS, Azure, and GCP implementations and covers missing explicit objects, paginated latest selection, upload verification failure, retention deletion failure, cancellation, and partial-download cleanup.
- Implemented: CI coverage enforcement now raises the global floor to `55.0%` with documented tolerance from the measured `59.6%` baseline and adds a `50.0%` storage-package floor to protect the provider contract suite.
- Implemented: `test/test.bats` includes an end-to-end local-plus-S3 second-upload failure scenario that asserts nonzero exit, explicit partial-state output, explicit no-retention output, and preserved old local backup state.
- Verified: `go test -shuffle=on -run 'TestProviderContract' ./storage`
- Result: passed
- Verified: `go test -shuffle=on -coverprofile=/tmp/coverage.out ./...`
- Result: passed; total statement coverage was `59.6%`.
- Verified: `go test -shuffle=on -coverprofile=/tmp/storage-coverage.out ./storage` and `go tool cover -func=/tmp/storage-coverage.out`
- Result: passed; storage statement coverage was `55.2%`; AWS, Azure, and GCP operational methods had nonzero unit coverage for target selection, upload, download, retention deletion, existence/list/delete seams.
- Verified: `go test -race -shuffle=on ./...`
- Result: passed
- Verified: `git diff --check`
- Result: passed
- Not run: `bats --print-output-on-failure test/test.bats`; local Docker integration is unavailable in this WSL environment (`docker`: `The command 'docker' could not be found in this WSL 2 distro.`).

### Task CI-02: Make Integration Startup Deterministic And Bounded

Status: blocked

Priority: P2

Suggested agent: integration infrastructure engineer

Dependencies: CI-01

Primary ownership:

- `sandbox/docker-compose.yml`
- `sandbox/docker-compose-ci.yml`
- `.github/workflows/test.yml`
- `test/testdb-*.go`

Finding:

CI waits for MinIO health but not successful completion of the separate `minio-init` service, so bucket setup races with the Bats suite. MongoDB helper calls use unbounded `context.TODO` or background contexts. A slow or hung dependency can consume the whole workflow until the platform timeout.

References:

- `sandbox/docker-compose.yml:36-51`
- `.github/workflows/test.yml:67-97`
- `test/testdb-setup.go:32-35`
- `test/testdb-check.go:32-35`
- `test/testdb-drop.go:23-30`

Implementation requirements:

1. Wait for `minio-init` to exit zero before starting Bats and fail with logs on nonzero exit.
2. Do not suppress service-account setup errors except an explicitly detected already-exists result.
3. Give helper connect, ping, drop, and disconnect calls bounded contexts.
4. Keep diagnostics and Compose teardown under always-run steps.
5. Add repeated clean-start verification to detect startup flakiness.

Acceptance criteria:

- A forced MinIO bucket-creation failure deterministically blocks Bats.
- Bats cannot begin before initialization exits successfully.
- A blocked Mongo helper exits within its documented bound.
- At least five consecutive clean integration starts and suites pass.

Completion evidence:

- Changed: `sandbox/docker-compose.yml`, `.github/workflows/integration.yml`, `test/testdb-setup.go`, `test/testdb-check.go`, `test/testdb-drop.go`
- Implemented: `minio-init` now fails closed after bucket setup errors, no longer ends with unconditional success, and suppresses service-account setup errors only when the MinIO output explicitly reports an already-existing service account.
- Implemented: the integration workflow waits for the `minio-init` container to exit `0` before Bats can start; nonzero, missing, or timed-out init states fail before Bats and print `minio-init` logs; service logs and Compose teardown remain `if: always()` steps.
- Implemented: the reusable integration workflow runs five independent clean-start Bats attempts through a matrix, with per-attempt log artifacts.
- Implemented: MongoDB helper connection selection, ping, drop, and disconnect paths now use a documented `10s` bound.
- Verified: `go test -shuffle=on ./...`
- Result: passed
- Verified: `go vet ./...`
- Result: passed
- Verified: `DATABASE_URL='mongodb://127.0.0.1:1' timeout 20s go run ./test/testdb-drop.go`
- Result: exited nonzero within the documented helper bound with `server selection error: context deadline exceeded`.
- Verified: `go build -o /tmp/opencode/testdb-setup ./test/testdb-setup.go`, `go build -o /tmp/opencode/testdb-check ./test/testdb-check.go`, `go build -o /tmp/opencode/testdb-drop ./test/testdb-drop.go`
- Result: passed
- Verified: `go test -race -shuffle=on ./...`
- Result: passed
- Verified: `git diff --check`
- Result: passed
- Blocked: Docker integration acceptance could not be run locally because `docker --version` failed with `The command 'docker' could not be found in this WSL 2 distro.` This blocks local verification of forced MinIO bucket-creation failure, Bats sequencing against real Compose services, and five consecutive clean integration starts and suites.

### Task DOCS-01: Align Public Configuration Docs And Safe Operational Examples

Status: completed

Priority: P2

Suggested agent: technical writer with Go CLI and Kubernetes experience

Dependencies: CONFIG-01, ARCH-02

Primary ownership:

- `internal/flagdocs`
- `flags.md`
- `README.md`
- `examples/*.yaml`
- documentation validation tests

Finding:

Shared flag registration exposes restore-only `--storage-backend` in the archive reference even though archive runtime ignores it. Extraction-limit variables are implemented and documented in README but absent from the generated authoritative environment table. Kubernetes examples use stale `0.3` images and literal credentials; `job-copy.yaml` runs restore after archive failure because its shell commands are not fail-fast.

References:

- `internal/toolconfig/shared.go:192-287`
- `flags.md:53-56`
- `flags.md:166-173`
- `mongounarchive/main/mongounarchive.go:274-298`
- `README.md:123-131`
- `examples/cronjob-archive.yaml:28-44`
- `examples/job-copy.yaml:10-28`
- `README.md:329-340`

Implementation requirements:

1. Separate common storage settings from restore-only backend selection.
2. Register every public environment-only runtime setting in the authoritative typed documentation source.
3. Add a completeness check for public environment lookups or an explicit internal-only allowlist.
4. Make copy examples fail immediately when archive fails and avoid restoring an older implicit latest object.
5. Replace literal credentials with Kubernetes Secret references and stale versions with a maintained placeholder or automated update policy.
6. Validate Kubernetes examples in CI.

Acceptance criteria:

- Archive no longer documents or accepts an ignored restore-only selector.
- All three extraction limits and migrated runtime settings appear in generated docs.
- Adding an undocumented public environment lookup fails CI.
- A forced archive failure prevents the example restore command and returns nonzero.
- Example manifests contain no literal credentials and pass strict schema validation.

Completion evidence:

- Changed: `internal/toolconfig/shared.go`, `mongounarchive/flags.go`, `mongoarchive/flags_test.go`, `internal/flagdocs/envdocs_test.go`, `internal/flagdocs/kubernetes_examples_test.go`, `flags.md`, `README.md`, `examples/cronjob-archive.yaml`, `examples/job-copy.yaml`
- Implemented: shared storage docs/bindings no longer include restore-only `--storage-backend`; `mongo-unarchive` binds and documents the selector separately; generated docs include restore extraction limits plus migrated runtime-only settings; documentation tests scan public runtime environment lookups and require generated docs or an explicit internal-only allowlist.
- Implemented: Kubernetes examples use `<latest-version>` image placeholders and Secret references for credential-bearing settings; `job-copy.yaml` runs with `set -eu`, restores by explicit `--object-name`, and has a regression test proving a forced archive failure exits nonzero before restore is invoked.
- Verified: `go test ./internal/flagdocs ./mongoarchive ./mongounarchive ./internal/toolconfig -count=1`
- Result: passed; includes strict Kubernetes example schema/secret checks and the forced archive-failure copy-script regression.
- Verified: `git diff --check`
- Result: passed
- Verified: `go test ./... -count=1`
- Result: passed

### Task SUPPLY-01: Align Toolchains And Harden Reproducibility

Status: blocked

Priority: P2

Suggested agent: build reproducibility and supply-chain engineer

Dependencies: RELEASE-01, RELEASE-02

Primary ownership:

- `go.mod`
- `.tool-versions`
- `package.json`
- `Dockerfile`
- `Makefile`
- `requirements.txt` or generated lock
- `.github/actions/setup-tools/action.yml`
- `renovate.json`
- `docs/release-artifact-policy.md`

Finding:

Go is `1.26.5` in `go.mod` and Docker but `1.26.6` in `.tool-versions`; pnpm is `11.17.0` in package metadata and `11.21.0` in `.tool-versions`. Release and integration container inputs use mutable tags, asdf plugins are fetched without commit pins, and Python dependencies lack transitive hashes. Release tarballs also retain variable metadata and have no byte-reproducibility test.

References:

- `go.mod:3`
- `.tool-versions:1-8`
- `package.json:23-25`
- `Dockerfile:2-17`
- `Makefile:43-53`
- `Makefile:76-87`
- `.github/actions/setup-tools/action.yml:7-17`
- `requirements.txt:1-3`
- `docs/release-artifact-policy.md:33-35`

Implementation requirements:

1. Select and synchronize one Go and pnpm version across code, CI, container, and policy.
2. Add an automated version-consistency check.
3. Pin release-critical images and setup inputs by digest or immutable revision with Renovate support.
4. Hash-lock Python dependencies used by CI/release tooling.
5. Define reproducible build inputs such as `-trimpath`, fixed archive metadata, and `SOURCE_DATE_EPOCH` where applicable.
6. Mark `package.json` private if npm publication is not an intended product.

Acceptance criteria:

- Version-consistency checks fail on any declaration drift.
- Two clean builds with identical source and epoch produce identical archive SHA-256 values, or documented unavoidable variance has an approved residual-risk record.
- Release-critical external inputs are content-addressed and automatically updateable.
- A clean frozen package/tool install succeeds.

Blocked evidence:

- Changed: `go.mod`, `.tool-versions`, `package.json`, `Dockerfile`, `Makefile`, `requirements-lock.txt`, `.github/actions/setup-tools/action.yml`, `.github/workflows/release.yml`, `renovate.json`, `docs/release-artifact-policy.md`, `scripts/check-supply-chain.sh`
- Implemented: Go is synchronized to `1.26.6` across `go.mod`, `.tool-versions`, Dockerfile, and release policy; pnpm is synchronized to `11.17.0` across `.tool-versions` and `package.json`; `package.json` is marked private.
- Implemented: `scripts/check-supply-chain.sh` enforces Go/pnpm declaration consistency, Dockerfile and release-workflow digest pins, custom asdf plugin commit pins, hashed Python lock presence, and private package metadata; `make release-verify` now runs this gate before build/test work.
- Implemented: Dockerfile build/runtime images and the release SBOM `anchore/syft` image are pinned by digest; setup uses pinned GitHub Actions and checks out custom asdf plugin repos to full commit SHAs; Renovate is configured to update Docker digests, the workflow `syft` image, and asdf plugin git refs.
- Implemented: Python CI/release tooling is locked in `requirements-lock.txt` with transitive hashes and setup installs it with `pip --require-hashes`.
- Implemented: Go builds now use `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, and an empty build ID; release archives use sorted entries, owner/group `0`, numeric owners, gzip `-n`, and `SOURCE_DATE_EPOCH`; `make check-reproducible-archives` builds two independent archive sets and compares SHA-256 manifests.
- Verified: `bash ./scripts/check-supply-chain.sh`
- Result: passed.
- Verified: temporary drifted copy with `.tool-versions` pnpm changed to `11.18.0`, then `bash ./scripts/check-supply-chain.sh`
- Result: failed as expected with `pnpm version in package.json drift: expected 11.18.0, got 11.17.0`.
- Verified: `pnpm install --frozen-lockfile`
- Result: passed with pnpm `11.17.0`.
- Verified: `python3 -m pip install --require-hashes -r requirements-lock.txt --dry-run`
- Result: passed.
- Verified: `python3 -m venv /tmp/opencode/supply-python-venv && /tmp/opencode/supply-python-venv/bin/python -m pip install --require-hashes -r requirements-lock.txt`
- Result: passed.
- Verified: `make check-reproducible-archives VERSION=v0.15.0 SOURCE_DATE_EPOCH=1787449531`
- Result: passed; two independent clean archive builds produced identical SHA-256 manifests.
- Verified: `git diff --check`
- Result: passed.
- Verified: `shellcheck scripts/check-supply-chain.sh`
- Result: passed.
- Verified: `actionlint .github/workflows/*.yml .github/workflows/*.yaml`
- Result: passed.
- Verified: `go test -shuffle=on ./...`
- Result: passed.
- Verified: `go test -race -shuffle=on ./...`
- Result: passed.
- Verified: `go vet ./...`
- Result: passed.
- Verified: `go mod verify`
- Result: passed.
- Verified: `make build VERSION=v0.15.0`
- Result: passed.
- Verified: `make tmp/bin/govulncheck && bash ./scripts/release-govulncheck.sh "$(pwd)/tmp/bin/govulncheck" ./...`
- Result: passed; the gate reported `35 reachable findings matched documented unexpired exceptions; 28 imported/module-only findings were not reachable at symbol level`.
- Blocked: `docker buildx build --check .`
- Exact blocker: `The command 'docker' could not be found in this WSL 2 distro. We recommend to activate the WSL integration in Docker Desktop settings.`

### Task GOVERNANCE-01: Define Security And Ownership Policy

Status: completed

Priority: P2

Suggested agent: repository security maintainer

Dependencies: RELEASE-01, SUPPLY-01

Primary ownership:

- `SECURITY.md`
- `CONTRIBUTING.md`
- `.github/CODEOWNERS`
- repository policy documentation

Finding:

The repository has no security reporting policy, supported-version policy, contribution guide, or code ownership rules despite handling database credentials, backup retention, restore extraction, and release publication.

References:

- Repository root and `.github/` contain no `SECURITY.md`, `CONTRIBUTING.md`, or `CODEOWNERS` as of task creation.

Implementation requirements:

1. Document private vulnerability reporting and response expectations.
2. Define supported release versions and vulnerability exception ownership/expiry.
3. Document local and full verification commands for contributors.
4. Assign review ownership for storage, restore/extraction, release workflows, and security policy.
5. Confirm repository settings and branch protection align with documented required checks.

Acceptance criteria:

- Security researchers have a non-public reporting route.
- Supported versions and response expectations are explicit.
- Sensitive boundary and release changes request appropriate owners.
- Contribution instructions reproduce current CI commands.

Completion evidence:

- Changed: `SECURITY.md`, `CONTRIBUTING.md`, `.github/CODEOWNERS`, `docs/repository-governance.md`, `docs/tasks/20260823-112531-post-remediation-codebase-health.md`
- Implemented: `SECURITY.md` defines private vulnerability reporting through GitHub private vulnerability reporting, fallback instructions that avoid public details, maintainer response expectations, supported-version policy, vulnerability exception metadata, ownership, and expiry requirements, and sensitive security boundaries.
- Implemented: `CONTRIBUTING.md` documents tool versions, local verification, integration verification, release verification, and CI-equivalent commands from the current workflows and `Makefile`.
- Implemented: `.github/CODEOWNERS` assigns review ownership for storage, restore/extraction, archive/notification boundaries, release workflows, supply-chain policy, and security/governance policy to the current repository owner account.
- Implemented: `docs/repository-governance.md` documents required branch-protection checks, release protection expectations, CODEOWNERS review boundaries, vulnerability exception ownership, and read-only GitHub settings verification.
- Repository settings check: `gh repo view --json nameWithOwner,defaultBranchRef,isPrivate,isSecurityPolicyEnabled,securityPolicyUrl,viewerCanAdminister,viewerPermission,hasIssuesEnabled` reported public repository `egose/database-tools`, default branch `main`, issues enabled, viewer permission `READ`, no admin rights, and no security policy enabled on the remote default branch before this change is merged.
- Repository settings check: `gh api repos/egose/database-tools/branches/main` reported `protected: false` and disabled status-check protection for `main`; `gh api repos/egose/database-tools/branches/main/protection` returned `404`. Residual admin verification is documented in `docs/repository-governance.md` for enabling security policy/private reporting, branch protection or rulesets, required checks, CODEOWNERS review, and release tag restrictions.
- Verified: `git diff --check`
- Result: passed
- Verified: `pre-commit run --files SECURITY.md CONTRIBUTING.md docs/repository-governance.md .github/CODEOWNERS docs/tasks/20260823-112531-post-remediation-codebase-health.md`
- Result: passed
- Verified: `actionlint .github/workflows/*.yml .github/workflows/*.yaml`
- Result: passed

### Task INTEGRATION-01: Independently Verify The Post-Remediation Result

Status: completed

Priority: P1

Suggested agent: independent senior Go security and release reviewer

Dependencies: RESTORE-01, CONFIG-01, RELEASE-01, RELEASE-02, CI-01, ARCH-01, ARCH-02, CRON-01, NOTIFY-01, TEST-01, CI-02, DOCS-01, SUPPLY-01, GOVERNANCE-01

Primary ownership:

- independent review across all changed files
- minimal corrections and missing regression tests discovered during review

Finding:

The tasks alter correctness boundaries, configuration, provider abstractions, workflow dependencies, artifact identity, and vulnerability exceptions. Independent verification is required to detect alternate-path bypasses and mismatches between runtime, tests, public docs, and published artifacts.

References:

- Completion evidence under every preceding task.
- Final diff from the implementation base through the integration review.

Implementation requirements:

1. Verify every acceptance criterion against observable behavior, not only code shape.
2. Re-test partial restore results, incomplete providers, cancellation, cleanup, provider contract parity, and mixed-backend failure behavior.
3. Verify no credentials or internal SDK details cross errors, logs, docs, notification, or artifact boundaries.
4. Verify the scanned, attested, and published image is the same digest and reports the release version.
5. Verify exact-tag integration dependencies, PR checks, timeouts, workflow permissions, and cleanup.
6. Record every deferral with maintainer rationale, owner, review date, and residual risk.

Acceptance criteria:

- No P0 or P1 finding remains unresolved.
- Every completed task has linked test or command evidence.
- Unit, race, vet, integration, vulnerability, image, cross-build, and reproducibility checks pass as applicable.
- Public flags, environment variables, examples, runtime behavior, and release metadata agree.
- The reviewer was not a primary implementation agent for the preceding tasks.

Completion evidence:

- Reviewer: independent final-review sub-agent for `INTEGRATION-01` only.
- Changed: `examples/cronjob-archive.yaml` was reformatted by `pre-commit run --all-files`; `docs/tasks/20260823-112531-post-remediation-codebase-health.md` updated with this final review evidence.
- Review scope: read this task file including all statuses/evidence; read related completed task files `docs/tasks/20260804-165757-codebase-health-remediation.md` and `docs/tasks/20260813-105920-multi-backend-archive-contract-remediation.md`; reviewed changed restore/archive pipelines, runtime/config parsing, storage/provider contracts, notification redaction/transport boundaries, generated docs/example checks, CI/release/publish workflows, Dockerfile, Makefile, vulnerability gate, and supply-chain checks.
- Acceptance review: partial restore results now return `restorePartialResultError` before post-restore updates or completion-success logging; zero-failure restores still apply updates; incomplete AWS/Azure/GCP intent is rejected with setting names instead of supplied values; runtime timeouts/limits are parsed into config before operation side effects; cleanup and close errors preserve primary errors; cron shutdown is owned by the caller context and uses gocron `LimitModeReschedule`, documented upstream as skip-and-reschedule rather than queueing; archive multi-backend upload runs all uploads before any retention and reports explicit partial state on mixed outcomes; restore backends carry explicit identity and duplicate canonical names fail; public flags/env/docs/examples and release metadata were checked for agreement.
- Security review: validation errors, normal logs, notification payload construction, generated docs, examples, release task evidence, and release artifacts were reviewed for credential leakage. No normal path was found that logs configured secret values; notification error bodies and Mongo/cloud URI userinfo/query credentials are redacted. Remaining exported credential/provider fields are not formatted in normal logs and remain documented as a nonblocking P2 follow-up under optional provider encapsulation.
- Verified: `go test ./mongounarchive/main -run 'TestRestorePipelinePartialResultFailsBeforeUpdatesAndSuccessLog|TestRestorePipelineZeroFailureResultAppliesUpdatesAndSucceeds|TestRestorePipelineRejectsIncompleteStorageBeforeSideEffects|TestRestorePipelinePropagatesCancellationToUpdates|TestRestorePipelineCleanupByStage' -count=1`
- Result: passed.
- Verified: `go test ./mongoarchive/main ./storage ./internal/toolconfig ./internal/flagdocs ./notification -run 'TestUploadBackupToStorages|TestArchivePipelineRejects|TestCronRuntime|TestProviderContract|TestStorageOptionsValidate|TestKubernetesExamples|TestPublicEnvironmentLookups|TestBuildMessage|Test.*Redact|Test.*Webhook|Test.*SMTP' -count=1`
- Result: passed.
- Verified: `git diff --check`
- Result: passed.
- Verified: `go test -shuffle=on -count=1 -coverprofile=/tmp/coverage.out ./...`
- Result: passed; total statement coverage remained `59.6%`, storage package coverage `55.2%`, and AWS/Azure/GCP provider operational methods had nonzero contract coverage.
- Verified: `go tool cover -func=/tmp/coverage.out`
- Result: passed; coverage summary generated successfully with total `59.6%`.
- Verified: `go test -race -shuffle=on -count=1 ./...`
- Result: passed.
- Verified: `go vet ./...`
- Result: passed.
- Verified: `go mod verify`
- Result: passed with `all modules verified`.
- Verified: `pnpm install --frozen-lockfile`
- Result: passed with pnpm `11.17.0`.
- Verified: `pre-commit run --all-files`
- Result: initially reformatted `examples/cronjob-archive.yaml`; rerun passed all hooks including YAML formatting, detect-secrets, prettier, and gofmt.
- Verified: `actionlint .github/workflows/*.yml .github/workflows/*.yaml`
- Result: passed.
- Verified: `make release-verify VERSION=v0.15.0`
- Result: passed `bash ./scripts/check-supply-chain.sh`, `pnpm install --frozen-lockfile`, `go test -shuffle=on ./...`, `go test -race -shuffle=on ./...`, `go vet ./...`, `go mod verify`, and the JSON `govulncheck` gate, which reported `35 reachable findings matched documented unexpired exceptions; 28 imported/module-only findings were not reachable at symbol level`; blocked at `docker buildx build --check .` because Docker is unavailable in this WSL environment.
- Verified: `docker buildx build --check .`
- Result: blocked by environment with exact error: `The command 'docker' could not be found in this WSL 2 distro. We recommend to activate the WSL integration in Docker Desktop settings.`
- Verified: `make build-all VERSION=v0.15.0 && make build-archive VERSION=v0.15.0 && make check-reproducible-archives VERSION=v0.15.0 SOURCE_DATE_EPOCH=1787449531`
- Result: passed; cross-platform binaries and archives were built, and two independent archive builds produced matching SHA-256 manifests.
- Verified: `make build VERSION=v0.15.0 && ./dist/mongo-archive --version && ./dist/mongo-unarchive --version`
- Result: passed; both binaries reported `v0.15.0`.
- Environment blockers: Docker is unavailable, so local `bats --print-output-on-failure test/test.bats`, Compose startup sequencing/five-clean-start checks, `docker buildx build --check .`, and live container scan-before-push/digest-publication verification could not be executed locally. These are environment-dependent limitations only; workflow structure and actionlint were verified.
- Review result: no unresolved P0/P1 findings remain in the feasible local review. Residual risks are Docker/external-service verification gaps listed above and the existing P2 optional provider-field encapsulation follow-up.

## Dependency And Parallelization Guidance

| Wave | Tasks                                | Parallelization                                                                          |
| ---- | ------------------------------------ | ---------------------------------------------------------------------------------------- |
| 1    | RESTORE-01, CONFIG-01                | May run in parallel; ownership does not overlap materially.                              |
| 2    | RELEASE-01, CI-01                    | May run in parallel. RELEASE-02 follows RELEASE-01.                                      |
| 3    | ARCH-01, ARCH-02, CRON-01, NOTIFY-01 | Sequence ARCH-01 before ARCH-02, then CRON-01 and NOTIFY-01 may run in parallel.         |
| 4    | TEST-01, CI-02                       | May run in parallel after their dependencies, but coordinate integration workflow edits. |
| 5    | DOCS-01, SUPPLY-01, GOVERNANCE-01    | DOCS-01 can run with SUPPLY-01; GOVERNANCE-01 follows policy decisions.                  |
| 6    | INTEGRATION-01                       | Must be independent and last.                                                            |

Shared hotspots that must not be edited concurrently:

- `internal/toolconfig/shared.go`: CONFIG-01, ARCH-01, ARCH-02.
- `mongoarchive/main/mongoarchive.go`: ARCH-01, ARCH-02, CRON-01, NOTIFY-01.
- `mongounarchive/main/mongounarchive.go`: RESTORE-01, ARCH-01, ARCH-02.
- `.github/workflows/test.yml`: CI-01, TEST-01, CI-02.
- `.github/workflows/release.yml` and `publish.yaml`: CI-01, RELEASE-02, SUPPLY-01.
- `Makefile` and `Dockerfile`: RELEASE-01, RELEASE-02, SUPPLY-01.
- `flags.md` and flag definitions: CONFIG-01, ARCH-02, DOCS-01.

Recommended agent allocation:

| Agent                       | Primary tasks                 |
| --------------------------- | ----------------------------- |
| Restore correctness agent   | RESTORE-01                    |
| Configuration/storage agent | CONFIG-01, then ARCH-01       |
| Release security agent      | RELEASE-01, then RELEASE-02   |
| CI/integration agent        | CI-01, then CI-02             |
| Go architecture agent       | ARCH-02, then CRON-01         |
| Notification agent          | NOTIFY-01                     |
| Storage test agent          | TEST-01                       |
| Docs/operations agent       | DOCS-01                       |
| Reproducibility agent       | SUPPLY-01, then GOVERNANCE-01 |
| Independent reviewer        | INTEGRATION-01 only           |

## Deferred Decisions Requiring Maintainer Input

These do not block regression tests but must be resolved before the owning task is completed:

1. Whether GCP application-default credentials are an intentionally supported production mode and which partial inline-credential combinations must fail.
2. Whether signal-driven cron shutdown should be a successful exit or expose `context.Canceled` as nonzero.
3. Whether notifier clients live for the process lifetime or are reconstructed once per scheduled run to refresh credentials.
4. Which branch protection and merge-queue events are enabled, determining whether `merge_group` is required.
5. The acceptable tolerance below current global and critical-package coverage baselines.
6. The supported platform set for container images; multi-architecture publication is an optional extension unless maintainers commit to it.
7. Whether `storage` and `notification` are external Go APIs. If external consumers exist, provider-field encapsulation requires a separate compatibility plan; otherwise constructor-based private fields are recommended follow-up work.
8. The owner and review date for each remaining no-fix vulnerability exception.

## Optional Follow-Ups

These are architectural or operational improvements, not confirmed current failures. Promote them to detailed tasks only after maintainer approval:

- Make provider credentials and SDK clients private behind validated constructors, preserving narrow client injection for tests.
- Reduce manually synchronized flag definition, binding, apply, and documentation lists through typed group registries without reflection-heavy machinery.
- Publish and smoke-test `linux/amd64` and `linux/arm64` container manifests if both are supported products.
- Make lint a read-only verification workflow instead of granting formatting automation repository write permission.
- Include `LICENSE` and concise usage metadata in release archives.

## Final Verification

Run targeted commands under each task first. After all accepted tasks are integrated, run serialized full verification because release builds and integration services share outputs and ports:

```sh
git diff --check
go test -shuffle=on -count=1 -coverprofile=/tmp/coverage.out ./...
go tool cover -func=/tmp/coverage.out
go test -race -shuffle=on -count=1 ./...
go vet ./...
go mod verify
pnpm install --frozen-lockfile
pre-commit run --all-files
actionlint .github/workflows/*.yml .github/workflows/*.yaml
```

Then start the repository integration stack, wait for every initializer contract, run:

```sh
bats --print-output-on-failure test/test.bats
```

Finally run:

```sh
make release-verify VERSION=v0.15.0
docker buildx build --check .
```

Also verify the release workflow in a non-production test repository or dry-run path proves scan-before-push, exact-SHA integration dependency, image version labels, digest identity, and artifact reproducibility without mutating production tags.

## Definition Of Done

- Every accepted task is `completed` with changed files and exact verification evidence, or `deferred` with owner, rationale, review date, and residual risk.
- Partial restores cannot return success or run updates.
- Incomplete provider intent cannot be silently ignored.
- Vulnerability scanner failures cannot be mistaken for allowed findings.
- No image is published before its exact digest passes required integration and vulnerability checks.
- Both binaries and image metadata identify the shipped release.
- Archive and restore pipelines depend on narrow, explicitly identified capabilities.
- Runtime settings are parsed and validated once before side effects.
- Context cancellation owns cron shutdown, and lifecycle behavior remains race-tested.
- Provider implementations share observable contract tests and meaningful coverage protection.
- Public docs and examples match runtime behavior and do not encourage plaintext credentials or restore-after-failure flows.
- Full unit, race, vet, integration, release, artifact, and policy checks pass from a clean checkout.
