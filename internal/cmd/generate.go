package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/render"
	"github.com/schmitthub/openclaw-docker/internal/versions"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Resolve version and generate Dockerfile",
		RunE:  runGenerate,
	}

	return cmd
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	opts, err := mergedOptions(cmd)
	if err != nil {
		return err
	}

	var meta versions.ReleaseMeta

	// If a versions file already exists and --openclaw-version was not
	// explicitly requested, reuse the cached manifest instead of resolving
	// from npm.
	if !cmd.Flags().Changed("openclaw-version") {
		if _, err := os.Stat(opts.VersionsFile); err == nil {
			meta, err = versions.ReadManifest(opts.VersionsFile)
			if err != nil {
				return err
			}
		}
	}

	// Resolve from npm when we don't yet have metadata.
	if meta.FullVersion == "" {
		meta, err = versions.Resolve(context.Background(), versions.ResolveOptions{
			Requested: opts.Version,
			Debug:     opts.Debug,
		})
		if err != nil {
			return err
		}

		if err := versions.WriteManifest(opts.VersionsFile, meta); err != nil {
			return err
		}
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
		SquidAllowedDomains:  opts.SquidAllowedDomains,
		ConfirmWrite: func(path string) error {
			return confirmWrite(cmd, opts.DangerousInline, path)
		},
	}); err != nil {
		return err
	}

	fmt.Printf("Generated deployment artifacts in %s\n", opts.OutputDir)
	return nil
}
