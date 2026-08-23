# Extra MongoDB Tools

This repository provides supplementary tools for MongoDB, supporting both backup and restoration workflows:

- **`mongo-archive`** – Dumps MongoDB data to disk and uploads it to supported cloud storage services.
- **`mongo-unarchive`** – Downloads archived dumps from cloud storage and restores them into a live MongoDB database.

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

You can install **mongo-archive** and **mongo-unarchive** in two ways:

### 1. Install via [asdf](https://asdf-vm.com/) (Recommended)

If you use [asdf](https://asdf-vm.com/) to manage CLI tools, you can install the `mongodb-database-tools` plugin and make the CLI available globally.

```bash
# Add the mongodb-database-tools plugin (only once)
asdf plugin add mongodb-database-tools

# Install the desired version
asdf install mongodb-database-tools <latest-version>

# Set it as the global version
asdf global mongodb-database-tools <latest-version>
# Or set it locally for a project
asdf local mongodb-database-tools <latest-version>
```

After installation, you can run:

```bash
mongo-archive --version
mongo-unarchive --version
```

### 2. Download from GitHub Releases

You can also manually download the prebuilt binaries from the official releases page:

**Releases:** [https://github.com/egose/database-tools/releases](https://github.com/egose/database-tools/releases)

1. Visit the release page for **version <latest-version>**.
2. Download the binary for your operating system and architecture.
3. Make the binary executable and move it into a directory in your `PATH`:

```bash
chmod +x mongo-archive
chmod +x mongo-unarchive
sudo mv mongo-archive /usr/local/bin/
sudo mv mongo-unarchive /usr/local/bin/
```

### Verify Installation

Run the following commands to confirm the installed version:

```bash
mongo-archive --version
mongo-unarchive --version
```

## ⚙️ Configuration: CLI Flags & Environment Variables

Both `mongo-archive` and `mongo-unarchive` follow the conventions of MongoDB’s native tools (e.g., `mongodump`, `mongorestore`), using similar command-line arguments. Configuration values can also be passed via environment variables for convenience or container-based execution.

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
  ghcr.io/egose/database-tools:latest \
  mongo-archive \
  --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
  --db=<dbname> \
  --az-account-name=<az_account_name> \
  --az-account-key=<az_account_key> \
  --az-container-name=<az_container_name> \
  --keep
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
