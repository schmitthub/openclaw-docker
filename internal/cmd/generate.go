package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/render"
	"github.com/schmitthub/openclaw-docker/internal/versions"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Resolve versions and generate Dockerfiles",
		RunE:  runGenerate,
	}

	return cmd
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	opts, err := mergedOptions(cmd)
	if err != nil {
		return err
	}

	manifest, err := versions.Resolve(context.Background(), versions.ResolveOptions{
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

	if err := confirmWrite(cmd, opts.DangerousInline, opts.VersionsFile); err != nil {
		return err
	}

	if err := versions.WriteManifest(opts.VersionsFile, manifest); err != nil {
		return err
	}

	if err := render.Generate(render.Options{
		Manifest:     manifest,
		OutputDir:    opts.OutputDir,
		TemplatesDir: opts.TemplatesDir,
		Cleanup:      opts.Cleanup,
		ConfirmWrite: func(path string) error {
			return confirmWrite(cmd, opts.DangerousInline, path)
		},
	}); err != nil {
		return err
	}

	fmt.Printf("Generated Dockerfiles in %s\n", opts.OutputDir)
	return nil
}
