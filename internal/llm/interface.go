package llm

type LLM interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Generate(prompt string) (string, error)
	DescribeImage(imagePath string) (string, error)
	Ping() error
	Type() string
}

type LLMExt interface {
	LLM
	ExtractTags(content string) ([]string, error)
	Summarize(content string) (string, error)
	ExtractKeyIdeas(content string) ([]string, error)
}

const (
	ProviderOllama = "ollama"
	ProviderGroq   = "groq"
	ProviderOpenAI = "openai"
)
