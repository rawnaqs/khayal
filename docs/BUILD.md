# Build Instructions

## Build Tags

This project requires specific build tags for SQLite features:

```bash
# Basic build (FTS5 disabled)
go build ./...

# With FTS5 (full-text search)
go build -tags "fts5" ./...

# With FTS5 and sqlite-vec (vector search)
go build -tags "fts5,sqlite_vec" ./...
```

## All Build Commands

Every `go build`, `go test`, and `go run` command must include the appropriate build tags:

```bash
# Build the binary
go build -tags "fts5" -o khayal ./cmd/khayal

# Run tests
go test -tags "fts5" ./...

# Run tests with verbose
go test -tags "fts5" -v ./...

# Run the binary
go run -tags "fts5" ./cmd/khayal
```

## Build Tag Matrix

| Tag | Features | Use Case |
|-----|----------|----------|
| (none) | Basic queue, jobs | Development without search |
| `fts5` | Full-text search | Production with FTS5 |
| `fts5,sqlite_vec` | Vector search | Production with embeddings |

## Requirements for FTS5

### macOS

The default SQLite on macOS doesn't include FTS5. Install with Homebrew:

```bash
brew install sqlite3
```

Or use the tagged mattn/go-sqlite3 build which includes FTS5:

```bash
go build -tags "fts5"
```

### Linux

Most Linux distributions include FTS5 by default. If not:

```bash
# Debian/Ubuntu
sudo apt-get install libsqlite3-dev

# Build with FTS5
go build -tags "fts5"
```

## Phase 7 Files

When Phase 7 is implemented, add build tags to:

- **Makefile**
  ```
  .PHONY: build test
  build: go build -tags "fts5" -o khayal ./cmd/khayal
  test: go test -tags "fts5" ./...
  ```

- **.goreleaser.yml**
  ```yaml
  builds:
    - id: khayal
      main: ./cmd/khayal
      env:
        - CGO_CFLAGS=-DSQLITE_ENABLE_FTS5
  ```

- **.github/workflows/ci.yml**
  ```yaml
  - name: Run tests
    run: go test -tags "fts5" -v ./...
  ```
