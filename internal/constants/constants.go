package constants

import "time"

// SQLite retry configuration
const (
	SQLiteMaxRetries = 3
	SQLiteRetrySleep = 100 * time.Millisecond
)

// Streak milestones
var Milestones = []int{7, 14, 21, 30, 50, 75, 100, 150, 200, 365}

// Search configuration
const (
	SearchOverFetchMultiplier = 2
	DefaultSearchLimit        = 10
	DefaultQueueLimit         = 20
)

// Worker timeouts
const (
	WorkerTickerInterval = 500 * time.Millisecond
	WorkerJobTimeout     = 120 * time.Second
	WorkerRetryBackoff   = 5 * time.Second
)

// Ollama timeouts
const (
	OllamaClientTimeout   = 120 * time.Second
	OllamaPingTimeout     = 5 * time.Second
	OllamaEmbedTimeout    = 60 * time.Second
	OllamaGenerateTimeout = 120 * time.Second
	OllamaVisionTimeout   = 120 * time.Second
)

// LLM defaults
const (
	DefaultTemperature           = 0.7
	DefaultMaxConcurrent         = 4
	DefaultTruncateTextTokens    = 2000
	DefaultTruncateImageTokens   = 3000
	DefaultTruncateArticleTokens = 12000
)

// System prompts define the model's persona and output expectations.
type SystemPrompts struct {
	ExtractTags     string `yaml:"extract_tags"`
	Summarize       string `yaml:"summarize"`
	ExtractKeyIdeas string `yaml:"extract_key_ideas"`
	DescribeImage   string `yaml:"describe_image"`
}

var DefaultSystemPrompts = SystemPrompts{
	ExtractTags:     `You are a precise knowledge extractor. Given a piece of content, extract the most relevant tags that categorize and describe it. Tags should be single words or short phrases (2-3 words max). Be specific, not generic. Output ONLY a JSON array of plain strings. No markdown, no formatting, no text outside the array.`,
	Summarize:       `You are a precise summarizer for a personal knowledge base. Summarize content concisely, capturing the essential meaning. Never use phrases like "this content" or "the text discusses". Write directly.`,
	ExtractKeyIdeas: `You are a knowledge extractor for a personal second brain. Extract the most important, actionable ideas from the content. Each idea should be a complete, standalone thought. Output ONLY a JSON array of plain strings. No markdown, no formatting, no text outside the array.`,
	DescribeImage:   `You are a visual knowledge extractor for a personal second brain. Describe images in detail, focusing on what's useful for later retrieval. Include any text visible in the image, charts, diagrams, and key visual elements.`,
}

// Prompt templates define per-bucket user prompts.
type PromptTemplates struct {
	ExtractTags     map[string]string `yaml:"extract_tags"`
	Summarize       map[string]string `yaml:"summarize"`
	ExtractKeyIdeas map[string]string `yaml:"extract_key_ideas"`
	DescribeImage   string            `yaml:"describe_image"`
}

var DefaultPromptTemplates = PromptTemplates{
	ExtractTags: map[string]string{
		"text": `Extract 3-5 tags for this thought. Return ONLY a JSON array.
Example: ["tag1", "tag2", "tag3"]

Content:
%s`,
		"article": `Extract 3-5 tags for this article. Return ONLY a JSON array.
Example: ["tag1", "tag2", "tag3"]

Article:
%s`,
		"image": `Extract 3-5 tags describing this image. Return ONLY a JSON array.
Example: ["tag1", "tag2", "tag3"]

Image description:
%s`,
	},
	Summarize: map[string]string{
		"text": `Summarize this thought in 1-2 sentences. Write directly, no preamble.

Content:
%s`,
		"article": `Summarize this article in 2-3 sentences. Focus on the key finding or argument.

Article:
%s`,
	},
	ExtractKeyIdeas: map[string]string{
		"text": `Extract 3-5 key ideas from this thought.
Respond with ONLY a JSON array, nothing else. Do not use markdown formatting, headers, or bullet points.
Example response: ["idea1", "idea2"]

Content:
%s`,
		"article": `Extract 3-5 key ideas from this article.
Respond with ONLY a JSON array, nothing else. Do not use markdown formatting, headers, or bullet points.
Example response: ["idea1", "idea2"]

Article:
%s`,
	},
	DescribeImage: `Describe this image in detail. Include all visible text, charts, diagrams, and key visual elements.`,
}
