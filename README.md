# Khayal

> Your private treasury of thought. Local, secure, yours.

A local-first, privacy-focused second brain. Capture anything — text, images, URLs. Process locally with your own LLM. Search semantically and by keyword. Your data never leaves your machine.

## Features

- **Capture** — Text, images, URLs with zero friction
- **Process** — Local LLM processing (Ollama)
- **Search** — Keyword + semantic hybrid search
- **Store** — Plain markdown, yours forever
- **Privacy** — No cloud, no data leaves your machine
- **Update Notifications** — Built-in update checker via GitHub releases

## Requirements

- Ollama (for LLM features)

## Quick Start

```bash
# Install (includes khayal + kl)
brew install rawnaqs/tap/khayal

# Or use the one-liner installer
curl -fsSL https://raw.githubusercontent.com/rawnaqs/khayal/main/install.sh | sh

# Initialize
khayal init
kl init

# Start (foreground)
khayal start

# Or run as a service (macOS/Linux)
brew services start khayal

# Capture a thought
kl "my first thought"

# Search
kl search "distributed systems"

# Or use the web UI
# Visit http://127.0.0.1:1133
```

## Installation

### Homebrew

```bash
brew install rawnaqs/tap/khayal
```

### One-liner

```bash
curl -fsSL https://raw.githubusercontent.com/rawnaqs/khayal/main/install.sh | sh
```

### Docker

```bash
docker compose up
```

## Environment Variables

- `KHAYAL_CONFIG` — Path to config file (default: `~/.config/khayal/config.yaml`)
- `KL_CONFIG` — Path to kl client config (default: `~/.config/khayal/kl.yaml`)

## Commands

### Server (`khayal`)

| Command | Description |
|---------|-------------|
| `khayal init` | First-run setup |
| `khayal start` | Start server + worker |
| `khayal stop` | Graceful shutdown |
| `khayal restart` | Stop + start |
| `khayal status` | Status dashboard |
| `khayal reindex` | Rebuild search index |
| `khayal config` | View config |

### Client (`kl`)

| Command | Description |
|---------|-------------|
| `kl "text"` | Capture text |
| `kl url "https://..."` | Capture URL |
| `kl image <imgpath>` | Capture image |
| `kl search "query"` | Search vault |
| `kl recent` | Recent captures |
| `kl stats` | Vault statistics |
| `kl status` | Server status + update check |

### Homebrew Service (macOS)

```bash
# Start as background service
brew services start khayal

# Check status
brew services list

# View logs
tail -f ~/.config/khayal/logs/khayal.log

# Stop
brew services stop khayal
```

## PWA Features

- Shows server version (not build version) in header
- Update notification icon when new version available
- Auto-refreshing server status indicator

## License

AGPLv3 — See [LICENSE](LICENSE)

---

Made by Rawnaqs
