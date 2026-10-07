package llm

import (
	"fmt"

	"github.com/rawnaqs/khayal/internal/config"
)

func NewLLM(cfg config.LLMConfig) (LLMExt, error) {
	return NewLLMWithPing(cfg, true)
}

// NewLLMWithPing builds the primary client. With ping=false the Ollama
// reachability check is skipped, so the server can start without Ollama
// (tests/CI); jobs that need the LLM simply fail until it is back.
func NewLLMWithPing(cfg config.LLMConfig, ping bool) (LLMExt, error) {
	return newOllamaLLM(cfg, cfg.TextModel, ping)
}

// NewConsolidationLLM returns a dedicated client for memory consolidation
// when a distinct consolidation_model is configured. Returns (nil, nil) when
// the consolidation model is unset or identical to the main text model, in
// which case callers should reuse their primary client.
func NewConsolidationLLM(cfg config.LLMConfig) (LLMExt, error) {
	return NewConsolidationLLMWithPing(cfg, true)
}

func NewConsolidationLLMWithPing(cfg config.LLMConfig, ping bool) (LLMExt, error) {
	if cfg.ConsolidationModel == "" || cfg.ConsolidationModel == cfg.TextModel {
		return nil, nil
	}
	return newOllamaLLM(cfg, cfg.ConsolidationModel, ping)
}

func newOllamaLLM(cfg config.LLMConfig, textModel string, ping bool) (LLMExt, error) {
	switch cfg.Provider {
	case ProviderOllama:
		client := NewOllamaClientWithConcurrency(
			cfg.OllamaHost,
			cfg.EmbedModel,
			textModel,
			cfg.VisionModel,
			cfg.MaxLLMConcurrency,
		)

		if cfg.Temperature > 0 {
			client.temperature = cfg.Temperature
		}
		client.SetTempTag(cfg.TemperatureTags)
		client.SetTempSummarize(cfg.TemperatureSummarize)
		client.SetTempKeyIdeas(cfg.TemperatureKeyIdeas)
		client.SetTempVision(cfg.TemperatureVision)

		if cfg.TruncateTextTokens > 0 {
			client.truncateTextTokens = cfg.TruncateTextTokens
		}
		if cfg.TruncateImageTokens > 0 {
			client.truncateImageTokens = cfg.TruncateImageTokens
		}
		if cfg.TruncateArticleTokens > 0 {
			client.truncateArticleTokens = cfg.TruncateArticleTokens
		}

		if cfg.Prompts != nil {
			applyPromptConfig(client, cfg.Prompts)
		}

		if ping {
			if err := client.Ping(); err != nil {
				return nil, fmt.Errorf("ollama unavailable at %s: %w", cfg.OllamaHost, err)
			}
		}

		return client, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}

func applyPromptConfig(client *OllamaClient, p *config.PromptConfig) {
	if p.ExtractTags != "" {
		client.systemPrompts.ExtractTags = p.ExtractTags
	}
	if p.Summarize != "" {
		client.systemPrompts.Summarize = p.Summarize
	}
	if p.ExtractKeyIdeas != "" {
		client.systemPrompts.ExtractKeyIdeas = p.ExtractKeyIdeas
	}
	if p.DescribeImage != "" {
		client.systemPrompts.DescribeImage = p.DescribeImage
	}

	perBucket := map[string]string{}
	setIf := func(k, v string) {
		if v != "" {
			perBucket[k] = v
		}
	}
	setIf("extract_tags:text", p.ExtractTagsText)
	setIf("extract_tags:article", p.ExtractTagsArticle)
	setIf("extract_tags:image", p.ExtractTagsImage)
	setIf("summarize:text", p.SummarizeText)
	setIf("summarize:article", p.SummarizeArticle)
	setIf("extract_key_ideas:text", p.ExtractKeyIdeasText)
	setIf("extract_key_ideas:article", p.ExtractKeyIdeasArticle)
	client.SetPerBucketSystem(perBucket)

	if p.ExtractTagsTextTemplate != "" {
		client.prompts.ExtractTags["text"] = p.ExtractTagsTextTemplate
	}
	if p.ExtractTagsArticleTemplate != "" {
		client.prompts.ExtractTags["article"] = p.ExtractTagsArticleTemplate
	}
	if p.ExtractTagsImageTemplate != "" {
		client.prompts.ExtractTags["image"] = p.ExtractTagsImageTemplate
	}
	if p.SummarizeTextTemplate != "" {
		client.prompts.Summarize["text"] = p.SummarizeTextTemplate
	}
	if p.SummarizeArticleTemplate != "" {
		client.prompts.Summarize["article"] = p.SummarizeArticleTemplate
	}
	if p.ExtractKeyIdeasTextTemplate != "" {
		client.prompts.ExtractKeyIdeas["text"] = p.ExtractKeyIdeasTextTemplate
	}
	if p.ExtractKeyIdeasArticleTemplate != "" {
		client.prompts.ExtractKeyIdeas["article"] = p.ExtractKeyIdeasArticleTemplate
	}
	if p.DescribeImageTemplate != "" {
		client.prompts.DescribeImage = p.DescribeImageTemplate
	}
}
