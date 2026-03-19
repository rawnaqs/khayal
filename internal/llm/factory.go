package llm

import (
	"fmt"

	"github.com/rawnaqs/khayal/internal/config"
)

func NewLLM(cfg config.LLMConfig) (LLMExt, error) {
	switch cfg.Provider {
	case ProviderOllama:
		client := NewOllamaClientWithConfig(
			cfg.OllamaHost,
			cfg.EmbedModel,
			cfg.TextModel,
			cfg.VisionModel,
			cfg.TruncateTextTokens,
			cfg.TruncateImageTokens,
			cfg.TruncateArticleTokens,
		)

		if err := client.Ping(); err != nil {
			return nil, fmt.Errorf("ollama unavailable at %s: %w", cfg.OllamaHost, err)
		}

		return client, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}
