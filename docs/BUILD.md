# Build Instructions

## Go Backend

**No build tags required!** Khayal uses:
- `modernc.org/sqlite` (pure Go SQLite) - no CGO, no system dependencies
- Pure Go cosine similarity for vector search

```bash
# Build the binary (includes embedded PWA)
go build -o khayal ./cmd/khayal

# Run tests
go test ./...

# Run the binary
./khayal
```

## Frontend (PWA)

```bash
cd external/react

# Install dependencies
npm install

# Development server (with hot reload + API proxy to :1133)
npm run dev

# Production build (outputs to ../../internal/api/ui/static/)
npm run build
```

### Test Commands

```bash
# Unit tests (Vitest) — watch mode
npm run test

# Unit tests — single run
npm run test:run

# E2E tests (Playwright) — headless
npm run test:e2e

# E2E tests — with UI
npm run test:e2e:ui
```

### Test Coverage

| Category | Files | Count |
|----------|-------|-------|
| Unit tests | `src/lib/__tests__/`, `src/hooks/__tests__/`, `src/components/*/` | 71 tests |
| E2E tests | `e2e/capture.spec.ts`, `e2e/search.spec.ts`, `e2e/offline.spec.ts` | 60 tests |

### PWA Build

The build uses `vite-plugin-pwa` which:
1. Bundles React app to `internal/api/ui/static/`
2. Generates `manifest.webmanifest` from `vite.config.ts`
3. Generates `sw.js` with Workbox precache + runtime caching
4. Generates `registerSW.js` for service worker registration

## Release

```bash
# Tag and push to trigger release
git tag v0.1.0
git push origin v0.1.0

# GitHub Actions runs goreleaser:
# 1. Builds React PWA (syncs version from git tag to package.json)
# 2. Builds khayal + kl (darwin/linux, amd64/arm64)
# 3. Creates GitHub release with archives
# 4. Updates rawnaqs/homebrew-tap
```

### Version Sync

GoReleaser reads the version from the git tag (`{{.Version}}`) and:
- Sets Go binary version via ldflags: `-X github.com/rawnaqs/khayal/internal/version.Version={{.Version}}`
- Updates package.json version: `npm version {{.Version}} --no-git-tag-version`

This ensures both Go and PWA binaries have matching versions.

### Homebrew Install

```bash
brew install rawnaqs/tap/khayal
```

### Homebrew Service (Background)

```bash
# Start as a background service
brew services start khayal

# Check status
brew services list | grep khayal

# View logs
tail -f ~/.config/khayal/logs/khayal.log

# Stop service
brew services stop khayal
```

### One-liner Install

```bash
curl -fsSL https://raw.githubusercontent.com/rawnaqs/khayal/main/install.sh | sh
```

### Docker

**Prerequisites:** Run [Ollama](https://ollama.com) locally for GPU acceleration.

```bash
docker run \
  -v ~/Documents/brain:/vault \
  -v ~/.config/khayal:/root/.config/khayal \
  -p 1133:1133 \
  ghcr.io/rawnaqs/khayal
```

## Requirements

### Go
None! Pure Go dependencies with no external requirements.
- FTS5 is included
- Vector search in pure Go (no external deps)
- No Homebrew sqlite3 needed
- No libsqlite3-dev needed
- Works on macOS, Linux, Windows out of the box

### Frontend
- Node.js 18+
- npm
