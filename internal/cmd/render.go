package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/render"
	"github.com/schmitthub/openclaw-docker/internal/versions"
)

func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render Dockerfiles from versions manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := mergedOptions(cmd)
			if err != nil {
				return err
			}

			manifest, err := versions.ReadManifest(opts.VersionsFile)
			if err != nil {
				return err
			}

			if err := render.Generate(render.Options{
				Manifest:     manifest,
				OutputDir:    opts.OutputDir,
				TemplatesDir: opts.TemplatesDir,
				Cleanup:      opts.Cleanup,
				Requested:    opts.Versions,
				ConfirmWrite: func(path string) error {
					return confirmWrite(cmd, opts.DangerousInline, path)
				},
			}); err != nil {
				return err
			}

			fmt.Printf("Rendered Dockerfiles to %s\n", opts.OutputDir)
			return nil
		},
	}

	return cmd
}
