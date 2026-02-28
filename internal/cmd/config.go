package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Config helpers",
	}

	cmd.AddCommand(newConfigInitCmd())

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write an annotated config template to disk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			target := strings.TrimSpace(filePath)
			if target == "" {
				return fmt.Errorf("config template path cannot be empty")
			}

			opts, err := mergedOptions(cmd)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create config directory: %w", err)
			}

			if err := confirmWrite(cmd, opts.DangerousInline, target); err != nil {
				return err
			}

			if err := os.WriteFile(target, []byte(config.DefaultTemplate()), 0o644); err != nil {
				return fmt.Errorf("write config template: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Wrote config template: %s\n", target)
			return nil
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "./openclaw-docker.yaml", "Path to write config template")

	return cmd
}
