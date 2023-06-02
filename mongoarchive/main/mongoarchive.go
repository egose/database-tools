package main

import (
	"bytes"
	"context"
	"os"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/junminahn/mongo-tools-ext/common"
	"github.com/junminahn/mongo-tools-ext/mongoarchive"
	"github.com/junminahn/mongo-tools-ext/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/progress"
	"github.com/mongodb/mongo-tools/common/signals"
	"github.com/mongodb/mongo-tools/mongodump"
)

const (
	progressBarLength   = 24
	progressBarWaitTime = time.Second * 3
	envPrefix           = "MONGOARCHIVE__"
)

func main() {
	mongoarchive.ParseFlags()

	if mongoarchive.HasCron() {
		runCronJob()
	} else {
		err := runTask()
		common.HandleError(err)
	}
}

func runCronJob() {
	cron := mongoarchive.GetCronScheduler()

	cron.Do(func() {
		err := runTask()
		if err != nil {
			mlog.Logvf(mlog.Always, "Failed: %v", err.Error())
		}
	})
	cron.StartBlocking()
}

func runTask() error {
	var dumpPath string
	if dumpPath = os.Getenv(envPrefix + "DUMP_PATH"); dumpPath == "" {
		dumpPath = "/tmp/datadump"
	}

	filename, uname := utils.GetNewFilename()
	destPath := path.Join(dumpPath, uname)
	tarfilePath := path.Join(dumpPath, filename)

	options := mongoarchive.GetMongodumpOptions()
	options = append(options, "--out="+destPath)

	opts, err := mongodump.ParseOptions(options, "", "")
	if err != nil {
		return err
	}

	mlog.SetVerbosity(opts.Verbosity)
	opts.URI.LogUnsupportedOptions()

	progressManager := progress.NewBarWriter(mlog.Writer(0), progressBarWaitTime, progressBarLength, false)
	progressManager.Start()
	defer progressManager.Stop()

	dump := mongodump.MongoDump{
		ToolOptions:     opts.ToolOptions,
		InputOptions:    opts.InputOptions,
		OutputOptions:   opts.OutputOptions,
		ProgressManager: progressManager,
	}

	finishedChan := signals.HandleWithInterrupt(dump.HandleInterrupt)
	defer close(finishedChan)

	err = dump.Init()
	if err != nil {
		return err
	}

	err = dump.Dump()
	if err != nil {
		return err
	}

	err = utils.Tar(destPath, tarfilePath)
	if err != nil {
		return err
	}

	containerClient, err := mongoarchive.GetAzBlobContainerClient()
	if err != nil {
		return err
	}

	buffer, err := utils.ReadFileToBuffer(tarfilePath)
	if err != nil {
		return err
	}

	blockBlobClient := containerClient.NewBlockBlobClient(filename)
	blockBlobUploadOptions := blockblob.UploadOptions{
		// Metadata: map[string]string{"meta": "value"},
		// Tags:     map[string]string{"tag": "value"},
	}
	uploadResp, err := blockBlobClient.Upload(context.Background(), streaming.NopCloser(bytes.NewReader(buffer)), &blockBlobUploadOptions)
	if err != nil {
		return err
	}

	if mongoarchive.HasKeep() != true {
		err = utils.DeleteDirectory(destPath)
		if err != nil {
			return err
		}

		err = utils.DeleteFile(tarfilePath)
		if err != nil {
			return err
		}
	}

	mlog.Logvf(mlog.Always, "Archive completed successfully; ETag: %v", uploadResp.ETag)

	return nil
}
