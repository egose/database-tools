Sure! Here's an updated, clean, and structured parameter list with flags for both `mongo-archive` and `mongo-unarchive` based on your provided Go flag definitions. This version is suitable to include in your documentation under **Parameters / Flags** section.

---

# MongoArchive CLI Flags

| Flag                                  | Env Variable                        | Type   | Description                                                                                                                       |
| ------------------------------------- | ----------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `--verbose`                           | `VERBOSE`                           | string | More detailed log output. Include multiple times for increased verbosity (e.g. `-vvvv`) or specify numeric value (`--verbose=3`). |
| `--quiet`                             | `QUIET`                             | bool   | Hide all log output.                                                                                                              |
| `--host`                              | `HOST`                              | string | MongoDB host to connect to. Can specify replica sets as `setname/host1,host2`.                                                    |
| `--port`                              | `PORT`                              | string | MongoDB port (can also be included in `--host` as hostname\:port).                                                                |
| **SSL Options**                       |                                     |        |                                                                                                                                   |
| `--ssl`                               | `SSL`                               | bool   | Connect using SSL/TLS.                                                                                                            |
| `--ssl-ca-file`                       | `SSL_CA_FILE`                       | string | Path to PEM file with root CA certificates.                                                                                       |
| `--ssl-pem-key-file`                  | `SSL_PEM_KEY_FILE`                  | string | Path to PEM file containing client certificate and key.                                                                           |
| `--ssl-pem-key-password`              | `SSL_PEM_KEY_PASSWORD`              | string | Password to decrypt the PEM key file, if encrypted.                                                                               |
| `--ssl-crl-file`                      | `SSL_CRL_FILE`                      | string | PEM file containing the certificate revocation list.                                                                              |
| `--ssl-allow-invalid-certificates`    | `SSL_ALLOW_INVALID_CERTIFICATES`    | bool   | Skip server certificate validation.                                                                                               |
| `--ssl-allow-invalid-hostnames`       | `SSL_ALLOW_INVALID_HOSTNAMES`       | bool   | Skip server hostname validation.                                                                                                  |
| `--ssl-fips-mode`                     | `SSL_FIPS_MODE`                     | bool   | Use FIPS mode of installed OpenSSL library.                                                                                       |
| **Authentication**                    |                                     |        |                                                                                                                                   |
| `--username`                          | `USERNAME`                          | string | Username for authentication.                                                                                                      |
| `--password`                          | `PASSWORD`                          | string | Password for authentication.                                                                                                      |
| `--authentication-database`           | `AUTHENTICATION_DATABASE`           | string | Database holding the user's credentials.                                                                                          |
| `--authentication-mechanism`          | `AUTHENTICATION_MECHANISM`          | string | Authentication mechanism to use.                                                                                                  |
| **Kerberos**                          |                                     |        |                                                                                                                                   |
| `--gssapi-service-name`               | `GSSAPI_SERVICE_NAME`               | string | Service name for GSSAPI/Kerberos authentication (default: `mongodb`).                                                             |
| `--gssapi-host-name`                  | `GSSAPI_HOST_NAME`                  | string | Hostname for GSSAPI/Kerberos authentication (default: remote server address).                                                     |
| **Namespace & Query Options**         |                                     |        |                                                                                                                                   |
| `--db`                                | `DB`                                | string | Database to dump.                                                                                                                 |
| `--collection`                        | `COLLECTION`                        | string | Collection to dump.                                                                                                               |
| `--query`                             | `QUERY`                             | string | Query filter as v2 Extended JSON string.                                                                                          |
| `--query-file`                        | `QUERY_FILE`                        | string | Path to file containing query filter (v2 Extended JSON).                                                                          |
| `--read-preference`                   | `READ_PREFERENCE`                   | string | Read preference mode or JSON object (e.g. `nearest`).                                                                             |
| `--force-table-scan`                  | `FORCE_TABLE_SCAN`                  | bool   | Force a table scan.                                                                                                               |
| **Cloud Storage - Azure**             |                                     |        |                                                                                                                                   |
| `--az-endpoint`                       | `AZ_ENDPOINT`                       | string | Azure Blob Storage endpoint (e.g. emulator hostname and port).                                                                    |
| `--az-account-name`                   | `AZ_ACCOUNT_NAME`                   | string | Azure Blob Storage account name.                                                                                                  |
| `--az-account-key`                    | `AZ_ACCOUNT_KEY`                    | string | Azure Blob Storage account key.                                                                                                   |
| `--az-container-name`                 | `AZ_CONTAINER_NAME`                 | string | Azure Blob Storage container name.                                                                                                |
| **Cloud Storage - AWS S3**            |                                     |        |                                                                                                                                   |
| `--aws-endpoint`                      | `AWS_ENDPOINT`                      | string | AWS S3 endpoint URL or hostname.                                                                                                  |
| `--aws-access-key-id`                 | `AWS_ACCESS_KEY_ID`                 | string | AWS access key ID.                                                                                                                |
| `--aws-secret-access-key`             | `AWS_SECRET_ACCESS_KEY`             | string | AWS secret access key.                                                                                                            |
| `--aws-region`                        | `AWS_REGION`                        | string | AWS region to use (default: `us-east-1`).                                                                                         |
| `--aws-bucket`                        | `AWS_BUCKET`                        | string | AWS S3 bucket name.                                                                                                               |
| `--aws-s3-force-path-style`           | `AWS_S3_FORCE_PATH_STYLE`           | bool   | Force path-style addressing for S3 requests instead of virtual-hosted style.                                                      |
| **Cloud Storage - GCP**               |                                     |        |                                                                                                                                   |
| `--gcp-endpoint`                      | `GCP_ENDPOINT`                      | string | GCP storage endpoint URL.                                                                                                         |
| `--gcp-bucket`                        | `GCP_BUCKET`                        | string | GCP storage bucket name.                                                                                                          |
| `--gcp-creds-file`                    | `GCP_CREDS_FILE`                    | string | Path to GCP service account credentials file.                                                                                     |
| `--gcp-project-id`                    | `GCP_PROJECT_ID`                    | string | GCP project ID.                                                                                                                   |
| `--gcp-private-key-id`                | `GCP_PRIVATE_KEY_ID`                | string | GCP service account private key ID.                                                                                               |
| `--gcp-private-key`                   | `GCP_PRIVATE_KEY`                   | string | GCP service account private key.                                                                                                  |
| `--gcp-client-email`                  | `GCP_CLIENT_EMAIL`                  | string | GCP service account client email.                                                                                                 |
| `--gcp-client-id`                     | `GCP_CLIENT_ID`                     | string | GCP service account client ID.                                                                                                    |
| **Local Options**                     |                                     |        |                                                                                                                                   |
| `--local-path`                        | `LOCAL_PATH`                        | string | Local directory path to store backups.                                                                                            |
| `--expiry-days`                       | `EXPIRY_DAYS`                       | string | Maximum retention age for archives in days.                                                                                       |
| **Notification**                      |                                     |        |                                                                                                                                   |
| `--rocketchat-webhook-url`            | `ROCKETCHAT_WEBHOOK_URL`            | string | Rocket.Chat webhook URL.                                                                                                          |
| `--rocketchat-webhook-prefix`         | `ROCKETCHAT_WEBHOOK_PREFIX`         | string | Rocket.Chat webhook message prefix.                                                                                               |
| `--rocketchat-notify-on-failure-only` | `ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY` | bool   | Only send Rocket.Chat notifications on failures.                                                                                  |
| **Cron and Scheduling**               |                                     |        |                                                                                                                                   |
| `--cron`                              | `CRON`                              | bool   | Run as a cron scheduler, blocking current execution.                                                                              |
| `--cron-expression`                   | `CRON_EXPRESSION`                   | string | Cron schedule expression string.                                                                                                  |
| `--tz`                                | `TZ`                                | string | Time zone for scheduling (see GNU TZ variable).                                                                                   |
| **Miscellaneous**                     |                                     |        |                                                                                                                                   |
| `--keep`                              | `KEEP`                              | bool   | Keep local data dump after upload.                                                                                                |
| `--version`                           |                                     | bool   | Show version information and exit.                                                                                                |

---

# MongoUnarchive CLI Flags

| Flag                                     | Env Variable                           | Type   | Description                                                                                                |
| ---------------------------------------- | -------------------------------------- | ------ | ---------------------------------------------------------------------------------------------------------- |
| `--verbose`                              | `VERBOSE`                              | string | More detailed log output; multiple times for increased verbosity or numeric value (same as mongo-archive). |
| `--quiet`                                | `QUIET`                                | bool   | Hide all log output.                                                                                       |
| `--host`                                 | `HOST`                                 | string | MongoDB host to connect to.                                                                                |
| `--port`                                 | `PORT`                                 | string | MongoDB port.                                                                                              |
| **SSL Options**                          |                                        |        |                                                                                                            |
| `--ssl`                                  | `SSL`                                  | bool   | Use SSL/TLS connection.                                                                                    |
| `--ssl-ca-file`                          | `SSL_CA_FILE`                          | string | Root CA PEM file path.                                                                                     |
| `--ssl-pem-key-file`                     | `SSL_PEM_KEY_FILE`                     | string | Client certificate and key PEM file.                                                                       |
| `--ssl-pem-key-password`                 | `SSL_PEM_KEY_PASSWORD`                 | string | Password for PEM key file.                                                                                 |
| `--ssl-crl-file`                         | `SSL_CRL_FILE`                         | string | Certificate revocation list PEM file.                                                                      |
| `--ssl-allow-invalid-certificates`       | `SSL_ALLOW_INVALID_CERTIFICATES`       | bool   | Skip server certificate validation.                                                                        |
| `--ssl-allow-invalid-hostnames`          | `SSL_ALLOW_INVALID_HOSTNAMES`          | bool   | Skip server hostname validation.                                                                           |
| `--ssl-fips-mode`                        | `SSL_FIPS_MODE`                        | bool   | Use FIPS mode for OpenSSL.                                                                                 |
| **Authentication**                       |                                        |        |                                                                                                            |
| `--username`                             | `USERNAME`                             | string | Username.                                                                                                  |
| `--password`                             | `PASSWORD`                             | string | Password.                                                                                                  |
| `--authentication-database`              | `AUTHENTICATION_DATABASE`              | string | Auth DB.                                                                                                   |
| `--authentication-mechanism`             | `AUTHENTICATION_MECHANISM`             | string | Authentication mechanism.                                                                                  |
| **Kerberos**                             |                                        |        |                                                                                                            |
| `--gssapi-service-name`                  | `GSSAPI_SERVICE_NAME`                  | string | Kerberos service name.                                                                                     |
| `--gssapi-host-name`                     | `GSSAPI_HOST_NAME`                     | string | Kerberos hostname.                                                                                         |
| **URI Options**                          |                                        |        |                                                                                                            |
| `--uri`                                  | `URI`                                  | string | MongoDB URI connection string.                                                                             |
| `--uri-prune`                            | `URI_PRUNE`                            | bool   | Prune MongoDB URI.                                                                                         |
| **Namespace Options**                    |                                        |        |                                                                                                            |
| `--db`                                   | `DB`                                   | string | Database to restore.                                                                                       |
| `--collection`                           | `COLLECTION`                           | string | Collection to restore.                                                                                     |
| `--ns-exclude`                           | `NS_EXCLUDE`                           | string | Exclude matching namespaces.                                                                               |
| `--ns-include`                           | `NS_INCLUDE`                           | string | Include matching namespaces.                                                                               |
| `--ns-from`                              | `NS_FROM`                              | string | Rename matching namespaces from this name (must be paired with `ns-to`).                                   |
| `--ns-to`                                | `NS_TO`                                | string | Rename matching namespaces to this name (must be paired with `ns-from`).                                   |
| **Restore Options**                      |                                        |        |                                                                                                            |
| `--drop`                                 | `DROP`                                 | bool   | Drop each collection before import.                                                                        |
| `--dry-run`                              | `DRY_RUN`                              | bool   | View summary without importing data.                                                                       |
| `--write-concern`                        | `WRITE_CONCERN`                        | string | Write concern options.                                                                                     |
| `--no-index-restore`                     | `NO_INDEX_RESTORE`                     | bool   | Skip restoring indexes.                                                                                    |
| `--no-options-restore`                   | `NO_OPTIONS_RESTORE`                   | bool   | Skip restoring collection options.                                                                         |
| `--keep-index-version`                   | `KEEP_INDEX_VERSION`                   | bool   | Don't update index version during restore.                                                                 |
| `--maintain-insertion-order`             | `MAINTAIN_INSERTION_ORDER`             | bool   | Restore documents in input order, disables parallel insertions, enables `--stop-on-error`.                 |
| `--num-parallel-collections`             | `NUM_PARALLEL_COLLECTIONS`             | string | Number of collections to restore in parallel (default: 4).                                                 |
| `--num-insertion-workers-per-collection` | `NUM_INSERTION_WORKERS_PER_COLLECTION` | string | Number of concurrent inserts per collection (default: 1).                                                  |
| `--stop-on-error`                        | `STOP_ON_ERROR`                        | bool   | Halt restore on first insertion error.                                                                     |
| `--bypass-document-validation`           | `BYPASS_DOCUMENT_VALIDATION`           | bool   | Bypass MongoDB document validation.                                                                        |
| `--preserve-uuid`                        | `PRESERVE_UUID`                        | bool   | Preserve original collection UUIDs (requires drop).                                                        |
| **Cloud Storage - Azure**                |                                        |        |                                                                                                            |
| `--az-endpoint`                          | `AZ_ENDPOINT`                          | string | Azure Blob Storage endpoint.                                                                               |
| `--az-account-name`                      | `AZ_ACCOUNT_NAME`                      | string | Azure Blob Storage account name.                                                                           |
| `--az-account-key`                       | `AZ_ACCOUNT_KEY`                       | string | Azure Blob Storage account key.                                                                            |
| `--az-container-name`                    | `AZ_CONTAINER_NAME`                    | string | Azure Blob Storage container name.                                                                         |
| **Cloud Storage - AWS S3**               |                                        |        |                                                                                                            |
| `--aws-endpoint`                         | `AWS_ENDPOINT`                         | string | AWS endpoint URL or hostname.                                                                              |
| `--aws-access-key-id`                    | `AWS_ACCESS_KEY_ID`                    | string | AWS access key ID.                                                                                         |
| `--aws-secret-access-key`                | `AWS_SECRET_ACCESS_KEY`                | string | AWS secret access key.                                                                                     |
| `--aws-region`                           | `AWS_REGION`                           | string | AWS region (default: `us-east-1`).                                                                         |
| `--aws-bucket`                           | `AWS_BUCKET`                           | string | AWS S3 bucket name.                                                                                        |
| `--aws-s3-force-path-style`              | `AWS_S3_FORCE_PATH_STYLE`              | bool   | Force path-style addressing for S3.                                                                        |
| **Cloud Storage - GCP**                  |                                        |        |                                                                                                            |
| `--gcp-endpoint`                         | `GCP_ENDPOINT`                         | string | GCP endpoint URL.                                                                                          |
| `--gcp-bucket`                           | `GCP_BUCKET`                           | string | GCP bucket name.                                                                                           |
| `--gcp-creds-file`                       | `GCP_CREDS_FILE`                       | string | Path to GCP credentials JSON file.                                                                         |
| `--gcp-project-id`                       | `GCP_PROJECT_ID`                       | string | GCP project ID.                                                                                            |
| `--gcp-private-key-id`                   | `GCP_PRIVATE_KEY_ID`                   | string | GCP private key ID.                                                                                        |
| `--gcp-private-key`                      | `GCP_PRIVATE_KEY`                      | string | GCP private key.                                                                                           |
| `--gcp-client-email`                     | `GCP_CLIENT_EMAIL`                     | string | GCP client email.                                                                                          |
| `--gcp-client-id`                        | `GCP_CLIENT_ID`                        | string | GCP client ID.                                                                                             |
| **Local Options**                        |                                        |        |                                                                                                            |
| `--local-path`                           | `LOCAL_PATH`                           | string | Local directory to store downloaded dump.                                                                  |
| `--object-name`                          | `OBJECT_NAME`                          | string | Object name of archived file in storage (optional).                                                        |
| `--dir`                                  | `DIR`                                  | string | Directory containing dumped files.                                                                         |
| **Updates**                              |                                        |        |                                                                                                            |
| `--updates`                              | `UPDATES`                              | string | JSON string with update specifications to apply after restore.                                             |
| `--updates-file`                         | `UPDATES_FILE`                         | string | Path to file containing array of update specifications.                                                    |
| **Miscellaneous**                        |                                        |        |                                                                                                            |
| `--keep`                                 | `KEEP`                                 | bool   | Keep local data dump after restore.                                                                        |
| `--version`                              |                                        | bool   | Show version info and exit.                                                                                |

---

If you want, I can also help you generate markdown tables ready to paste, or prepare this for your project README or man pages. Just say the word!
