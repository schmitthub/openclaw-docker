package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newVersionCmd(version, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show build version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), formatVersion(version, buildDate))
		},
	}

	return cmd
}

func formatVersion(version, buildDate string) string {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		version = "DEV"
	}

	if strings.TrimSpace(buildDate) != "" {
		return fmt.Sprintf("openclaw-docker version %s (%s)\n", version, strings.TrimSpace(buildDate))
	}

	return fmt.Sprintf("openclaw-docker version %s\n", version)
}
