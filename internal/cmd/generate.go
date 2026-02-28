package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/render"
	"github.com/schmitthub/openclaw-docker/internal/versions"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Resolve version and generate deployment artifacts",
		RunE:  runGenerate,
	}

	cmd.Flags().StringVarP(&rootOpts.OutputDir, "output", "o", "", "Dockerfile output directory (defaults to ./openclaw-deploy)")
	cmd.Flags().StringVar(&rootOpts.Version, "openclaw-version", "", "Requested OpenClaw version/tag (dist-tag like 'latest' or semver partial like '2026.2')")
	cmd.Flags().BoolVar(&rootOpts.Cleanup, "cleanup", false, "Show defensive cleanup warning (deletes are disabled; generation is overwrite-only)")
	cmd.Flags().StringVar(&rootOpts.DockerAptPackages, "docker-apt-packages", "", "Additional apt packages to install in generated Dockerfile")
	cmd.Flags().StringVar(&rootOpts.OpenClawConfigDir, "openclaw-config-dir", "", "Default OPENCLAW_CONFIG_DIR value baked into generated Dockerfile")
	cmd.Flags().StringVar(&rootOpts.OpenClawWorkspaceDir, "openclaw-workspace-dir", "", "Default OPENCLAW_WORKSPACE_DIR value baked into generated Dockerfile")
	cmd.Flags().StringVar(&rootOpts.OpenClawGatewayPort, "openclaw-gateway-port", "", "Default OPENCLAW_GATEWAY_PORT value baked into generated Dockerfile")
	cmd.Flags().StringVar(&rootOpts.OpenClawBridgePort, "openclaw-bridge-port", "", "Default OPENCLAW_BRIDGE_PORT value baked into generated Dockerfile")
	cmd.Flags().StringVar(&rootOpts.OpenClawGatewayBind, "openclaw-gateway-bind", "", "Default OPENCLAW_GATEWAY_BIND value baked into generated Dockerfile")
	cmd.Flags().StringVar(&rootOpts.OpenClawImage, "openclaw-image", "", "Default OPENCLAW_IMAGE value used in generated compose/.env.openclaw")
	cmd.Flags().StringVar(&rootOpts.OpenClawGatewayToken, "openclaw-gateway-token", "", "Default OPENCLAW_GATEWAY_TOKEN value used in generated compose/.env.openclaw")
	cmd.Flags().StringVar(&rootOpts.AllowedDomains, "allowed-domains", "", "Comma-separated domains to whitelist in egress proxy")
	cmd.Flags().StringVar(&rootOpts.ExternalOrigin, "external-origin", "", "External origin for server deployments (e.g. https://myclaw.example.com)")

	return cmd
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	opts, err := mergedOptions(cmd)
	if err != nil {
		return err
	}

	var meta versions.ReleaseMeta

	// OPENCLAW_DOCKER_VERSIONS_FILE allows test environments to provide a
	// pre-resolved manifest, bypassing npm resolution entirely.
	if envFile, ok := os.LookupEnv("OPENCLAW_DOCKER_VERSIONS_FILE"); ok {
		meta, err = versions.ReadManifest(envFile)
		if err != nil {
			return err
		}
	} else {
		meta, err = versions.Resolve(context.Background(), versions.ResolveOptions{
			Requested: opts.Version,
			Debug:     opts.Debug,
		})
		if err != nil {
			return err
		}
	}

	// Write manifest to output dir so subsequent runs can detect what
	// version is already rendered there.
	manifestPath := filepath.Join(opts.OutputDir, "manifest.json")
	if err := versions.WriteManifest(manifestPath, meta); err != nil {
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
		AllowedDomains:       opts.AllowedDomains,
		ExternalOrigin:       opts.ExternalOrigin,
		ConfirmWrite: func(path string) error {
			return confirmWrite(cmd, opts.DangerousInline, path)
		},
	}); err != nil {
		return err
	}

	fmt.Printf("Generated deployment artifacts in %s\n", opts.OutputDir)
	return nil
}
