# Build Instructions

## Build

**No build tags required!** Khayal uses:
- `modernc.org/sqlite` (pure Go SQLite) - no CGO, no system dependencies
- Pure Go cosine similarity for vector search

```bash
# Build the binary
go build -o khayal ./cmd/khayal

# Run tests
go test ./...

# Run the binary
./khayal
```

## Requirements

None! Pure Go dependencies with no external requirements.

- FTS5 is included
- Vector search in pure Go (no external deps)
- No Homebrew sqlite3 needed
- No libsqlite3-dev needed
- Works on macOS, Linux, Windows out of the box

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
  - name: Run tests
    run: go test -v ./...
  ```
