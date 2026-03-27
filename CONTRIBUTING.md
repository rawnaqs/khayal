# Contributing to Khayal

## Development Setup

```bash
git clone https://github.com/rawnaqs/khayal
cd khayal
go mod download

# Build both binaries
go build -o khayal ./cmd/khayal
go build -o kl ./cmd/kl
```

### Frontend

```bash
cd external/react
npm install
npm run dev    # Dev server at localhost:5173
npm run build  # Build to internal/api/ui/static/
```

## Running Tests

```bash
# Go tests
go test ./...
go vet ./...

# Frontend tests
cd external/react
npm run test:run   # Unit tests
npm run test:e2e   # E2E tests
```

## Code Style

- Use `go fmt`
- Run `go vet` and fix warnings
- Use `golangci-lint run` for static analysis
- Add tests for new features
- Document public APIs

## Commit Messages

Use conventional commits:

- `feat:` — New feature
- `fix:` — Bug fix
- `docs:` — Documentation
- `refactor:` — Code change (no feature/fix)
- `test:` — Test changes
- `chore:` — Build, CI, tooling

These generate the changelog automatically via GoReleaser.

## Submitting PRs

1. Fork the repo
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make changes
4. Run tests (`go test ./...`)
5. Run lint (`golangci-lint run`)
6. Commit with conventional message
7. Push and create PR

## Issues

Use [GitHub Issues](https://github.com/rawnaqs/khayal/issues) for bugs and feature requests.
