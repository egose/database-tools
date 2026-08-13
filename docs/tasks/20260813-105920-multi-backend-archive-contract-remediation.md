# Multi-Backend Archive Contract Remediation

Created: 2026-08-13T10:59:20

Related task file: `docs/tasks/20260804-165757-codebase-health-remediation.md`

## Objective

Define, implement, and independently verify an explicit runtime contract for `mongo-archive` when multiple storage backends are configured, so archive uploads, retention, exit status, and operator-visible results are internally consistent.

## Scope

- Multi-backend archive uploads in `mongo-archive`.
- Retention invocation ordering relative to upload verification.
- CLI exit semantics, logs, and notification wording for mixed backend outcomes.
- Regression tests and documentation for the chosen contract.

## Working Rules And Non-Goals

- Preserve unrelated worktree changes. Never reset or revert files outside this follow-up.
- Keep storage initialization fail-closed unless an explicit new mode is documented and tested.
- Do not broaden this ticket into restore-path, archive-format, or notification-transport redesign work.
- Prefer the smallest shared enforcement point in archive orchestration over provider-specific branching.
- If public CLI behavior changes, update `README.md`, `flags.md`, and release-facing notes in the same change.

## Baseline Verification

This follow-up was created from a source review of the current workspace; verification was not rerun during ticket creation.

Confirmed code references:

- `mongoarchive/main/mongoarchive.go:382-404`
- `internal/toolconfig/shared.go:537-579`
- `README.md:133`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:320-321`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:338`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:797`

Relevant repository verification commands:

```sh
go test ./mongoarchive/main ./mongoarchive ./storage ./internal/toolconfig
go test -race ./...
go vet ./...
bats --print-output-on-failure test/test.bats
make release-verify VERSION=<tag>
```

## Priority Definitions

- P0: credible data-loss or security boundary break.
- P1: serious correctness or operational-safety gap with misleading success/failure behavior.
- P2: maintainability, observability, or documentation improvement.

## Execution Waves

1. Lock the desired runtime contract and add failing regression tests.
2. Implement orchestration behavior, exit semantics, and user-visible reporting.
3. Align documentation and run focused plus full verification.
4. Perform an independent review of mixed-backend behavior.

## Detailed Tasks

### Task CONTRACT-01: Define And Enforce Multi-Backend Archive Outcome Semantics

Status: completed

Priority: P1

Suggested agent: Go storage and orchestration engineer

Dependencies: none

Primary ownership:

- `mongoarchive/main/mongoarchive.go`
- `mongoarchive/main/mongoarchive_test.go`
- `mongoarchive/flags.go`
- `mongoarchive/flags_test.go`
- focused storage/orchestration tests

References:

- `mongoarchive/main/mongoarchive.go:382-404`
- `internal/toolconfig/shared.go:537-579`
- `README.md:133`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:320-321`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:338`

Finding:

Configured storage initialization is fail-closed, but runtime archive delivery is not. `uploadBackupToStorages` uploads and runs retention one backend at a time, returning immediately on the first later failure. If backend A uploads successfully and deletes expired objects, then backend B fails, the process exits nonzero after only a subset of configured backends has been mutated. This is neither true all-or-nothing behavior nor an explicit partial-success contract.

Implementation requirements:

1. Define one explicit supported runtime contract for multi-backend archive execution and apply it consistently to upload, retention, exit status, logs, and notifications.
2. Supported outcomes are limited to:
   - strict fail-closed/all-or-nothing behavior for multi-backend archive execution, or
   - an explicit opt-in best-effort mode with documented partial-success semantics.
3. If best-effort behavior exists, it must not be implicit. Require an explicit flag or configuration mode, and define the non-success exit contract plus per-backend operator-visible reporting.
4. Never run retention for a backend whose upload failed.
5. If a later backend failure occurs after an earlier backend was already mutated, surface that partial state explicitly instead of implying global success.
6. Add regression tests for at least:
   - first backend upload succeeds, second backend upload fails
   - first backend upload fails before any later backend mutation
   - retention runs only after that backend's verified upload
   - single-backend archive behavior remains unchanged
7. Preserve the existing fail-closed initialization behavior from `StorageOptions.GetStorages` unless a new explicit mode is introduced and documented together with its verification.

Acceptance criteria:

- A two-backend regression test fails against the old orchestration behavior and passes after the fix.
- When mixed backend outcomes occur, the CLI exit status and operator-visible result reflect the chosen contract rather than reporting a generic failure after silent partial mutation.
- Retention is never invoked for a backend that did not complete a verified upload.
- `go test ./mongoarchive/main ./mongoarchive ./storage ./internal/toolconfig` passes.

Completion evidence:

- Changed: `mongoarchive/main/mongoarchive.go`, `mongoarchive/main/mongoarchive_test.go`
- Implemented: a strict two-phase multi-backend contract for `mongo-archive` that uploads every configured backend first, runs retention only after all uploads succeed, preserves single-backend behavior, and returns explicit partial-state errors naming the backends that already uploaded or completed retention
- Verified: `go test ./mongoarchive/main ./mongoarchive ./storage ./internal/toolconfig`
- Verified: `go test -race ./...`
- Verified: `go vet ./...`
- Result: all commands passed

### Task DOCS-01: Align Public Documentation With The Chosen Contract

Status: completed

Priority: P2

Suggested agent: technical writer with Go CLI context

Dependencies: CONTRACT-01

Primary ownership:

- `README.md`
- `flags.md`
- release-facing notes if the public contract changes

References:

- `README.md:133`
- `flags.md`
- `docs/tasks/20260804-165757-codebase-health-remediation.md:797`

Finding:

The current documentation states only that multiple backends can be enabled at the same time. It does not tell operators whether mixed backend results are fail-closed, partially successful, retriable, or tied to a dedicated mode/exit contract.

Implementation requirements:

1. Document the chosen multi-backend archive contract in operator-facing terms.
2. If partial success is supported, document how to enable it, what exit result to expect, and what retention guarantees still hold.
3. If behavior remains strict by default, document what happens when one configured backend fails.
4. Keep `README.md`, `flags.md`, and any release-facing notes synchronized.

Acceptance criteria:

- A user can tell from the docs what happens when one archive backend succeeds and another fails.
- Any new flag or mode is present in `flags.md` and described in `README.md`.
- Documentation changes are verified by the existing flag-doc checks if definitions change.

Completion evidence:

- Changed: `README.md`, `CHANGELOG.md`
- Documented: the multi-backend two-phase upload/retention contract, explicit partial-state reporting, one-shot versus cron failure behavior, and cron singleton scheduling that skips overlapping runs
- Notes: no new flag or mode was introduced, so `flags.md` did not require regeneration

### Task REVIEW-01: Independently Verify Mixed-Backend Archive Behavior

Status: completed

Priority: P1

Suggested agent: independent senior Go reviewer

Dependencies: CONTRACT-01, DOCS-01

Primary ownership:

- review only across changed archive, storage, and docs files
- minimal missing regression tests discovered during review

References:

- Completion evidence from `CONTRACT-01` and `DOCS-01`
- `mongoarchive/main/mongoarchive.go`
- `mongoarchive/main/mongoarchive_test.go`

Finding:

This defect sits at the orchestration boundary between multiple storage backends and retention. Independent verification is needed to ensure the implemented contract is observable, documented, and not contradicted by logs, notifications, or tests.

Implementation requirements:

1. Verify each acceptance criterion from `CONTRACT-01` and `DOCS-01` against runtime behavior, not only code shape.
2. Re-test mixed backend outcomes with failing first backend, failing later backend, and all-success execution.
3. Confirm notifications and logs do not overstate success when the chosen contract reports a non-success outcome.
4. Confirm docs, flags, and runtime behavior agree.

Acceptance criteria:

- Mixed-backend tests demonstrate the chosen contract end-to-end.
- Targeted, race, and vet checks required by this ticket pass.
- No unresolved P1 ambiguity remains in archive multi-backend semantics.

Completion evidence:

- Reviewer: independent general review agent session `ses_003b2dfdaffe8QNgz7uX62TK7Q`
- Result: no findings were identified in the final reviewed change set
- Residual risk: reviewed coverage is strong at the unit level, but cron mode with the real scheduler plus concrete notifications and storage backends still lacks an end-to-end test for partial multi-backend failures

## Dependency And Parallelization Guidance

- `CONTRACT-01` must run before any documentation finalization.
- `DOCS-01` can start once the runtime contract is chosen and tests prove it.
- `REVIEW-01` must be performed independently after implementation and documentation are complete.

Shared hotspots that must not be edited concurrently:

- `mongoarchive/main/mongoarchive.go`
- `mongoarchive/main/mongoarchive_test.go`
- `README.md`
- `flags.md`

## Deferred Decisions Requiring Maintainer Input

None for this remediation. Resolved during implementation:

1. The default runtime contract remains strict and fail-closed for archive results: multi-backend uploads run in two phases, and any upload or retention failure is reported as a non-success outcome.
2. No explicit best-effort mode or new exit code was added; one-shot runs return nonzero on failure, while cron runs log and notify failed executions without stopping the scheduler.
3. No compensating rollback is attempted. If a later backend fails after an earlier backend already uploaded the new archive, the command reports that partial state explicitly and retention is not started unless the upload phase completed across all configured backends.

## Definition Of Done

- The runtime contract for multi-backend archive execution is explicit in code, tests, and docs.
- Mixed-backend outcomes no longer leave operators guessing whether a failed command changed a subset of configured backends.
- Retention behavior remains tied to verified uploads and never runs for failed uploads.
- Targeted tests, race tests, and vet pass for the final implementation.
- An independent reviewer verifies the final behavior and records any residual risk.
