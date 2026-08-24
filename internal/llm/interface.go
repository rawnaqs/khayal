package llm

type LLM interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Generate(prompt string) (string, error)
	GenerateWithSystem(system, user string) (string, error)
	DescribeImage(imagePath string) (string, error)
	Ping() error
	Type() string
}

type LLMExt interface {
	LLM
	ExtractTags(content string, bucket string) ([]string, error)
	Summarize(content string, bucket string) (string, error)
	ExtractKeyIdeas(content string, bucket string) ([]string, error)
	ExtractEntities(content string, bucket string) (EntityResult, error)
}

// EntityResult is the raw structured-entity output from the LLM,
// before any normalization. Defined here rather than in ingest to
// avoid an import cycle.
type EntityResult struct {
	People  []string `json:"people"`
	Amounts []string `json:"amounts"`
	Dates   []string `json:"dates"`
	Places  []string `json:"places"`
	Orgs    []string `json:"orgs"`
	URLs    []string `json:"urls"`
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
