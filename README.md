# Extra MongoDB Tools

This repository provides additional MongoDB Tools with the following functionalities:

- **mongo-archive** - dump MongoDB backups to disk and upload them to cloud storage.
- **mongo-unarchive** - download MongoDB dump files from cloud storage and restore them to a live database.

## Building Tools

To build the MongoDB Tools, follow these steps:

1. Clone the repository:

   ```sh
   git clone https://github.com/egose/database-tools
   cd database-tools
   ```

1. Install dependencies and build the Go binaries:

   ```sh
   go mod tidy
   make build
   ```

This will ensure that all the necessary dependencies are installed and then build the Go binaries in `dist` directory.

## Binary Arguments and Environment Variables

The binaries provided in this repository utilize MongoDB Tools directly, ensuring a familiar interface for users with minimal modifications to the command arguments. The design closely resembles the behavior and command structure of MongoDB's native tools such as `mongodump` and `mongorestore`.

### mongo-archive

| flags                          | environments                                   | type   | description                                                        |
| ------------------------------ | ---------------------------------------------- | ------ | ------------------------------------------------------------------ |
| uri                            | MONGOARCHIVE\_\_URI                            | string | MongoDB uri connection string                                      |
| db                             | MONGOARCHIVE\_\_DB                             | string | database to use                                                    |
| collection                     | MONGOARCHIVE\_\_COLLECTION                     | string | collection to use                                                  |
| host                           | MONGOARCHIVE\_\_HOST                           | string | MongoDB host to connect to                                         |
| port                           | MONGOARCHIVE\_\_PORT                           | string | MongoDB port                                                       |
| ssl                            | MONGOARCHIVE\_\_VERBOSE                        | bool   | connect to a mongod or mongos that has ssl enabled                 |
| ssl-ca-file                    | MONGOARCHIVE\_\_SSL_CA_FILE                    | string | the .pem file containing the root certificate chain                |
| ssl-pem-key-file               | MONGOARCHIVE\_\_SSL_PEM_KEY_FILE               | string | the .pem file containing the certificate and key                   |
| ssl-pem-key-password           | MONGOARCHIVE\_\_SSL_PEM_KEY_PASSWORD           | string | the password to decrypt the sslPEMKeyFile, if necessary            |
| ssl-crl-file                   | MONGOARCHIVE\_\_SSL_CRL_File                   | string | the .pem file containing the certificate revocation list           |
| ssl-allow-invalid-certificates | MONGOARCHIVE\_\_SSL_ALLOW_INVALID_CERTIFICATES | bool   | bypass the validation for server certificates                      |
| ssl-allow-invalid-hostnames    | MONGOARCHIVE\_\_SSL_ALLOW_INVALID_HOSTNAMES    | bool   | bypass the validation for server name                              |
| ssl-fips-mode                  | MONGOARCHIVE\_\_SSL_FIPS_MODE                  | bool   | use FIPS mode of the installed openssl library                     |
| username                       | MONGOARCHIVE\_\_USERNAME                       | string | username for authentication                                        |
| password                       | MONGOARCHIVE\_\_PASSWORD                       | string | password for authentication                                        |
| authentication-database        | MONGOARCHIVE\_\_AUTHENTICATION_DATABASE        | string | database that holds the user's credentials                         |
| authentication-mechanism       | MONGOARCHIVE\_\_AUTHENTICATION_MECHANISM       | string | authentication mechanism to use                                    |
| gssapi-service-name            | MONGOARCHIVE\_\_GSSAPI_SERVICE_NAME            | string | service name to use when authenticating using GSSAPI/Kerberos      |
| gssapi-host-name               | MONGOARCHIVE\_\_GSSAPI_HOST_NAME               | string | hostname to use when authenticating using GSSAPI/Kerberos          |
| query                          | MONGOARCHIVE\_\_QUERY                          | string | query filter, as a v2 Extended JSON string                         |
| query-file                     | MONGOARCHIVE\_\_QUERY_FILE                     | string | path to a file containing a query filter (v2 Extended JSON)        |
| read-preference                | MONGOARCHIVE\_\_READ_PREFERENCE                | string | specify either a preference mode or a preference json objectoutput |
| force-table-scan               | MONGOARCHIVE\_\_FORCE_TABLE_SCAN               | bool   | force a table scanoutput                                           |
| verbose                        | MONGOARCHIVE\_\_VERBOSE                        | string | more detailed log output                                           |
| quiet                          | MONGOARCHIVE\_\_QUIET                          | bool   | hide all log output                                                |
| az-account-name                | MONGOARCHIVE\_\_AZ_ACCOUNT_NAME                | string | Azure Blob Storage Account Name                                    |
| az-account-key                 | MONGOARCHIVE\_\_AZ_ACCOUNT_KEY                 | string | Azure Blob Storage Account Key                                     |
| az-container-name              | MONGOARCHIVE\_\_AZ_CONTAINER_NAME              | string | Azure Blob Storage Container Name                                  |
| aws-access-key-id              | MONGOARCHIVE\_\_AWS_ACCESS_KEY_ID              | string | AWS access key associated with an IAM account                      |
| aws-secret-access-key          | MONGOARCHIVE\_\_AWS_SECRET_ACCESS_KEY          | string | AWS secret key associated with the access keyName                  |
| aws-region                     | MONGOARCHIVE\_\_AWS_REGION                     | string | AWS Region whose servers you want to send your requests to         |
| aws-bucket                     | MONGOARCHIVE\_\_AWS_BUCKET                     | string | AWS S3 bucket name                                                 |
| gcp-bucket                     | MONGOARCHIVE\_\_GCP_BUCKET                     | string | GCP storage bucket name                                            |
| gcp-creds-file                 | MONGOARCHIVE\_\_GCP_CREDS_FILE                 | string | GCP service account's credentials file                             |
| gcp-project-id                 | MONGOARCHIVE\_\_GCP_PROJECT_ID                 | string | GCP service account's project id                                   |
| gcp-private-key-id             | MONGOARCHIVE\_\_GCP_PRIVATE_KEY_ID             | string | GCP service account's private key id                               |
| gcp-private-key                | MONGOARCHIVE\_\_GCP_PRIVATE_KEY                | string | GCP service account's private key                                  |
| gcp-client-email               | MONGOARCHIVE\_\_GCP_CLIENT_EMAIL               | string | GCP service account's client email                                 |
| gcp-client-id                  | MONGOARCHIVE\_\_GCP_CLIENT_ID                  | string | GCP service account's client id                                    |
| cron                           | MONGOARCHIVE\_\_CRON                           | bool   | run a cron schedular and block current execution path              |
| cron-expression                | MONGOARCHIVE\_\_CRON_EXPRESSION                | string | a string describes individual details of the cron schedule         |
| tz                             | MONGOARCHIVE\_\_TZ                             | string | user-specified time zone                                           |
| keep                           | MONGOARCHIVE\_\_KEEP                           | bool   | keep data dump                                                     |
| uri-prune                      | MONGOARCHIVE\_\_URI_PRUNE                      | bool   | prune MongoDB uri connection string                                |

### mongo-unarchive

| flags                                | environments                                           | type   | description                                                         |
| ------------------------------------ | ------------------------------------------------------ | ------ | ------------------------------------------------------------------- |
| uri                                  | MONGOUNARCHIVE\_\_URI                                  | string | MongoDB uri connection string                                       |
| db                                   | MONGOUNARCHIVE\_\_DB                                   | string | database to use                                                     |
| collection                           | MONGOUNARCHIVE\_\_COLLECTION                           | string | collection to use                                                   |
| ns-exclude                           | MONGOUNARCHIVE\_\_NS_EXCLUDE                           | string | exclude matching namespaces                                         |
| ns-include                           | MONGOUNARCHIVE\_\_NS_INCLUDE                           | string | include matching namespaces                                         |
| ns-from                              | MONGOUNARCHIVE\_\_NS_FROM                              | string | rename matching namespaces, must have matching nsTo                 |
| ns-to                                | MONGOUNARCHIVE\_\_NS_TO                                | string | rename matched namespaces, must have matching nsFrom                |
| host                                 | MONGOUNARCHIVE\_\_HOST                                 | string | MongoDB host to connect to                                          |
| port                                 | MONGOUNARCHIVE\_\_PORT                                 | string | MongoDB port                                                        |
| ssl                                  | MONGOUNARCHIVE\_\_VERBOSE                              | bool   | connect to a mongod or mongos that has ssl enabled                  |
| ssl-ca-file                          | MONGOUNARCHIVE\_\_SSL_CA_FILE                          | string | the .pem file containing the root certificate chain                 |
| ssl-pem-key-file                     | MONGOUNARCHIVE\_\_SSL_PEM_KEY_FILE                     | string | the .pem file containing the certificate and key                    |
| ssl-pem-key-password                 | MONGOUNARCHIVE\_\_SSL_PEM_KEY_PASSWORD                 | string | the password to decrypt the sslPEMKeyFile, if necessary             |
| ssl-crl-file                         | MONGOUNARCHIVE\_\_SSL_CRL_File                         | string | the .pem file containing the certificate revocation list            |
| ssl-allow-invalid-certificates       | MONGOUNARCHIVE\_\_SSL_ALLOW_INVALID_CERTIFICATES       | bool   | bypass the validation for server certificates                       |
| ssl-allow-invalid-hostnames          | MONGOUNARCHIVE\_\_SSL_ALLOW_INVALID_HOSTNAMES          | bool   | bypass the validation for server name                               |
| ssl-fips-mode                        | MONGOUNARCHIVE\_\_SSL_FIPS_MODE                        | bool   | use FIPS mode of the installed openssl library                      |
| username                             | MONGOUNARCHIVE\_\_USERNAME                             | string | username for authentication                                         |
| password                             | MONGOUNARCHIVE\_\_PASSWORD                             | string | password for authentication                                         |
| authentication-database              | MONGOUNARCHIVE\_\_AUTHENTICATION_DATABASE              | string | database that holds the user's credentials                          |
| authentication-mechanism             | MONGOUNARCHIVE\_\_AUTHENTICATION_MECHANISM             | string | authentication mechanism to use                                     |
| gssapi-service-name                  | MONGOUNARCHIVE\_\_GSSAPI_SERVICE_NAME                  | string | service name to use when authenticating using GSSAPI/Kerberos       |
| gssapi-host-name                     | MONGOUNARCHIVE\_\_GSSAPI_HOST_NAME                     | string | hostname to use when authenticating using GSSAPI/Kerberos           |
| drop                                 | MONGOUNARCHIVE\_\_DROP                                 | bool   | drop each collection before import                                  |
| dry-run                              | MONGOUNARCHIVE\_\_DRY_RUN                              | bool   | view summary without importing anything. recommended with verbosity |
| write-concern                        | MONGOUNARCHIVE\_\_WRITE_CONCERN                        | string | write concern options                                               |
| no-index-restore                     | MONGOUNARCHIVE\_\_NO_INDEX_RESTORE                     | bool   | don't restore indexes                                               |
| no-options-restore                   | MONGOUNARCHIVE\_\_NO_OPTIONS_RESTORE                   | bool   | don't restore collection options                                    |
| keep-index-version                   | MONGOUNARCHIVE\_\_KEEP_INDEX_VERSION                   | bool   | don't update index version                                          |
| maintain-insertion-order             | MONGOUNARCHIVE\_\_MAINTAIN_INSERTION_ORDER             | bool   | restore the documents in the order of the input source              |
| num-parallel-collections             | MONGOUNARCHIVE\_\_NUM_PARALLEL_COLLECTIONS             | string | number of collections to restore in parallel                        |
| num-insertion-workers-per-collection | MONGOUNARCHIVE\_\_NUM_INSERTION_WORKERS_PER_COLLECTION | string | number of insert operations to run concurrently per collection      |
| stop-on-error                        | MONGOUNARCHIVE\_\_STOP_ON_ERROR                        | string | halt after encountering any error during insertion                  |
| bypass-document-validation           | MONGOUNARCHIVE\_\_BYPASS_DOCUMENT_VALIDATION           | string | bypass document validation                                          |
| preserve-uuid                        | MONGOUNARCHIVE\_\_PRESERVE_UUID                        | string | preserve original collection UUIDs                                  |
| verbose                              | MONGOUNARCHIVE\_\_VERBOSE                              | string | more detailed log output                                            |
| quiet                                | MONGOUNARCHIVE\_\_QUIET                                | bool   | hide all log output                                                 |
| az-account-name                      | MONGOUNARCHIVE\_\_AZ_ACCOUNT_NAME                      | string | Azure Blob Storage Account Name                                     |
| az-account-key                       | MONGOUNARCHIVE\_\_AZ_ACCOUNT_KEY                       | string | Azure Blob Storage Account Key                                      |
| az-container-name                    | MONGOUNARCHIVE\_\_AZ_CONTAINER_NAME                    | string | Azure Blob Storage Container Name                                   |
| aws-access-key-id                    | MONGOUNARCHIVE\_\_AWS_ACCESS_KEY_ID                    | string | AWS access key associated with an IAM account                       |
| aws-secret-access-key                | MONGOUNARCHIVE\_\_AWS_SECRET_ACCESS_KEY                | string | AWS secret key associated with the access keyName                   |
| aws-region                           | MONGOUNARCHIVE\_\_AWS_REGION                           | string | AWS Region whose servers you want to send your requests to          |
| aws-bucket                           | MONGOUNARCHIVE\_\_AWS_BUCKET                           | string | AWS S3 bucket name                                                  |
| gcp-bucket                           | MONGOUNARCHIVE\_\_GCP_BUCKET                           | string | GCP storage bucket name                                             |
| gcp-creds-file                       | MONGOUNARCHIVE\_\_GCP_CREDS_FILE                       | string | GCP service account's credentials file                              |
| gcp-project-id                       | MONGOUNARCHIVE\_\_GCP_PROJECT_ID                       | string | GCP service account's project id                                    |
| gcp-private-key-id                   | MONGOUNARCHIVE\_\_GCP_PRIVATE_KEY_ID                   | string | GCP service account's private key id                                |
| gcp-private-key                      | MONGOUNARCHIVE\_\_GCP_PRIVATE_KEY                      | string | GCP service account's private key                                   |
| gcp-client-email                     | MONGOUNARCHIVE\_\_GCP_CLIENT_EMAIL                     | string | GCP service account's client email                                  |
| gcp-client-id                        | MONGOUNARCHIVE\_\_GCP_CLIENT_ID                        | string | GCP service account's client id                                     |
| object-name                          | MONGOUNARCHIVE\_\_OBJECT_NAME                          | bool   | Object name of the archived file in the storage                     |
| dir                                  | MONGOUNARCHIVE\_\_DIR                                  | bool   | directory name that contains the dumped files                       |
| updates                              | MONGOUNARCHIVE\_\_UPDATES                              | bool   | array of update specifications in JSON string                       |
| updates-file                         | MONGOUNARCHIVE\_\_UPDATES_FILE                         | bool   | path to a file containing an array of update specifications         |
| keep                                 | MONGOUNARCHIVE\_\_KEEP                                 | bool   | keep data dump                                                      |
| uri-prune                            | MONGOUNARCHIVE\_\_URI_PRUNE                            | bool   | prune MongoDB uri connection string                                 |

## Examples

### Dump Database and Upload to Azure Storage

```sh
mongo-archive \
--uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
--db=<dbname> \
--az-account-name=<az_account_name> \
--az-account-key=<az_account_key> \
--az-container-name=<az_container_name>
```

This example demonstrates how to dump the data from a specified database and upload it to Azure storage. Replace <username>, <password>, <dbname>, <az_account_name>, <az_account_key>, and <az_container_name> with the appropriate values for your setup.

### Run Persistent Server for Regular Database Archival

```sh
mongo-archive \
--uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
--db=<dbname> \
--az-account-name=<az_account_name> \
--az-account-key=<az_account_key> \
--az-container-name=<az_container_name> \
--cron \
--cron-expression="* * * * *"
```

This example demonstrates how to run a persistent server that regularly archives a database. The server will execute the archival process based on the specified cron expression. Replace <username>, <password>, <dbname>, <az_account_name>, <az_account_key>, <az_container_name>, and <cron_expression> with your own values.

### Restore the Target Database from Azure Storage

```sh
mongo-unarchive \
--uri="mongodb://localhost:27017" \
--db=<dbname> \
--az-account-name=<az_account_name> \
--az-account-key=<az_account_key> \
--az-container-name=<az_container_name>
```

This example shows how to restore the target database from Azure storage. Replace <dbname>, <az_account_name>, <az_account_key>, and <az_container_name> with your own values. The database will be restored to the MongoDB instance running on localhost:27017.

### Restore the Target Database from Azure Storage and Apply Changes

```sh
mongo-unarchive \
--uri="mongodb://localhost:27017" \
--db=<dbname> \
--az-account-name=<az_account_name> \
--az-account-key=<az_account_key> \
--az-container-name=<az_container_name> \
--updates-file=/home/nonroot/updates.json
```

This example demonstrates how to restore the target database from Azure storage and apply changes contained in an updates file. Replace <dbname>, <az_account_name>, <az_account_key>, <az_container_name>, and /home/nonroot/updates.json with your own values. The updates file should contain the necessary instructions to modify the restored database.

An example of `updates.json`:

```json
[
  {
    "collection": "users",
    "filter": {
      "email": {
        "$exists": true
      }
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

This JSON file provides an example of updating the users collection in the restored database.

### Execute Binary Using Docker Container Image

To execute a binary using a Docker container image, you can use the following command:

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

### Run Kubernetes CronJob with Mounted Volume

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
              imagePullPolicy: IfNotPresent
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
              image: ghcr.io/egose/database-tools:0.2.6
              imagePullPolicy: IfNotPresent
              command: ["/bin/sh", "-c"]
              args:
                - |
                  mongo-archive --db=mydb --read-preference=primary --force-table-scan
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

## Backlog
...
