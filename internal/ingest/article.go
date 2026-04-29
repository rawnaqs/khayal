package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/PuerkitoBio/goquery"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

func IngestArticle(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt) (string, error) {
	title, content, err := scrapeArticle(job.SourceURL)
	if err != nil {
		return "", fmt.Errorf("failed to scrape article: %w", err)
	}

	combinedContent := title + "\n\n" + content

	g, _ := errgroup.WithContext(ctx)

	var summary string
	var keyIdeas []string
	var tags []string

	g.Go(func() error {
		var err error
		summary, err = llmClient.Summarize(combinedContent, llm.BucketArticle)
		return err
	})

	g.Go(func() error {
		var err error
		keyIdeas, err = llmClient.ExtractKeyIdeas(combinedContent, llm.BucketArticle)
		return err
	})

	g.Go(func() error {
		var err error
		tags, err = llmClient.ExtractTags(combinedContent, llm.BucketArticle)
		return err
	})

	if err := g.Wait(); err != nil {
		return "", fmt.Errorf("llm extraction failed: %w", err)
	}

	if tags == nil {
		tags = []string{"article"}
	}

	now := time.Now().UTC()
	note := &vault.Note{
		Metadata: vault.NoteMetadata{
			Created:   job.CreatedAt,
			Updated:   &now,
			Type:      "article",
			Status:    "done",
			SourceURL: job.SourceURL,
			Tags:      tags,
			History: []vault.HistoryEvent{
				{At: now, Event: "processed"},
			},
		},
		Title:    title,
		Summary:  summary,
		KeyIdeas: keyIdeas,
		Raw:      combinedContent,
	}

	notePath, err := v.WriteNote(note, job.ID)
	if err != nil {
		return "", fmt.Errorf("failed to write note: %w", err)
	}

	if err := q.IndexNote(ctx, notePath, title, combinedContent, strings.Join(tags, ",")); err != nil {
		return "", fmt.Errorf("failed to index note: %w", err)
	}

	embedContent := title + "\n\n" + summary + "\n\n" + strings.Join(keyIdeas, "\n")
	embedding, err := llmClient.Embed(embedContent)
	if err != nil {
		return notePath, nil
	}

	if err := q.SaveChunk(ctx, notePath, 0, combinedContent, embedding); err != nil {
		return notePath, nil
	}

	return notePath, nil
}

func scrapeArticle(url string) (title, content string, err error) {
	logger := slog.Default()

	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch URL: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	title = doc.Find("title").First().Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}
	title = strings.TrimSpace(title)

	doc.Find("script, style, nav, header, footer, aside, .advertisement, .sidebar, .comments").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	var paragraphs []string
	doc.Find("article, main, .content, .post, .entry, .article-content, .post-content, .story-body").Each(func(i int, s *goquery.Selection) {
		s.Find("p, h2, h3, h4, h5, h6, li, blockquote, pre, code, figure, figcaption").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" && len(text) > 10 {
				paragraphs = append(paragraphs, text)
			}
		})
	})

	if len(paragraphs) == 0 {
		doc.Find("p").Each(func(i int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" && len(text) > 10 {
				paragraphs = append(paragraphs, text)
			}
		})
	}

	content = strings.Join(paragraphs, "\n\n")



	logger.Debug("scraped article", "title", title, "content_length", len(content))

	return title, content, nil
}
