package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	envPrefix           = "MONGOUNARCHIVE__"
)

type update struct {
	Collection string      `json:"collection"`
	Filter     interface{} `json:"filter"`
	Update     interface{} `json:"update"`
}

func main() {
	cfg, showVersion := mongounarchive.ParseFlags()
	if showVersion {
		fmt.Println("mongo-unarchive version:", version)
		return
	}

	if err := runTask(cfg); err != nil {
		mlog.Logvf(mlog.Always, "Failed: %v", err)
		os.Exit(1)
	}
}

func runTask(cfg *mongounarchive.Config) error {
	restorePath := getRestorePath()

	storages := cfg.GetStorages()
	if len(storages) == 0 {
		return fmt.Errorf("no storage backends configured")
	}

	storage := storages[0]

	objectName, err := storage.GetTargetObjectName(cfg.GetObjectName())
	if err != nil {
		return err
	}

	tarfilePath, err := utils.ResolvePathWithinRoot(restorePath, objectName)
	if err != nil {
		return err
	}

	destPath := filepath.Join(restorePath, utils.GetFileNameWithoutExtension(objectName))

	mlog.Logvf(mlog.Always, "Downloading archive...")
	err = storage.Download(objectName, tarfilePath)
	if err != nil {
		return err
	}

	mlog.Logvf(mlog.Always, "Extracting files...")
	err = utils.UnTar(tarfilePath, destPath)
	if err != nil {
		return err
	}

	options := cfg.GetMongounarchiveOptions(destPath)
	opts, err := mongorestore.ParseOptions(options, "", "")
	if err != nil {
		return err
	}

	restore, err := mongorestore.New(opts)
	if err != nil {
		return err
	}

	defer restore.Close()

	finishedChan := signals.HandleWithInterrupt(restore.HandleInterrupt)
	defer close(finishedChan)

	mlog.Logvf(mlog.Always, "Restoring database...")
	result := restore.Restore()
	if result.Err != nil {
		return result.Err
	}

	if restore.ToolOptions.WriteConcern.Acknowledged() {
		mlog.Logvf(mlog.Always, "%v document(s) restored successfully. %v document(s) failed to restore.", result.Successes, result.Failures)
	} else {
		mlog.Logvf(mlog.Always, "done")
	}

	if !cfg.HasKeep() {
		err = utils.DeleteDirectory(destPath)
		if err != nil {
			return err
		}

		err = utils.DeleteFile(tarfilePath)
		if err != nil {
			return err
		}
	}

	if cfg.HasUpdates() {
		mlog.Logvf(mlog.Always, "Applying updates...")
		if err := applyUpdates(cfg); err != nil {
			return err
		}
	}

	mlog.Logvf(mlog.Always, "Unarchive completed successfully")
	return nil
}

func getRestorePath() string {
	restorePath := os.Getenv(envPrefix + "RESTORE_PATH")
	if restorePath == "" {
		restorePath = "/tmp/datarestore"
	}
	return restorePath
}

func applyUpdates(cfg *mongounarchive.Config) error {
	client, dbClient, err := cfg.GetMongoClient()
	if err != nil {
		return err
	}

	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	updates := []update{}
	bytes, err := cfg.GetUpdates()
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &updates)
	if err != nil {
		return err
	}

	for i, u := range updates {
		coll := dbClient.Collection(u.Collection)
		result, err := coll.UpdateMany(context.Background(), u.Filter, u.Update)
		if err != nil {
			return err
		}

		mlog.Logvf(mlog.Always, "Update[%d]: matched count: %d", i, result.MatchedCount)
		mlog.Logvf(mlog.Always, "Update[%d]: modified count: %d", i, result.ModifiedCount)
	}

	return nil
}
