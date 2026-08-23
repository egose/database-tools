# Security Policy

This repository handles MongoDB connection strings, cloud storage credentials, backup retention, archive extraction, restore updates, and release publication. Treat all reports involving those areas as security-sensitive until maintainers decide otherwise.

## Private Reporting

Report vulnerabilities through GitHub private vulnerability reporting for this repository:

https://github.com/egose/database-tools/security/advisories/new

Do not open a public issue, pull request, discussion, or commit with exploit details, credentials, private backup names, logs that contain secrets, or proof-of-concept archives. If the private reporting button is unavailable, open a public issue that says only `security contact request` and wait for a maintainer to provide a private channel. Do not include vulnerability details in that public issue.

## Response Expectations

Maintainers should acknowledge a private report within 3 business days and provide an initial triage decision within 10 business days. Confirmed exploitable issues should receive a severity assessment, an owner, and a remediation target before public disclosure.

The expected handling model is:

- Keep the report private while the issue is triaged and fixed.
- Coordinate disclosure timing with the reporter when practical.
- Release a patched version before public details are published for exploitable issues.
- Credit reporters when requested and safe.
- Avoid sharing proof-of-concept archives, credentials, private object names, or customer data outside the private advisory.

## Supported Versions

Only the latest released minor line is supported for security fixes. For example, after `v0.15.x` is released, `v0.15.x` is the supported line and earlier `v0.14.x` or older lines receive fixes only if maintainers explicitly document an exception.

Security fixes should normally ship as a patch release on the supported minor line. If a fix requires a breaking change, maintainers must document the upgrade path in the release notes and this policy or a linked advisory.

## Vulnerability Exceptions

Release vulnerability exceptions are temporary and must remain narrow. Every exception must include all of the following metadata in the release gate implementation or its documented policy:

- Vulnerability ID.
- Scan mode.
- Affected module, package, and symbol where available.
- Rationale.
- Owner.
- Creation date.
- Review or expiry date.
- Live issue, advisory, or task reference.

Expired, malformed, or broader-than-documented exceptions must fail release verification. Current exception policy and ownership are documented in `docs/release-artifact-policy.md`; the release gate enforces this through `scripts/release-govulncheck.sh` and `internal/releasegovulncheck`.

## Sensitive Boundaries

Changes in these areas require security-oriented review before merge:

- Storage backends, object selection, retention, upload verification, download atomicity, and backend identity.
- Restore archive extraction, path containment, extraction limits, restore result handling, and post-restore updates.
- Secret parsing, credential redaction, notification transport, and any log or error text that can contain credentials.
- CI, release verification, vulnerability exceptions, artifact provenance, SBOM generation, container scanning, or publish workflows.
- This security policy and repository governance documentation.
