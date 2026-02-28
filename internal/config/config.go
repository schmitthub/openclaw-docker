package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FileConfig struct {
	Version              string `yaml:"version"`
	OutputDir            string `yaml:"output"`
	Cleanup              *bool  `yaml:"cleanup"`
	Debug                *bool  `yaml:"debug"`
	DockerAptPackages    string `yaml:"docker_apt_packages"`
	OpenClawConfigDir    string `yaml:"openclaw_config_dir"`
	OpenClawWorkspaceDir string `yaml:"openclaw_workspace_dir"`
	OpenClawGatewayPort  string `yaml:"openclaw_gateway_port"`
	OpenClawBridgePort   string `yaml:"openclaw_bridge_port"`
	OpenClawGatewayBind  string `yaml:"openclaw_gateway_bind"`
	OpenClawImage        string `yaml:"openclaw_image"`
	OpenClawGatewayToken string `yaml:"openclaw_gateway_token"`
	OpenClawExtraMounts  string `yaml:"openclaw_extra_mounts"`
	OpenClawHomeVolume   string `yaml:"openclaw_home_volume"`
	SquidAllowedDomains  string `yaml:"squid_allowed_domains"`
}

func Load(path string) (FileConfig, error) {
	if path == "" {
		return FileConfig{}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("read config: %w", err)
	}

	var cfg FileConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("parse config YAML: %w", err)
	}

	return cfg, nil
}

func FromString(s string) (FileConfig, error) {
	var cfg FileConfig
	if err := yaml.Unmarshal([]byte(s), &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("parse config YAML: %w", err)
	}
	return cfg, nil
}
