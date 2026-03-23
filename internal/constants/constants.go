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

// LLM Prompt templates
type PromptConfig struct {
	DescribeImage   string `json:"describe_image" yaml:"describe_image"`
	ExtractTags     string `json:"extract_tags" yaml:"extract_tags"`
	Summarize       string `json:"summarize" yaml:"summarize"`
	ExtractKeyIdeas string `json:"extract_key_ideas" yaml:"extract_key_ideas"`
	VisionPrompt    string `json:"vision_prompt" yaml:"vision_prompt"`
}

var DefaultPrompts = PromptConfig{
	DescribeImage: "Describe this image in detail. Include any text visible in the image.",
	ExtractTags: `Extract 3-5 relevant tags for the following content.
Return only a JSON array of strings, nothing else. No markdown.

Content:
%s

Tags:`,
	Summarize: `Summarize the following content in 2-3 sentences.

Content:
%s

Summary:`,
	ExtractKeyIdeas: `Extract 3-5 key ideas from the following content.
Return only a JSON array of strings, nothing else. No markdown.

Content:
%s

Key Ideas:`,
	VisionPrompt: "Describe this image in detail. Include any text visible in the image.",
}
