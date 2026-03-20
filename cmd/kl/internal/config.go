package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rawnaqs/khayal/internal/config"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

var configPath = func() string {
	if envPath := os.Getenv("KL_CONFIG"); envPath != "" {
		return envPath
	}
	return config.ExpandTilde("~/.config/khayal/kl.yaml")
}()

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found: %s (run 'kl init' first)", configPath)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func GetConfigPath() string {
	return configPath
}
