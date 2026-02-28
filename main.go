package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/schmitthub/openclaw-docker/internal/build"
	"github.com/schmitthub/openclaw-docker/internal/cmd"
	"github.com/schmitthub/openclaw-docker/internal/update"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(exitCode(err))
	}
}

func run() error {
	buildDate := build.Date
	buildVersion := build.Version

	rootCmd := cmd.NewRootCmd(buildVersion, buildDate)
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	_, err := rootCmd.ExecuteC()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	statePath, stateErr := update.DefaultStatePath()
	if stateErr != nil {
		return fmt.Errorf("resolve update state path: %w", stateErr)
	}

	result, checkErr := update.CheckForUpdate(ctx, statePath, buildVersion, "schmitthub/openclaw-docker")
	if checkErr != nil {
		return fmt.Errorf("check for update: %w", checkErr)
	}
	if result == nil || !result.UpdateAvailable {
		return nil
	}

	if _, err := fmt.Fprintf(
		rootCmd.OutOrStdout(),
		"\nUpdate available: %s -> %s\nInstall with:\n  brew upgrade openclaw-docker\n  curl -fsSL https://raw.githubusercontent.com/schmitthub/openclaw-docker/main/scripts/install.sh | bash\n\n",
		result.CurrentVersion,
		result.LatestVersion,
	); err != nil {
		return fmt.Errorf("print update message: %w", err)
	}

	return nil
}

type exitCoder interface {
	ExitCode() int
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var coded exitCoder
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}

	return 1
}
