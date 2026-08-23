# Repository Governance

This document defines the repository settings expected to protect the security and release policies in `SECURITY.md`, `CONTRIBUTING.md`, and `docs/release-artifact-policy.md`.

## Required Pull Request Checks

Branch protection for the default branch should require pull requests and the current CI checks before merge:

- `Test codebase / go-test-cover`
- `Test codebase / go-test-race`
- `Test codebase / go-vet`
- `Integration verification / integration-bats (clean start 1/5)`
- `Integration verification / integration-bats (clean start 2/5)`
- `Integration verification / integration-bats (clean start 3/5)`
- `Integration verification / integration-bats (clean start 4/5)`
- `Integration verification / integration-bats (clean start 5/5)`
- `Check code conventions / pre-commit-check`

If merge queue is enabled, keep the existing `merge_group` triggers in `.github/workflows/test.yml` and `.github/workflows/lint.yml` required by branch protection. If merge queue is not enabled, the triggers are still harmless and keep the workflows ready for future enablement.

## Release Protection

Version tags matching `v*.*.*` should be created only by maintainers. The `Release` workflow must remain the only path that creates GitHub Release assets or publishes GHCR images for those tags.

The release workflow must keep these dependencies:

- The `release` job requires both `verify` and `integration`.
- The `publish-image` job requires both `verify` and `integration`.
- The `publish-image` workflow builds and scans a local image before authenticating to GHCR or pushing tags.
- Release artifact, SBOM, image digest, and provenance evidence are written by the workflow.

Repository administrators should protect `v*.*.*` tags or otherwise restrict tag creation to trusted maintainers if the hosting plan supports tag protection or repository rulesets.

## CODEOWNERS Review Boundaries

`.github/CODEOWNERS` routes review requests for the sensitive areas below. The current repository owner account is the reviewer for each area until separate maintainer teams exist.

- Storage: `storage/`, `internal/toolconfig/`, and filesystem helpers that affect object selection, retention, upload verification, download atomicity, and backend identity.
- Restore and extraction: `mongounarchive/`, archive extraction helpers, restore update handling, and restore-facing documentation.
- Release and CI: `.github/workflows/`, `.github/actions/`, `Dockerfile`, `Makefile`, `scripts/`, `internal/releasegovulncheck/`, toolchain files, dependency locks, and release policy.
- Security policy: `SECURITY.md`, `CONTRIBUTING.md`, this governance document, and `.github/CODEOWNERS`.

## Vulnerability Exception Ownership

Vulnerability exceptions are owned by `release-security-maintainers` in policy and by CODEOWNERS review in this repository. Each exception must expire or be reviewed by its documented review date. Owners are responsible for removing exceptions as soon as upstream fixes or local mitigations make them unnecessary.

Current exception details live in `docs/release-artifact-policy.md` and are enforced by `scripts/release-govulncheck.sh`.

## Repository Settings Verification

The following read-only checks were run on 2026-08-23 with the available GitHub token:

- `gh repo view --json nameWithOwner,defaultBranchRef,isPrivate,isSecurityPolicyEnabled,securityPolicyUrl,viewerCanAdminister,viewerPermission,hasIssuesEnabled`
- `gh api repos/egose/database-tools/branches/main`
- `gh api repos/egose/database-tools/branches/main/protection`

Observed results:

- Repository: `egose/database-tools`.
- Default branch: `main`.
- Visibility: public.
- Issues: enabled.
- Current viewer permission: `READ`.
- Current viewer can administer: `false`.
- Security policy was not enabled on the remote default branch before this policy file is merged.
- Branch API reported `main` as `protected: false`, with protection status checks disabled.
- Branch-protection API returned `404`, consistent with missing protection or insufficient admin visibility.

Residual administrator verification is required after this change merges:

- Confirm GitHub recognizes `SECURITY.md` and enables the repository security policy link.
- Enable private vulnerability reporting, or publish another maintainer-approved private reporting channel in `SECURITY.md` before relying on public disclosure workflows.
- Enable branch protection or a repository ruleset for `main` requiring the checks listed above.
- Require CODEOWNERS review for pull requests that touch sensitive boundaries.
- Restrict `v*.*.*` release tag creation to trusted maintainers where supported.
