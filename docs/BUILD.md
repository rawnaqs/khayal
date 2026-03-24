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

## Phase 7 Files

When Phase 7 is implemented, add build commands to:

- **Makefile**
  ```
  .PHONY: build test
  build: go build -o khayal ./cmd/khayal
  test: go test ./...
  ```

- **.goreleaser.yml**
  ```yaml
  builds:
    - id: khayal
      main: ./cmd/khayal
  ```

- **.github/workflows/ci.yml**
  ```yaml
  - name: Run Go tests
    run: go test -v ./...

  - name: Install frontend dependencies
    run: cd external/react && npm ci

  - name: Run frontend tests
    run: cd external/react && npm run test:run

  - name: Build frontend
    run: cd external/react && npm run build

  - name: Build binary (includes embedded PWA)
    run: go build -o khayal ./cmd/khayal
  ```
