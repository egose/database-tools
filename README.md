# Database Tools

This repository provides supplementary tools for MongoDB and PostgreSQL backup and restoration workflows:

- **`mongo-archive`** – Dumps MongoDB data to disk and uploads it to supported cloud storage services.
- **`mongo-unarchive`** – Downloads archived dumps from cloud storage and restores them into a live MongoDB database.
- **`postgres-archive`** – Runs `pg_dump` in custom format, packages the dump with a manifest, and uploads it to supported storage services.
- **`postgres-unarchive`** – Downloads a PostgreSQL archive, validates its manifest and payload, and restores it with `pg_restore --exit-on-error`.

Native release archives install the four Go wrappers only. PostgreSQL operations on native installs require compatible `pg_dump` and `pg_restore` executables in `PATH`; the wrappers' `--version` commands do not require those clients. The published container image includes pinned PostgreSQL client tools and verifies `pg_dump` and `pg_restore` during release.

## 🚀 Building the Tools

To build the binaries from source:

1. **Clone the repository**:

   ```sh
   git clone https://github.com/egose/database-tools
   cd database-tools
   ```

2. **Install dependencies and build**:

   ```sh
   go mod tidy
   make build
   ```

   This will install dependencies and build the binaries into the `dist/` directory.

## Installation

You can install `mongo-archive`, `mongo-unarchive`, `postgres-archive`, and `postgres-unarchive` in two ways:

### 1. Install via [asdf](https://asdf-vm.com/) (Recommended)

If you use [asdf](https://asdf-vm.com/) to manage CLI tools, install the embedded `database-tools` plugin from this repository.

```bash
# Add the database-tools plugin (only once)
asdf plugin add database-tools https://github.com/egose/database-tools.git

# Install the desired version
asdf install database-tools <latest-version>

# Set it as the global version
asdf set -u database-tools <latest-version>
# Or set it locally for a project
asdf set database-tools <latest-version>
```

After installation, you can run:

```bash
mongo-archive --version
mongo-unarchive --version
postgres-archive --version
postgres-unarchive --version
```

### 2. Download from GitHub Releases

You can also manually download the prebuilt binaries from the official releases page:

**Releases:** [https://github.com/egose/database-tools/releases](https://github.com/egose/database-tools/releases)

1. Visit the release page for **version <latest-version>**.
2. Download and extract the `.tar.gz` archive for your operating system and architecture.
3. Make the extracted binaries executable and move them into a directory in your `PATH`:

```bash
chmod +x mongo-archive mongo-unarchive postgres-archive postgres-unarchive
sudo mv mongo-archive mongo-unarchive postgres-archive postgres-unarchive /usr/local/bin/
```

### Verify Installation

Run the following commands to confirm the installed version:

```bash
mongo-archive --version
mongo-unarchive --version
postgres-archive --version
postgres-unarchive --version
pg_dump --version
pg_restore --version
```

## ⚙️ Configuration: CLI Flags & Environment Variables

MongoDB commands follow the conventions of MongoDB’s native tools. PostgreSQL commands use typed libpq-style connection options (`--host`, `--port`, `--user`, `--database`, `--ssl-mode`, `--uri`, and `--password`) and execute PostgreSQL clients directly without a shell. Configuration values can also be passed via environment variables for convenience or container-based execution.

PostgreSQL environment lookup checks command-specific variables first, then shared PostgreSQL variables, then unprefixed variables. For example, `postgres-archive` checks `POSTGRESARCHIVE__DATABASE`, then `POSTGRES__DATABASE`, then `DATABASE`; `postgres-unarchive` checks `POSTGRESUNARCHIVE__DATABASE`, then `POSTGRES__DATABASE`, then `DATABASE`.

The authoritative flag reference lives in [`flags.md`](./flags.md). It is verified by tests against the current flag definitions so documentation drift is caught during CI.

## Documentation Site

The Docusaurus documentation app lives in [`website/`](./website). From the repository root, run:

```sh
pnpm docs:start
pnpm docs:build
pnpm docs:typecheck
```

## 📦 `mongo-archive`

### Functionality

- Dumps MongoDB data locally.
- Uploads the dump to cloud storage (Azure Blob, AWS S3, or Google Cloud Storage).
- Can be run once or as a cron-scheduled job. Scheduled runs skip overlapping executions for the same job while a prior run is still active.

### Managed Backup Object Contract

`mongo-archive` now stores managed backups under a dedicated prefix. By default that prefix is `mongo-archive/`, and it can be overridden with `--backup-prefix` or `MONGOARCHIVE__BACKUP_PREFIX`.

- Managed backup object names use `<backup-prefix><generated-name>.tar.gz`.
- Automatic latest-object selection and retention only consider objects inside that prefix whose filename matches the generated backup format.
- Objects outside the prefix, or malformed objects inside the prefix, are ignored by automatic selection and retention.
- New uploads are verified before retention runs, so a failed upload does not trigger deletions.
- Existing legacy backups stored outside the managed prefix are no longer selected automatically; restore them by passing `--object-name` explicitly during `mongo-unarchive`.

### Multi-Backend Archive Contract

When more than one archive backend is configured, `mongo-archive` now runs in two phases:

1. Upload the new archive to every configured backend.
2. Run retention on each backend only after every upload succeeds.

In one-shot mode, any upload or retention failure returns a nonzero exit. In cron mode, the scheduled run is logged as failed and failure notifications are sent while the scheduler keeps running. In both cases, the error output names which backends already received the new archive or completed retention so operators can see any partial state. A later backend failure can still leave the freshly uploaded archive on an earlier backend, but retention never starts until the upload phase succeeds for all configured backends.

## 🔄 `mongo-unarchive`

### Functionality

- Downloads archived MongoDB dumps from supported cloud storage.
- Restores the data to a MongoDB database.
- Supports applying update operations post-restore using a JSON configuration.

### Restore Result Contract

`mongo-unarchive` treats a restore as successful only when `mongorestore` reports no top-level error and zero document-level failures. If any document fails to restore, the command returns a nonzero exit status and reports the successful and failed document counts. Post-restore update operations are not applied after any top-level restore error or document-level restore failure. The tool does not attempt transactional rollback of documents that were already restored before the failure was reported.

### Archive Extraction Limits

`mongo-unarchive` extracts only regular files and directories from `.tar.gz` backups. Absolute paths, `..` traversal, symlinks, hard links, devices, FIFOs, and other unsupported archive entries are rejected. Extraction is staged in a private directory and only moved into place after a full successful extract.

| Environment Variable                      | Default        | Description                                                       |
| ----------------------------------------- | -------------- | ----------------------------------------------------------------- |
| `MONGOUNARCHIVE__ARCHIVE_MAX_ENTRIES`     | `100000`       | Maximum number of archive entries to extract.                     |
| `MONGOUNARCHIVE__ARCHIVE_MAX_ENTRY_BYTES` | `34359738368`  | Maximum size in bytes for a single extracted file (32 GiB).       |
| `MONGOUNARCHIVE__ARCHIVE_MAX_TOTAL_BYTES` | `274877906944` | Maximum combined size in bytes for all extracted files (256 GiB). |

## 🔔 Notifications

`mongo-archive` can notify one or more destinations after each run. The current notification backends are:

- Rocket.Chat webhook
- Slack webhook
- SMTP email
- AWS SES email

Each backend can be enabled independently, and multiple backends can be enabled at the same time. For archive uploads, multi-backend runs follow the two-phase contract above: upload all configured backends first, then run retention. One-shot runs return nonzero on backend failure, while cron runs log and notify the failed execution without stopping the scheduler.

### Failure-Only Notifications

Each backend supports its own `*-notify-on-failure-only` flag/env var. When enabled, success notifications are skipped for that backend while failure notifications are still sent.

## 🐘 PostgreSQL Operations

`postgres-archive` archives exactly one PostgreSQL database per run. Cluster-global roles, tablespaces, physical backups, WAL archiving, point-in-time recovery, replication slots, and multi-database dumps are outside the initial scope.

PostgreSQL managed objects use `postgres-archive/` by default. MongoDB managed objects use `mongo-archive/`. Latest selection and retention are prefix-scoped, so PostgreSQL does not automatically select or delete MongoDB backups and MongoDB does not automatically select or delete PostgreSQL backups. If you set `--backup-prefix`, keep prefixes separated by database family and environment.

The outer storage object remains a managed `.tar.gz` file. Inside it, PostgreSQL archives contain a custom-format `pg_dump` payload and a JSON manifest with format version, database family, dump format, creation time, source database name, and PostgreSQL client version. Credentials and password-bearing connection strings are not written to the manifest.

`postgres-unarchive` restores into an existing target database. It validates the manifest and custom-format dump before invoking `pg_restore --exit-on-error`. It does not pass `--clean` or `--create` by default, does not create a database, and does not promise rollback. A failed restore can leave partial database changes.

PostgreSQL client compatibility follows PostgreSQL's client/server rules: use a `pg_dump` major version compatible with the source server and a `pg_restore` version compatible with the dump and target server. The container image currently includes pinned PostgreSQL 18 clients; native users choose and patch their host clients.

### Suggested Setup

For most production setups, a good pattern is:

- Slack or Rocket.Chat for fast operational visibility
- SMTP email for broader failure distribution
- AWS SES when you want provider-backed email delivery instead of raw SMTP

## 🧪 Usage Examples

### Dump a Database to Azure Storage

```sh
mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name>
```

### Schedule Regular Backups with Cron

```sh
mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --cron \
  --cron-expression="0 * * * *"
```

### Send Failure Alerts to Slack

```sh
mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --slack-webhook-url="https://hooks.slack.com/services/<path>" \
  --slack-webhook-prefix="[prod-backups]" \
  --slack-notify-on-failure-only
```

### Send Failure Alerts by Email

```sh
mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --smtp-host="smtp.example.com" \
  --smtp-port="587" \
  --smtp-username="alerts@example.com" \
  --smtp-password="<smtp_password>" \
  --smtp-from="alerts@example.com" \
  --smtp-to="dba@example.com,ops@example.com" \
  --smtp-subject-prefix="[prod-backups]" \
  --smtp-notify-on-failure-only
```

### Send Failure Alerts with AWS SES

```sh
mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --ses-region="us-east-1" \
  --ses-access-key-id="<aws_access_key_id>" \
  --ses-secret-access-key="<aws_secret_access_key>" \
  --ses-from="alerts@example.com" \
  --ses-to="dba@example.com,ops@example.com" \
  --ses-subject-prefix="[prod-backups]" \
  --ses-notify-on-failure-only
```

### Restore from Azure Storage

```sh
mongo-unarchive \
  --uri="mongodb://localhost:27017" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name>
```

### Restore and Apply Updates

```sh
mongo-unarchive \
  --uri="mongodb://localhost:27017" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --updates-file=/home/nonroot/updates.json
```

### Archive a PostgreSQL Database to Local Storage

```sh
set -eu
postgres-archive \
  --host=postgres.example.com \
  --port=5432 \
  --user=<username> \
  --database=appdb \
  --ssl-mode=require \
  --local-path=/var/backups/database-tools
```

Supply `POSTGRESARCHIVE__PASSWORD` from your shell's secret manager or job secret injection rather than putting it in the command line.

### Restore a Specific PostgreSQL Object

```sh
set -eu
postgres-unarchive \
  --host=postgres.example.com \
  --port=5432 \
  --user=<username> \
  --database=appdb_restore \
  --ssl-mode=require \
  --local-path=/var/backups/database-tools \
  --object-name="postgres-archive/<generated-name>.tar.gz"
```

### Restore the Latest PostgreSQL Archive from S3

```sh
set -eu
postgres-unarchive \
  --database=appdb_restore \
  --host=postgres.example.com \
  --user=<username> \
  --ssl-mode=require \
  --aws-region=us-east-1 \
  --aws-bucket=<bucket_name>
```

#### Sample `updates.json`

```json
[
  {
    "collection": "users",
    "filter": {
      "email": { "$exists": true }
    },
    "update": [
      {
        "$set": {
          "email": {
            "$replaceOne": {
              "input": "$email",
              "find": "@",
              "replacement": "_"
            }
          }
        }
      }
    ]
  }
]
```

## 🐳 Running with Docker

```sh
docker run --rm \
  -v "$(pwd)/tmp:/tmp" \
  -e MONGOARCHIVE__DUMP_PATH=/tmp/datadump \
  ghcr.io/egose/database-tools:0.15.0 \
  mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --keep
```

The container image includes `pg_dump` and `pg_restore`, so PostgreSQL container jobs do not need host PostgreSQL clients:

```sh
docker run --rm \
  -e POSTGRESARCHIVE__HOST=postgres.example.com \
  -e POSTGRESARCHIVE__USER=<username> \
  -e POSTGRESARCHIVE__DATABASE=appdb \
  -e POSTGRESARCHIVE__SSL_MODE=require \
  -e POSTGRESARCHIVE__PASSWORD \
  -v "$(pwd)/backups:/backups" \
  ghcr.io/egose/database-tools:<latest-version> \
  postgres-archive --local-path=/backups
```

## ☁️ Running as a Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mongo-archive
spec:
  schedule: '0 12 * * *'
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      backoffLimit: 3
      template:
        spec:
          restartPolicy: Never
          initContainers:
            - name: backup-permission
              image: alpine:3.18
              command: ['/bin/sh', '-c']
              args:
                - |
                  rm -rf /tmp/*;
                  adduser -D -u 1000 nonroot;
                  chown nonroot:nonroot /tmp;
              volumeMounts:
                - mountPath: /tmp
                  name: backup-volume
          containers:
            - name: backup-job
              image: ghcr.io/egose/database-tools:<latest-version>
              command: ['/bin/sh', '-c']
              args:
                - mongo-archive --db=mydb --read-preference=primary --force-table-scan
              env:
                - name: MONGOARCHIVE__URI
                  valueFrom:
                    secretKeyRef:
                      name: mongo-archive-secrets
                      key: mongodb-uri
                - name: MONGOARCHIVE__AZ_ACCOUNT_NAME
                  valueFrom:
                    secretKeyRef:
                      name: mongo-archive-secrets
                      key: azure-account-name
                - name: MONGOARCHIVE__AZ_ACCOUNT_KEY
                  valueFrom:
                    secretKeyRef:
                      name: mongo-archive-secrets
                      key: azure-account-key
                - name: MONGOARCHIVE__AZ_CONTAINER_NAME
                  valueFrom:
                    secretKeyRef:
                      name: mongo-archive-secrets
                      key: azure-container-name
              volumeMounts:
                - mountPath: /tmp
                  name: backup-volume
          volumes:
            - name: backup-volume
              persistentVolumeClaim:
                claimName: backup-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: backup-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

Use a maintained image tag in place of `<latest-version>` and provide the referenced `mongo-archive-secrets` Secret separately; example manifests in [`examples/`](./examples) are validated by the Go test suite.

## 🗂️ Backlog

> _To be documented._
