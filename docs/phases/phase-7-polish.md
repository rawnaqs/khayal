# Phase 7: Polish

> Release preparation, CI/CD, documentation. Updated: 2026-03-17

## Goals

- [ ] Dependency checker
- [ ] CI workflow
- [ ] GoReleaser config
- [ ] Docker Compose
- [ ] README, CONTRIBUTING
- [ ] Example config

## Step 7.1: Dependency Checker

**File:** `install/check.go`

### Requirements

- Run on `khayal start`
- Check Ollama, ffmpeg, yt-dlp
- Show install instructions
- Allow continue anyway for optional deps

```go
package install

import (
    "fmt"
    "os/exec"
    "strings"
)

type Dependency struct {
    Name        string
    Required    bool
    InstallCmd  string
    CheckArgs   []string
    CheckOutput string // substring to find
}

var Dependencies = []Dependency{
    {
        Name:       "Ollama",
        Required:   true,
        InstallCmd: "brew install ollama",
        CheckArgs:  []string{"--version"},
        CheckOutput: "ollama",
    },
    {
        Name:       "ffmpeg",
        Required:   false,
        InstallCmd: "brew install ffmpeg",
        CheckArgs:  []string{"-version"},
        CheckOutput: "ffmpeg version",
    },
    {
        Name:       "yt-dlp",
        Required:   false,
        InstallCmd: "brew install yt-dlp",
        CheckArgs:  []string{"--version"},
        CheckOutput: "yt-dlp",
    },
}

type CheckResult struct {
    Name     string
    Found    bool
    Location string
    Error    string
}

func CheckDependencies() ([]CheckResult, error) {
    var results []CheckResult
    
    for _, dep := range Dependencies {
        result := CheckResult{Name: dep.Name}
        
        cmd := exec.Command(dep.CheckArgs[0], dep.CheckArgs[1:]...)
        output, err := cmd.Output()
        
        if err != nil {
            result.Found = false
            result.Error = fmt.Sprintf("not found. Install: %s", dep.InstallCmd)
            results = append(results, result)
            continue
        }
        
        if dep.CheckOutput != "" && !strings.Contains(string(output), dep.CheckOutput) {
            result.Found = false
            results = append(results, result)
            continue
        }
        
        result.Found = true
        result.Location = "found"
        results = append(results, result)
    }
    
    return results, nil
}

func CheckOllamaConnection(host string) error {
    // Try to connect to Ollama API
    resp, err := http.Get(host + "/api/tags")
    if err != nil {
        return fmt.Errorf("ollama not reachable at %s", host)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("ollama returned status %d", resp.StatusCode)
    }
    
    return nil
}
```

### Usage in main

```go
func main() {
    // Check dependencies
    results, err := install.CheckDependencies()
    if err != nil {
        log.Error().Err(err).Msg("dependency check failed")
    }
    
    for _, r := range results {
        if r.Found {
            log.Info().Str("dep", r.Name).Msg("✓ found")
        } else {
            log.Warn().Str("dep", r.Name).Str("msg", r.Error).Msg("✗ not found")
        }
    }
    
    // Continue with startup...
}
```

## Step 7.2: CI Workflow

**File:** `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          
      - name: Download dependencies
        run: go mod download
        
      - name: Run tests
        run: go test -v -race ./...
        
      - name: Run vet
        run: go vet ./...
        
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          
  build:
    needs: test
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          
      - name: Build
        run: go build -o khayal ./cmd/khayal
        
      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: khayal
          path: khayal
```

## Step 7.3: Release Config

**File:** `.goreleaser.yml`

```yaml
project_name: khayal

before:
  hooks:
    - go mod download

builds:
  # Full server binary - requires CGO for SQLite
  - id: khayal
    main: ./cmd/khayal
    binary: khayal
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/rawnaqs/khayal/internal/config.Version={{.Version}}

  # Thin HTTP client binary - no CGO needed
  - id: kl
    main: ./cmd/kl
    binary: kl
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w

archives:
  - id: khayal
    builds:
      - khayal
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

  - id: kl
    builds:
      - kl
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ .Tag }}-next"

changelog:
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'

release:
  github:
    owner: rawnaqs
    name: khayal
  draft: false
  prerelease: auto

# Two Homebrew taps
brew:
  # Server binary
  - name: khayal
    tap:
      owner: rawnaqs
      name: homebrew-tap
    folder: Formula
    commit_author:
      name: goreleaser
      email: noreply@rawnaqs.github.com

  # Client binary
  - name: kl
    tap:
      owner: rawnaqs
      name: homebrew-tap
    folder: Formula
    commit_author:
      name: goreleaser
      email: noreply@rawnaqs.github.com
  folder: Formula
  name: khayal

docker:
  - image_templates:
      - ghcr.io/rawnaqs/khayal:{{ .Tag }}
      - ghcr.io/rawnaqs/khayal:latest
    dockerfile: Dockerfile
    context: .
```

### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o khayal ./cmd/khayal

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/khayal .
EXPOSE 7766
ENTRYPOINT ["./khayal", "start"]
```

## Step 7.4: Docker Compose

**File:** `docker-compose.yml`

```yaml
version: '3.8'

services:
  khayal:
    image: ghcr.io/rawnaqs/khayal:latest
    ports:
      - "7766:7766"
    volumes:
      - ./brain:/brain
      - ./config:/config
      - khayal-data:/root/.config/khayal
    environment:
      - KHAYAL_VAULT_PATH=/brain
      - KHAYAL_SERVER_HOST=0.0.0.0
      - KHAYAL_LOG_FILE=/config/khayal.log
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama

volumes:
  khayal-data:
  ollama-data:
```

## Step 7.5: README

**File:** `README.md`

```markdown
# Khayal

> Your private treasury of thought. Local, secure, yours.

A local-first, privacy-focused second brain. Capture anything — text, images, articles, URLs. Process it locally with your own LLM. Search it semantically and by keyword. Your data never leaves your machine.

## Features

- **Capture** — Text, images, URLs with zero friction
- **Process** — Local LLM processing (Ollama)
- **Search** — Keyword + semantic hybrid search
- **Store** — Plain markdown, yours forever
- **Privacy** — No cloud, no data leaves your machine

## Quick Start

```bash
# Install
brew install rawnaqs/tap/khayal

# Initialize
kl init

# Capture a thought
kl "my first thought"

# Search
kl search "distributed systems"

# Or use the web UI
# Visit http://127.0.0.1:7766
```

## Installation

### Homebrew

```bash
brew install rawnaqs/tap/khayal
```

### Direct Download

```bash
curl -fsSL https://github.com/rawnaqs/khayal/releases/latest/download/install.sh | sh
```

### Docker

```bash
docker pull ghcr.io/rawnaqs/khayal
docker compose up
```

## Requirements

- Go 1.22+
- Ollama (for full functionality)

## Documentation

- [Spec](docs/khayal-spec.md)
- [Tech Stack](docs/TECH_STACK.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Implementation Plan](docs/PLAN.md)

## License

AGPLv3 — See [LICENSE](LICENSE)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

---

Made with 💜 by Rawnaqs
```

## Step 7.6: CONTRIBUTING

**File:** `CONTRIBUTING.md`

```markdown
# Contributing to Khayal

## Development Setup

```bash
git clone github.com/rawnaqs/khayal
cd khayal
go mod download
go build -o khayal ./cmd/khayal
```

## Running Tests

```bash
go test ./...
go vet ./...
golangci-lint run
```

## Code Style

- Use `go fmt`
- Run `go vet` and fix warnings
- Add tests for new features
- Document public APIs

## Submitting PRs

1. Fork the repo
2. Create a feature branch
3. Make changes
4. Run tests
5. Submit PR

## Issues

Use GitHub Issues for bugs and feature requests.
```

## Step 7.7: Example Config

**File:** `config.example.yaml`

```yaml
vault:
  path: ~/Documents/brain
  inbox_dir: inbox
  media:
    default_dir: inbox/media
    strategy:
      image: vault
      pdf: vault
      audio: config
      video: config

server:
  host: 127.0.0.1
  port: 7766
  token: ""  # Auto-generated on first run
  log_file: ~/.config/khayal/logs/khayal.log

llm:
  provider: ollama
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: llama3.2:3b
  vision_model: moondream
  fallback_provider: ""
  fallback_api_key: ""

worker:
  max_workers: 1
  max_retries: 3
  retry_backoff: exponential

db:
  path: ~/.config/khayal/khayal.db
```

## Step 7.8: .gitignore

**File:** `.gitignore`

```
# Binaries
khayal
kl
*.exe

# Config (sensitive)
config.yaml
*.log

# Database
*.db
*.db-journal

# IDE
.vscode/
.idea/

# OS
.DS_Store
Thumbs.db

# Node (PWA)
ui/react/node_modules/
ui/react/dist/

# Secrets
.env
*.pem
```

## Testing

Full integration tests:

```bash
go test -tags=integration ./...
```

## Checklist

- [ ] Dependency checker
- [ ] CI workflow (test, vet, lint)
- [ ] GoReleaser config
- [ ] Docker + Compose
- [ ] README with quick start
- [ ] CONTRIBUTING guide
- [ ] Example config
- [ ] .gitignore
- [ ] All tests passing
- [ ] go vet clean
- [ ] golangci-lint clean

## Final Build

```bash
# Build for release
goreleaser release --clean

# Or local build
go build -o khayal -ldflags="-X main.version=$(git describe --tags)" ./cmd/khayal
```

## Next Steps

- Publish to GitHub releases
- Update Homebrew tap
- Push Docker image
- Announce! 🎉

## Notes

- Version: Semantic (v0.1.0)
- Default bind: 127.0.0.1:7766
- Token: Auto-generated, shown once
