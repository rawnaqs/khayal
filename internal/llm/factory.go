package llm

import (
	"fmt"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/constants"
)

func NewLLM(cfg config.LLMConfig) (LLMExt, error) {
	switch cfg.Provider {
	case ProviderOllama:
		// Convert config prompts to constants.PromptConfig
		var prompts *constants.PromptConfig
		if cfg.Prompts != nil {
			prompts = &constants.PromptConfig{
				DescribeImage:   cfg.Prompts.DescribeImage,
				ExtractTags:     cfg.Prompts.ExtractTags,
				Summarize:       cfg.Prompts.Summarize,
				ExtractKeyIdeas: cfg.Prompts.ExtractKeyIdeas,
				VisionPrompt:    cfg.Prompts.VisionPrompt,
			}
		}

		client := NewOllamaClientWithConfig(
			cfg.OllamaHost,
			cfg.EmbedModel,
			cfg.TextModel,
			cfg.VisionModel,
			cfg.MaxLLMConcurrency,
			cfg.Temperature,
			prompts,
		)

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
