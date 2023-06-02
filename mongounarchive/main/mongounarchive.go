package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"

	"github.com/junminahn/mongo-tools-ext/common"
	"github.com/junminahn/mongo-tools-ext/mongounarchive"
	"github.com/junminahn/mongo-tools-ext/utils"

	mlog "github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/signals"
	"github.com/mongodb/mongo-tools/mongorestore"
)

const (
	progressBarLength   = 24
	progressBarWaitTime = time.Second * 3
	envPrefix           = "MONGOARCHIVE__"
)

type update struct {
	Collection string      `json:"collection"`
	Filter     interface{} `json:"filter"`
	Update     interface{} `json:"update"`
}

func main() {
	mongounarchive.ParseFlags()

	runTask()
}

func runTask() {
	var restorePath string
	if restorePath = os.Getenv(envPrefix + "RESTORE_PATH"); restorePath == "" {
		restorePath = "/tmp/datarestore"
	}

	containerClient, err := mongounarchive.GetAzBlobContainerClient()
	common.HandleError(err)

	blobName, err := mongounarchive.GetTargetAzBlobName(containerClient)
	common.HandleError(err)

	tarfilePath := path.Join(restorePath, blobName)
	destPath := path.Join(restorePath, utils.GetFileNameWithoutExtension(blobName))

	blockBlobClient := mongounarchive.GetAzBlockBlobClient(containerClient, blobName)

	destFile, err := utils.CreateFile(tarfilePath)
	common.HandleError(err)

	_, err = blockBlobClient.DownloadFile(context.Background(),
		destFile,
		&blob.DownloadFileOptions{
			// If Progress is non-nil, this function is called periodically as bytes are uploaded.
			Progress: func(bytesTransferred int64) {
				fmt.Printf("Downloaded %d.\n", bytesTransferred)
			}})

	destFile.Close()
	common.HandleError(err)

	err = utils.UnTar(tarfilePath, destPath)
	common.HandleError(err)

	options := mongounarchive.GetMongounarchiveOptions(destPath)
	opts, err := mongorestore.ParseOptions(options, "", "")
	common.HandleError(err)

	restore, err := mongorestore.New(opts)
	common.HandleError(err)

	defer restore.Close()

	finishedChan := signals.HandleWithInterrupt(restore.HandleInterrupt)
	defer close(finishedChan)

	result := restore.Restore()
	common.HandleError(result.Err)

	if restore.ToolOptions.WriteConcern.Acknowledged() {
		mlog.Logvf(mlog.Always, "%v document(s) restored successfully. %v document(s) failed to restore.", result.Successes, result.Failures)
	} else {
		mlog.Logvf(mlog.Always, "done")
	}

	if mongounarchive.HasKeep() != true {
		err = utils.DeleteDirectory(destPath)
		common.HandleError(result.Err)

		err = utils.DeleteFile(tarfilePath)
		common.HandleError(result.Err)
	}

	// updates
	if mongounarchive.HasUpdates() == true {
		client, dbClient, err := mongounarchive.GetMongoClient()
		common.HandleError(err)

		defer func() {
			err = client.Disconnect(context.Background())
			common.HandleError(err)
		}()

		updates := []update{}
		bytes, err := mongounarchive.GetUpdates()
		common.HandleError(err)

		err = json.Unmarshal(bytes, &updates)
		common.HandleError(err)

		for i, u := range updates {
			coll := dbClient.Collection(u.Collection)
			result, err := coll.UpdateMany(context.Background(), u.Filter, u.Update)
			common.HandleError(err)

			mlog.Logvf(mlog.Always, "Update[%d]: matched count: %d", i, result.MatchedCount)
			mlog.Logvf(mlog.Always, "Update[%d]: modified count: %d", i, result.ModifiedCount)
		}

	}

	mlog.Logvf(mlog.Always, "Unarchive completed successfully")
}
