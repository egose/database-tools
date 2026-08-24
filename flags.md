# CLI Flags

This file is generated from the CLI flag definitions. Update the definitions and re-run the verification tests when flags change.

## `mongo-archive`

| Flag | Environment Variable | Type | Description |
| ---- | -------------------- | ---- | ----------- |
| `--verbose` | `MONGOARCHIVE__VERBOSE` | string | more detailed log output (include multiple times for more verbosity, e.g. -vvvvv, or specify a numeric value, e.g. --verbose=N) |
| `--quiet` | `MONGOARCHIVE__QUIET` | bool | hide all log output |
| `--host` | `MONGOARCHIVE__HOST` | string | MongoDB host to connect to (setname/host1,host2 for replica sets) |
| `--port` | `MONGOARCHIVE__PORT` | string | MongoDB port (can also use --host hostname:port) |
| `--ssl` | `MONGOARCHIVE__SSL` | bool | connect to a mongod or mongos that has ssl enabled |
| `--ssl-ca-file` | `MONGOARCHIVE__SSL_CA_FILE` | string | the .pem file containing the root certificate chain from the certificate authority |
| `--ssl-pem-key-file` | `MONGOARCHIVE__SSL_PEM_KEY_FILE` | string | the .pem file containing the certificate and key |
| `--ssl-pem-key-password` | `MONGOARCHIVE__SSL_PEM_KEY_PASSWORD` | string | the password to decrypt the sslPEMKeyFile, if necessary |
| `--ssl-crl-file` | `MONGOARCHIVE__SSL_CRL_FILE` | string | the .pem file containing the certificate revocation list |
| `--ssl-allow-invalid-certificates` | `MONGOARCHIVE__SSL_ALLOW_INVALID_CERTIFICATES` | bool | bypass the validation for server certificates |
| `--ssl-allow-invalid-hostnames` | `MONGOARCHIVE__SSL_ALLOW_INVALID_HOSTNAMES` | bool | bypass the validation for server name |
| `--ssl-fips-mode` | `MONGOARCHIVE__SSL_FIPS_MODE` | bool | use FIPS mode of the installed openssl library |
| `--username` | `MONGOARCHIVE__USERNAME` | string | username for authentication |
| `--password` | `MONGOARCHIVE__PASSWORD` | string | password for authentication |
| `--authentication-database` | `MONGOARCHIVE__AUTHENTICATION_DATABASE` | string | database that holds the user's credentials |
| `--authentication-mechanism` | `MONGOARCHIVE__AUTHENTICATION_MECHANISM` | string | authentication mechanism to use |
| `--gssapi-service-name` | `MONGOARCHIVE__GSSAPI_SERVICE_NAME` | string | service name to use when authenticating using GSSAPI/Kerberos (default: mongodb) |
| `--gssapi-host-name` | `MONGOARCHIVE__GSSAPI_HOST_NAME` | string | hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>) |
| `--db` | `MONGOARCHIVE__DB` | string | database to use |
| `--collection` | `MONGOARCHIVE__COLLECTION` | string | collection to use |
| `--uri` | `MONGOARCHIVE__URI` | string | MongoDB uri connection string |
| `--uri-prune` | `MONGOARCHIVE__URI_PRUNE` | bool | prune MongoDB uri connection string |
| `--query` | `MONGOARCHIVE__QUERY` | string | query filter, as a v2 Extended JSON string |
| `--query-file` | `MONGOARCHIVE__QUERY_FILE` | string | path to a file containing a query filter (v2 Extended JSON) |
| `--read-preference` | `MONGOARCHIVE__READ_PREFERENCE` | string | specify either a preference mode (e.g. 'nearest') or a preference json object |
| `--force-table-scan` | `MONGOARCHIVE__FORCE_TABLE_SCAN` | bool | force a table scan |
| `--az-endpoint` | `MONGOARCHIVE__AZ_ENDPOINT` | string | specify the emulator hostname and Azure Blob Storage port |
| `--az-account-name` | `MONGOARCHIVE__AZ_ACCOUNT_NAME` | string | Azure Blob Storage Account Name |
| `--az-account-key` | `MONGOARCHIVE__AZ_ACCOUNT_KEY` | string | Azure Blob Storage Account Key |
| `--az-container-name` | `MONGOARCHIVE__AZ_CONTAINER_NAME` | string | Azure Blob Storage Container Name |
| `--aws-endpoint` | `MONGOARCHIVE__AWS_ENDPOINT` | string | AWS endpoint URL (hostname only or fully qualified URI) |
| `--aws-access-key-id` | `MONGOARCHIVE__AWS_ACCESS_KEY_ID` | string | AWS access key associated with an IAM account |
| `--aws-secret-access-key` | `MONGOARCHIVE__AWS_SECRET_ACCESS_KEY` | string | AWS secret key associated with the access key |
| `--aws-region` | `MONGOARCHIVE__AWS_REGION` | string | AWS Region whose servers you want to send your requests to |
| `--aws-bucket` | `MONGOARCHIVE__AWS_BUCKET` | string | AWS S3 bucket name |
| `--aws-s3-force-path-style` | `MONGOARCHIVE__AWS_S3_FORCE_PATH_STYLE` | bool | force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`) |
| `--gcp-endpoint` | `MONGOARCHIVE__GCP_ENDPOINT` | string | GCP endpoint URL |
| `--gcp-bucket` | `MONGOARCHIVE__GCP_BUCKET` | string | GCP storage bucket name |
| `--gcp-creds-file` | `MONGOARCHIVE__GCP_CREDS_FILE` | string | GCP service account's credentials file |
| `--gcp-project-id` | `MONGOARCHIVE__GCP_PROJECT_ID` | string | GCP service account's project id |
| `--gcp-private-key-id` | `MONGOARCHIVE__GCP_PRIVATE_KEY_ID` | string | GCP service account's private key id |
| `--gcp-private-key` | `MONGOARCHIVE__GCP_PRIVATE_KEY` | string | GCP service account's private key |
| `--gcp-client-email` | `MONGOARCHIVE__GCP_CLIENT_EMAIL` | string | GCP service account's client email |
| `--gcp-client-id` | `MONGOARCHIVE__GCP_CLIENT_ID` | string | GCP service account's client id |
| `--local-path` | `MONGOARCHIVE__LOCAL_PATH` | string | Local directory path to store backups |
| `--backup-prefix` | `MONGOARCHIVE__BACKUP_PREFIX` | string | Prefix/namespace used for managed backup objects |
| `--expiry-days` | `MONGOARCHIVE__EXPIRY_DAYS` | string | The maximum age, in days, for archives to be retained |
| `--rocketchat-webhook-url` | `MONGOARCHIVE__ROCKETCHAT_WEBHOOK_URL` | string | Rocket Chat Webhook URL |
| `--rocketchat-webhook-prefix` | `MONGOARCHIVE__ROCKETCHAT_WEBHOOK_PREFIX` | string | Rocket Chat Webhook Prefix |
| `--rocketchat-notify-on-failure-only` | `MONGOARCHIVE__ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY` | bool | Send Rocket Chat notifications only when something goes wrong during the execution |
| `--slack-webhook-url` | `MONGOARCHIVE__SLACK_WEBHOOK_URL` | string | Slack webhook URL |
| `--slack-webhook-prefix` | `MONGOARCHIVE__SLACK_WEBHOOK_PREFIX` | string | Slack message prefix |
| `--slack-notify-on-failure-only` | `MONGOARCHIVE__SLACK_NOTIFY_ON_FAILURE_ONLY` | bool | Send Slack notifications only when something goes wrong during the execution |
| `--smtp-host` | `MONGOARCHIVE__SMTP_HOST` | string | SMTP server host |
| `--smtp-port` | `MONGOARCHIVE__SMTP_PORT` | string | SMTP server port |
| `--smtp-username` | `MONGOARCHIVE__SMTP_USERNAME` | string | SMTP username |
| `--smtp-password` | `MONGOARCHIVE__SMTP_PASSWORD` | string | SMTP password |
| `--smtp-from` | `MONGOARCHIVE__SMTP_FROM` | string | SMTP from address |
| `--smtp-to` | `MONGOARCHIVE__SMTP_TO` | string | Comma-separated SMTP recipient addresses |
| `--smtp-subject-prefix` | `MONGOARCHIVE__SMTP_SUBJECT_PREFIX` | string | SMTP email subject prefix |
| `--smtp-notify-on-failure-only` | `MONGOARCHIVE__SMTP_NOTIFY_ON_FAILURE_ONLY` | bool | Send SMTP notifications only when something goes wrong during the execution |
| `--smtp-allow-insecure-no-tls-in-development` | `MONGOARCHIVE__SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT` | bool | Allow SMTP without STARTTLS only for local development or emulator use |
| `--ses-endpoint` | `MONGOARCHIVE__SES_ENDPOINT` | string | AWS SES endpoint override |
| `--ses-region` | `MONGOARCHIVE__SES_REGION` | string | AWS SES region |
| `--ses-access-key-id` | `MONGOARCHIVE__SES_ACCESS_KEY_ID` | string | AWS SES access key ID |
| `--ses-secret-access-key` | `MONGOARCHIVE__SES_SECRET_ACCESS_KEY` | string | AWS SES secret access key |
| `--ses-from` | `MONGOARCHIVE__SES_FROM` | string | AWS SES sender address |
| `--ses-to` | `MONGOARCHIVE__SES_TO` | string | Comma-separated AWS SES recipient addresses |
| `--ses-subject-prefix` | `MONGOARCHIVE__SES_SUBJECT_PREFIX` | string | AWS SES email subject prefix |
| `--ses-notify-on-failure-only` | `MONGOARCHIVE__SES_NOTIFY_ON_FAILURE_ONLY` | bool | Send AWS SES notifications only when something goes wrong during the execution |
| `--notification-allow-insecure-http-in-development` | `MONGOARCHIVE__NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT` | bool | Allow HTTP notification webhooks or endpoint overrides only for local development or emulator use |
| `--cron` | `MONGOARCHIVE__CRON` | bool | run a cron schedular and block current execution path |
| `--cron-expression` | `MONGOARCHIVE__CRON_EXPRESSION` | string | a string describes individual details of the cron schedule |
| `--tz` | `MONGOARCHIVE__TZ` | string | user-specified time zone |
| `--keep` | `MONGOARCHIVE__KEEP` | bool | keep data dump |
| `--version` | _(no env var)_ | bool | Show the version |

### Environment-Only Variables

| Environment Variable | Default | Description |
| -------------------- | ------- | ----------- |
| `MONGOARCHIVE__DUMP_PATH` | _(none)_ | Base directory for per-run dump workspaces before uploads |
| `MONGOARCHIVE__STORAGE_OPERATION_TIMEOUT` | _(none)_ | Optional timeout applied to storage lookup, upload, and retention operations |
| `MONGOARCHIVE__NOTIFICATION_TIMEOUT` | _(none)_ | Optional timeout applied to outbound notification sends |


## `mongo-unarchive`

| Flag | Environment Variable | Type | Description |
| ---- | -------------------- | ---- | ----------- |
| `--verbose` | `MONGOUNARCHIVE__VERBOSE` | string | more detailed log output (include multiple times for more verbosity, e.g. -vvvvv, or specify a numeric value, e.g. --verbose=N) |
| `--quiet` | `MONGOUNARCHIVE__QUIET` | bool | hide all log output |
| `--host` | `MONGOUNARCHIVE__HOST` | string | MongoDB host to connect to (setname/host1,host2 for replica sets) |
| `--port` | `MONGOUNARCHIVE__PORT` | string | MongoDB port (can also use --host hostname:port) |
| `--ssl` | `MONGOUNARCHIVE__SSL` | bool | connect to a mongod or mongos that has ssl enabled |
| `--ssl-ca-file` | `MONGOUNARCHIVE__SSL_CA_FILE` | string | the .pem file containing the root certificate chain from the certificate authority |
| `--ssl-pem-key-file` | `MONGOUNARCHIVE__SSL_PEM_KEY_FILE` | string | the .pem file containing the certificate and key |
| `--ssl-pem-key-password` | `MONGOUNARCHIVE__SSL_PEM_KEY_PASSWORD` | string | the password to decrypt the sslPEMKeyFile, if necessary |
| `--ssl-crl-file` | `MONGOUNARCHIVE__SSL_CRL_FILE` | string | the .pem file containing the certificate revocation list |
| `--ssl-allow-invalid-certificates` | `MONGOUNARCHIVE__SSL_ALLOW_INVALID_CERTIFICATES` | bool | bypass the validation for server certificates |
| `--ssl-allow-invalid-hostnames` | `MONGOUNARCHIVE__SSL_ALLOW_INVALID_HOSTNAMES` | bool | bypass the validation for server name |
| `--ssl-fips-mode` | `MONGOUNARCHIVE__SSL_FIPS_MODE` | bool | use FIPS mode of the installed openssl library |
| `--username` | `MONGOUNARCHIVE__USERNAME` | string | username for authentication |
| `--password` | `MONGOUNARCHIVE__PASSWORD` | string | password for authentication |
| `--authentication-database` | `MONGOUNARCHIVE__AUTHENTICATION_DATABASE` | string | database that holds the user's credentials |
| `--authentication-mechanism` | `MONGOUNARCHIVE__AUTHENTICATION_MECHANISM` | string | authentication mechanism to use |
| `--gssapi-service-name` | `MONGOUNARCHIVE__GSSAPI_SERVICE_NAME` | string | service name to use when authenticating using GSSAPI/Kerberos (default: mongodb) |
| `--gssapi-host-name` | `MONGOUNARCHIVE__GSSAPI_HOST_NAME` | string | hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>) |
| `--db` | `MONGOUNARCHIVE__DB` | string | database to use |
| `--collection` | `MONGOUNARCHIVE__COLLECTION` | string | collection to use |
| `--uri` | `MONGOUNARCHIVE__URI` | string | MongoDB uri connection string |
| `--uri-prune` | `MONGOUNARCHIVE__URI_PRUNE` | bool | prune MongoDB uri connection string |
| `--ns-exclude` | `MONGOUNARCHIVE__NS_EXCLUDE` | string | exclude matching namespaces |
| `--ns-include` | `MONGOUNARCHIVE__NS_INCLUDE` | string | include matching namespaces |
| `--ns-from` | `MONGOUNARCHIVE__NS_FROM` | string | rename matching namespaces, must have matching nsTo |
| `--ns-to` | `MONGOUNARCHIVE__NS_TO` | string | rename matched namespaces, must have matching nsFrom |
| `--drop` | `MONGOUNARCHIVE__DROP` | bool | drop each collection before import |
| `--dry-run` | `MONGOUNARCHIVE__DRY_RUN` | bool | view summary without importing anything; cannot be combined with updates |
| `--write-concern` | `MONGOUNARCHIVE__WRITE_CONCERN` | string | write concern options |
| `--no-index-restore` | `MONGOUNARCHIVE__NO_INDEX_RESTORE` | bool | don't restore indexes |
| `--no-options-restore` | `MONGOUNARCHIVE__NO_OPTIONS_RESTORE` | bool | don't restore collection options |
| `--keep-index-version` | `MONGOUNARCHIVE__KEEP_INDEX_VERSION` | bool | don't update index version |
| `--maintain-insertion-order` | `MONGOUNARCHIVE__MAINTAIN_INSERTION_ORDER` | bool | restore the documents in the order of their appearance in the input source. By default the insertions will be performed in an arbitrary order. Setting this flag also enables the behavior of --stopOnError and restricts NumInsertionWorkersPerCollection to 1 |
| `--num-parallel-collections` | `MONGOUNARCHIVE__NUM_PARALLEL_COLLECTIONS` | string | number of collections to restore in parallel (default: 4) |
| `--num-insertion-workers-per-collection` | `MONGOUNARCHIVE__NUM_INSERTION_WORKERS_PER_COLLECTION` | string | number of insert operations to run concurrently per collection (default: 1) |
| `--stop-on-error` | `MONGOUNARCHIVE__STOP_ON_ERROR` | bool | halt after encountering any error during insertion. By default, mongorestore will attempt to continue through document validation and DuplicateKey errors, but with this option enabled, the tool will stop instead. A small number of documents may be inserted after encountering an error even with this option enabled; use --maintainInsertionOrder to halt immediately after an error |
| `--bypass-document-validation` | `MONGOUNARCHIVE__BYPASS_DOCUMENT_VALIDATION` | bool | bypass document validation |
| `--preserve-uuid` | `MONGOUNARCHIVE__PRESERVE_UUID` | bool | preserve original collection UUIDs (off by default, requires drop) |
| `--az-endpoint` | `MONGOUNARCHIVE__AZ_ENDPOINT` | string | specify the emulator hostname and Azure Blob Storage port |
| `--az-account-name` | `MONGOUNARCHIVE__AZ_ACCOUNT_NAME` | string | Azure Blob Storage Account Name |
| `--az-account-key` | `MONGOUNARCHIVE__AZ_ACCOUNT_KEY` | string | Azure Blob Storage Account Key |
| `--az-container-name` | `MONGOUNARCHIVE__AZ_CONTAINER_NAME` | string | Azure Blob Storage Container Name |
| `--aws-endpoint` | `MONGOUNARCHIVE__AWS_ENDPOINT` | string | AWS endpoint URL (hostname only or fully qualified URI) |
| `--aws-access-key-id` | `MONGOUNARCHIVE__AWS_ACCESS_KEY_ID` | string | AWS access key associated with an IAM account |
| `--aws-secret-access-key` | `MONGOUNARCHIVE__AWS_SECRET_ACCESS_KEY` | string | AWS secret key associated with the access key |
| `--aws-region` | `MONGOUNARCHIVE__AWS_REGION` | string | AWS Region whose servers you want to send your requests to |
| `--aws-bucket` | `MONGOUNARCHIVE__AWS_BUCKET` | string | AWS S3 bucket name |
| `--aws-s3-force-path-style` | `MONGOUNARCHIVE__AWS_S3_FORCE_PATH_STYLE` | bool | force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`) |
| `--gcp-endpoint` | `MONGOUNARCHIVE__GCP_ENDPOINT` | string | GCP endpoint URL |
| `--gcp-bucket` | `MONGOUNARCHIVE__GCP_BUCKET` | string | GCP storage bucket name |
| `--gcp-creds-file` | `MONGOUNARCHIVE__GCP_CREDS_FILE` | string | GCP service account's credentials file |
| `--gcp-project-id` | `MONGOUNARCHIVE__GCP_PROJECT_ID` | string | GCP service account's project id |
| `--gcp-private-key-id` | `MONGOUNARCHIVE__GCP_PRIVATE_KEY_ID` | string | GCP service account's private key id |
| `--gcp-private-key` | `MONGOUNARCHIVE__GCP_PRIVATE_KEY` | string | GCP service account's private key |
| `--gcp-client-email` | `MONGOUNARCHIVE__GCP_CLIENT_EMAIL` | string | GCP service account's client email |
| `--gcp-client-id` | `MONGOUNARCHIVE__GCP_CLIENT_ID` | string | GCP service account's client id |
| `--local-path` | `MONGOUNARCHIVE__LOCAL_PATH` | string | Local directory path to store backups |
| `--backup-prefix` | `MONGOUNARCHIVE__BACKUP_PREFIX` | string | Prefix/namespace used for managed backup objects |
| `--storage-backend` | `MONGOUNARCHIVE__STORAGE_BACKEND` | string | Storage backend to use for restore when multiple backends are configured (azure, aws, gcp, local) |
| `--object-name` | `MONGOUNARCHIVE__OBJECT_NAME` | string | Object name of the archived file in the storage (optional) |
| `--dir` | `MONGOUNARCHIVE__DIR` | string | directory name that contains the dumped files |
| `--updates` | `MONGOUNARCHIVE__UPDATES` | string | array of update specifications in JSON string |
| `--updates-file` | `MONGOUNARCHIVE__UPDATES_FILE` | string | path to a file containing an array of update specifications |
| `--keep` | `MONGOUNARCHIVE__KEEP` | bool | keep data dump |
| `--version` | _(no env var)_ | bool | Show the version |

### Environment-Only Variables

| Environment Variable | Default | Description |
| -------------------- | ------- | ----------- |
| `MONGOUNARCHIVE__RESTORE_PATH` | _(none)_ | Base directory for per-run restore workspaces before extraction |
| `MONGOUNARCHIVE__ARCHIVE_MAX_ENTRIES` | 100000 | Maximum number of entries allowed while extracting an archive |
| `MONGOUNARCHIVE__ARCHIVE_MAX_ENTRY_BYTES` | 34359738368 | Maximum size in bytes allowed for a single extracted archive entry |
| `MONGOUNARCHIVE__ARCHIVE_MAX_TOTAL_BYTES` | 274877906944 | Maximum combined size in bytes allowed across all extracted archive entries |
| `MONGOUNARCHIVE__UPDATE_MAX_BYTES` | 1048576 | Maximum size in bytes allowed for inline or file-based update specifications |
| `MONGOUNARCHIVE__STORAGE_OPERATION_TIMEOUT` | _(none)_ | Optional timeout applied to storage lookup and download operations |
| `MONGOUNARCHIVE__UPDATE_TIMEOUT` | _(none)_ | Optional timeout applied to MongoDB update connections and update operations |


## `postgres-archive`

| Flag | Environment Variable | Type | Description |
| ---- | -------------------- | ---- | ----------- |
| `--host` | `POSTGRESARCHIVE__HOST` | string | PostgreSQL server host |
| `--port` | `POSTGRESARCHIVE__PORT` | string | PostgreSQL server port |
| `--user` | `POSTGRESARCHIVE__USER` | string | PostgreSQL user |
| `--database` | `POSTGRESARCHIVE__DATABASE` | string | PostgreSQL database to archive |
| `--ssl-mode` | `POSTGRESARCHIVE__SSL_MODE` | string | libpq SSL mode (disable, allow, prefer, require, verify-ca, verify-full) |
| `--uri` | `POSTGRESARCHIVE__URI` | string | PostgreSQL connection URI |
| `--password` | `POSTGRESARCHIVE__PASSWORD` | string | PostgreSQL password |
| `--az-endpoint` | `POSTGRESARCHIVE__AZ_ENDPOINT` | string | specify the emulator hostname and Azure Blob Storage port |
| `--az-account-name` | `POSTGRESARCHIVE__AZ_ACCOUNT_NAME` | string | Azure Blob Storage Account Name |
| `--az-account-key` | `POSTGRESARCHIVE__AZ_ACCOUNT_KEY` | string | Azure Blob Storage Account Key |
| `--az-container-name` | `POSTGRESARCHIVE__AZ_CONTAINER_NAME` | string | Azure Blob Storage Container Name |
| `--aws-endpoint` | `POSTGRESARCHIVE__AWS_ENDPOINT` | string | AWS endpoint URL (hostname only or fully qualified URI) |
| `--aws-access-key-id` | `POSTGRESARCHIVE__AWS_ACCESS_KEY_ID` | string | AWS access key associated with an IAM account |
| `--aws-secret-access-key` | `POSTGRESARCHIVE__AWS_SECRET_ACCESS_KEY` | string | AWS secret key associated with the access key |
| `--aws-region` | `POSTGRESARCHIVE__AWS_REGION` | string | AWS Region whose servers you want to send your requests to |
| `--aws-bucket` | `POSTGRESARCHIVE__AWS_BUCKET` | string | AWS S3 bucket name |
| `--aws-s3-force-path-style` | `POSTGRESARCHIVE__AWS_S3_FORCE_PATH_STYLE` | bool | force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`) |
| `--gcp-endpoint` | `POSTGRESARCHIVE__GCP_ENDPOINT` | string | GCP endpoint URL |
| `--gcp-bucket` | `POSTGRESARCHIVE__GCP_BUCKET` | string | GCP storage bucket name |
| `--gcp-creds-file` | `POSTGRESARCHIVE__GCP_CREDS_FILE` | string | GCP service account's credentials file |
| `--gcp-project-id` | `POSTGRESARCHIVE__GCP_PROJECT_ID` | string | GCP service account's project id |
| `--gcp-private-key-id` | `POSTGRESARCHIVE__GCP_PRIVATE_KEY_ID` | string | GCP service account's private key id |
| `--gcp-private-key` | `POSTGRESARCHIVE__GCP_PRIVATE_KEY` | string | GCP service account's private key |
| `--gcp-client-email` | `POSTGRESARCHIVE__GCP_CLIENT_EMAIL` | string | GCP service account's client email |
| `--gcp-client-id` | `POSTGRESARCHIVE__GCP_CLIENT_ID` | string | GCP service account's client id |
| `--local-path` | `POSTGRESARCHIVE__LOCAL_PATH` | string | Local directory path to store backups |
| `--backup-prefix` | `POSTGRESARCHIVE__BACKUP_PREFIX` | string | Prefix/namespace used for managed backup objects |
| `--expiry-days` | `POSTGRESARCHIVE__EXPIRY_DAYS` | string | maximum archive age in days |
| `--rocketchat-webhook-url` | `POSTGRESARCHIVE__ROCKETCHAT_WEBHOOK_URL` | string | Rocket.Chat webhook URL |
| `--rocketchat-webhook-prefix` | `POSTGRESARCHIVE__ROCKETCHAT_WEBHOOK_PREFIX` | string | Rocket.Chat message prefix |
| `--rocketchat-notify-on-failure-only` | `POSTGRESARCHIVE__ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY` | bool | send Rocket.Chat notifications only on failure |
| `--slack-webhook-url` | `POSTGRESARCHIVE__SLACK_WEBHOOK_URL` | string | Slack webhook URL |
| `--slack-webhook-prefix` | `POSTGRESARCHIVE__SLACK_WEBHOOK_PREFIX` | string | Slack message prefix |
| `--slack-notify-on-failure-only` | `POSTGRESARCHIVE__SLACK_NOTIFY_ON_FAILURE_ONLY` | bool | send Slack notifications only on failure |
| `--smtp-host` | `POSTGRESARCHIVE__SMTP_HOST` | string | SMTP server host |
| `--smtp-port` | `POSTGRESARCHIVE__SMTP_PORT` | string | SMTP server port |
| `--smtp-username` | `POSTGRESARCHIVE__SMTP_USERNAME` | string | SMTP username |
| `--smtp-password` | `POSTGRESARCHIVE__SMTP_PASSWORD` | string | SMTP password |
| `--smtp-from` | `POSTGRESARCHIVE__SMTP_FROM` | string | SMTP sender |
| `--smtp-to` | `POSTGRESARCHIVE__SMTP_TO` | string | comma-separated SMTP recipients |
| `--smtp-subject-prefix` | `POSTGRESARCHIVE__SMTP_SUBJECT_PREFIX` | string | SMTP subject prefix |
| `--smtp-notify-on-failure-only` | `POSTGRESARCHIVE__SMTP_NOTIFY_ON_FAILURE_ONLY` | bool | send SMTP notifications only on failure |
| `--smtp-allow-insecure-no-tls-in-development` | `POSTGRESARCHIVE__SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT` | bool | allow SMTP without STARTTLS for development |
| `--ses-endpoint` | `POSTGRESARCHIVE__SES_ENDPOINT` | string | AWS SES endpoint override |
| `--ses-region` | `POSTGRESARCHIVE__SES_REGION` | string | AWS SES region |
| `--ses-access-key-id` | `POSTGRESARCHIVE__SES_ACCESS_KEY_ID` | string | AWS SES access key ID |
| `--ses-secret-access-key` | `POSTGRESARCHIVE__SES_SECRET_ACCESS_KEY` | string | AWS SES secret access key |
| `--ses-from` | `POSTGRESARCHIVE__SES_FROM` | string | AWS SES sender |
| `--ses-to` | `POSTGRESARCHIVE__SES_TO` | string | comma-separated AWS SES recipients |
| `--ses-subject-prefix` | `POSTGRESARCHIVE__SES_SUBJECT_PREFIX` | string | AWS SES subject prefix |
| `--ses-notify-on-failure-only` | `POSTGRESARCHIVE__SES_NOTIFY_ON_FAILURE_ONLY` | bool | send AWS SES notifications only on failure |
| `--notification-allow-insecure-http-in-development` | `POSTGRESARCHIVE__NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT` | bool | allow HTTP notification endpoints for development |
| `--cron` | `POSTGRESARCHIVE__CRON` | bool | run as a cron scheduler |
| `--cron-expression` | `POSTGRESARCHIVE__CRON_EXPRESSION` | string | cron schedule expression |
| `--tz` | `POSTGRESARCHIVE__TZ` | string | scheduler time zone |
| `--keep` | `POSTGRESARCHIVE__KEEP` | bool | keep the private per-run workspace |
| `--version` | _(no env var)_ | bool | show the version |

### Environment-Only Variables

| Environment Variable | Default | Description |
| -------------------- | ------- | ----------- |
| `POSTGRESARCHIVE__DUMP_PATH` | _(none)_ | Base directory for private per-run PostgreSQL archive workspaces |
| `POSTGRESARCHIVE__STORAGE_OPERATION_TIMEOUT` | _(none)_ | Optional timeout for storage upload and retention operations |
| `POSTGRESARCHIVE__NOTIFICATION_TIMEOUT` | _(none)_ | Optional timeout applied to outbound notification sends |

### PostgreSQL Environment Fallbacks

PostgreSQL commands read command-specific variables first, then shared PostgreSQL variables, then unprefixed variables.

| Key | Lookup Order |
| --- | ------------ |
| `HOST` | `POSTGRESARCHIVE__HOST`, `POSTGRES__HOST`, `HOST` |
| `PORT` | `POSTGRESARCHIVE__PORT`, `POSTGRES__PORT`, `PORT` |
| `USER` | `POSTGRESARCHIVE__USER`, `POSTGRES__USER`, `USER` |
| `DATABASE` | `POSTGRESARCHIVE__DATABASE`, `POSTGRES__DATABASE`, `DATABASE` |
| `SSL_MODE` | `POSTGRESARCHIVE__SSL_MODE`, `POSTGRES__SSL_MODE`, `SSL_MODE` |
| `URI` | `POSTGRESARCHIVE__URI`, `POSTGRES__URI`, `URI` |
| `PASSWORD` | `POSTGRESARCHIVE__PASSWORD`, `POSTGRES__PASSWORD`, `PASSWORD` |
| `AZ_ENDPOINT` | `POSTGRESARCHIVE__AZ_ENDPOINT`, `POSTGRES__AZ_ENDPOINT`, `AZ_ENDPOINT` |
| `AZ_ACCOUNT_NAME` | `POSTGRESARCHIVE__AZ_ACCOUNT_NAME`, `POSTGRES__AZ_ACCOUNT_NAME`, `AZ_ACCOUNT_NAME` |
| `AZ_ACCOUNT_KEY` | `POSTGRESARCHIVE__AZ_ACCOUNT_KEY`, `POSTGRES__AZ_ACCOUNT_KEY`, `AZ_ACCOUNT_KEY` |
| `AZ_CONTAINER_NAME` | `POSTGRESARCHIVE__AZ_CONTAINER_NAME`, `POSTGRES__AZ_CONTAINER_NAME`, `AZ_CONTAINER_NAME` |
| `AWS_ENDPOINT` | `POSTGRESARCHIVE__AWS_ENDPOINT`, `POSTGRES__AWS_ENDPOINT`, `AWS_ENDPOINT` |
| `AWS_ACCESS_KEY_ID` | `POSTGRESARCHIVE__AWS_ACCESS_KEY_ID`, `POSTGRES__AWS_ACCESS_KEY_ID`, `AWS_ACCESS_KEY_ID` |
| `AWS_SECRET_ACCESS_KEY` | `POSTGRESARCHIVE__AWS_SECRET_ACCESS_KEY`, `POSTGRES__AWS_SECRET_ACCESS_KEY`, `AWS_SECRET_ACCESS_KEY` |
| `AWS_REGION` | `POSTGRESARCHIVE__AWS_REGION`, `POSTGRES__AWS_REGION`, `AWS_REGION` |
| `AWS_BUCKET` | `POSTGRESARCHIVE__AWS_BUCKET`, `POSTGRES__AWS_BUCKET`, `AWS_BUCKET` |
| `AWS_S3_FORCE_PATH_STYLE` | `POSTGRESARCHIVE__AWS_S3_FORCE_PATH_STYLE`, `POSTGRES__AWS_S3_FORCE_PATH_STYLE`, `AWS_S3_FORCE_PATH_STYLE` |
| `GCP_ENDPOINT` | `POSTGRESARCHIVE__GCP_ENDPOINT`, `POSTGRES__GCP_ENDPOINT`, `GCP_ENDPOINT` |
| `GCP_BUCKET` | `POSTGRESARCHIVE__GCP_BUCKET`, `POSTGRES__GCP_BUCKET`, `GCP_BUCKET` |
| `GCP_CREDS_FILE` | `POSTGRESARCHIVE__GCP_CREDS_FILE`, `POSTGRES__GCP_CREDS_FILE`, `GCP_CREDS_FILE` |
| `GCP_PROJECT_ID` | `POSTGRESARCHIVE__GCP_PROJECT_ID`, `POSTGRES__GCP_PROJECT_ID`, `GCP_PROJECT_ID` |
| `GCP_PRIVATE_KEY_ID` | `POSTGRESARCHIVE__GCP_PRIVATE_KEY_ID`, `POSTGRES__GCP_PRIVATE_KEY_ID`, `GCP_PRIVATE_KEY_ID` |
| `GCP_PRIVATE_KEY` | `POSTGRESARCHIVE__GCP_PRIVATE_KEY`, `POSTGRES__GCP_PRIVATE_KEY`, `GCP_PRIVATE_KEY` |
| `GCP_CLIENT_EMAIL` | `POSTGRESARCHIVE__GCP_CLIENT_EMAIL`, `POSTGRES__GCP_CLIENT_EMAIL`, `GCP_CLIENT_EMAIL` |
| `GCP_CLIENT_ID` | `POSTGRESARCHIVE__GCP_CLIENT_ID`, `POSTGRES__GCP_CLIENT_ID`, `GCP_CLIENT_ID` |
| `LOCAL_PATH` | `POSTGRESARCHIVE__LOCAL_PATH`, `POSTGRES__LOCAL_PATH`, `LOCAL_PATH` |
| `BACKUP_PREFIX` | `POSTGRESARCHIVE__BACKUP_PREFIX`, `POSTGRES__BACKUP_PREFIX`, `BACKUP_PREFIX` |
| `EXPIRY_DAYS` | `POSTGRESARCHIVE__EXPIRY_DAYS`, `POSTGRES__EXPIRY_DAYS`, `EXPIRY_DAYS` |
| `ROCKETCHAT_WEBHOOK_URL` | `POSTGRESARCHIVE__ROCKETCHAT_WEBHOOK_URL`, `POSTGRES__ROCKETCHAT_WEBHOOK_URL`, `ROCKETCHAT_WEBHOOK_URL` |
| `ROCKETCHAT_WEBHOOK_PREFIX` | `POSTGRESARCHIVE__ROCKETCHAT_WEBHOOK_PREFIX`, `POSTGRES__ROCKETCHAT_WEBHOOK_PREFIX`, `ROCKETCHAT_WEBHOOK_PREFIX` |
| `ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY` | `POSTGRESARCHIVE__ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY`, `POSTGRES__ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY`, `ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY` |
| `SLACK_WEBHOOK_URL` | `POSTGRESARCHIVE__SLACK_WEBHOOK_URL`, `POSTGRES__SLACK_WEBHOOK_URL`, `SLACK_WEBHOOK_URL` |
| `SLACK_WEBHOOK_PREFIX` | `POSTGRESARCHIVE__SLACK_WEBHOOK_PREFIX`, `POSTGRES__SLACK_WEBHOOK_PREFIX`, `SLACK_WEBHOOK_PREFIX` |
| `SLACK_NOTIFY_ON_FAILURE_ONLY` | `POSTGRESARCHIVE__SLACK_NOTIFY_ON_FAILURE_ONLY`, `POSTGRES__SLACK_NOTIFY_ON_FAILURE_ONLY`, `SLACK_NOTIFY_ON_FAILURE_ONLY` |
| `SMTP_HOST` | `POSTGRESARCHIVE__SMTP_HOST`, `POSTGRES__SMTP_HOST`, `SMTP_HOST` |
| `SMTP_PORT` | `POSTGRESARCHIVE__SMTP_PORT`, `POSTGRES__SMTP_PORT`, `SMTP_PORT` |
| `SMTP_USERNAME` | `POSTGRESARCHIVE__SMTP_USERNAME`, `POSTGRES__SMTP_USERNAME`, `SMTP_USERNAME` |
| `SMTP_PASSWORD` | `POSTGRESARCHIVE__SMTP_PASSWORD`, `POSTGRES__SMTP_PASSWORD`, `SMTP_PASSWORD` |
| `SMTP_FROM` | `POSTGRESARCHIVE__SMTP_FROM`, `POSTGRES__SMTP_FROM`, `SMTP_FROM` |
| `SMTP_TO` | `POSTGRESARCHIVE__SMTP_TO`, `POSTGRES__SMTP_TO`, `SMTP_TO` |
| `SMTP_SUBJECT_PREFIX` | `POSTGRESARCHIVE__SMTP_SUBJECT_PREFIX`, `POSTGRES__SMTP_SUBJECT_PREFIX`, `SMTP_SUBJECT_PREFIX` |
| `SMTP_NOTIFY_ON_FAILURE_ONLY` | `POSTGRESARCHIVE__SMTP_NOTIFY_ON_FAILURE_ONLY`, `POSTGRES__SMTP_NOTIFY_ON_FAILURE_ONLY`, `SMTP_NOTIFY_ON_FAILURE_ONLY` |
| `SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT` | `POSTGRESARCHIVE__SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT`, `POSTGRES__SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT`, `SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT` |
| `SES_ENDPOINT` | `POSTGRESARCHIVE__SES_ENDPOINT`, `POSTGRES__SES_ENDPOINT`, `SES_ENDPOINT` |
| `SES_REGION` | `POSTGRESARCHIVE__SES_REGION`, `POSTGRES__SES_REGION`, `SES_REGION` |
| `SES_ACCESS_KEY_ID` | `POSTGRESARCHIVE__SES_ACCESS_KEY_ID`, `POSTGRES__SES_ACCESS_KEY_ID`, `SES_ACCESS_KEY_ID` |
| `SES_SECRET_ACCESS_KEY` | `POSTGRESARCHIVE__SES_SECRET_ACCESS_KEY`, `POSTGRES__SES_SECRET_ACCESS_KEY`, `SES_SECRET_ACCESS_KEY` |
| `SES_FROM` | `POSTGRESARCHIVE__SES_FROM`, `POSTGRES__SES_FROM`, `SES_FROM` |
| `SES_TO` | `POSTGRESARCHIVE__SES_TO`, `POSTGRES__SES_TO`, `SES_TO` |
| `SES_SUBJECT_PREFIX` | `POSTGRESARCHIVE__SES_SUBJECT_PREFIX`, `POSTGRES__SES_SUBJECT_PREFIX`, `SES_SUBJECT_PREFIX` |
| `SES_NOTIFY_ON_FAILURE_ONLY` | `POSTGRESARCHIVE__SES_NOTIFY_ON_FAILURE_ONLY`, `POSTGRES__SES_NOTIFY_ON_FAILURE_ONLY`, `SES_NOTIFY_ON_FAILURE_ONLY` |
| `NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT` | `POSTGRESARCHIVE__NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT`, `POSTGRES__NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT`, `NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT` |
| `CRON` | `POSTGRESARCHIVE__CRON`, `POSTGRES__CRON`, `CRON` |
| `CRON_EXPRESSION` | `POSTGRESARCHIVE__CRON_EXPRESSION`, `POSTGRES__CRON_EXPRESSION`, `CRON_EXPRESSION` |
| `TZ` | `POSTGRESARCHIVE__TZ`, `POSTGRES__TZ`, `TZ` |
| `KEEP` | `POSTGRESARCHIVE__KEEP`, `POSTGRES__KEEP`, `KEEP` |
| `DUMP_PATH` | `POSTGRESARCHIVE__DUMP_PATH`, `POSTGRES__DUMP_PATH`, `DUMP_PATH` |
| `STORAGE_OPERATION_TIMEOUT` | `POSTGRESARCHIVE__STORAGE_OPERATION_TIMEOUT`, `POSTGRES__STORAGE_OPERATION_TIMEOUT`, `STORAGE_OPERATION_TIMEOUT` |
| `NOTIFICATION_TIMEOUT` | `POSTGRESARCHIVE__NOTIFICATION_TIMEOUT`, `POSTGRES__NOTIFICATION_TIMEOUT`, `NOTIFICATION_TIMEOUT` |


## `postgres-unarchive`

| Flag | Environment Variable | Type | Description |
| ---- | -------------------- | ---- | ----------- |
| `--host` | `POSTGRESUNARCHIVE__HOST` | string | PostgreSQL server host |
| `--port` | `POSTGRESUNARCHIVE__PORT` | string | PostgreSQL server port |
| `--user` | `POSTGRESUNARCHIVE__USER` | string | PostgreSQL user |
| `--database` | `POSTGRESUNARCHIVE__DATABASE` | string | existing PostgreSQL database to restore into |
| `--ssl-mode` | `POSTGRESUNARCHIVE__SSL_MODE` | string | libpq SSL mode (disable, allow, prefer, require, verify-ca, verify-full) |
| `--uri` | `POSTGRESUNARCHIVE__URI` | string | PostgreSQL connection URI |
| `--password` | `POSTGRESUNARCHIVE__PASSWORD` | string | PostgreSQL password |
| `--az-endpoint` | `POSTGRESUNARCHIVE__AZ_ENDPOINT` | string | specify the emulator hostname and Azure Blob Storage port |
| `--az-account-name` | `POSTGRESUNARCHIVE__AZ_ACCOUNT_NAME` | string | Azure Blob Storage Account Name |
| `--az-account-key` | `POSTGRESUNARCHIVE__AZ_ACCOUNT_KEY` | string | Azure Blob Storage Account Key |
| `--az-container-name` | `POSTGRESUNARCHIVE__AZ_CONTAINER_NAME` | string | Azure Blob Storage Container Name |
| `--aws-endpoint` | `POSTGRESUNARCHIVE__AWS_ENDPOINT` | string | AWS endpoint URL (hostname only or fully qualified URI) |
| `--aws-access-key-id` | `POSTGRESUNARCHIVE__AWS_ACCESS_KEY_ID` | string | AWS access key associated with an IAM account |
| `--aws-secret-access-key` | `POSTGRESUNARCHIVE__AWS_SECRET_ACCESS_KEY` | string | AWS secret key associated with the access key |
| `--aws-region` | `POSTGRESUNARCHIVE__AWS_REGION` | string | AWS Region whose servers you want to send your requests to |
| `--aws-bucket` | `POSTGRESUNARCHIVE__AWS_BUCKET` | string | AWS S3 bucket name |
| `--aws-s3-force-path-style` | `POSTGRESUNARCHIVE__AWS_S3_FORCE_PATH_STYLE` | bool | force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`) |
| `--gcp-endpoint` | `POSTGRESUNARCHIVE__GCP_ENDPOINT` | string | GCP endpoint URL |
| `--gcp-bucket` | `POSTGRESUNARCHIVE__GCP_BUCKET` | string | GCP storage bucket name |
| `--gcp-creds-file` | `POSTGRESUNARCHIVE__GCP_CREDS_FILE` | string | GCP service account's credentials file |
| `--gcp-project-id` | `POSTGRESUNARCHIVE__GCP_PROJECT_ID` | string | GCP service account's project id |
| `--gcp-private-key-id` | `POSTGRESUNARCHIVE__GCP_PRIVATE_KEY_ID` | string | GCP service account's private key id |
| `--gcp-private-key` | `POSTGRESUNARCHIVE__GCP_PRIVATE_KEY` | string | GCP service account's private key |
| `--gcp-client-email` | `POSTGRESUNARCHIVE__GCP_CLIENT_EMAIL` | string | GCP service account's client email |
| `--gcp-client-id` | `POSTGRESUNARCHIVE__GCP_CLIENT_ID` | string | GCP service account's client id |
| `--local-path` | `POSTGRESUNARCHIVE__LOCAL_PATH` | string | Local directory path to store backups |
| `--backup-prefix` | `POSTGRESUNARCHIVE__BACKUP_PREFIX` | string | Prefix/namespace used for managed backup objects |
| `--storage-backend` | `POSTGRESUNARCHIVE__STORAGE_BACKEND` | string | Storage backend to use for restore when multiple backends are configured (azure, aws, gcp, local) |
| `--object-name` | `POSTGRESUNARCHIVE__OBJECT_NAME` | string | object name to restore; omit to select the latest eligible PostgreSQL archive |
| `--keep` | `POSTGRESUNARCHIVE__KEEP` | bool | keep the private per-run workspace |
| `--version` | _(no env var)_ | bool | show the version |

### Environment-Only Variables

| Environment Variable | Default | Description |
| -------------------- | ------- | ----------- |
| `POSTGRESUNARCHIVE__RESTORE_PATH` | _(none)_ | Base directory for private per-run restore workspaces |
| `POSTGRESUNARCHIVE__ARCHIVE_MAX_ENTRIES` | 100000 | Maximum number of archive entries |
| `POSTGRESUNARCHIVE__ARCHIVE_MAX_ENTRY_BYTES` | 34359738368 | Maximum bytes in one archive entry |
| `POSTGRESUNARCHIVE__ARCHIVE_MAX_TOTAL_BYTES` | 274877906944 | Maximum total extracted bytes |
| `POSTGRESUNARCHIVE__STORAGE_OPERATION_TIMEOUT` | _(none)_ | Optional timeout for storage lookup and download operations |

### PostgreSQL Environment Fallbacks

PostgreSQL commands read command-specific variables first, then shared PostgreSQL variables, then unprefixed variables.

| Key | Lookup Order |
| --- | ------------ |
| `HOST` | `POSTGRESUNARCHIVE__HOST`, `POSTGRES__HOST`, `HOST` |
| `PORT` | `POSTGRESUNARCHIVE__PORT`, `POSTGRES__PORT`, `PORT` |
| `USER` | `POSTGRESUNARCHIVE__USER`, `POSTGRES__USER`, `USER` |
| `DATABASE` | `POSTGRESUNARCHIVE__DATABASE`, `POSTGRES__DATABASE`, `DATABASE` |
| `SSL_MODE` | `POSTGRESUNARCHIVE__SSL_MODE`, `POSTGRES__SSL_MODE`, `SSL_MODE` |
| `URI` | `POSTGRESUNARCHIVE__URI`, `POSTGRES__URI`, `URI` |
| `PASSWORD` | `POSTGRESUNARCHIVE__PASSWORD`, `POSTGRES__PASSWORD`, `PASSWORD` |
| `AZ_ENDPOINT` | `POSTGRESUNARCHIVE__AZ_ENDPOINT`, `POSTGRES__AZ_ENDPOINT`, `AZ_ENDPOINT` |
| `AZ_ACCOUNT_NAME` | `POSTGRESUNARCHIVE__AZ_ACCOUNT_NAME`, `POSTGRES__AZ_ACCOUNT_NAME`, `AZ_ACCOUNT_NAME` |
| `AZ_ACCOUNT_KEY` | `POSTGRESUNARCHIVE__AZ_ACCOUNT_KEY`, `POSTGRES__AZ_ACCOUNT_KEY`, `AZ_ACCOUNT_KEY` |
| `AZ_CONTAINER_NAME` | `POSTGRESUNARCHIVE__AZ_CONTAINER_NAME`, `POSTGRES__AZ_CONTAINER_NAME`, `AZ_CONTAINER_NAME` |
| `AWS_ENDPOINT` | `POSTGRESUNARCHIVE__AWS_ENDPOINT`, `POSTGRES__AWS_ENDPOINT`, `AWS_ENDPOINT` |
| `AWS_ACCESS_KEY_ID` | `POSTGRESUNARCHIVE__AWS_ACCESS_KEY_ID`, `POSTGRES__AWS_ACCESS_KEY_ID`, `AWS_ACCESS_KEY_ID` |
| `AWS_SECRET_ACCESS_KEY` | `POSTGRESUNARCHIVE__AWS_SECRET_ACCESS_KEY`, `POSTGRES__AWS_SECRET_ACCESS_KEY`, `AWS_SECRET_ACCESS_KEY` |
| `AWS_REGION` | `POSTGRESUNARCHIVE__AWS_REGION`, `POSTGRES__AWS_REGION`, `AWS_REGION` |
| `AWS_BUCKET` | `POSTGRESUNARCHIVE__AWS_BUCKET`, `POSTGRES__AWS_BUCKET`, `AWS_BUCKET` |
| `AWS_S3_FORCE_PATH_STYLE` | `POSTGRESUNARCHIVE__AWS_S3_FORCE_PATH_STYLE`, `POSTGRES__AWS_S3_FORCE_PATH_STYLE`, `AWS_S3_FORCE_PATH_STYLE` |
| `GCP_ENDPOINT` | `POSTGRESUNARCHIVE__GCP_ENDPOINT`, `POSTGRES__GCP_ENDPOINT`, `GCP_ENDPOINT` |
| `GCP_BUCKET` | `POSTGRESUNARCHIVE__GCP_BUCKET`, `POSTGRES__GCP_BUCKET`, `GCP_BUCKET` |
| `GCP_CREDS_FILE` | `POSTGRESUNARCHIVE__GCP_CREDS_FILE`, `POSTGRES__GCP_CREDS_FILE`, `GCP_CREDS_FILE` |
| `GCP_PROJECT_ID` | `POSTGRESUNARCHIVE__GCP_PROJECT_ID`, `POSTGRES__GCP_PROJECT_ID`, `GCP_PROJECT_ID` |
| `GCP_PRIVATE_KEY_ID` | `POSTGRESUNARCHIVE__GCP_PRIVATE_KEY_ID`, `POSTGRES__GCP_PRIVATE_KEY_ID`, `GCP_PRIVATE_KEY_ID` |
| `GCP_PRIVATE_KEY` | `POSTGRESUNARCHIVE__GCP_PRIVATE_KEY`, `POSTGRES__GCP_PRIVATE_KEY`, `GCP_PRIVATE_KEY` |
| `GCP_CLIENT_EMAIL` | `POSTGRESUNARCHIVE__GCP_CLIENT_EMAIL`, `POSTGRES__GCP_CLIENT_EMAIL`, `GCP_CLIENT_EMAIL` |
| `GCP_CLIENT_ID` | `POSTGRESUNARCHIVE__GCP_CLIENT_ID`, `POSTGRES__GCP_CLIENT_ID`, `GCP_CLIENT_ID` |
| `LOCAL_PATH` | `POSTGRESUNARCHIVE__LOCAL_PATH`, `POSTGRES__LOCAL_PATH`, `LOCAL_PATH` |
| `BACKUP_PREFIX` | `POSTGRESUNARCHIVE__BACKUP_PREFIX`, `POSTGRES__BACKUP_PREFIX`, `BACKUP_PREFIX` |
| `STORAGE_BACKEND` | `POSTGRESUNARCHIVE__STORAGE_BACKEND`, `POSTGRES__STORAGE_BACKEND`, `STORAGE_BACKEND` |
| `OBJECT_NAME` | `POSTGRESUNARCHIVE__OBJECT_NAME`, `POSTGRES__OBJECT_NAME`, `OBJECT_NAME` |
| `KEEP` | `POSTGRESUNARCHIVE__KEEP`, `POSTGRES__KEEP`, `KEEP` |
| `RESTORE_PATH` | `POSTGRESUNARCHIVE__RESTORE_PATH`, `POSTGRES__RESTORE_PATH`, `RESTORE_PATH` |
| `ARCHIVE_MAX_ENTRIES` | `POSTGRESUNARCHIVE__ARCHIVE_MAX_ENTRIES`, `POSTGRES__ARCHIVE_MAX_ENTRIES`, `ARCHIVE_MAX_ENTRIES` |
| `ARCHIVE_MAX_ENTRY_BYTES` | `POSTGRESUNARCHIVE__ARCHIVE_MAX_ENTRY_BYTES`, `POSTGRES__ARCHIVE_MAX_ENTRY_BYTES`, `ARCHIVE_MAX_ENTRY_BYTES` |
| `ARCHIVE_MAX_TOTAL_BYTES` | `POSTGRESUNARCHIVE__ARCHIVE_MAX_TOTAL_BYTES`, `POSTGRES__ARCHIVE_MAX_TOTAL_BYTES`, `ARCHIVE_MAX_TOTAL_BYTES` |
| `STORAGE_OPERATION_TIMEOUT` | `POSTGRESUNARCHIVE__STORAGE_OPERATION_TIMEOUT`, `POSTGRES__STORAGE_OPERATION_TIMEOUT`, `STORAGE_OPERATION_TIMEOUT` |
