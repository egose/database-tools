# Contributing

Thank you for helping improve `database-tools`. This project contains backup and restore tooling, so correctness and failure behavior matter as much as feature behavior.

## Before You Start

- Read `SECURITY.md` before working on storage, restore, credential handling, release, or policy changes.
- Do not put real credentials, private object names, customer data, or exploitable archive samples in commits, tests, logs, issues, or pull requests.
- Keep changes minimal and focused. Do not rewrite unrelated code or reformat unrelated files.
- If your change touches public flags, environment variables, examples, release behavior, or operational contracts, update the matching documentation in the same pull request.

## Tooling

Use the versions in `.tool-versions` when possible. The repository currently expects Go `1.26.6`, pnpm `11.17.0`, Node.js `26.7.0`, Python `3.14.6`, Bats `1.14.0`, Docker Compose `5.4.0`, actionlint `1.7.12`, and shellcheck `0.11.0`.

Install JavaScript tooling with:

```sh
pnpm install --frozen-lockfile
```

Python CI/release tooling is locked with hashes:

```sh
python3 -m pip install --require-hashes -r requirements-lock.txt
```

## Local Verification

Run focused tests for the packages you changed, then run the normal local checks:

```sh
git diff --check
go test -shuffle=on -coverprofile=/tmp/coverage.out ./...
go tool cover -func=/tmp/coverage.out
go test -race -shuffle=on ./...
go vet ./...
go mod verify
pnpm install --frozen-lockfile
pre-commit run --all-files
actionlint .github/workflows/*.yml .github/workflows/*.yaml
```

The coverage check currently protects a total coverage floor of `55.0%` and a storage package floor of `50.0%`, matching `.github/workflows/test.yml`.

## Integration Verification

Integration tests require Docker services and a test environment file:

```sh
cp .env.example .env.test
mkdir -p ./sandbox/mnt/mongodb ./sandbox/mnt/minio ./sandbox/mnt/azurite ./sandbox/mnt/fake-gcs-server
docker-compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml up -d
bats --print-output-on-failure test/test.bats
docker-compose --env-file .env.test -f sandbox/docker-compose.yml -f sandbox/docker-compose-ci.yml down -v --remove-orphans
```

CI runs the Bats suite through `.github/workflows/integration.yml` against five clean starts. If integration setup fails locally, include the exact Docker, Compose, and service-log error in the pull request.

## Release Verification

Release candidates must pass the release gate before a tag is published:

```sh
make release-verify VERSION=v0.15.0
```

The release gate runs supply-chain checks, frozen pnpm install, unit tests, race tests, vet, module verification, `govulncheck -json` through the repository gate, Dockerfile build checks, cross-platform builds, release archive creation, and reproducible archive verification. See `docs/release-artifact-policy.md` for the release and vulnerability exception policy.

To exercise only the release artifact pipeline, pass the release tag and deterministic timestamp through the trusted configuration:

```sh
VERSION=v0.15.0 SOURCE_DATE_EPOCH=0 pnpm run release:build
VERSION=v0.15.0 SOURCE_DATE_EPOCH=0 pnpm run release:verify
```

## Review Expectations

Pull requests that touch sensitive boundaries should request the matching CODEOWNERS review. In particular, storage behavior, restore extraction, release workflows, security policy, vulnerability exceptions, credential handling, and generated public docs should not merge without owner review.

Repository administrators should keep branch protection aligned with `docs/repository-governance.md` so required checks match the commands above.
