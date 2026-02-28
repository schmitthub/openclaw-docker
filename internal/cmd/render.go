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
		Short: "Render Dockerfile from versions manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := mergedOptions(cmd)
			if err != nil {
				return err
			}

			meta, err := versions.ReadManifest(opts.VersionsFile)
			if err != nil {
				return err
			}

			if err := render.Generate(render.Options{
				Meta:                 meta,
				OutputDir:            opts.OutputDir,
				Cleanup:              opts.Cleanup,
				DockerAptPackages:    opts.DockerAptPackages,
				OpenClawConfigDir:    opts.OpenClawConfigDir,
				OpenClawWorkspaceDir: opts.OpenClawWorkspaceDir,
				OpenClawGatewayPort:  opts.OpenClawGatewayPort,
				OpenClawBridgePort:   opts.OpenClawBridgePort,
				OpenClawGatewayBind:  opts.OpenClawGatewayBind,
				OpenClawImage:        opts.OpenClawImage,
				OpenClawGatewayToken: opts.OpenClawGatewayToken,
				OpenClawExtraMounts:  opts.OpenClawExtraMounts,
				OpenClawHomeVolume:   opts.OpenClawHomeVolume,
				ConfirmWrite: func(path string) error {
					return confirmWrite(cmd, opts.DangerousInline, path)
				},
			}); err != nil {
				return err
			}

			fmt.Printf("Rendered deployment artifacts to %s\n", opts.OutputDir)
			return nil
		},
	}

	return cmd
}
