# Phase 2.6: Search Overview

> Agent instruction file. Read all of it before writing any code.

---

## What This Phase Does

Adds an on-demand AI answer above search results — the "Google + Gemini"
pattern. The user explicitly triggers it; it is never automatic and there
is deliberately **no config option**.

- **PWA**: an "AI Answer" button on the search results view. Clicking
  re-fetches the same query with `overview=true` and renders the answer in
  a box above the results, with `[1]`-style citations linking to entries.
- **kl**: `kl search "query" --answer` prints a themed overview block above
  the result list.
- **API**: `GET /v1/search?q=…&overview=true` returns
  `{results: [...], overview: {"text": "...", "citations": [0, 2]} | null}`.

Generation reuses the search we already run: the top-K results' chunk
excerpts and titles become the RAG context for one LLM call. No new
retrieval machinery, no reranker dependency.

Fail-open contract: if overview generation fails (LLM down, timeout,
garbage), the response still carries full `results` with `overview: null`
and a warn log. Search itself must NEVER fail because of the answer.

---

## Files To Read First

```
docs/CLI_RULES.md            ← output theming, spinner rules (2-4s op → spinner ok)
internal/api/search.go       ← current handler + hybrid merge
internal/llm/interface.go    ← where the overview call lives
docs/phases/v1.1/phase-1-entities.md   ← fail-open precedent (JSON parse)
```

---

## Files To Modify / Create

```
internal/constants/constants.go     ← SearchOverview system prompt
internal/llm/interface.go           ← SummarizeWithContext(context, query) or reuse Generate path
internal/llm/ollama.go              ← implementation
internal/api/search.go              ← overview param handling + assembly
cmd/kl/... (search command)         ← --answer flag + themed rendering
internal/api/ui/                    ← AI Answer button + box (PWA source, rebuild assets)
docs/api/openapi.yaml               ← parameter + response schema
```

---

## Step 1 — Prompt

Add to `DefaultSystemPrompts`:

```
You are a precise answer engine over a personal knowledge base. Given a
question and numbered note excerpts, write a short standalone answer
(3-6 sentences) grounded ONLY in those excerpts. Cite sources inline as
[n] matching the excerpt numbers. If the excerpts do not contain enough
information, say so plainly — never speculate. Plain prose, no markdown
headers, no bullet lists unless comparing items.
```

User template: `"Question: %s\n\nExcerpts:\n%s"` where excerpts are
`[1] (title, date) excerpt-text` lines, top-K with K = min(5, len(results)).

## Step 2 — API

In the search handler:

1. Parse `overview` bool query param (absent/false → exactly today's behavior;
   assert zero extra LLM calls in tests).
2. Run search unchanged first.
3. If overview requested AND len(results) > 0:
   - Assemble excerpt block from `results[i].Excerpt` (+ Title, CreatedAt)
   - One `GenerateWithSystemTemp(system, user, default temp)` call
   - Extract cited indices by scanning for `[n]` tokens within range;
     clamp/drop out-of-range refs
   - On ANY error: log warn, return null overview — do not error the request
4. Empty results → null overview without calling the LLM.

Response shape (additive — old clients ignore unknown fields):

```json
{
  "results": [ ...unchanged... ],
  "overview": { "text": "...", "citations": [0, 2] }
}
```

## Step 3 — kl

`--answer` flag: after printing nothing yet, fetch with overview=true;
render overview in a rounded panel (`theme.Panel` border) above results,
muted citation markers, then the normal list. Spinner while generating
(covered by existing >200ms spinner rule). On null overview print a muted
one-liner ("AI answer unavailable") instead of failing.

## Step 4 — PWA

Button beside the search input ("AI Answer", sparkles icon acceptable).
Behavior: disabled until ≥1 result exists; click → second fetch with
`&overview=true`; render box above results; `[n]` tokens clickable → scroll
to that result card. Loading state on button during generation.

---

## Tests

- API handler (mock LLM):
  - no param → llm.Generate* never called; response identical to today's shape
  - param + results → overview text returned; citations extracted/clamped
    (`[7]` with 3 results → dropped)
  - param + LLM error → 200, results intact, overview null
  - param + zero results → 200, overview null, no LLM call
- Citation extraction: pure function table test
- kl rendering: manual per MANUAL_TESTING.md conventions

## Checklist

- [ ] Prompt added; extraction helper red-green
- [ ] Handler param wiring + four mock-LLM cases green
- [ ] openapi.yaml updated
- [ ] kl --answer themed output
- [ ] PWA button + box; `go build` embeds new assets
- [ ] go vet / golangci-lint / go test ./... green
- [ ] Manual: search → click AI Answer → box appears with working citation links

## Hard Rules

1. No config option. The param/button IS the switch.
2. Overview failure never fails search (fail-open, warn logged).
3. Never call the LLM when results are empty or param absent.
4. Citations out of range are dropped, never passed through blindly.
5. Answer prompts get ONLY result excerpts/titles — never raw vault paths
   beyond what results already expose.
