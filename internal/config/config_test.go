package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Vault.Path != "~/Documents/brain" {
		t.Errorf("expected default vault path ~/Documents/brain, got %s", cfg.Vault.Path)
	}
	if cfg.Server.Port != 7766 {
		t.Errorf("expected default port 7766, got %d", cfg.Server.Port)
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
			cfg:     &Config{Vault: VaultConfig{Path: ""}, Server: ServerConfig{Port: 7766}},
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
				Vault:  VaultConfig{Path: "~/test"},
				Server: ServerConfig{Port: 7766},
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

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Vault.Path = tmpDir
	cfg.DB.Path = filepath.Join(tmpDir, "test.db")

	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := LoadFromPath(configPath)
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

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
