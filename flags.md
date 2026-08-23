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
