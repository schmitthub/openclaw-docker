package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/versions"
)

func newResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve OpenClaw version and write manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := mergedOptions(cmd)
			if err != nil {
				return err
			}

			meta, err := versions.Resolve(context.Background(), versions.ResolveOptions{
				Requested: opts.Version,
				Debug:     opts.Debug,
			})
			if err != nil {
				return err
			}

			if err := confirmWrite(cmd, opts.DangerousInline, opts.VersionsFile); err != nil {
				return err
			}

			if err := versions.WriteManifest(opts.VersionsFile, meta); err != nil {
				return err
			}

			fmt.Printf("Resolved %s → %s\nWrote manifest: %s\n", opts.Version, meta.FullVersion, opts.VersionsFile)
			return nil
		},
	}

	return cmd
}
