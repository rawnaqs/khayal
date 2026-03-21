package config

import (
	"crypto/rand"
	"errors"
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
	Search SearchConfig `yaml:"search"`
	Log    LogConfig    `yaml:"log"`
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
	Host             string `yaml:"host"`
	Port             int    `yaml:"port"`
	Token            string `yaml:"token"`
	MaxTextBodyMB    int    `yaml:"max_text_body_mb"`
	MaxImageBodyMB   int    `yaml:"max_image_body_mb"`
	ShutdownTimeoutS int    `yaml:"shutdown_timeout_s"`
}

type LLMConfig struct {
	Provider              string `yaml:"provider"`
	OllamaHost            string `yaml:"ollama_host"`
	EmbedModel            string `yaml:"embed_model"`
	TextModel             string `yaml:"text_model"`
	VisionModel           string `yaml:"vision_model"`
	FallbackProvider      string `yaml:"fallback_provider"`
	FallbackAPIKey        string `yaml:"fallback_api_key"`
	TruncateTextTokens    int    `yaml:"truncate_text_tokens"`
	TruncateImageTokens   int    `yaml:"truncate_image_tokens"`
	TruncateArticleTokens int    `yaml:"truncate_article_tokens"`
	MaxLLMConcurrency     int    `yaml:"max_llm_concurrency"`
}

type WorkerConfig struct {
	MaxWorkers   int    `yaml:"max_workers"`
	MaxRetries   int    `yaml:"max_retries"`
	RetryBackoff string `yaml:"retry_backoff"`
}

type DBConfig struct {
	Path string `yaml:"path"`
}

type SearchConfig struct {
	MaxResults       int     `yaml:"max_results"`
	MaxExcerpt       int     `yaml:"max_excerpt"`
	RRFK             int     `yaml:"rrf_k"`
	MinSemanticScore float64 `yaml:"min_semantic_score"`
}

type LogConfig struct {
	Level             string `yaml:"level"`
	WorkerLevel       string `yaml:"worker_level"`
	File              string `yaml:"file"`
	RotationMaxSizeMB int    `yaml:"rotation_max_size_mb"`
	RotationMaxFiles  int    `yaml:"rotation_max_files"`
}

func DefaultConfig() *Config {
	return &Config{
		Vault: VaultConfig{
			Path:     "~/Documents/brain",
			InboxDir: "khayal",
			Media: MediaConfig{
				DefaultDir: "media",
				Strategy: StrategyConfig{
					Image: "vault",
					PDF:   "vault",
					Audio: "config",
					Video: "config",
				},
			},
		},
		Server: ServerConfig{
			Host:             "127.0.0.1",
			Port:             1133,
			Token:            "",
			MaxTextBodyMB:    1,
			MaxImageBodyMB:   10,
			ShutdownTimeoutS: 30,
		},
		LLM: LLMConfig{
			Provider:              "ollama",
			OllamaHost:            "http://localhost:11434",
			EmbedModel:            "nomic-embed-text",
			TextModel:             "llama3.2:3b",
			VisionModel:           "moondream",
			TruncateTextTokens:    2000,
			TruncateImageTokens:   3000,
			TruncateArticleTokens: 12000,
			MaxLLMConcurrency:     4,
		},
		Worker: WorkerConfig{
			MaxWorkers:   1,
			MaxRetries:   3,
			RetryBackoff: "exponential",
		},
		DB: DBConfig{
			Path: DefaultDBPath,
		},
		Search: SearchConfig{
			MaxResults: 50,
			MaxExcerpt: 500,
			RRFK:       60,
		},
	}
}

func Load() (*Config, string, error) {
	return LoadFromPath(DefaultConfigPath)
}

func LoadFromPath(path string) (*Config, string, error) {
	absPath, err := filepath.Abs(ExpandTilde(path))
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg := DefaultConfig()

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, absPath, nil
		}
		return nil, "", fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, "", fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.ApplyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, absPath, nil
}

func (c *Config) ApplyDefaults() {
	if c.Vault.InboxDir == "" {
		c.Vault.InboxDir = "khayal"
	}
	if c.Server.MaxTextBodyMB == 0 {
		c.Server.MaxTextBodyMB = 1
	}
	if c.Server.MaxImageBodyMB == 0 {
		c.Server.MaxImageBodyMB = 10
	}
	if c.Server.Port == 0 {
		c.Server.Port = 1133
	}
	if c.Server.ShutdownTimeoutS == 0 {
		c.Server.ShutdownTimeoutS = 30
	}
	if c.Search.MaxResults == 0 {
		c.Search.MaxResults = 50
	}
	if c.Search.MaxExcerpt == 0 {
		c.Search.MaxExcerpt = 500
	}
	if c.Search.RRFK == 0 {
		c.Search.RRFK = 60
	}
	if c.Search.MinSemanticScore == 0 {
		c.Search.MinSemanticScore = 0.5
	}
	if c.LLM.TruncateTextTokens == 0 {
		c.LLM.TruncateTextTokens = 2000
	}
	if c.LLM.TruncateImageTokens == 0 {
		c.LLM.TruncateImageTokens = 3000
	}
	if c.LLM.TruncateArticleTokens == 0 {
		c.LLM.TruncateArticleTokens = 12000
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.File == "" {
		c.Log.File = DefaultLogPath
	}
	if c.Log.RotationMaxSizeMB == 0 {
		c.Log.RotationMaxSizeMB = 10
	}
	if c.Log.RotationMaxFiles == 0 {
		c.Log.RotationMaxFiles = 5
	}
}

func (c *Config) Validate() error {
	if c.Vault.Path == "" {
		return fmt.Errorf("vault.path is required")
	}
	if err := ValidateVaultSubPath(c.Vault.InboxDir); err != nil {
		return fmt.Errorf("vault.inbox_dir: %w", err)
	}
	if err := ValidateVaultSubPath(c.Vault.Media.DefaultDir); err != nil {
		return fmt.Errorf("vault.media.default_dir: %w", err)
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.Token == "" {
		c.Server.Token = GenerateToken()
	}
	return nil
}

func ValidateVaultSubPath(path string) error {
	if path == "" {
		return errors.New("cannot be empty")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("must be relative: %s", path)
	}
	if strings.HasPrefix(path, "~") || strings.Contains(path, "$") {
		return fmt.Errorf("must not contain ~ or env vars: %s", path)
	}
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") || cleaned == ".." {
		return fmt.Errorf("must not escape vault: %s", path)
	}
	if strings.HasPrefix(filepath.Base(path), ".") {
		return fmt.Errorf("must not be hidden: %s", path)
	}
	return nil
}

func (c *Config) EnsureDirectories(configPath string) error {
	cfgDir := filepath.Dir(configPath)
	paths := []string{
		cfgDir,
		MakeAbsolute(c.Vault.Path, configPath),
		filepath.Dir(MakeAbsolute(c.DB.Path, configPath)),
		filepath.Dir(MakeAbsolute(c.Log.File, configPath)),
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
	absPath := ExpandTilde(path)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(absPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func MakeAbsolute(path, configPath string) string {
	path = os.ExpandEnv(path)
	path = ExpandTilde(path)

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(filepath.Dir(configPath), path)
}

func ExpandTilde(path string) string {
	if len(path) >= 2 && path[0] == '~' && path[1] == '/' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
