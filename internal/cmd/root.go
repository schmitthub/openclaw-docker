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
	OpenClawGatewayToken string
	AllowedDomains       string
	ExternalOrigin       string
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
	cmd.PersistentFlags().BoolVar(&rootOpts.Debug, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().BoolVar(&rootOpts.DangerousInline, "dangerous-inline", false, "Skip write confirmation prompts and perform writes inline")

	cmd.AddCommand(newVersionCmd(buildVersion, buildDate))
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newGenerateCmd())

	return cmd
}

func mergedOptions(cmd *cobra.Command) (runtimeOptions, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return runtimeOptions{}, fmt.Errorf("get cwd: %w", err)
	}

	merged := runtimeOptions{
		OutputDir:            filepath.Join(cwd, "openclaw-deploy"),
		Version:              "latest",
		Cleanup:              false,
		DockerAptPackages:    "",
		OpenClawConfigDir:    "/home/node/.openclaw",
		OpenClawWorkspaceDir: "/home/node/.openclaw/workspace",
		OpenClawGatewayPort:  "18789",
		OpenClawBridgePort:   "18790",
		OpenClawGatewayBind:  "lan",
		OpenClawGatewayToken: "",
		AllowedDomains:       "",
		ExternalOrigin:       "",
	}

	if rootOpts.ConfigPath != "" {
		fileCfg, err := config.Load(rootOpts.ConfigPath)
		if err != nil {
			return runtimeOptions{}, err
		}

		if fileCfg.Version != "" {
			merged.Version = fileCfg.Version
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
		if fileCfg.OpenClawGatewayToken != "" {
			merged.OpenClawGatewayToken = fileCfg.OpenClawGatewayToken
		}
		if fileCfg.AllowedDomains != "" {
			merged.AllowedDomains = fileCfg.AllowedDomains
		}
		if fileCfg.ExternalOrigin != "" {
			merged.ExternalOrigin = fileCfg.ExternalOrigin
		}
	}

	if err := applyEnvOverrides(&merged); err != nil {
		return runtimeOptions{}, err
	}

	if cmd.Flags().Changed("output") {
		merged.OutputDir = rootOpts.OutputDir
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
	if cmd.Flags().Changed("openclaw-gateway-token") {
		merged.OpenClawGatewayToken = rootOpts.OpenClawGatewayToken
	}
	if cmd.Flags().Changed("allowed-domains") {
		merged.AllowedDomains = rootOpts.AllowedDomains
	}
	if cmd.Flags().Changed("openclaw-version") {
		merged.Version = rootOpts.Version
	}
	if cmd.Flags().Changed("external-origin") {
		merged.ExternalOrigin = rootOpts.ExternalOrigin
	}

	merged.OutputDir = strings.TrimSpace(merged.OutputDir)
	merged.Version = strings.TrimSpace(merged.Version)
	merged.DockerAptPackages = strings.TrimSpace(merged.DockerAptPackages)
	merged.OpenClawConfigDir = strings.TrimSpace(merged.OpenClawConfigDir)
	merged.OpenClawWorkspaceDir = strings.TrimSpace(merged.OpenClawWorkspaceDir)
	merged.OpenClawGatewayPort = strings.TrimSpace(merged.OpenClawGatewayPort)
	merged.OpenClawBridgePort = strings.TrimSpace(merged.OpenClawBridgePort)
	merged.OpenClawGatewayBind = strings.TrimSpace(merged.OpenClawGatewayBind)
	merged.OpenClawGatewayToken = strings.TrimSpace(merged.OpenClawGatewayToken)
	merged.AllowedDomains = strings.TrimSpace(merged.AllowedDomains)
	merged.ExternalOrigin = strings.TrimSpace(merged.ExternalOrigin)

	if merged.Version == "" {
		merged.Version = "latest"
	}

	return merged, nil
}

func applyEnvOverrides(opts *runtimeOptions) error {
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OUTPUT"); ok {
		opts.OutputDir = value
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
	if value, ok := getenvTrim("OPENCLAW_DOCKER_OPENCLAW_GATEWAY_TOKEN"); ok {
		opts.OpenClawGatewayToken = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_ALLOWED_DOMAINS"); ok {
		opts.AllowedDomains = value
	}
	if value, ok := getenvTrim("OPENCLAW_DOCKER_EXTERNAL_ORIGIN"); ok {
		opts.ExternalOrigin = value
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
