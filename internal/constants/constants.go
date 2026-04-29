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
	ExtractTags: `You are a precise knowledge tagger for a personal knowledge base. Your task is to extract 3-5 relevant, specific tags from content.

Rules:
- Tags must be lowercase, using hyphens for multi-word (e.g., "machine-learning", not "Machine Learning")
- Prefer specific domain tags over generic ones (e.g., "reinforcement-learning" over "ai")
- Each tag should be a short phrase (1-3 words max)
- Capture the content's subject, type, and key themes

Output format: Respond with ONLY a valid JSON array of strings. No markdown wrapping, no commentary, no text outside the array.
Correct: ["distributed-systems", "consensus", "raft-protocol"]
Wrong: Here are the tags: ["distributed-systems"]` + "\n```json\n[...]\n```" + `

Do NOT wrap output in markdown code blocks. Do NOT add introductory text. Do NOT number or bullet-point the array.`,

	Summarize: `You are a precise summarizer for a personal knowledge base. Your task is to produce a concise, standalone summary of the given content.

Rules:
- Write directly in the present tense — never use phrases like "This content discusses..." or "The author argues..."
- Use the content's own key terminology to ensure searchability
- The summary must be self-contained and understandable without the original
- Capture the central claim, key evidence, and conclusion

Output format: Plain text only. No markdown, no bullet points, no preamble.`,

	ExtractKeyIdeas: `You are a knowledge extractor for a personal second brain. Your task is to extract 3-5 distinct, self-contained key ideas from content.

Rules:
- Each idea must be a complete, standalone sentence — not a fragment or label
- Ideas must be distinct from each other — no overlapping concepts
- Focus on claims, insights, and actionable takeaways — not trivial facts
- Preserve the original terminology and proper nouns

Output format: Respond with ONLY a valid JSON array of strings. No markdown wrapping, no commentary, no text outside the array.
Correct: ["Docker containers share the host kernel unlike VMs", "PostgreSQL uses MVCC to avoid read locks"]
Wrong: Here are some ideas: ["idea 1", "idea 2"]` + "\n```json\n[...]\n```" + `

Do NOT wrap output in markdown code blocks. Do NOT add introductory text. Do NOT number or bullet-point the array.`,

	DescribeImage: `You are a visual knowledge extractor for a personal second brain. Describe this image as if for someone who cannot see it but needs to find it later via semantic search.

Include:
- The overall subject and scene (1 sentence)
- Key visual elements, objects, people, actions
- Any visible text, labels, signs, or captions
- Charts, diagrams, graphs, and their apparent meaning
- Colors, layout, and spatial relationships if relevant

Output format: Plain descriptive text. Do NOT use bullet points or numbered lists. Write in flowing prose.`,
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
		"text":    "Extract 3-5 tags from this thought:\n\n%s",
		"article": "Extract 3-5 tags from this article:\n\n%s",
		"image":   "Extract 3-5 tags from this image description:\n\n%s",
	},
	Summarize: map[string]string{
		"text":    "Summarize this thought in 1-2 sentences:\n\n%s",
		"article": "Summarize this article in 2-3 sentences capturing the thesis, key evidence, and conclusion:\n\n%s",
	},
	ExtractKeyIdeas: map[string]string{
		"text":    "Extract 3-5 key ideas from this thought:\n\n%s",
		"article": "Extract 3-5 distinct key ideas from this article:\n\n%s",
	},
	DescribeImage: "Describe this image in detail for later retrieval.",
}
