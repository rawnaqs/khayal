package llm

import (
	"fmt"

	"github.com/rawnaqs/khayal/internal/config"
)

func NewLLM(cfg config.LLMConfig) (LLMExt, error) {
	switch cfg.Provider {
	case ProviderOllama:
		client := NewOllamaClientWithConcurrency(
			cfg.OllamaHost,
			cfg.EmbedModel,
			cfg.TextModel,
			cfg.VisionModel,
			cfg.MaxLLMConcurrency,
		)

		// Apply temperature override
		if cfg.Temperature > 0 {
			client.temperature = cfg.Temperature
		}

		// Apply truncation overrides if configured
		if cfg.TruncateTextTokens > 0 {
			client.truncateTextTokens = cfg.TruncateTextTokens
		}
		if cfg.TruncateImageTokens > 0 {
			client.truncateImageTokens = cfg.TruncateImageTokens
		}
		if cfg.TruncateArticleTokens > 0 {
			client.truncateArticleTokens = cfg.TruncateArticleTokens
		}

		if err := client.Ping(); err != nil {
			return nil, fmt.Errorf("ollama unavailable at %s: %w", cfg.OllamaHost, err)
		}

		return client, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}
