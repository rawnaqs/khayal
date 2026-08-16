# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

A single technical user who runs the khayal server on their primary machine (typically macOS) and captures thoughts from any device on the local network or via Tailscale. They work with ideas, code, meetings, research, and personal knowledge daily. Their tools include a terminal, a browser, and markdown-based note systems like Obsidian. They value privacy, ownership, and speed over collaboration features or cloud convenience.

Non-technical users are an explicit v2.0 concern, not a current design target.

## Product Purpose

Khayal is a local-first, privacy-respecting second brain. It makes knowledge capture and retrieval frictionless: capture text, images, or URLs through a CLI (`kl`) or PWA; process everything locally with your own LLM via Ollama; search by keyword and meaning; and store everything as plain markdown you own forever. Success means a user captures more thoughts than they lose, finds things faster than they forget them, and never worries about where their data lives.

## Positioning

The only second brain that combines zero-friction multi-surface capture (CLI + PWA), fully local LLM processing, hybrid FTS5 + semantic search, and plain markdown storage with no cloud dependency. Complementary to Obsidian (which opens the same vault). Not a SaaS, not a chat interface, not a graph database.

A competing product could not truthfully claim: local LLM processing on the user's own hardware, zero cloud dependency, and plain markdown output — all three simultaneously.

## Operating Context

- The server runs on a single machine (typically a Mac), started as a foreground process or Homebrew service
- Clients talk to the server over HTTP with token auth, from laptops, phones, or the same machine
- Capture happens in moments: a thought during a meeting, a URL from a browser, a photo of a whiteboard
- Search happens during work: finding a past decision, retrieving a person's name, checking what was said
- The vault is plain markdown — openable by Obsidian, grep, or any text tool
- Processing is asynchronous for media/URLs, synchronous for text

## Capabilities and Constraints

**Capabilities:**
- Capture text (CLI + PWA), URLs (CLI + PWA), and images (CLI + PWA)
- Hybrid search: FTS5 keyword (BM25) + semantic (embeddings) merged via Reciprocal Rank Fusion
- Local LLM processing via Ollama: tag extraction, summarization, key idea extraction, image description
- Groq and OpenAI as optional LLM fallbacks
- PWA with offline capture queue (IndexedDB + background sync)
- CLI client (`kl`) for rapid capture and search from the terminal
- Server admin CLI (`khayal`) for setup, monitoring, reindexing, and vault maintenance
- Plain markdown vault with YAML frontmatter, atomic writes, file locking
- Automatic update checking via GitHub releases
- Homebrew distribution, Docker support, single-binary deployment

**Constraints:**
- macOS and Linux only (server). PWA works on any device
- Ollama required for full LLM functionality
- Single-user, single-server architecture
- Never writes outside the configured inbox directory within the vault
- Plain markdown format is non-negotiable
- Vault safety contract (10 rules) governs all file operations
- Token auth required on every API request
- Dark-only visual theme
- AGPLv3 license

**Undecided:**
- Formal accessibility standard (best-effort for now)
- When or whether to introduce multi-user support

## Brand Commitments

- **Name:** Khayal (Arabic: خيال — imagination, thought). CLI shortcut: `kl`
- **Organization:** Rawnaqs — "the luster of craftsmanship"
- **Voice:** Direct, warm, personal. The CLI feels like a trusted tool, not enterprise software. Error messages tell you what to do next, never dump stack traces.
- **Visual identity:** Gold-on-near-black dark palette. Abstract flame/lamp icon representing illumination of thought. IBM Plex Mono for UI, Bricolage Grotesque for display and headings. Gold gradient accents. Minimal, luxury aesthetic.
- **Design system:** `github.com/rawnaqs/theme` is the single source of truth for all colors and typography. No color or type is defined inline in Khayal code.
- **Module path:** `github.com/rawnaqs/khayal`

## Evidence on Hand

- `docs/SPEC.md` — Full v1.0 specification with CLI UX, API, architecture, and roadmap through v2.0
- `docs/ARCHITECTURE.md` — System design, component responsibilities, data flow
- `docs/UI_SPEC.md` — PWA implementation specification with component structure and design tokens
- `docs/Vault.md` — Vault structure, path handling, and safety guarantees
- `external/react/` — Fully built React PWA source
- `internal/api/ui/static/` — Built PWA assets (embedded in binary)
- `internal/` — Complete Go backend implementation
- `cli/` — Complete `khayal` and `kl` CLI implementations
- `config.example.yaml` — Full configuration reference
- `README.md` — Public project readme
- `LICENSE` — AGPLv3
- Icon assets: `icon.svg`, `icon-192.png`, `icon-512.png`

**Absences design work must respect:**
- No user testimonials or case studies exist — do not fabricate
- No usage metrics or adoption data — do not invent
- No customer logos or third-party endorsements — do not imply

## Product Principles

1. **Your data, your machine.** Every byte of user content stays local. The cloud is never a dependency, only an optional fallback.
2. **Friction kills capture.** A thought must land in the vault in under 100ms from the CLI, in one tap from the PWA. Processing happens after, never before.
3. **Plain text is the ultimate format.** Markdown in a real folder, openable by anything, durable for decades. No proprietary lock-in.
4. **The tool recedes.** Capture, search, done. The interface should feel inevitable, not demanding. Brand lives in precise details, not ornament.
5. **Privacy is not a feature — it's the foundation.** Local-first is the architecture, not a checkbox. Token auth, no telemetry, no accounts.

## Accessibility & Inclusion

Best-effort accessible. The PWA targets reasonable usability with touch targets >= 44px, minimum 16px font size on inputs (prevents iOS zoom), keyboard-operable search, and semantic HTML structure. No formal WCAG conformance level is mandated at this stage.
