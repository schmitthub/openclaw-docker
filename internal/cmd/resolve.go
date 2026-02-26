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
		Short: "Resolve OpenClaw versions and write versions manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := mergedOptions(cmd)
			if err != nil {
				return err
			}

			manifest, err := versions.Resolve(context.Background(), versions.ResolveOptions{
				PackageName:   opts.PackageName,
				Requested:     opts.Versions,
				DebianDefault: opts.DebianDefault,
				AlpineDefault: opts.AlpineDefault,
				Variants:      opts.Variants,
				Arches:        opts.Arches,
				Debug:         opts.Debug,
			})
			if err != nil {
				return err
			}

			if err := versions.WriteManifest(opts.VersionsFile, manifest); err != nil {
				return err
			}

			fmt.Printf("Wrote versions manifest: %s\n", opts.VersionsFile)
			return nil
		},
	}

	return cmd
}
