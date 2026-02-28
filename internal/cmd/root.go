package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/schmitthub/openclaw-docker/internal/config"
)

type runtimeOptions struct {
	ConfigPath           string
	OutputDir            string
	VersionsFile         string
	Version              string
	Cleanup              bool
	Debug                bool
	DangerousInline      bool
	DockerAptPackages    string
	OpenClawConfigDir    string
	OpenClawWorkspaceDir string
	OpenClawGatewayPort  string
	OpenClawBridgePort   string
	OpenClawGatewayBind  string
	OpenClawImage        string
	OpenClawGatewayToken string
	OpenClawExtraMounts  string
	OpenClawHomeVolume   string
	SquidAllowedDomains  string
}

var rootOpts runtimeOptions

func NewRootCmd(buildVersion, buildDate string) *cobra.Command {
	showVersion := false

	cmd := &cobra.Command{
		Use:           "openclaw-docker",
		Short:         "Generate OpenClaw Docker deployment artifacts",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprint(cmd.OutOrStdout(), formatVersion(buildVersion, buildDate))
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVarP(&rootOpts.ConfigPath, "config", "f", "", "Path to YAML config file")
	cmd.Flags().BoolVar(&showVersion, "version", false, "Print CLI version")
	cmd.PersistentFlags().StringVarP(&rootOpts.OutputDir, "output", "o", "", "Dockerfile output directory (defaults to ./openclaw-deploy)")
	cmd.PersistentFlags().StringVar(&rootOpts.VersionsFile, "versions-file", "", "Path to versions manifest JSON")
	cmd.PersistentFlags().BoolVar(&rootOpts.Debug, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().BoolVar(&rootOpts.Cleanup, "cleanup", false, "Show defensive cleanup warning (deletes are disabled; generation is overwrite-only)")
	cmd.PersistentFlags().BoolVar(&rootOpts.DangerousInline, "dangerous-inline", false, "Skip write confirmation prompts and perform writes inline")
	cmd.PersistentFlags().StringVar(&rootOpts.DockerAptPackages, "docker-apt-packages", "", "Additional apt packages to install in generated Dockerfile")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawConfigDir, "openclaw-config-dir", "", "Default OPENCLAW_CONFIG_DIR value baked into generated Dockerfile")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawWorkspaceDir, "openclaw-workspace-dir", "", "Default OPENCLAW_WORKSPACE_DIR value baked into generated Dockerfile")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawGatewayPort, "openclaw-gateway-port", "", "Default OPENCLAW_GATEWAY_PORT value baked into generated Dockerfile")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawBridgePort, "openclaw-bridge-port", "", "Default OPENCLAW_BRIDGE_PORT value baked into generated Dockerfile")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawGatewayBind, "openclaw-gateway-bind", "", "Default OPENCLAW_GATEWAY_BIND value baked into generated Dockerfile")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawImage, "openclaw-image", "", "Default OPENCLAW_IMAGE value used in generated compose/.env.openclaw")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawGatewayToken, "openclaw-gateway-token", "", "Default OPENCLAW_GATEWAY_TOKEN value used in generated compose/.env.openclaw")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawExtraMounts, "openclaw-extra-mounts", "", "Default OPENCLAW_EXTRA_MOUNTS value used in generated compose/.env.openclaw")
	cmd.PersistentFlags().StringVar(&rootOpts.OpenClawHomeVolume, "openclaw-home-volume", "", "Default OPENCLAW_HOME_VOLUME value used in generated compose/.env.openclaw")
	cmd.PersistentFlags().StringVar(&rootOpts.SquidAllowedDomains, "squid-allowed-domains", "", "Comma-separated domains to whitelist in squid egress proxy")
	cmd.PersistentFlags().StringVar(&rootOpts.Version, "openclaw-version", "", "Requested OpenClaw version/tag (dist-tag like 'latest' or semver partial like '2026.2')")

	cmd.AddCommand(newVersionCmd(buildVersion, buildDate))
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newResolveCmd())

	return cmd
}

func mergedOptions(cmd *cobra.Command) (runtimeOptions, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return runtimeOptions{}, fmt.Errorf("get cwd: %w", err)
	}

	merged := runtimeOptions{
		OutputDir:            filepath.Join(cwd, "openclaw-deploy"),
		VersionsFile:         defaultVersionsFilePath(),
		Version:              "latest",
		Cleanup:              false,
		DockerAptPackages:    "",
		OpenClawConfigDir:    "/home/node/.openclaw",
		OpenClawWorkspaceDir: "/home/node/.openclaw/workspace",
		OpenClawGatewayPort:  "18789",
		OpenClawBridgePort:   "18790",
		OpenClawGatewayBind:  "lan",
		OpenClawImage:        "openclaw:local",
		OpenClawGatewayToken: "",
		OpenClawExtraMounts:  "",
		OpenClawHomeVolume:   "",
		SquidAllowedDomains:  "",
	}

	if rootOpts.ConfigPath != "" {
		fileCfg, err := config.Load(rootOpts.ConfigPath)
		if err != nil {
			return runtimeOptions{}, err
		}

		if fileCfg.Version != "" {
			merged.Version = fileCfg.Version
		}
		if fileCfg.VersionsFile != "" {
			merged.VersionsFile = fileCfg.VersionsFile
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
		if fileCfg.DockerAptPackages != "" {
			merged.DockerAptPackages = fileCfg.DockerAptPackages
		}
		if fileCfg.OpenClawConfigDir != "" {
			merged.OpenClawConfigDir = fileCfg.OpenClawConfigDir
		}
		if fileCfg.OpenClawWorkspaceDir != "" {
			merged.OpenClawWorkspaceDir = fileCfg.OpenClawWorkspaceDir
		}
		if fileCfg.OpenClawGatewayPort != "" {
			merged.OpenClawGatewayPort = fileCfg.OpenClawGatewayPort
		}
		if fileCfg.OpenClawBridgePort != "" {
			merged.OpenClawBridgePort = fileCfg.OpenClawBridgePort
		}
		if fileCfg.OpenClawGatewayBind != "" {
			merged.OpenClawGatewayBind = fileCfg.OpenClawGatewayBind
		}
		if fileCfg.OpenClawImage != "" {
			merged.OpenClawImage = fileCfg.OpenClawImage
		}
		if fileCfg.OpenClawGatewayToken != "" {
			merged.OpenClawGatewayToken = fileCfg.OpenClawGatewayToken
		}
		if fileCfg.OpenClawExtraMounts != "" {
			merged.OpenClawExtraMounts = fileCfg.OpenClawExtraMounts
		}
		if fileCfg.OpenClawHomeVolume != "" {
			merged.OpenClawHomeVolume = fileCfg.OpenClawHomeVolume
		}
		if fileCfg.SquidAllowedDomains != "" {
			merged.SquidAllowedDomains = fileCfg.SquidAllowedDomains
		}
	}

	if err := applyEnvOverrides(&merged); err != nil {
		return runtimeOptions{}, err
	}

	if cmd.Flags().Changed("output") {
		merged.OutputDir = rootOpts.OutputDir
	}
	if cmd.Flags().Changed("versions-file") {
		merged.VersionsFile = rootOpts.VersionsFile
	}
	if cmd.Flags().Changed("cleanup") {
		merged.Cleanup = rootOpts.Cleanup
	}
	if cmd.Flags().Changed("debug") {
		merged.Debug = rootOpts.Debug
	}
	if cmd.Flags().Changed("dangerous-inline") {
		merged.DangerousInline = rootOpts.DangerousInline
	}
	if cmd.Flags().Changed("docker-apt-packages") {
		merged.DockerAptPackages = rootOpts.DockerAptPackages
	}
	if cmd.Flags().Changed("openclaw-config-dir") {
		merged.OpenClawConfigDir = rootOpts.OpenClawConfigDir
	}
	if cmd.Flags().Changed("openclaw-workspace-dir") {
		merged.OpenClawWorkspaceDir = rootOpts.OpenClawWorkspaceDir
	}
	if cmd.Flags().Changed("openclaw-gateway-port") {
		merged.OpenClawGatewayPort = rootOpts.OpenClawGatewayPort
	}
	if cmd.Flags().Changed("openclaw-bridge-port") {
		merged.OpenClawBridgePort = rootOpts.OpenClawBridgePort
	}
	if cmd.Flags().Changed("openclaw-gateway-bind") {
		merged.OpenClawGatewayBind = rootOpts.OpenClawGatewayBind
	}
	if cmd.Flags().Changed("openclaw-image") {
		merged.OpenClawImage = rootOpts.OpenClawImage
	}
	if cmd.Flags().Changed("openclaw-gateway-token") {
		merged.OpenClawGatewayToken = rootOpts.OpenClawGatewayToken
	}
	if cmd.Flags().Changed("openclaw-extra-mounts") {
		merged.OpenClawExtraMounts = rootOpts.OpenClawExtraMounts
	}
	if cmd.Flags().Changed("openclaw-home-volume") {
		merged.OpenClawHomeVolume = rootOpts.OpenClawHomeVolume
	}
	if cmd.Flags().Changed("squid-allowed-domains") {
		merged.SquidAllowedDomains = rootOpts.SquidAllowedDomains
	}
	if cmd.Flags().Changed("openclaw-version") {
		merged.Version = rootOpts.Version
	}

	merged.OutputDir = strings.TrimSpace(merged.OutputDir)
	merged.VersionsFile = strings.TrimSpace(merged.VersionsFile)
	merged.Version = strings.TrimSpace(merged.Version)
	merged.DockerAptPackages = strings.TrimSpace(merged.DockerAptPackages)
	merged.OpenClawConfigDir = strings.TrimSpace(merged.OpenClawConfigDir)
	merged.OpenClawWorkspaceDir = strings.TrimSpace(merged.OpenClawWorkspaceDir)
	merged.OpenClawGatewayPort = strings.TrimSpace(merged.OpenClawGatewayPort)
	merged.OpenClawBridgePort = strings.TrimSpace(merged.OpenClawBridgePort)
	merged.OpenClawGatewayBind = strings.TrimSpace(merged.OpenClawGatewayBind)
	merged.OpenClawImage = strings.TrimSpace(merged.OpenClawImage)
	merged.OpenClawGatewayToken = strings.TrimSpace(merged.OpenClawGatewayToken)
	merged.OpenClawExtraMounts = strings.TrimSpace(merged.OpenClawExtraMounts)
	merged.OpenClawHomeVolume = strings.TrimSpace(merged.OpenClawHomeVolume)
	merged.SquidAllowedDomains = strings.TrimSpace(merged.SquidAllowedDomains)

	if merged.Version == "" {
		merged.Version = "latest"
	}

	return merged, nil
}

func applyEnvOverrides(opts *runtimeOptions) error {
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OUTPUT"); ok {
		opts.OutputDir = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_VERSIONS_FILE"); ok {
		opts.VersionsFile = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_VERSION"); ok {
		opts.Version = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_APT_PACKAGES"); ok {
		opts.DockerAptPackages = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_CONFIG_DIR"); ok {
		opts.OpenClawConfigDir = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_WORKSPACE_DIR"); ok {
		opts.OpenClawWorkspaceDir = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_GATEWAY_PORT"); ok {
		opts.OpenClawGatewayPort = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_BRIDGE_PORT"); ok {
		opts.OpenClawBridgePort = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_GATEWAY_BIND"); ok {
		opts.OpenClawGatewayBind = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_IMAGE"); ok {
		opts.OpenClawImage = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_GATEWAY_TOKEN"); ok {
		opts.OpenClawGatewayToken = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_EXTRA_MOUNTS"); ok {
		opts.OpenClawExtraMounts = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_HOME_VOLUME"); ok {
		opts.OpenClawHomeVolume = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_SQUID_ALLOWED_DOMAINS"); ok {
		opts.SquidAllowedDomains = value
	}

	if value, ok := getenvTrim("OPENCLAW_DOCKER_CLEANUP"); ok {
		parsed, err := parseBoolEnv("OPENCLAW_DOCKER_CLEANUP", value)
		if err != nil {
			return err
		}
		opts.Cleanup = parsed
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_DEBUG"); ok {
		parsed, err := parseBoolEnv("OPENCLAW_DOCKER_DEBUG", value)
		if err != nil {
			return err
		}
		opts.Debug = parsed
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_DANGEROUS_INLINE"); ok {
		parsed, err := parseBoolEnv("OPENCLAW_DOCKER_DANGEROUS_INLINE", value)
		if err != nil {
			return err
		}
		opts.DangerousInline = parsed
	}
	return nil
}

func getenvTrim(name string) (string, bool) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func parseBoolEnv(name, raw string) (bool, error) {
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("parse %s as bool: %w", name, err)
	}
	return parsed, nil
}

func defaultVersionsFilePath() string {
	if cacheDir := strings.TrimSpace(os.Getenv("OPENCLAW_DOCKER_CACHE_DIR")); cacheDir != "" {
		return filepath.Join(cacheDir, "openclaw-docker", "versions.json")
	}

	if xdgCache := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); xdgCache != "" {
		return filepath.Join(xdgCache, "openclaw-docker", "versions.json")
	}

	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".cache", "openclaw-docker", "versions.json")
	}

	return filepath.Join(".cache", "openclaw-docker", "versions.json")
}
