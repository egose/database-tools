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

## 📦 `mongo-archive`

### Functionality

- Dumps MongoDB data locally.
- Uploads the dump to cloud storage (Azure Blob, AWS S3, or Google Cloud Storage).
- Can be run once or as a cron-scheduled job.

### Parameters

| Flag                                | Environment Variable                              | Type   | Description                                                          |
| ----------------------------------- | ------------------------------------------------- | ------ | -------------------------------------------------------------------- |
| `uri`                               | `MONGOARCHIVE__URI`                               | string | MongoDB URI connection string                                        |
| `db`                                | `MONGOARCHIVE__DB`                                | string | Database to use                                                      |
| `collection`                        | `MONGOARCHIVE__COLLECTION`                        | string | Collection to use                                                    |
| `host`                              | `MONGOARCHIVE__HOST`                              | string | MongoDB host to connect to (for replica sets: `setname/host1,host2`) |
| `port`                              | `MONGOARCHIVE__PORT`                              | string | MongoDB port (can also use `--host hostname:port`)                   |
| `ssl`                               | `MONGOARCHIVE__SSL`                               | bool   | Connect to a mongod or mongos that has SSL enabled                   |
| `ssl-ca-file`                       | `MONGOARCHIVE__SSL_CA_FILE`                       | string | `.pem` file containing the root certificate chain from the CA        |
| `ssl-pem-key-file`                  | `MONGOARCHIVE__SSL_PEM_KEY_FILE`                  | string | `.pem` file containing the certificate and key                       |
| `ssl-pem-key-password`              | `MONGOARCHIVE__SSL_PEM_KEY_PASSWORD`              | string | Password to decrypt the `sslPEMKeyFile`                              |
| `ssl-crl-file`                      | `MONGOARCHIVE__SSL_CRL_FILE`                      | string | `.pem` file containing the certificate revocation list               |
| `ssl-allow-invalid-certificates`    | `MONGOARCHIVE__SSL_ALLOW_INVALID_CERTIFICATES`    | bool   | Bypass validation for server certificates                            |
| `ssl-allow-invalid-hostnames`       | `MONGOARCHIVE__SSL_ALLOW_INVALID_HOSTNAMES`       | bool   | Bypass validation for server hostnames                               |
| `ssl-fips-mode`                     | `MONGOARCHIVE__SSL_FIPS_MODE`                     | bool   | Use FIPS mode of the installed OpenSSL library                       |
| `username`                          | `MONGOARCHIVE__USERNAME`                          | string | Username for authentication                                          |
| `password`                          | `MONGOARCHIVE__PASSWORD`                          | string | Password for authentication                                          |
| `authentication-database`           | `MONGOARCHIVE__AUTHENTICATION_DATABASE`           | string | Database that holds the user's credentials                           |
| `authentication-mechanism`          | `MONGOARCHIVE__AUTHENTICATION_MECHANISM`          | string | Authentication mechanism to use                                      |
| `gssapi-service-name`               | `MONGOARCHIVE__GSSAPI_SERVICE_NAME`               | string | Service name for GSSAPI/Kerberos auth (default: `mongodb`)           |
| `gssapi-host-name`                  | `MONGOARCHIVE__GSSAPI_HOST_NAME`                  | string | Hostname for GSSAPI/Kerberos auth (default: server address)          |
| `uri-prune`                         | `MONGOARCHIVE__URI_PRUNE`                         | bool   | Prune MongoDB URI connection string (remove credentials etc.)        |
| `query`                             | `MONGOARCHIVE__QUERY`                             | string | Query filter as v2 Extended JSON string                              |
| `query-file`                        | `MONGOARCHIVE__QUERY_FILE`                        | string | Path to file containing query filter (v2 Extended JSON)              |
| `read-preference`                   | `MONGOARCHIVE__READ_PREFERENCE`                   | string | Preference mode (e.g., `nearest`) or preference JSON object          |
| `force-table-scan`                  | `MONGOARCHIVE__FORCE_TABLE_SCAN`                  | bool   | Force a table scan                                                   |
| `verbose`                           | `MONGOARCHIVE__VERBOSE`                           | string | More detailed log output (`-vvvvv` or `--verbose=N`)                 |
| `quiet`                             | `MONGOARCHIVE__QUIET`                             | bool   | Hide all log output                                                  |
| `az-endpoint`                       | `MONGOARCHIVE__AZ_ENDPOINT`                       | string | Azure Blob Storage emulator hostname and port                        |
| `az-account-name`                   | `MONGOARCHIVE__AZ_ACCOUNT_NAME`                   | string | Azure Blob Storage account name                                      |
| `az-account-key`                    | `MONGOARCHIVE__AZ_ACCOUNT_KEY`                    | string | Azure Blob Storage account key                                       |
| `az-container-name`                 | `MONGOARCHIVE__AZ_CONTAINER_NAME`                 | string | Azure Blob Storage container name                                    |
| `aws-endpoint`                      | `MONGOARCHIVE__AWS_ENDPOINT`                      | string | AWS endpoint URL (hostname only or fully qualified URI)              |
| `aws-access-key-id`                 | `MONGOARCHIVE__AWS_ACCESS_KEY_ID`                 | string | AWS access key associated with an IAM account                        |
| `aws-secret-access-key`             | `MONGOARCHIVE__AWS_SECRET_ACCESS_KEY`             | string | AWS secret key associated with the access key                        |
| `aws-region`                        | `MONGOARCHIVE__AWS_REGION`                        | string | AWS region to send requests to                                       |
| `aws-bucket`                        | `MONGOARCHIVE__AWS_BUCKET`                        | string | AWS S3 bucket name                                                   |
| `aws-s3-force-path-style`           | `MONGOARCHIVE__AWS_S3_FORCE_PATH_STYLE`           | bool   | Force path-style S3 addressing instead of virtual-hosted             |
| `gcp-endpoint`                      | `MONGOARCHIVE__GCP_ENDPOINT`                      | string | GCP endpoint URL                                                     |
| `gcp-bucket`                        | `MONGOARCHIVE__GCP_BUCKET`                        | string | GCP storage bucket name                                              |
| `gcp-creds-file`                    | `MONGOARCHIVE__GCP_CREDS_FILE`                    | string | Path to GCP service account credentials file                         |
| `gcp-project-id`                    | `MONGOARCHIVE__GCP_PROJECT_ID`                    | string | GCP project ID                                                       |
| `gcp-private-key-id`                | `MONGOARCHIVE__GCP_PRIVATE_KEY_ID`                | string | GCP private key ID                                                   |
| `gcp-private-key`                   | `MONGOARCHIVE__GCP_PRIVATE_KEY`                   | string | GCP private key                                                      |
| `gcp-client-email`                  | `MONGOARCHIVE__GCP_CLIENT_EMAIL`                  | string | GCP client email                                                     |
| `gcp-client-id`                     | `MONGOARCHIVE__GCP_CLIENT_ID`                     | string | GCP client ID                                                        |
| `local-path`                        | `MONGOARCHIVE__LOCAL_PATH`                        | string | Local directory path to store backups                                |
| `expiry-days`                       | `MONGOARCHIVE__EXPIRY_DAYS`                       | string | Max age (in days) for archives to be retained                        |
| `rocketchat-webhook-url`            | `MONGOARCHIVE__ROCKETCHAT_WEBHOOK_URL`            | string | Rocket.Chat webhook URL                                              |
| `rocketchat-webhook-prefix`         | `MONGOARCHIVE__ROCKETCHAT_WEBHOOK_PREFIX`         | string | Prefix for Rocket.Chat webhook messages                              |
| `rocketchat-notify-on-failure-only` | `MONGOARCHIVE__ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY` | bool   | Send Rocket.Chat notifications only on failure                       |
| `cron`                              | `MONGOARCHIVE__CRON`                              | bool   | Run a cron scheduler and block current execution path                |
| `cron-expression`                   | `MONGOARCHIVE__CRON_EXPRESSION`                   | string | Cron schedule expression                                             |
| `tz`                                | `MONGOARCHIVE__TZ`                                | string | User-specified time zone (see GNU `TZ` variable format)              |
| `keep`                              | `MONGOARCHIVE__KEEP`                              | bool   | Keep data dump after completion                                      |
| `version`                           | _(no env var)_                                    | bool   | Show the version and exit                                            |

## 🔄 `mongo-unarchive`

### Functionality

- Downloads archived MongoDB dumps from supported cloud storage.
- Restores the data to a MongoDB database.
- Supports applying update operations post-restore using a JSON configuration.

### Parameters

| Flag                                   | Environment Variable                                   | Type   | Description                                                                                                                       |
| -------------------------------------- | ------------------------------------------------------ | ------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `verbose`                              | `MONGOUNARCHIVE__VERBOSE`                              | string | More detailed log output (`-vvvvv` or `--verbose=N`)                                                                              |
| `quiet`                                | `MONGOUNARCHIVE__QUIET`                                | bool   | Hide all log output                                                                                                               |
| `host`                                 | `MONGOUNARCHIVE__HOST`                                 | string | MongoDB host to connect to (for replica sets: `setname/host1,host2`)                                                              |
| `port`                                 | `MONGOUNARCHIVE__PORT`                                 | string | MongoDB port (can also use `--host hostname:port`)                                                                                |
| `ssl`                                  | `MONGOUNARCHIVE__SSL`                                  | bool   | Connect to a mongod or mongos that has SSL enabled                                                                                |
| `ssl-ca-file`                          | `MONGOUNARCHIVE__SSL_CA_FILE`                          | string | `.pem` file containing the root certificate chain from the CA                                                                     |
| `ssl-pem-key-file`                     | `MONGOUNARCHIVE__SSL_PEM_KEY_FILE`                     | string | `.pem` file containing the certificate and key                                                                                    |
| `ssl-pem-key-password`                 | `MONGOUNARCHIVE__SSL_PEM_KEY_PASSWORD`                 | string | Password to decrypt the `sslPEMKeyFile`                                                                                           |
| `ssl-crl-file`                         | `MONGOUNARCHIVE__SSL_CRL_FILE`                         | string | `.pem` file containing the certificate revocation list                                                                            |
| `ssl-allow-invalid-certificates`       | `MONGOUNARCHIVE__SSL_ALLOW_INVALID_CERTIFICATES`       | bool   | Bypass validation for server certificates                                                                                         |
| `ssl-allow-invalid-hostnames`          | `MONGOUNARCHIVE__SSL_ALLOW_INVALID_HOSTNAMES`          | bool   | Bypass validation for server hostnames                                                                                            |
| `ssl-fips-mode`                        | `MONGOUNARCHIVE__SSL_FIPS_MODE`                        | bool   | Use FIPS mode of the installed OpenSSL library                                                                                    |
| `username`                             | `MONGOUNARCHIVE__USERNAME`                             | string | Username for authentication                                                                                                       |
| `password`                             | `MONGOUNARCHIVE__PASSWORD`                             | string | Password for authentication                                                                                                       |
| `authentication-database`              | `MONGOUNARCHIVE__AUTHENTICATION_DATABASE`              | string | Database that holds the user's credentials                                                                                        |
| `authentication-mechanism`             | `MONGOUNARCHIVE__AUTHENTICATION_MECHANISM`             | string | Authentication mechanism to use                                                                                                   |
| `gssapi-service-name`                  | `MONGOUNARCHIVE__GSSAPI_SERVICE_NAME`                  | string | Service name for GSSAPI/Kerberos auth (default: `mongodb`)                                                                        |
| `gssapi-host-name`                     | `MONGOUNARCHIVE__GSSAPI_HOST_NAME`                     | string | Hostname for GSSAPI/Kerberos auth (default: server address)                                                                       |
| `uri`                                  | `MONGOUNARCHIVE__URI`                                  | string | MongoDB URI connection string                                                                                                     |
| `uri-prune`                            | `MONGOUNARCHIVE__URI_PRUNE`                            | bool   | Prune MongoDB URI connection string (remove credentials etc.)                                                                     |
| `db`                                   | `MONGOUNARCHIVE__DB`                                   | string | Database to use                                                                                                                   |
| `collection`                           | `MONGOUNARCHIVE__COLLECTION`                           | string | Collection to use                                                                                                                 |
| `ns-exclude`                           | `MONGOUNARCHIVE__NS_EXCLUDE`                           | string | Exclude matching namespaces                                                                                                       |
| `ns-include`                           | `MONGOUNARCHIVE__NS_INCLUDE`                           | string | Include matching namespaces                                                                                                       |
| `ns-from`                              | `MONGOUNARCHIVE__NS_FROM`                              | string | Rename matching namespaces (requires matching `ns-to`)                                                                            |
| `ns-to`                                | `MONGOUNARCHIVE__NS_TO`                                | string | Rename matched namespaces (requires matching `ns-from`)                                                                           |
| `drop`                                 | `MONGOUNARCHIVE__DROP`                                 | bool   | Drop each collection before import                                                                                                |
| `dry-run`                              | `MONGOUNARCHIVE__DRY_RUN`                              | bool   | View summary without importing anything (recommended with verbosity)                                                              |
| `write-concern`                        | `MONGOUNARCHIVE__WRITE_CONCERN`                        | string | Write concern options                                                                                                             |
| `no-index-restore`                     | `MONGOUNARCHIVE__NO_INDEX_RESTORE`                     | bool   | Do not restore indexes                                                                                                            |
| `no-options-restore`                   | `MONGOUNARCHIVE__NO_OPTIONS_RESTORE`                   | bool   | Do not restore collection options                                                                                                 |
| `keep-index-version`                   | `MONGOUNARCHIVE__KEEP_INDEX_VERSION`                   | bool   | Do not update index version                                                                                                       |
| `maintain-insertion-order`             | `MONGOUNARCHIVE__MAINTAIN_INSERTION_ORDER`             | bool   | Restore documents in the order they appear in the input source; also enables `--stopOnError` and restricts insertion workers to 1 |
| `num-parallel-collections`             | `MONGOUNARCHIVE__NUM_PARALLEL_COLLECTIONS`             | string | Number of collections to restore in parallel (default: 4)                                                                         |
| `num-insertion-workers-per-collection` | `MONGOUNARCHIVE__NUM_INSERTION_WORKERS_PER_COLLECTION` | string | Number of insert operations to run concurrently per collection (default: 1)                                                       |
| `stop-on-error`                        | `MONGOUNARCHIVE__STOP_ON_ERROR`                        | bool   | Halt after any insertion error instead of continuing                                                                              |
| `bypass-document-validation`           | `MONGOUNARCHIVE__BYPASS_DOCUMENT_VALIDATION`           | bool   | Bypass document validation                                                                                                        |
| `preserve-uuid`                        | `MONGOUNARCHIVE__PRESERVE_UUID`                        | bool   | Preserve original collection UUIDs (requires `--drop`)                                                                            |
| `az-endpoint`                          | `MONGOUNARCHIVE__AZ_ENDPOINT`                          | string | Azure Blob Storage emulator hostname and port                                                                                     |
| `az-account-name`                      | `MONGOUNARCHIVE__AZ_ACCOUNT_NAME`                      | string | Azure Blob Storage account name                                                                                                   |
| `az-account-key`                       | `MONGOUNARCHIVE__AZ_ACCOUNT_KEY`                       | string | Azure Blob Storage account key                                                                                                    |
| `az-container-name`                    | `MONGOUNARCHIVE__AZ_CONTAINER_NAME`                    | string | Azure Blob Storage container name                                                                                                 |
| `aws-endpoint`                         | `MONGOUNARCHIVE__AWS_ENDPOINT`                         | string | AWS endpoint URL (hostname only or fully qualified URI)                                                                           |
| `aws-access-key-id`                    | `MONGOUNARCHIVE__AWS_ACCESS_KEY_ID`                    | string | AWS access key associated with an IAM account                                                                                     |
| `aws-secret-access-key`                | `MONGOUNARCHIVE__AWS_SECRET_ACCESS_KEY`                | string | AWS secret key associated with the access key                                                                                     |
| `aws-region`                           | `MONGOUNARCHIVE__AWS_REGION`                           | string | AWS region to send requests to                                                                                                    |
| `aws-bucket`                           | `MONGOUNARCHIVE__AWS_BUCKET`                           | string | AWS S3 bucket name                                                                                                                |
| `aws-s3-force-path-style`              | `MONGOUNARCHIVE__AWS_S3_FORCE_PATH_STYLE`              | bool   | Force path-style S3 addressing instead of virtual-hosted                                                                          |
| `gcp-endpoint`                         | `MONGOUNARCHIVE__GCP_ENDPOINT`                         | string | GCP endpoint URL                                                                                                                  |
| `gcp-bucket`                           | `MONGOUNARCHIVE__GCP_BUCKET`                           | string | GCP storage bucket name                                                                                                           |
| `gcp-creds-file`                       | `MONGOUNARCHIVE__GCP_CREDS_FILE`                       | string | Path to GCP service account credentials file                                                                                      |
| `gcp-project-id`                       | `MONGOUNARCHIVE__GCP_PROJECT_ID`                       | string | GCP project ID                                                                                                                    |
| `gcp-private-key-id`                   | `MONGOUNARCHIVE__GCP_PRIVATE_KEY_ID`                   | string | GCP private key ID                                                                                                                |
| `gcp-private-key`                      | `MONGOUNARCHIVE__GCP_PRIVATE_KEY`                      | string | GCP private key                                                                                                                   |
| `gcp-client-email`                     | `MONGOUNARCHIVE__GCP_CLIENT_EMAIL`                     | string | GCP client email                                                                                                                  |
| `gcp-client-id`                        | `MONGOUNARCHIVE__GCP_CLIENT_ID`                        | string | GCP client ID                                                                                                                     |
| `local-path`                           | `MONGOUNARCHIVE__LOCAL_PATH`                           | string | Local directory path to store backups                                                                                             |
| `object-name`                          | `MONGOUNARCHIVE__OBJECT_NAME`                          | string | Object name of the archived file in the storage (optional)                                                                        |
| `dir`                                  | `MONGOUNARCHIVE__DIR`                                  | string | Directory containing the dumped files                                                                                             |
| `updates`                              | `MONGOUNARCHIVE__UPDATES`                              | string | Array of update specifications in JSON string format                                                                              |
| `updates-file`                         | `MONGOUNARCHIVE__UPDATES_FILE`                         | string | Path to a file containing an array of update specifications                                                                       |
| `keep`                                 | `MONGOUNARCHIVE__KEEP`                                 | bool   | Keep data dump after completion                                                                                                   |
| `version`                              | _(no env var)_                                         | bool   | Show the version and exit                                                                                                         |

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
  schedule: "0 12 * * *"
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
              command: ["/bin/sh", "-c"]
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
              command: ["/bin/sh", "-c"]
              args:
                - mongo-archive --db=mydb --read-preference=primary --force-table-scan
              env:
                - name: MONGOARCHIVE__URI
                  value: "mongodb+srv://user:password@cluster0.my.mongodb.net"
                - name: MONGOARCHIVE__AZ_ACCOUNT_NAME
                  value: mystorage
                - name: MONGOARCHIVE__AZ_ACCOUNT_KEY
                  value: myaccountkey
                - name: MONGOARCHIVE__AZ_CONTAINER_NAME
                  value: mybackup
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

## 🗂️ Backlog

> _To be documented._
