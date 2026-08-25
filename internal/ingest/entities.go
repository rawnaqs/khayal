package ingest

import (
	"context"
	"math"
	"time"
	"strconv"
	"strings"

	"github.com/rawnaqs/khayal/internal/dates"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

// toQueue converts normalized entities into the store's mirror type.
func (e Entities) toQueue() queue.NoteEntities {
	return queue.NoteEntities{
		People:        e.People,
		Amounts:       e.Amounts,
		Dates:         e.Dates,
		ResolvedDates: e.ResolvedDates,
		Places:        e.Places,
		Orgs:          e.Orgs,
		URLs:          e.URLs,
	}
}

// toVaultBlock converts normalized entities into the frontmatter block type.
func (e Entities) toVaultBlock() *vault.EntitiesBlock {
	return &vault.EntitiesBlock{
		People:          e.People,
		Amounts:         e.Amounts,
		Dates:           e.Dates,
		DateResolutions: e.ResolvedDates,
		Places:          e.Places,
		Orgs:            e.Orgs,
		URLs:            e.URLs,
	}
}

// ExtractAndSaveEntities runs entity extraction for one ingest path.
// Called sequentially after the parallel tags/summary/key-ideas phase —
// deliberately not added to the errgroup, which already loads Ollama
// with three concurrent calls. An LLM error propagates (fail-fast);
// only malformed JSON degrades to empty inside the client itself.
// notePath is assigned by the vault write that follows extraction, so the
// caller passes it back in via SaveEntitiesForNote after WriteNote.
func ExtractAndSaveEntities(llmClient llm.LLMExt, content, bucket string) (Entities, error) {
	raw, err := llmClient.ExtractEntities(content, bucket)
	if err != nil {
		return Entities{}, err
	}
	return NormalizeEntities(raw), nil
}

// SaveEntitiesForNote persists a note's normalized entities once its
// notePath is known.
func SaveEntitiesForNote(ctx context.Context, q *queue.Queue, notePath string, ents Entities) error {
	return q.SaveEntities(ctx, notePath, ents.toQueue())
}

// Entities holds structured entities extracted from note content.
// All fields are slices — empty slice means no entities of that type.
type Entities struct {
	People  []string `json:"people"`
	Amounts []string `json:"amounts"`
	Dates   []string `json:"dates"`
	// ResolvedDates is index-aligned with Dates: absolute dates for
	// relative references, empty string when a date did not resolve.
	ResolvedDates []string `json:"resolved_dates,omitempty"`
	Places        []string `json:"places"`
	Orgs          []string `json:"orgs"`
	URLs          []string `json:"urls"`
}

// ResolveRelativeDates fills ResolvedDates for any relative date
// references ("tomorrow", "in 3 days") against the given capture time.
func (e *Entities) ResolveRelativeDates(now time.Time) {
	e.ResolvedDates = make([]string, len(e.Dates))
	for i, d := range e.Dates {
		if t, ok := dates.ResolveRelative(d, now); ok {
			e.ResolvedDates[i] = t.Format("2006-01-02")
		}
	}
}

// personStoplist are pronouns/author references LLMs occasionally emit as
// "people". Single letters are rejected outright.
var personStoplist = map[string]bool{
	"i": true, "me": true, "we": true, "us": true,
	"he": true, "she": true, "it": true, "they": true, "them": true,
	"him": true, "her": true,
	"you": true, "myself": true, "ourselves": true,
}

// isPlausiblePerson filters junk the extractor sometimes emits: single
// letters and first-person/pronoun entries.
func isPlausiblePerson(name string) bool {
	t := strings.TrimSpace(name)
	if len(t) <= 1 {
		return false
	}
	return !personStoplist[strings.ToLower(t)]
}

// NormalizeEntities applies normalization to raw LLM entity output:
// amounts become plain integer strings, people lose short-form
// duplicates and pronoun junk, and the remaining types pass through
// unchanged.
func NormalizeEntities(raw llm.EntityResult) Entities {
	people := normalizeNames(raw.People)
	filtered := make([]string, 0, len(people))
	for _, p := range people {
		if isPlausiblePerson(p) {
			filtered = append(filtered, p)
		}
	}
	return Entities{
		People:  filtered,
		Amounts: normalizeAmounts(raw.Amounts),
		Dates:   raw.Dates,
		Places:  raw.Places,
		Orgs:    raw.Orgs,
		URLs:    raw.URLs,
	}
}

func normalizeAmounts(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if n := normalizeAmount(s); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// normalizeAmount converts an amount string to a plain integer string.
// Examples: "$2,000" → "2000", "2k" → "2000", "2.5k" → "2500",
// "£1m" → "1000000", "1,234.56" → "1235". Unparseable input returns "".
func normalizeAmount(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "$£€¥")
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return ""
	}

	mult := 1.0
	switch suffix := strings.ToLower(s[len(s)-1:]); suffix {
	case "k":
		mult = 1000
		s = s[:len(s)-1]
	case "m":
		mult = 1000000
		s = s[:len(s)-1]
	case "b":
		mult = 1000000000
		s = s[:len(s)-1]
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 || math.IsInf(v, 0) {
		return ""
	}
	return strconv.FormatInt(int64(math.Round(v*mult)), 10)
}

// normalizeNames deduplicates person names: identical entries collapse,
// and a shorter name is dropped when all its words appear as whole words
// inside a longer one ("John" ⊂ "John Doe", but "Jo" ⊄ "John").
func normalizeNames(names []string) []string {
	cleaned := make([]string, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		t := strings.TrimSpace(n)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		cleaned = append(cleaned, t)
	}

	out := make([]string, 0, len(cleaned))
	for i, a := range cleaned {
		aWords := strings.Fields(strings.ToLower(a))
		subset := false
		for j, b := range cleaned {
			if i == j {
				continue
			}
			bWords := strings.Fields(strings.ToLower(b))
			if len(bWords) <= len(aWords) {
				continue
			}
			if containsAllWords(bWords, aWords) {
				subset = true
				break
			}
		}
		if !subset {
			out = append(out, a)
		}
	}
	return out
}

func containsAllWords(haystack, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, w := range haystack {
		set[w] = true
	}
	for _, w := range needles {
		if !set[w] {
			return false
		}
	}
	return true
}
