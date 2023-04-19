// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.elastic.co/ecszap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"

	libconf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/file"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/configure"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/filelock"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/paths"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/upgrade"
	"github.com/elastic/elastic-agent/internal/pkg/agent/configuration"
	"github.com/elastic/elastic-agent/internal/pkg/agent/errors"
	"github.com/elastic/elastic-agent/internal/pkg/cli"
	"github.com/elastic/elastic-agent/internal/pkg/config"
	"github.com/elastic/elastic-agent/internal/pkg/release"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

const (
	// period during which we monitor for failures resulting in a rollback
	gracePeriodDuration = 10 * time.Minute

	watcherName     = "elastic-agent-watcher"
	watcherLockFile = "watcher.lock"
)

func newWatchCommandWithArgs(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch the Elastic Agent for failures and initiate rollback",
		Long:  `This command watches Elastic Agent for failures and initiates rollback if necessary.`,
		Run: func(_ *cobra.Command, _ []string) {
			log, err := configuredLogger()
			if err != nil {
				fmt.Fprintf(streams.Err, "Error configuring logger: %v\n%s\n", err, troubleshootMessage())
			}
			if err := watchCmd(log); err != nil {
				log.Errorw("Watch command failed", "error.message", err)
				fmt.Fprintf(streams.Err, "Watch command failed: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	return cmd
}

func watchCmd(log *logp.Logger) error {
	log.Info("hey look here")
	marker, err := upgrade.LoadMarker()
	if err != nil {
		log.Error("failed to load marker", err)
		return err
	}
	if marker == nil {
		// no marker found we're not in upgrade process
		log.Debugf("update marker not present at '%s'", paths.Data())
		return nil
	}

	locker := filelock.NewAppLocker(paths.Top(), watcherLockFile)
	if err := locker.TryLock(); err != nil {
		if errors.Is(err, filelock.ErrAppAlreadyRunning) {
			log.Debugf("exiting, lock already exists")
			return nil
		}

		log.Error("failed to acquire lock", err)
		return err
	}
	defer func() {
		_ = locker.Unlock()
	}()

	isWithinGrace, tilGrace := gracePeriod(marker)
	if !isWithinGrace {
		log.Debugf("not within grace [updatedOn %v] %v", marker.UpdatedOn.String(), time.Since(marker.UpdatedOn).String())
		// if it is started outside of upgrade loop
		// if we're not within grace and marker is still there it might mean
		// that cleanup was not performed ok, cleanup everything except current version
		// hash is the same as hash of agent which initiated watcher.
		if err := upgrade.Cleanup(log, release.ShortCommit(), true, false); err != nil {
			log.Error("rollback failed", err)
		}
		// exit nicely
		return nil
	}

	ctx := context.Background()
	if err := watch(ctx, tilGrace, log); err != nil {
		log.Error("Error detected proceeding to rollback: %v", err)
		err = upgrade.Rollback(ctx, log, marker.PrevHash, marker.Hash)
		if err != nil {
			log.Error("rollback failed", err)
		}
		return err
	}

	// cleanup older versions,
	// in windows it might leave self untouched, this will get cleaned up
	// later at the start, because for windows we leave marker untouched.
	removeMarker := !isWindows()
	err = upgrade.Cleanup(log, marker.Hash, removeMarker, false)
	if err != nil {
		log.Error("rollback failed", err)
	}
	return err
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func watch(ctx context.Context, tilGrace time.Duration, log *logger.Logger) error {
	errChan := make(chan error)
	crashChan := make(chan error)

	ctx, cancel := context.WithCancel(ctx)

	//cleanup
	defer func() {
		cancel()
		close(errChan)
		close(crashChan)
	}()

	errorChecker, err := upgrade.NewErrorChecker(errChan, log)
	if err != nil {
		return err
	}

	crashChecker, err := upgrade.NewCrashChecker(ctx, crashChan, log)
	if err != nil {
		return err
	}

	go errorChecker.Run(ctx)
	go crashChecker.Run(ctx)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	t := time.NewTimer(tilGrace)
	defer t.Stop()

WATCHLOOP:
	for {
		select {
		case <-signals:
			// ignore
			continue
		case <-ctx.Done():
			break WATCHLOOP
		// grace period passed, agent is considered stable
		case <-t.C:
			log.Info("Grace period passed, not watching")
			break WATCHLOOP
		// Agent in degraded state.
		case err := <-errChan:
			log.Error("Agent Error detected", err)
			return err
		// Agent keeps crashing unexpectedly
		case err := <-crashChan:
			log.Error("Agent crash detected", err)
			return err
		}
	}

	return nil
}

// gracePeriod returns true if it is within grace period and time until grace period ends.
// otherwise it returns false and 0
func gracePeriod(marker *upgrade.UpdateMarker) (bool, time.Duration) {
	sinceUpdate := time.Since(marker.UpdatedOn)

	if 0 < sinceUpdate && sinceUpdate < gracePeriodDuration {
		return true, gracePeriodDuration - sinceUpdate
	}

	return false, gracePeriodDuration
}

func configuredLogger() (*logger.Logger, error) {
	pathConfigFile := paths.ConfigFile()
	rawConfig, err := config.LoadFile(pathConfigFile)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not read configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not parse configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	internal, err := makeInternalFileOutput()
	if err != nil {
		return nil, err
	}

	libC, err := toCommonConfig(cfg.Settings.LoggingConfig)
	if err != nil {
		return nil, err
	}

	if err := configure.LoggingWithOutputs("", libC, internal); err != nil {
		return nil, fmt.Errorf("error initializing logging: %w", err)
	}
	return logp.NewLogger(""), nil
}

func toCommonConfig(cfg *logger.Config) (*libconf.C, error) {
	// work around custom types and common config
	// when custom type is transformed to common.Config
	// value is determined based on reflect value which is incorrect
	// enum vs human readable form
	yamlCfg, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	commonLogp, err := libconf.NewConfigFrom(string(yamlCfg))
	if err != nil {
		return nil, errors.New(err, errors.TypeConfig)
	}

	return commonLogp, nil
}

func makeInternalFileOutput() (zapcore.Core, error) {
	// defaultCfg is used to set the defaults for the file rotation of the internal logging
	// these settings cannot be changed by a user configuration
	defaultCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	filename := filepath.Join(paths.Home(), logger.DefaultLogDirectory, watcherName)
	rotator, err := file.NewFileRotator(filename,
		file.MaxSizeBytes(defaultCfg.Files.MaxSize),
		file.MaxBackups(defaultCfg.Files.MaxBackups),
		file.Permissions(os.FileMode(defaultCfg.Files.Permissions)),
		file.Interval(defaultCfg.Files.Interval),
		file.RotateOnStartup(defaultCfg.Files.RotateOnStartup),
		file.RedirectStderr(defaultCfg.Files.RedirectStderr),
	)
	if err != nil {
		return nil, errors.New("failed to create internal file rotator")
	}

	encoderConfig := ecszap.ECSCompatibleEncoderConfig(logp.JSONEncoderConfig())
	encoderConfig.EncodeTime = logger.UtcTimestampEncode
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	return ecszap.WrapCore(zapcore.NewCore(encoder, rotator, zapcore.DebugLevel)), nil
}
