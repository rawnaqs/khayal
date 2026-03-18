package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "~/.config/khayal/config.yaml"
	DefaultDBPath     = "~/.config/khayal/khayal.db"
	DefaultLogPath    = "~/.config/khayal/logs/khayal.log"
)

type Config struct {
	Vault  VaultConfig  `yaml:"vault"`
	Server ServerConfig `yaml:"server"`
	LLM    LLMConfig    `yaml:"llm"`
	Worker WorkerConfig `yaml:"worker"`
	DB     DBConfig     `yaml:"db"`
}

type VaultConfig struct {
	Path     string      `yaml:"path"`
	InboxDir string      `yaml:"inbox_dir"`
	Media    MediaConfig `yaml:"media"`
}

type MediaConfig struct {
	DefaultDir string         `yaml:"default_dir"`
	Strategy   StrategyConfig `yaml:"strategy"`
}

type StrategyConfig struct {
	Image string `yaml:"image"`
	PDF   string `yaml:"pdf"`
	Audio string `yaml:"audio"`
	Video string `yaml:"video"`
}

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Token   string `yaml:"token"`
	LogFile string `yaml:"log_file"`
}

type LLMConfig struct {
	Provider         string `yaml:"provider"`
	OllamaHost       string `yaml:"ollama_host"`
	EmbedModel       string `yaml:"embed_model"`
	TextModel        string `yaml:"text_model"`
	VisionModel      string `yaml:"vision_model"`
	FallbackProvider string `yaml:"fallback_provider"`
	FallbackAPIKey   string `yaml:"fallback_api_key"`
}

type WorkerConfig struct {
	MaxWorkers   int    `yaml:"max_workers"`
	MaxRetries   int    `yaml:"max_retries"`
	RetryBackoff string `yaml:"retry_backoff"`
}

type DBConfig struct {
	Path string `yaml:"path"`
}

func DefaultConfig() *Config {
	return &Config{
		Vault: VaultConfig{
			Path:     "~/Documents/brain",
			InboxDir: "inbox",
			Media: MediaConfig{
				DefaultDir: "inbox/media",
				Strategy: StrategyConfig{
					Image: "vault",
					PDF:   "vault",
					Audio: "config",
					Video: "config",
				},
			},
		},
		Server: ServerConfig{
			Host:    "127.0.0.1",
			Port:    7766,
			Token:   "",
			LogFile: DefaultLogPath,
		},
		LLM: LLMConfig{
			Provider:    "ollama",
			OllamaHost:  "http://localhost:11434",
			EmbedModel:  "nomic-embed-text",
			TextModel:   "llama3.2:3b",
			VisionModel: "moondream",
		},
		Worker: WorkerConfig{
			MaxWorkers:   1,
			MaxRetries:   3,
			RetryBackoff: "exponential",
		},
		DB: DBConfig{
			Path: DefaultDBPath,
		},
	}
}

func Load() (*Config, error) {
	return LoadFromPath(DefaultConfigPath)
}

func LoadFromPath(path string) (*Config, error) {
	path = expandPath(path)

	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Vault.Path == "" {
		return fmt.Errorf("vault.path is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.Token == "" {
		c.Server.Token = GenerateToken()
	}
	return nil
}

func (c *Config) EnsureDirectories() error {
	paths := []string{
		filepath.Dir(expandPath(DefaultConfigPath)),
		expandPath(c.Vault.Path),
		filepath.Dir(expandPath(c.DB.Path)),
		filepath.Dir(expandPath(c.Server.LogFile)),
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", p, err)
		}
	}

	return nil
}

func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func Save(cfg *Config, path string) error {
	path = expandPath(path)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}
