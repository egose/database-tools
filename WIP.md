Here’s an improved version of your documentation with clearer structure, improved readability, and some consistency fixes in terminology and formatting:

---

# Extra MongoDB Tools

This repository provides supplementary tools for MongoDB, supporting both backup and restoration workflows:

* **`mongo-archive`** – Dumps MongoDB data to disk and uploads it to supported cloud storage services.
* **`mongo-unarchive`** – Downloads archived dumps from cloud storage and restores them into a live MongoDB database.

---

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

---

## ⚙️ Configuration: CLI Flags & Environment Variables

Both `mongo-archive` and `mongo-unarchive` follow the conventions of MongoDB’s native tools (e.g., `mongodump`, `mongorestore`), using similar command-line arguments. Configuration values can also be passed via environment variables for convenience or container-based execution.

---

## 📦 `mongo-archive`

### Functionality

* Dumps MongoDB data locally.
* Uploads the dump to cloud storage (Azure Blob, AWS S3, or Google Cloud Storage).
* Can be run once or as a cron-scheduled job.

### Configuration

<details>
<summary>Click to expand full table of options</summary>

<!-- Retain the original table without changes here for brevity. Let me know if you'd like me to refine the table formatting too. -->

</details>

---

## 🔄 `mongo-unarchive`

### Functionality

* Downloads archived MongoDB dumps from supported cloud storage.
* Restores the data to a MongoDB database.
* Supports applying update operations post-restore using a JSON configuration.

### Configuration

<details>
<summary>Click to expand full table of options</summary>

<!-- Retain the original table without changes here for brevity. Let me know if you'd like me to refine the table formatting too. -->

</details>

---

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

---

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

---

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
              image: ghcr.io/egose/database-tools:0.2.6
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
```

```yaml
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

---

## 🗂️ Backlog

> *To be documented.*

---

Let me know if you'd like help updating the parameter tables with improved layout or validation descriptions, or if you'd prefer this broken up into separate Markdown files!
