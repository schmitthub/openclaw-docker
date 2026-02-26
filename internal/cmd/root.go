package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/config"
)

type runtimeOptions struct {
	ConfigPath     string
	OutputDir      string
	VersionsFile   string
	TemplatesDir   string
	Versions       []string
	Cleanup        bool
	Debug          bool
	DebianDefault  string
	AlpineDefault  string
	Variants       map[string][]string
	Arches         []string
	PackageName    string
	VersionsCSVRaw string
}

var rootOpts runtimeOptions

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openclaw-docker",
		Short: "Generate OpenClaw Dockerfiles",
		RunE:  runGenerate,
	}

	cmd.PersistentFlags().StringVarP(&rootOpts.ConfigPath, "config", "f", "", "Path to YAML config file")
	cmd.PersistentFlags().StringVarP(&rootOpts.OutputDir, "output", "o", "", "Dockerfile output directory (defaults to current working directory)")
	cmd.PersistentFlags().StringVar(&rootOpts.VersionsFile, "versions-file", "", "Path to versions manifest JSON")
	cmd.PersistentFlags().StringVar(&rootOpts.TemplatesDir, "templates-dir", "", "Template helper scripts directory used in generated Dockerfiles")
	cmd.PersistentFlags().BoolVar(&rootOpts.Debug, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().BoolVar(&rootOpts.Cleanup, "cleanup", true, "Delete obsolete version output directories")
	cmd.PersistentFlags().StringArrayVar(&rootOpts.Versions, "version", nil, "Requested version/tag (repeatable)")
	cmd.PersistentFlags().StringVar(&rootOpts.VersionsCSVRaw, "versions", "", "Requested versions/tags as comma-separated list")

	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newResolveCmd())
	cmd.AddCommand(newRenderCmd())

	return cmd
}

func mergedOptions(cmd *cobra.Command) (runtimeOptions, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return runtimeOptions{}, fmt.Errorf("get cwd: %w", err)
	}

	merged := runtimeOptions{
		OutputDir:     cwd,
		VersionsFile:  "versions.json",
		TemplatesDir:  "./build/templates",
		Cleanup:       true,
		DebianDefault: "trixie",
		AlpineDefault: "alpine3.23",
		Variants: map[string][]string{
			"trixie":     {},
			"bookworm":   {},
			"alpine3.23": {},
			"alpine3.22": {},
		},
		Arches:      []string{"amd64", "arm64v8"},
		PackageName: "openclaw",
	}

	if rootOpts.ConfigPath != "" {
		fileCfg, err := config.Load(rootOpts.ConfigPath)
		if err != nil {
			return runtimeOptions{}, err
		}

		if len(fileCfg.Versions) > 0 {
			merged.Versions = append([]string(nil), fileCfg.Versions...)
		}
		if fileCfg.VersionsFile != "" {
			merged.VersionsFile = fileCfg.VersionsFile
		}
		if fileCfg.TemplatesDir != "" {
			merged.TemplatesDir = fileCfg.TemplatesDir
		}
		if fileCfg.OutputDir != "" {
			merged.OutputDir = fileCfg.OutputDir
		}
		if fileCfg.Cleanup != nil {
			merged.Cleanup = *fileCfg.Cleanup
		}
		if fileCfg.Debug != nil {
			merged.Debug = *fileCfg.Debug
		}
		if fileCfg.DebianDefault != "" {
			merged.DebianDefault = fileCfg.DebianDefault
		}
		if fileCfg.AlpineDefault != "" {
			merged.AlpineDefault = fileCfg.AlpineDefault
		}
		if len(fileCfg.Variants) > 0 {
			merged.Variants = fileCfg.Variants
		}
		if len(fileCfg.Arches) > 0 {
			merged.Arches = append([]string(nil), fileCfg.Arches...)
		}
		if fileCfg.PackageName != "" {
			merged.PackageName = fileCfg.PackageName
		}
	}

	if cmd.Flags().Changed("output") {
		merged.OutputDir = rootOpts.OutputDir
	}
	if cmd.Flags().Changed("versions-file") {
		merged.VersionsFile = rootOpts.VersionsFile
	}
	if cmd.Flags().Changed("templates-dir") {
		merged.TemplatesDir = rootOpts.TemplatesDir
	}
	if cmd.Flags().Changed("cleanup") {
		merged.Cleanup = rootOpts.Cleanup
	}
	if cmd.Flags().Changed("debug") {
		merged.Debug = rootOpts.Debug
	}

	if cmd.Flags().Changed("version") {
		merged.Versions = append([]string(nil), rootOpts.Versions...)
	}
	if cmd.Flags().Changed("versions") {
		merged.Versions = splitCSV(rootOpts.VersionsCSVRaw)
	}

	merged.OutputDir = strings.TrimSpace(merged.OutputDir)
	merged.VersionsFile = strings.TrimSpace(merged.VersionsFile)
	merged.TemplatesDir = strings.TrimSpace(merged.TemplatesDir)

	if len(merged.Versions) == 0 {
		merged.Versions = []string{"latest"}
	}

	return merged, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}
