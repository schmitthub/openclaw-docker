package main

import (
	"os"

	"github.com/schmitthub/openclaw-docker/internal/build"
	"github.com/schmitthub/openclaw-docker/internal/cmd"
)

func main() {
	if err := cmd.Execute(build.Version, build.Date); err != nil {
		os.Exit(1)
	}
}
