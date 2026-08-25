package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/chunk"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Vault.Path != "" {
		t.Errorf("expected empty vault path, got %s", cfg.Vault.Path)
	}
	if cfg.Server.Port != 1133 {
		t.Errorf("expected default port 1133, got %d", cfg.Server.Port)
	}
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected default provider ollama, got %s", cfg.LLM.Provider)
	}
	if cfg.Worker.MaxWorkers != 1 {
		t.Errorf("expected default max workers 1, got %d", cfg.Worker.MaxWorkers)
	}
}

func TestGenerateToken(t *testing.T) {
	token := GenerateToken()

	if len(token) != 64 {
		t.Errorf("expected token length 64 (32 bytes hex), got %d", len(token))
	}

	token2 := GenerateToken()
	if token == token2 {
		t.Error("expected different tokens, got same")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "empty vault path",
			cfg:     &Config{Vault: VaultConfig{Path: ""}, Server: ServerConfig{Port: 1133}},
			wantErr: true,
		},
		{
			name:    "invalid port",
			cfg:     &Config{Vault: VaultConfig{Path: "~/test"}, Server: ServerConfig{Port: 0}},
			wantErr: true,
		},
		{
			name:    "invalid port too high",
			cfg:     &Config{Vault: VaultConfig{Path: "~/test"}, Server: ServerConfig{Port: 70000}},
			wantErr: true,
		},
		{
			name: "valid config - token auto-generated",
			cfg: &Config{
				Vault: VaultConfig{
					Path:     "~/test",
					InboxDir: "inbox",
					Media: MediaConfig{
						DefaultDir: "media",
					},
				},
				Server: ServerConfig{Port: 1133},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.cfg.Server.Token == "" {
				t.Error("expected token to be auto-generated")
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Search = SearchConfig{} // simulates a config file with no search section
	cfg.ApplyDefaults()

	if cfg.Search.MaxResults != 50 {
		t.Errorf("MaxResults = %d, want 50", cfg.Search.MaxResults)
	}
	if cfg.Search.ChunkTargetWords != 175 {
		t.Errorf("ChunkTargetWords = %d, want 175", cfg.Search.ChunkTargetWords)
	}
	if cfg.Search.ChunkMinWords != 50 {
		t.Errorf("ChunkMinWords = %d, want 50", cfg.Search.ChunkMinWords)
	}
	if cfg.Search.ChunkOverlapWords != 35 {
		t.Errorf("ChunkOverlapWords = %d, want 35", cfg.Search.ChunkOverlapWords)
	}

	d := chunk.DefaultOptions()
	got := cfg.Search.ChunkOptions()
	if got != d {
		t.Errorf("ChunkOptions() = %+v, want %+v", got, d)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Vault.Path = tmpDir
	cfg.DB.Path = filepath.Join(tmpDir, "test.db")

	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, _, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if loaded.Vault.Path != cfg.Vault.Path {
		t.Errorf("expected vault path %s, got %s", cfg.Vault.Path, loaded.Vault.Path)
	}
	if loaded.Server.Port != cfg.Server.Port {
		t.Errorf("expected port %d, got %d", cfg.Server.Port, loaded.Server.Port)
	}
}

func TestMakeAbsolute(t *testing.T) {
	home, _ := os.UserHomeDir()
	configPath := "/home/user/config.yaml"

	tests := []struct {
		path       string
		configPath string
		expected   string
	}{
		{"~/test", configPath, filepath.Join(home, "test")},
		{"/absolute/path", configPath, "/absolute/path"},
		{"relative/path", configPath, "/home/user/relative/path"},
		{"$HOME/test", configPath, filepath.Join(home, "test")},
	}

	for _, tt := range tests {
		result := MakeAbsolute(tt.path, tt.configPath)
		if result != tt.expected {
			t.Errorf("MakeAbsolute(%q, %q) = %q, want %q", tt.path, tt.configPath, result, tt.expected)
		}
	}
}

func TestMemoryThrottleDefaults(t *testing.T) {
	var m MemoryConfig
	if got := m.ConsolidationInterval(); got != 24*time.Hour {
		t.Errorf("default interval = %v, want 24h", got)
	}
	if got := m.PersonsThreshold(); got != 5 {
		t.Errorf("default threshold = %d, want 5", got)
	}

	zero := 0
	m.ConsolidationIntervalHours = &zero
	m.NewPersonsThreshold = &zero
	if m.ConsolidationInterval() != 0 {
		t.Error("explicit zero interval must be honored")
	}
	if m.PersonsThreshold() != 0 {
		t.Error("explicit zero threshold must be honored")
	}
}
