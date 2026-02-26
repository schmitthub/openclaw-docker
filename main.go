package main

import (
	"os"

	"github.com/schmitthub/openclaw-docker/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
