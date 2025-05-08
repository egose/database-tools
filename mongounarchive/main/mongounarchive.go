package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/egose/database-tools/common"
	"github.com/egose/database-tools/mongounarchive"
	"github.com/egose/database-tools/utils"

	mlog "github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/signals"
	"github.com/mongodb/mongo-tools/mongorestore"
)

var version string

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
	showVersion := mongounarchive.ParseFlags()
	if *showVersion {
		fmt.Println("mongo-unarchive version:", version)
		return
	}

	runTask()
}

func runTask() {
	restorePath := getRestorePath()

	storages := mongounarchive.GetStorages()
	if len(storages) == 0 {
		common.HandleErrorToPanic(fmt.Errorf("no storage backends configured"))
	}

	storage := storages[0]

	objectName, err := storage.GetTargetObjectName(mongounarchive.GetObjectName())
	common.HandleErrorToPanic(err)

	tarfilePath := path.Join(restorePath, objectName)
	destPath := path.Join(restorePath, utils.GetFileNameWithoutExtension(objectName))

	mlog.Logvf(mlog.Always, "Downloading archive...")
	err = storage.Download(objectName, tarfilePath)
	common.HandleErrorToPanic(err)

	mlog.Logvf(mlog.Always, "Extracting files...")
	err = utils.UnTar(tarfilePath, destPath)
	common.HandleErrorToPanic(err)

	options := mongounarchive.GetMongounarchiveOptions(destPath)
	opts, err := mongorestore.ParseOptions(options, "", "")
	common.HandleErrorToPanic(err)

	restore, err := mongorestore.New(opts)
	common.HandleErrorToPanic(err)

	defer restore.Close()

	finishedChan := signals.HandleWithInterrupt(restore.HandleInterrupt)
	defer close(finishedChan)

	mlog.Logvf(mlog.Always, "Restoring database...")
	result := restore.Restore()
	common.HandleErrorToPanic(result.Err)

	if restore.ToolOptions.WriteConcern.Acknowledged() {
		mlog.Logvf(mlog.Always, "%v document(s) restored successfully. %v document(s) failed to restore.", result.Successes, result.Failures)
	} else {
		mlog.Logvf(mlog.Always, "done")
	}

	if !mongounarchive.HasKeep() {
		err = utils.DeleteDirectory(destPath)
		common.HandleErrorToPanic(err)

		err = utils.DeleteFile(tarfilePath)
		common.HandleErrorToPanic(err)
	}

	if mongounarchive.HasUpdates() {
		mlog.Logvf(mlog.Always, "Applying updates...")
		applyUpdates()
	}

	mlog.Logvf(mlog.Always, "Unarchive completed successfully")
}

func getRestorePath() string {
	restorePath := os.Getenv(envPrefix + "RESTORE_PATH")
	if restorePath == "" {
		restorePath = "/tmp/datarestore"
	}
	return restorePath
}

func applyUpdates() error {
	client, dbClient, err := mongounarchive.GetMongoClient()
	common.HandleErrorToPanic(err)

	defer func() {
		err = client.Disconnect(context.Background())
		common.HandleErrorToPanic(err)
	}()

	updates := []update{}
	bytes, err := mongounarchive.GetUpdates()
	common.HandleErrorToPanic(err)

	err = json.Unmarshal(bytes, &updates)
	common.HandleErrorToPanic(err)

	for i, u := range updates {
		coll := dbClient.Collection(u.Collection)
		result, err := coll.UpdateMany(context.Background(), u.Filter, u.Update)
		common.HandleErrorToPanic(err)

		mlog.Logvf(mlog.Always, "Update[%d]: matched count: %d", i, result.MatchedCount)
		mlog.Logvf(mlog.Always, "Update[%d]: modified count: %d", i, result.ModifiedCount)
	}

	return nil
}
