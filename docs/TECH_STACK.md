# Khayal Tech Stack

> Technology decisions for Khayal. Updated: 2026-09-03

## Core

| Category | Choice | Rationale |
|----------|--------|------------|
| Language | Go 1.25+ | Performance, single binary, native HTTP |
| License | AGPLv3 | Copyleft, protects modifications |
| Org | Rawnaqs | "The luster of craftsmanship" |

## Server & API

| Component | Choice | Version | Rationale |
|-----------|--------|---------|------------|
| HTTP Router | Chi | latest | Lightweight, idiomatic Go, middleware support |
| Server | Go net/http | stdlib | No external dependency |
| Auth | Token (X-Khayal-Token header) | - | Simple, effective, no session management |
| Logging | log/slog | stdlib | Structured, multi-handler (console + rotating file) |
| WebSocket | gorilla/websocket | v1.5 | Live queue updates (/v1/queue/ws) |

## Database

| Component | Choice | Rationale |
|-----------|--------|------------|
| SQLite Driver | modernc.org/sqlite | Pure Go, no CGO, no system dependencies |
| Job Queue | SQLite | Built-in, reliable |
| Full-Text Search | SQLite FTS5 | Built-in |
| Vector Search | Pure Go cosine similarity | No external dependencies, batch processing |
| Backup Encryption | filippo.io/age | v1.2.0 | Embedded, pure Go; armored X25519 |

**Notes:**
- Uses `modernc.org/sqlite` for pure Go SQLite (no CGO, no system dependencies)
- Vector search implemented in Go with batch processing (1000 chunks/batch) and cosine similarity
- Results deduplicated by note_path (best scoring chunk per note)

## LLM

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Primary | Ollama | Local, private, free |
| Fallback 1 | Groq | Fast inference, good API |
| Fallback 2 | OpenAI | Universal fallback |
| Embedding Model | nomic-embed-text | Ollama default, good quality |
| Text Model | qwen2.5:3b (default config: llama3.2:3b) | Balanced size/performance |
| Consolidation Model | optional (e.g. qwen2.5:7b) | Dedicated model for memory consolidation; temp 0.2 |
| Vision Model | moondream | Lightweight, effective |

## CLI

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Command Structure | Cobra | Standard, battle-tested |
| Output Styling | Lip Gloss + rawnaqs/theme | Consistent design system |
| Markdown Rendering | Glamour | Beautiful terminal output |
| Interactive Prompts | Huh | Simple, Go-native |
| TUI Dashboard | Bubble Tea | Interactive, live updates |

## PWA (Frontend)

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Framework | React 18+ | Ecosystem, familiarity |
| Build Tool | Vite | Fast, simple, HMR |
| Routing | Tab state (single-page) | No router dependency needed |
| State | React hooks | Minimal surface |
| HTTP Client | Fetch (native) | No extra dependency |
| Animation | framer-motion | Sheet/queue/list transitions |
| Testing | Vitest + Testing Library | Unit/component |
| E2E | Playwright | Capture/search/queue flows |
| Offline | IndexedDB (idb-keyval) | Simple promise-based API |
| Styling | CSS Modules + rawnaqs/theme | Scoped, design system |

## PWA (Go Integration)

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Static Serving | Go net/http | Native, no额外 dependency |
| Embedding | embed.FS | Compile-time inclusion |
| SPA Fallback | index.html for unknown routes | Standard SPA behavior |

## Testing

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Unit Tests | Go testing | stdlib |
| Integration | Go + testutil | - |
| Frontend | Vitest | Fast, Vite integration |
| E2E | Playwright | Reliable, cross-browser |

## CI/CD

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Test | GitHub Actions | Free for open source |
| Lint | golangci-lint | Comprehensive |
| Release | GoReleaser | Binary + Homebrew + Docker |
| Docker | docker-compose | Local dev |

## External Tools (Optional)

| Tool | Purpose | Status |
|------|---------|--------|
| Ollama | Local LLM | Required for full functionality |

## Design System

| Resource | Path | Purpose |
|----------|------|---------|
| Theme | github.com/rawnaqs/theme | Colors, typography, tokens |
| Theme CSS | github.com/rawnaqs/theme/theme.css | Web variables |
| Theme Go | github.com/rawnaqs/theme/theme.go | CLI constants |

---

## Version Constraints

All Go dependencies should use major version pinning (v1, v2) to avoid breaking changes. Check `go.mod` for exact versions.

## Security

- Token: 32-byte hex, auto-generated
- Config: 600 permissions
- Default bind: 127.0.0.1 (never 0.0.0.0)
- Never log token or request body
