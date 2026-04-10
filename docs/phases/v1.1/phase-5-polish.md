# Phase 5: Polish

> Final integration, testing, documentation. Updated: 2026-04-11

## Goals

- [ ] Run lint (go vet, golangci-lint)
- [ ] Run tests
- [ ] Update PLAN.md
- [ ] Update SPEC.md for v1.1
- [ ] Build verification
- [ ] Run go mod tidy

## Existing Code (Verified)

- **go.mod** — Already exists (go 1.25.8)
- **Build commands** — Already work: `go build -o khayal ./cmd/khayal`
- **Test framework** — Standard Go tests: `go test ./...`

## Step 5.1: Lint

```bash
go vet ./...
golangci-lint run
```

Fix any errors. Remember per RULES.md:
- Never `io.ReadAll` — always stream decode
- Never accumulate all embeddings — use heap
- Store path only in jobs, not content
- HTTP body: MaxBytesReader + defer Close
- strings.Builder in loops
- Never defer in loops

## Step 5.2: Tests

```bash
go test ./internal/connections/...
go test ./internal/backup/...
go test ./cmd/khayal/commands/...
go test ./...
```

## Step 5.3: Update PLAN.md

Update docs/PLAN.md for v1.1:

```markdown
### Phase 1: Chunking
...

### Phase 2: Entity Extraction
...

### Phase 3: Proactive Connections
...

### Phase 4: Vault Commands
...

### Phase 5: Backup/Restore
...
```

## Step 5.4: Update SPEC.md

Mark v1.1 items as complete:

```markdown
### 1. Chunking
- [x] Target: 150-200 words per chunk
- [x] Overlap: 30 words
- [x] Split on paragraphs
- [x] Minimum chunk size: 50 words

### 2. Entity Extraction
- [x] Extract people, amounts, dates, places, orgs, urls
- [x] Frontmatter entities field
- [x] entities table

### 3. Proactive Connections (v1.1)
- [x] Semantic similar (similarity > 0.85)
- [x] Same person
- [x] Same amount
```

## Step 5.5: Build Verification

```bash
go build -o khayal ./cmd/khayal
go build -o kl ./cmd/kl

./khayal version
./kl version
```

Verify both binaries build and show version.

## Checklist

- [ ] go vet clean
- [ ] golangci-lint clean
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] PLAN.md updated
- [ ] SPEC.md updated
- [ ] Binary builds (khayal + kl)
- [ ] go mod tidy run

## Notes

- All output via theme package per CLI_RULES.md
- RULES.md memory management requirements
- Build both khayal and kl binaries
- Update PLAN.md v1.1 section after all phases complete
- Mark SPEC.md items as [x] for v1.1 features