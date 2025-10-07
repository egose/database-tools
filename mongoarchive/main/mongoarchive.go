package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/egose/database-tools/common"
	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/utils"
	"github.com/go-co-op/gocron/v2"
	mlog "github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/progress"
	"github.com/mongodb/mongo-tools/common/signals"
	"github.com/mongodb/mongo-tools/mongodump"
)

var version string

const (
	progressBarLength   = 24
	progressBarWaitTime = time.Second * 3
	envPrefix           = "MONGOARCHIVE__"
)

func main() {
	showVersion := mongoarchive.ParseFlags()
	if showVersion {
		fmt.Println("mongo-archive version:", version)
		return
	}

	if mongoarchive.HasCron() {
		runCronJob()
	} else {
		err := runTask()
		if err != nil {
			sendNotification(false, err.Error())
			mlog.Logvf(mlog.Always, "Failed: %v", err.Error())
		}

		common.HandleErrorToPanic(err)
	}
}

// See https://github.com/go-co-op/gocron
func runCronJob() {
	loc := mongoarchive.GetLocation()
	if loc == nil {
		mlog.Logvf(mlog.Always, "Failed: invalid timezone location")
		return
	}

	exp := mongoarchive.GetCronExpression()
	if exp == "" {
		mlog.Logvf(mlog.Always, "Failed: empty cron expression")
		return
	}

	mlog.Logvf(mlog.Always, "Using Cron Expression: %v", exp)

	s, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		mlog.Logvf(mlog.Always, "Failed to create scheduler: %v", err)
		return
	}
	defer s.Shutdown() // Ensure cleanup even if Start() panics

	_, err = s.NewJob(
		gocron.CronJob(exp, false),
		gocron.NewTask(func() {
			startTime := time.Now()
			mlog.Logvf(mlog.Always, "Task started at: %v", startTime)

			// Run the actual task
			if err := runTask(); err != nil {
				mlog.Logvf(mlog.Always, "Task failed: %v", err)
				sendNotification(false, err.Error())
			} else {
				mlog.Logvf(mlog.Always, "Task completed successfully at: %v (Duration: %v)", time.Now(), time.Since(startTime))
			}
		}),
	)

	if err != nil {
		mlog.Logvf(mlog.Always, "Failed to schedule job: %v", err)
		return
	}

	s.Start()
	mlog.Logvf(mlog.Always, "Scheduler started.")

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan // Block until a signal is received
	mlog.Logvf(mlog.Always, "Shutting down scheduler...")
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

	storages := mongoarchive.GetStorages()
	if len(storages) == 0 {
		return fmt.Errorf("no storage backends configured")
	}

	if err := dump.Init(); err != nil {
		return err
	}

	if err := dump.Dump(); err != nil {
		return err
	}

	if err := utils.Tar(destPath, tarfilePath); err != nil {
		return err
	}

	buffer, err := utils.ReadFileToBuffer(tarfilePath)
	if err != nil {
		return err
	}

	for _, s := range storages {
		err := s.DeleteOldObjects()
		if err != nil {
			return fmt.Errorf("failed to delete old objects in %T: %w", s, err)
		}

		result, err := s.Upload(filename, buffer)
		if err != nil {
			return fmt.Errorf("failed to upload to %T: %w", s, err)
		}
		mlog.Logvf(mlog.Always, "Successfully uploaded backup to %T: %v", s, result)
	}

	if !mongoarchive.HasKeep() {
		if err := utils.DeleteDirectory(destPath); err != nil {
			return err
		}

		if err := utils.DeleteFile(tarfilePath); err != nil {
			return err
		}
	}

	sendNotification(true, filename)

	return nil
}

func sendNotification(success bool, filenameOrError string) {
	notifications := mongoarchive.GetNotifications()
	for _, notification := range notifications {
		notification.Send(success, mongoarchive.GetTZ(), filenameOrError)
	}
}
