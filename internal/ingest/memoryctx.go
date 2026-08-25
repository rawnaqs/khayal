package ingest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/memory"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

// assembleMemoryContext builds the per-capture injection block from static
// config, the LLM-maintained memory file, the entity glossary, and topical
// recall. Every retrieval failure degrades silently — enrichment must never
// break because memory gathering hiccuped.
func assembleMemoryContext(ctx context.Context, q *queue.Queue, v *vault.Writer,
	llmClient interface {
		Embed(text string) ([]float32, error)
	},
	memCfg config.MemoryConfig, content string) string {

	if !config.IsOn(memCfg.Enabled) || q == nil || strings.TrimSpace(content) == "" {
		return ""
	}

	var recall []string
	if emb, err := llmClient.Embed(content); err == nil && len(emb) > 0 {
		if matches, err := q.TopSimilarChunks(ctx, emb, memory.MaxRecall, 0.40,
			time.Now().UTC(), ""); err == nil {
			for _, m := range matches {
				recall = append(recall, m.Content)
			}
		}
	}

	glossary, _ := q.GetEntityGlossary(ctx, 30)

	memFile := ""
	if path := filepath.Join(v.InboxPath(), filepath.Base(memCfg.File)); memCfg.File != "" {
		if b, err := os.ReadFile(path); err == nil {
			memFile = string(b)
		}
	}

	var matchedPeople, matchedOrgs []string
	lower := strings.ToLower(content)
	for name := range memCfg.People {
		if strings.Contains(lower, strings.ToLower(name)) {
			matchedPeople = append(matchedPeople, name)
		}
	}
	for name := range memCfg.Orgs {
		if strings.Contains(lower, strings.ToLower(name)) {
			matchedOrgs = append(matchedOrgs, name)
		}
	}

	return memory.BuildContextBlock(memory.Sources{
		UserContext:   memCfg.UserContext,
		People:        memCfg.People,
		Orgs:          memCfg.Orgs,
		MatchedPeople: matchedPeople,
		MatchedOrgs:   matchedOrgs,
		MemoryFile:    memFile,
		Glossary:      glossary,
		Recall:        recall,
	})
}

// setCallContext attaches the block to clients that support per-job context
// (Ollama) and returns a cleanup func. Other implementations are unaffected.
func setCallContext(llmClient llm.LLMExt, block string) (clear func()) {
	type ctxClient interface {
		SetCallContext(string)
		ClearCallContext()
	}
	if cc, ok := llmClient.(ctxClient); ok && block != "" {
		cc.SetCallContext(block)
		return cc.ClearCallContext
	}
	return func() {}
}
