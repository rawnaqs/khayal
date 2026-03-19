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
	ExtractTags(content string, bucket string) ([]string, error)
	Summarize(content string, bucket string) (string, error)
	ExtractKeyIdeas(content string, bucket string) ([]string, error)
}

const (
	BucketText    = "text"
	BucketImage   = "image"
	BucketArticle = "article"
)

const (
	ProviderOllama = "ollama"
	ProviderGroq   = "groq"
	ProviderOpenAI = "openai"
)
