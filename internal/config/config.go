package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FileConfig struct {
	Versions      []string            `yaml:"versions"`
	VersionsFile  string              `yaml:"versions_file"`
	TemplatesDir  string              `yaml:"templates_dir"`
	OutputDir     string              `yaml:"output"`
	Cleanup       *bool               `yaml:"cleanup"`
	Debug         *bool               `yaml:"debug"`
	DebianDefault string              `yaml:"debian_default"`
	AlpineDefault string              `yaml:"alpine_default"`
	Variants      map[string][]string `yaml:"variants"`
	Arches        []string            `yaml:"arches"`
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
