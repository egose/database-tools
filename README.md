# Extra MongoDB Tools

This repository provides additional MongoDB Tools with the following functionalities:

- **mongoarchive** - dump MongoDB backups to disk and upload them to cloud storage.
- **mongounarchive** - download MongoDB dump files from cloud storage and restore them to a live database.

## Building Tools

To build the MongoDB Tools, follow these steps:

1. Clone the repository:

   ```sh
   git clone https://github.com/junminahn/mongo-tools-ext
   cd mongo-tools-ext
   ```

1. Install dependencies and build the Go binaries:

   ```sh
   go mod tidy
   make build
   ```

This will ensure that all the necessary dependencies are installed and then build the Go binaries in `dist` directory.

## Binary Arguments and Environment Variables

The binaries provided in this repository utilize MongoDB Tools directly, ensuring a familiar interface for users with minimal modifications to the command arguments. The design closely resembles the behavior and command structure of MongoDB's native tools such as `mongodump` and `mongorestore`.

### mongoarchive

| flags                       | environments                                   | type   | description                                                        |
| --------------------------- | ---------------------------------------------- | ------ | ------------------------------------------------------------------ |
| uri                         | MONGOARCHIVE\_\_URI                            | string | MongoDB uri connection string                                      |
| db                          | MONGOARCHIVE\_\_DB                             | string | database to use                                                    |
| collection                  | MONGOARCHIVE\_\_COLLECTION                     | string | collection to use                                                  |
| host                        | MONGOARCHIVE\_\_HOST                           | string | MongoDB host to connect to                                         |
| port                        | MONGOARCHIVE\_\_PORT                           | string | MongoDB port                                                       |
| ssl                         | MONGOARCHIVE\_\_VERBOSE                        | bool   | connect to a mongod or mongos that has ssl enabled                 |
| sslCAFile                   | MONGOARCHIVE\_\_SSL_CA_FILE                    | string | the .pem file containing the root certificate chain                |
| sslPEMKeyFile               | MONGOARCHIVE\_\_SSL_PEM_KEY_FILE               | string | the .pem file containing the certificate and key                   |
| sslPEMKeyPassword           | MONGOARCHIVE\_\_SSL_PEM_KEY_PASSWORD           | string | the password to decrypt the sslPEMKeyFile, if necessary            |
| sslCRLFile                  | MONGOARCHIVE\_\_SSL_CRL_File                   | string | the .pem file containing the certificate revocation list           |
| sslAllowInvalidCertificates | MONGOARCHIVE\_\_SSL_ALLOW_INVALID_CERTIFICATES | bool   | bypass the validation for server certificates                      |
| sslAllowInvalidHostnames    | MONGOARCHIVE\_\_SSL_ALLOW_INVALID_HOSTNAMES    | bool   | bypass the validation for server name                              |
| sslFIPSMode                 | MONGOARCHIVE\_\_SSL_FIPS_MODE                  | bool   | use FIPS mode of the installed openssl library                     |
| username                    | MONGOARCHIVE\_\_USERNAME                       | string | username for authentication                                        |
| password                    | MONGOARCHIVE\_\_PASSWORD                       | string | password for authentication                                        |
| authenticationDatabase      | MONGOARCHIVE\_\_AUTHENTICATION_DATABASE        | string | database that holds the user's credentials                         |
| authenticationMechanism     | MONGOARCHIVE\_\_AUTHENTICATION_MECHANISM       | string | authentication mechanism to use                                    |
| gssapiServiceName           | MONGOARCHIVE\_\_GSSAPI_SERVICE_NAME            | string | service name to use when authenticating using GSSAPI/Kerberos      |
| gssapiHostName              | MONGOARCHIVE\_\_GSSAPI_HOST_NAME               | string | hostname to use when authenticating using GSSAPI/Kerberos          |
| query                       | MONGOARCHIVE\_\_QUERY                          | string | query filter, as a v2 Extended JSON string                         |
| queryFile                   | MONGOARCHIVE\_\_QUERY_FILE                     | string | path to a file containing a query filter (v2 Extended JSON)        |
| readPreference              | MONGOARCHIVE\_\_READ_PREFERENCE                | string | specify either a preference mode or a preference json objectoutput |
| forceTableScan              | MONGOARCHIVE\_\_FORCE_TABLE_SCAN               | bool   | force a table scanoutput                                           |
| verbose                     | MONGOARCHIVE\_\_VERBOSE                        | string | more detailed log output                                           |
| quiet                       | MONGOARCHIVE\_\_QUIET                          | bool   | hide all log output                                                |
| azAccountName               | MONGOARCHIVE\_\_AZ_ACCOUNT_NAME                | string | Azure Blob Storage Account Name                                    |
| azAccountKey                | MONGOARCHIVE\_\_AZ_ACCOUNT_KEY                 | string | Azure Blob Storage Account Key                                     |
| azContainerName             | MONGOARCHIVE\_\_AZ_CONTAINER_NAME              | string | Azure Blob Storage Container Name                                  |
| cron                        | MONGOARCHIVE\_\_CRON                           | bool   | run a cron schedular and block current execution path              |
| cronExpression              | MONGOARCHIVE\_\_CRON_EXPRESSION                | string | a string describes individual details of the cron schedule         |
| tz                          | MONGOARCHIVE\_\_TZ                             | string | user-specified time zone                                           |
| keep                        | MONGOARCHIVE\_\_KEEP                           | bool   | keep data dump                                                     |

### mongounarchive

| flags                            | environments                                           | type   | description                                                         |
| -------------------------------- | ------------------------------------------------------ | ------ | ------------------------------------------------------------------- |
| uri                              | MONGOUNARCHIVE\_\_URI                                  | string | MongoDB uri connection string                                       |
| db                               | MONGOUNARCHIVE\_\_DB                                   | string | database to use                                                     |
| collection                       | MONGOUNARCHIVE\_\_COLLECTION                           | string | collection to use                                                   |
| nsExclude                        | MONGOUNARCHIVE\_\_NS_EXCLUDE                           | string | exclude matching namespaces                                         |
| nsInclude                        | MONGOUNARCHIVE\_\_NS_INCLUDE                           | string | include matching namespaces                                         |
| nsFrom                           | MONGOUNARCHIVE\_\_NS_FROM                              | string | rename matching namespaces, must have matching nsTo                 |
| nsTo                             | MONGOUNARCHIVE\_\_NS_TO                                | string | rename matched namespaces, must have matching nsFrom                |
| host                             | MONGOUNARCHIVE\_\_HOST                                 | string | MongoDB host to connect to                                          |
| port                             | MONGOUNARCHIVE\_\_PORT                                 | string | MongoDB port                                                        |
| ssl                              | MONGOUNARCHIVE\_\_VERBOSE                              | bool   | connect to a mongod or mongos that has ssl enabled                  |
| sslCAFile                        | MONGOUNARCHIVE\_\_SSL_CA_FILE                          | string | the .pem file containing the root certificate chain                 |
| sslPEMKeyFile                    | MONGOUNARCHIVE\_\_SSL_PEM_KEY_FILE                     | string | the .pem file containing the certificate and key                    |
| sslPEMKeyPassword                | MONGOUNARCHIVE\_\_SSL_PEM_KEY_PASSWORD                 | string | the password to decrypt the sslPEMKeyFile, if necessary             |
| sslCRLFile                       | MONGOUNARCHIVE\_\_SSL_CRL_File                         | string | the .pem file containing the certificate revocation list            |
| sslAllowInvalidCertificates      | MONGOUNARCHIVE\_\_SSL_ALLOW_INVALID_CERTIFICATES       | bool   | bypass the validation for server certificates                       |
| sslAllowInvalidHostnames         | MONGOUNARCHIVE\_\_SSL_ALLOW_INVALID_HOSTNAMES          | bool   | bypass the validation for server name                               |
| sslFIPSMode                      | MONGOUNARCHIVE\_\_SSL_FIPS_MODE                        | bool   | use FIPS mode of the installed openssl library                      |
| username                         | MONGOARCHIVE\_\_USERNAME                               | string | username for authentication                                         |
| password                         | MONGOARCHIVE\_\_PASSWORD                               | string | password for authentication                                         |
| authenticationDatabase           | MONGOUNARCHIVE\_\_AUTHENTICATION_DATABASE              | string | database that holds the user's credentials                          |
| authenticationMechanism          | MONGOUNARCHIVE\_\_AUTHENTICATION_MECHANISM             | string | authentication mechanism to use                                     |
| gssapiServiceName                | MONGOUNARCHIVE\_\_GSSAPI_SERVICE_NAME                  | string | service name to use when authenticating using GSSAPI/Kerberos       |
| gssapiHostName                   | MONGOUNARCHIVE\_\_GSSAPI_HOST_NAME                     | string | hostname to use when authenticating using GSSAPI/Kerberos           |
| drop                             | MONGOUNARCHIVE\_\_DROP                                 | bool   | drop each collection before import                                  |
| dryRun                           | MONGOUNARCHIVE\_\_DRY_RUN                              | bool   | view summary without importing anything. recommended with verbosity |
| writeConcern                     | MONGOUNARCHIVE\_\_WRITE_CONCERN                        | string | write concern options                                               |
| noIndexRestore                   | MONGOUNARCHIVE\_\_NO_INDEX_RESTORE                     | bool   | don't restore indexes                                               |
| noOptionsRestore                 | MONGOUNARCHIVE\_\_NO_OPTIONS_RESTORE                   | bool   | don't restore collection options                                    |
| keepIndexVersion                 | MONGOUNARCHIVE\_\_KEEP_INDEX_VERSION                   | bool   | don't update index version                                          |
| maintainInsertionOrder           | MONGOUNARCHIVE\_\_MAINTAIN_INSERTION_ORDER             | bool   | restore the documents in the order of the input source              |
| numParallelCollections           | MONGOUNARCHIVE\_\_NUM_PARALLEL_COLLECTIONS             | string | number of collections to restore in parallel                        |
| numInsertionWorkersPerCollection | MONGOUNARCHIVE\_\_NUM_INSERTION_WORKERS_PER_COLLECTION | string | number of insert operations to run concurrently per collection      |
| stopOnError                      | MONGOUNARCHIVE\_\_STOP_ON_ERROR                        | string | halt after encountering any error during insertion                  |
| bypassDocumentValidation         | MONGOUNARCHIVE\_\_BYPASS_DOCUMENT_VALIDATION           | string | bypass document validation                                          |
| preserveUUID                     | MONGOUNARCHIVE\_\_PRESERVE_UUID                        | string | preserve original collection UUIDs                                  |
| verbose                          | MONGOUNARCHIVE\_\_VERBOSE                              | string | more detailed log output                                            |
| quiet                            | MONGOUNARCHIVE\_\_QUIET                                | bool   | hide all log output                                                 |
| azAccountName                    | MONGOUNARCHIVE\_\_AZ_ACCOUNT_NAME                      | string | Azure Blob Storage Account Name                                     |
| azAccountKey                     | MONGOUNARCHIVE\_\_AZ_ACCOUNT_KEY                       | string | Azure Blob Storage Account Key                                      |
| azContainerName                  | MONGOUNARCHIVE\_\_AZ_CONTAINER_NAME                    | string | Azure Blob Storage Container Name                                   |
| blobName                         | MONGOUNARCHIVE\_\_BLOB_NAME                            | bool   | Blob name of the archived file in the storage                       |
| dir                              | MONGOUNARCHIVE\_\_DIR                                  | bool   | directory name that contains the dumped files                       |
| updates                          | MONGOUNARCHIVE\_\_UPDATES                              | bool   | array of update specifications in JSON string                       |
| updatesFile                      | MONGOUNARCHIVE\_\_UPDATES_FILE                         | bool   | path to a file containing an array of update specifications         |
| keep                             | MONGOARCHIVE\_\_KEEP                                   | bool   | keep data dump                                                      |

## Examples

### Dump Database and Upload to Azure Storage

```sh
mongoarchive --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" --db=<dbname> --azAccountName=<az_account_name> --azAccountKey=<az_account_key> --azContainerName=<az_container_name>
```

This example demonstrates how to dump the data from a specified database and upload it to Azure storage. Replace <username>, <password>, <dbname>, <az_account_name>, <az_account_key>, and <az_container_name> with the appropriate values for your setup.

### Run Persistent Server for Regular Database Archival

```sh
mongoarchive --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" --db=<dbname> --azAccountName=<az_account_name> --azAccountKey=<az_account_key> --azContainerName=<az_container_name> --cron --cronExpression="* * * * *"
```

This example demonstrates how to run a persistent server that regularly archives a database. The server will execute the archival process based on the specified cron expression. Replace <username>, <password>, <dbname>, <az_account_name>, <az_account_key>, <az_container_name>, and <cron_expression> with your own values.

### Restore the Target Database from Azure Storage

```sh
mongounarchive --uri="mongodb://localhost:27017" --db=<dbname> --azAccountName=<az_account_name> --azAccountKey=<az_account_key> --azContainerName=<az_container_name>
```

This example shows how to restore the target database from Azure storage. Replace <dbname>, <az_account_name>, <az_account_key>, and <az_container_name> with your own values. The database will be restored to the MongoDB instance running on localhost:27017.

### Restore the Target Database from Azure Storage and Apply Changes

```sh
mongounarchive --uri="mongodb://localhost:27017" --db=<dbname> --azAccountName=<az_account_name> --azAccountKey=<az_account_key> --azContainerName=<az_container_name> --updatesFile=/home/mongotool/updates.json
```

This example demonstrates how to restore the target database from Azure storage and apply changes contained in an updates file. Replace <dbname>, <az_account_name>, <az_account_key>, <az_container_name>, and /home/mongotool/updates.json with your own values. The updates file should contain the necessary instructions to modify the restored database.

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
    ghcr.io/junminahn/mongo-tools-ext:latest \
    mongoarchive \
    --uri="mongodb://<username>:<password>@cluster0.mongodb.net/" \
    --db=<dbname> \
    --azAccountName=<az_account_name> \
    --azAccountKey=<az_account_key> \
    --azContainerName=<az_container_name> \
    --keep
```

## Backlog

1. Support for Other Major Cloud Storage Services
   - Add functionality to support AWS S3 storage as a target for database archival.
   - Implement integration with Google Cloud Storage to enable archiving databases to Google Cloud Storage.
   - Provide options and configuration parameters to specify the desired cloud storage service for database archival.
   - Ensure compatibility with the existing command structure and interface to maintain consistency for users.
