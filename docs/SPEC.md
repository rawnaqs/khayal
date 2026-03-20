# Khayal — v1 Project Specification

> Your private treasury of thought. Local, secure, yours.

---

## Identity

| | |
|---|---|
| **Tool name** | Khayal |
| **CLI short** | `kl` (short for khayal, short for knowledge) |
| **Org** | Rawnaqs — the luster of craftsmanship |
| **Module path** | `github.com/rawnaqs/khayal` |
| **License** | AGPLv3 |
| **Language** | Go 1.22+ |
| **Theme** | `github.com/rawnaqs/theme` |

---

## What It Is

A local-first, privacy-focused second brain. Capture anything — text, images, articles, URLs. Process it locally with your own LLM. Search it semantically and by keyword. Your data never leaves your machine.

## What It Is Not

- Not a chat interface over your notes
- Not a graph database
- Not a SaaS, subscription, or cloud service

---

## Core Philosophy

```
Capture  → zero friction, any device
Process  → immediate, local, private
Search   → fast, semantic + keyword
Store    → plain markdown, yours forever
Connect  → proactive discovery of related thoughts
```

---

## Vault Safety Contract

khayal treats your vault with extreme care. These rules are non-negotiable:

1. **Never write invalid YAML** — validate frontmatter before every write
2. **Never overwrite user edits** — check mtime, abort if modified externally
3. **Never write invalid UTF-8** — sanitize all LLM output
4. **Never create illegal filenames** — strict sanitization on all paths
5. **Never write broken wikilinks** — verify targets exist before writing
6. **Never grow frontmatter unbounded** — hard caps on all list fields
7. **Never race with Obsidian** — file locking on all writes
8. **Never hard-delete from vault** — soft-delete to `<inbox>/.khayal-trash/` only (NOT vault root)
9. **Never write outside <inbox_dir>/** — khayal's blast radius is contained
10. **Always write atomically** — temp file + rename, never partial writes

---

## CLI UX

Two distinct CLI tools with distinct audiences, jobs, and UX philosophies.

### The Two Tools

#### khayal — server administration CLI

**Audience:** You, on the Mac Air, managing the server.
**Job:** Run, monitor, configure, and maintain the khayal server.
**Feel:** Sysadmin tool. Dense, verbose, operational. You run it when something needs attention.
**When used:** Setup, troubleshooting, reindexing, checking server health.

**Mental model — khayal thinks about:**
```
the server · the vault health · the worker · the database · the backups
dependencies · logs · media · wikilinks · duplicates
```

#### kl — capture and retrieval CLI

**Audience:** You, on any machine, capturing and finding thoughts.
**Job:** Get thoughts into the vault and find them again.
**Feel:** Personal tool. Minimal, fast, warm. Used dozens of times per day.
**When used:** Every time you have a thought, find something, check what you captured.

**Mental model — kl thinks about:**
```
my thoughts · finding things · what I captured recently
my tags and people · is my capture landing
```

**Rule:** If a command requires SSH access to the Mac Air to be useful — it belongs in khayal. If a command is useful from a laptop or phone — it belongs in kl.

### kl commands — user actions only

```bash
kl "thought"              # capture text (default command)
kl --url https://...      # capture URL
kl --image ~/file.png     # capture image
kl search "query"         # search vault
kl recent                 # recent captures
kl recent --days 7        # last 7 days
kl recent --type image    # recent images only
kl browse --tag react     # all notes tagged react
kl browse --person "John" # all notes mentioning John
kl browse --amount 2000   # all notes mentioning $2000
kl stats                  # vault statistics (read-only)
kl status                 # lightweight — server reachable, queue health
kl init                   # setup ~/.config/khayal/kl.yaml (client config only)
kl config                 # view/set client config (host, token)
kl completion             # generate shell completion scripts
kl version               # kl version + connected khayal server version
```

**Removed from kl:** vault maintenance, reindex, logs — these require server access and belong in khayal.

### khayal commands — server administration only

```bash
khayal init         # first-run setup — generates config.yaml + token
khayal start        # starts server + worker, runs dep checker
khayal stop         # graceful shutdown
khayal restart      # stop + start
khayal status       # full admin dashboard (Bubble Tea TUI)
khayal reindex      # rebuild all chunk embeddings from vault
khayal backup      # backup vault, db + config (v1.1)
khayal restore      # restore from backup (v1.1)
khayal logs         # tail server logs
khayal config       # view current config (token redacted)
khayal version      # version + build info
khayal completion   # generate shell completion scripts
khayal vault        # vault maintenance subcommands
  khayal vault health          # vault health report
  khayal vault fix-links       # remove broken wikilinks
  khayal vault clean-media     # delete orphaned media files
  khayal vault show-duplicates # show potential duplicate notes
```

### khayal UX — per command

#### khayal init

Generates `~/.config/khayal/config.yaml` with 600 permissions. Generates token. Prints token once — never again.

```
khayal init

creating config directory...  ~/.config/khayal/
generating token...           a3f9c2e1d7b4f892... (save this — shown once)
writing config...             ~/.config/khayal/config.yaml (600)
creating log directory...     ~/.config/khayal/logs/

next steps:
  1. edit ~/.config/khayal/config.yaml
     set vault.path to your vault location

  2. khayal start

  3. on client machines:
     kl init → enter server address + token
```

#### khayal start

Verbose. Every check shown. User knows exactly what happened.

```
khayal v0.1.0

loading config...
checking dependencies...
  ✓ ollama        http://localhost:11434

  ✓ vault         /absolute/path/to/vault
  ✓ db            /absolute/path/to/khayal.db
  ✓ log           /absolute/path/to/logs/khayal.log
  ✓ queue         ready
  ✓ llm           ollama
  ✓ worker        started
  ✓ server        127.0.0.1:1133
  ✓ pid           12345

khayal is running.
press ctrl+c to stop
press ctrl+c to stop
```

**Vault auto-scan on start:**

```
scanning vault...
  2,847 notes found
    644 not yet indexed → queuing for processing
    147 modified since last index → re-queuing

khayal is running.
processing 791 notes in background — kl status to track
```

Rules:
- Show every dep check — pass and fail
- On dep failure: show install commands, say what's unavailable
- Redact token in all output — show first 8 chars + "..."
- End with "khayal is running" — unambiguous success signal
- If anything critical fails (vault not found, db error) → exit with error, never start
- Auto-scan vault on every start: queue new/modified notes, skip unchanged indexed notes
- Works with any existing markdown folder — no import command needed

#### khayal stop

```
khayal stop

stopping worker...    ✓ (waited for current job to finish)
stopping server...    ✓
khayal stopped.
```

Waits for current job to complete before stopping worker. Never kills mid-processing.

#### khayal status — Bubble Tea TUI, refreshes every 3 seconds

```
  khayal · admin                                         v0.1.0

  ┌─ server ────────────────────────────────────────────────────┐
  │  host      http://127.0.0.1:1133                            │
  │  uptime    3h 24m                                           │
  │  pid       12847                                            │
  └─────────────────────────────────────────────────────────────┘

  ┌─ vault ─────────────────────────────────────────────────────┐
  │  path      ~/brain                                          │
  │  notes     2,847                                            │
  │  indexed   2,203  (77%)   ████████████████░░░░             │
  │  pending     644           → khayal reindex                 │
  └─────────────────────────────────────────────────────────────┘

  ┌─ dependencies ──────────────────────────────────────────────┐
  │  ollama    ✓   qwen2.5:3b · moondream · nomic-embed-text    │
  │  ffmpeg    ✓                                                │
  │  yt-dlp    ✗   video ingestion unavailable                  │
  └─────────────────────────────────────────────────────────────┘

  ┌─ queue ─────────────────────────────────────────────────────┐
  │  pending      2    ██░░░░░░░░░░░░░░░░░░░░                   │
  │  processing   1    ██░░░░░░░░░░░░░░░░░░░░  image · abc123   │
  │  done       147                                             │
  │  failed       0                                             │
  └─────────────────────────────────────────────────────────────┘

  ┌─ system ────────────────────────────────────────────────────┐
  │  memory    187 MB                                           │
  │  db size    24 MB                                           │
  │  log size   1.2 MB                                          │
  └─────────────────────────────────────────────────────────────┘

  q quit   r refresh   l logs   c config   ? help
```

Rules:
- Refresh every 3 seconds automatically
- `q` quits, `r` forces immediate refresh, `l` opens log tail, `c` shows config
- Progress bar on indexed % — visual signal of vault health
- Processing row shows job type + ID — user knows what worker is doing
- Failed > 0 → highlight in error color — needs attention
- Keyboard hints always visible at bottom

#### khayal reindex

Long operation. Must show progress and be resumable.

```
khayal reindex

scanning vault...
  2,847 notes found
  2,203 already indexed  (skipped)
    644 to reindex

starting reindex...

  [████████████████░░░░░░░░░░░░░░]  412/644  64%  ~2m 18s remaining

  ✓ 2024-03-16-useeffect-cleanup.md
  ✓ 2024-03-15-meeting-sarah.md
  ✓ 2024-03-14-cap-theorem.md
  ✓ 2024-03-13-rust-cli-thoughts.md
  ...

  done.
  644 notes reindexed
  4m 32s elapsed
```

Rules:
- Progress bar with percentage + estimated time remaining
- Show each note as it completes (last 4 visible, scrolls)
- If interrupted (ctrl+c): graceful stop, show progress so far, resumable on next run
- Never re-embed already indexed notes — check mtime
- `khayal reindex --force` → reindex everything regardless of mtime

#### khayal logs

```
khayal logs

  tailing ~/.config/khayal/logs/khayal.log
  ctrl+c to stop

  2024-03-16T14:23:01 POST /v1/capture 200 47ms
  2024-03-16T14:23:04 worker: processed text abc123 3.2s
  2024-03-16T14:23:18 POST /v1/capture 200 52ms
  2024-03-16T14:23:19 worker: queued image def456
  2024-03-16T14:24:01 worker: processed image def456 11.4s
  2024-03-16T14:25:33 GET /v1/search 200 89ms
```

Simple log tail. No flags needed. ctrl+c to exit.

#### khayal version

```
khayal version

  khayal v0.1.0
  commit  a3f9c2e
  built   2024-03-16T10:00:00Z
  go      1.22.1
```

#### khayal config

Shows current config with token redacted. Read-only view — edit the file directly.

```
khayal config

  config at ~/.config/khayal/config.yaml

  vault
    path        ~/brain
    inbox_dir   khayal
    media
      image     vault
      pdf       vault
      audio     config
      video     config

  server
    host        127.0.0.1
    port        1133
    token       a3f9c2e1... (redacted)

  llm
    provider    ollama
    embed       nomic-embed-text
    text        qwen2.5:3b
    vision      moondream
    fallback    (none)

  worker
    max_workers 1
    max_retries 3
    backoff     exponential
```

#### khayal vault — vault maintenance subcommands

```bash
khayal vault health          # show vault health report
khayal vault fix-links       # remove broken wikilinks
khayal vault clean-media     # delete orphaned media files
khayal vault show-duplicates # show potential duplicate notes
```

##### khayal vault health

```
khayal vault health

  vault · ~/brain
  notes     2,847
  indexed   2,203  (77%)
  orphans   12 media files not referenced
  links     4 broken wikilinks found

  health    ⚠ needs attention
  → fix links:      khayal vault fix-links
  → clean media:   khayal vault clean-media
  → reindex:       khayal reindex
```

##### khayal vault fix-links

```
khayal vault fix-links

  scanning for broken wikilinks...
  4 broken links found in 3 files

  khayal/2024-03-10-project.md
    → khayal/old-note.md (deleted)
    → khayal/renamed-note.md (exists, will update)

  [dry run — use --fix to apply]
```

With `--fix`: rewrites wikilinks to point to existing files or removes broken links.

##### khayal vault clean-media

```
khayal vault clean-media

  scanning for orphaned media...
  12 orphaned files found · 34 MB

  khayal/media/unused-1.png    2.1 MB
  khayal/media/unused-2.jpg    1.8 MB
  ...

  [dry run — use --fix to apply]
```

With `--fix`: moves orphaned files to `.khayal-trash/`.

##### khayal vault show-duplicates

```
khayal vault show-duplicates

  checking for duplicates...
  potential duplicates found:

  khayal/2024-03-15-rust-thoughts.md
  khayal/2024-03-10-rust-notes.md
    similarity: 0.87 · 234 shared words

  khayal/2024-02-20-meeting-notes.md
  khayal/2024-02-19-meeting.md
    similarity: 0.91 · 189 shared words

  3 pairs total
  → review manually to merge or delete
```

#### khayal backup (v1.1)

Backup vault, database, and config.

```bash
khayal backup --dest /Volumes/BackupDrive/khayal
khayal backup --dest /Volumes/BackupDrive/khayal --encrypt  # encrypt db + config
```

**What gets backed up:**

| Item | Method | Encryption |
|------|--------|------------|
| vault/ | rsync | Plain (user's choice) |
| khayal.db | snapshot | Optional with `--encrypt` |
| config.yaml | copy | Optional with `--encrypt` |

Vault is plain markdown — no encryption needed. DB and config contain the token — encrypt with `--encrypt`.

**Encryption:** Uses `age` (https://github.com/FiloSottile/age) — simple, modern, no GPG complexity.

```bash
# First run: generate key
khayal backup --init-key
→ key saved to ~/.config/khayal/backup.key

# Backup with encryption
khayal backup --dest /Volumes/Backup --encrypt
→ encrypts db + config with backup.key
```

**Output:**

```
khayal backup --dest /Volumes/BackupDrive/khayal

  backing up vault...
    ~/brain → /Volumes/BackupDrive/khayal/vault/
    2,847 files · 124 MB · 8s

  backing up database...
    khayal.db → /Volumes/BackupDrive/khayal/khayal-2024-03-16.db
    24 MB · encrypted ✓

  backing up config...
    config.yaml → /Volumes/BackupDrive/khayal/config-2024-03-16.yaml
    encrypted ✓

  backup complete · 148 MB · 14s
  dest: /Volumes/BackupDrive/khayal/
```

**Scheduled backups:** `khayal init` optionally sets up a weekly launchd job.

#### khayal restore (v1.1)

Restore from backup.

```bash
khayal restore --from /Volumes/BackupDrive/khayal
khayal restore --from /Volumes/BackupDrive/khayal --date 2024-03-10
```

**Behavior:**

```
khayal restore --from /Volumes/BackupDrive/khayal

  available backups
    2024-03-16  vault: 2,847 notes · db: 24MB  ← latest
    2024-03-09  vault: 2,801 notes · db: 23MB
    2024-03-02  vault: 2,756 notes · db: 21MB

  restoring latest backup...

  ! khayal must be stopped before restore
    → run: khayal stop
    → then: khayal restore --from /Volumes/BackupDrive/khayal
```

If khayal is stopped:

```
  restoring vault...
    /Volumes/BackupDrive/khayal/vault/ → ~/brain
    2,847 files · 124 MB · 9s

  restoring database...
    khayal-2024-03-16.db → ~/.config/khayal/khayal.db
    decrypting... ✓
    24 MB · 2s

  restoring config...
    config-2024-03-16.yaml → ~/.config/khayal/config.yaml
    decrypting... ✓

  restore complete.
  run: khayal start
```

**Rules:**
- Refuses to run if khayal server is running
- `--date` selects specific backup (default: latest)
- Decrypts automatically if backup.key exists
- Vault restore: additive by default — never overwrites newer files
- `--overwrite` forces full vault overwrite — explicit user intent
- DB + config: always fully replaced
- `--dry-run` shows what would be restored

---

### kl UX — per command

#### kl capture (default)

The most used command. Output must be instant and minimal.

**Text — done immediately:**
```
✓ saved · #react #performance · 3ms
```

**URL — queued:**
```
⏳ queued · article · id: abc123
```

**Image — queued:**
```
⏳ queued · image · id: def456
```

**With connections (v1.1) — arrives async after capture:**
```
✓ saved · #react #performance · 3ms

  ┌─ connections ────────────────────────────────────┐
  │ ~ 2 years ago · you thought about this           │
  │   "useEffect with wrong deps causes loops"       │
  │                                                  │
  │ ⚡ Jan 2023 · contradicts something you wrote    │
  │   "cleanup functions are rarely needed"           │
  └──────────────────────────────────────────────────┘
```

**Ollama down — capture still works:**
```
✓ saved · unprocessed
  ollama unreachable — will tag when server is back
```

**Server unreachable:**
```
✗ cannot reach khayal at http://100.x.x.x:1133

  → is khayal running?     ssh mac-air khayal start
  → wrong address?         kl config set host <address>
  → check logs             ssh mac-air khayal logs
```

Rules:
- No spinner for text capture — returns in <100ms, spinner is jarring
- Spinner only for image/URL capture (uploading file) if >200ms
- Connections box: only shown if connections exist — never show empty box
- Connections box: never blocks — shown when async job completes
- Error messages: always actionable, never raw Go errors
- `kl "thought" --verbose` shows full processing details

#### kl search

```
$ kl search "paid john money"

  3 results · hybrid · 42ms

  ──────────────────────────────────────────────────────────
  khayal/2019-03-03-designer.md                          0.94
  March 3, 2019 · #finance #design

  ...paid John Doe $2,000 for logo design work.
  Follow-up: brand guidelines next week...

  ──────────────────────────────────────────────────────────
  khayal/2019-04-10-contractor.md                        0.81
  April 10, 2019 · #finance

  ...second payment to John, $500 for revisions...

  ──────────────────────────────────────────────────────────
  khayal/2018-12-01-branding.md                          0.68
  December 1, 2018 · #design

  ...initial quote from John was $2,500 but...
```

Rules:
- Score shown right-aligned — user learns what scores mean over time
- Date prominent — temporal context matters more than filename
- Tags on second line — category at a glance
- Excerpt shows the matched text — not random 200 chars
- Dividers between results — scannable
- No color on content — only on metadata (score, date, tags)
- `kl search "query" --mode keyword` forces keyword only
- `kl search "query" --mode semantic` forces semantic only
- `kl search "query" --from 2024-01-01 --to 2024-03-16` date filter

**No results:**
```
$ kl search "unicorn rainbow"

  0 results · hybrid · 12ms

  nothing found for "unicorn rainbow"
  → try different keywords
  → try: kl search "unicorn rainbow" --mode keyword
```

#### kl recent

```
$ kl recent

  today

  14:23  text     useEffect cleanup runs after every render    #react
  14:18  image    whiteboard-arch.png                         #system-design
  13:55  article  CAP theorem explained                       #distributed
  13:02  text     thinking about rust for the cli             #go #rust

  yesterday

  18:44  text     meeting with Sarah re contract              #work
  16:20  article  https://fasterthanli.me/articles/...        #rust
  11:05  image    notebook-sketch.jpg                         #ideas

  2 days ago

  ...3 more  →  kl recent --days 7
```

Rules:
- Grouped by day — "today", "yesterday", "2 days ago", then date
- Type badge: text / image / article / url — consistent width
- Title truncated at ~50 chars
- Tags right-aligned
- Truncate at 7 items per day, show "N more → kl recent --days X"
- `kl recent --days 7` shows full last 7 days
- `kl recent --type image` filters by type

#### kl browse

**By tag:**
```
$ kl browse --tag react

  #react · 23 notes

  2024-03-16  useEffect cleanup runs after every render
  2024-03-10  React Server Components mental model
  2024-02-28  Why I stopped using useCallback everywhere
  2024-01-15  [article] React 19 new hooks overview
  2023-12-03  [image] component architecture whiteboard

  18 more · kl browse --tag react --all
```

**By person:**
```
$ kl browse --person "John Doe"

  John Doe · 8 mentions

  2024-03-16  paid contractor 500 for revisions
  2024-03-02  meeting with John re new project
  2023-11-14  [article] saved John's blog post on systems
  2019-04-10  second payment to John, $500
  2019-03-03  paid John Doe $2,000 for logo design

  3 more · kl browse --person "John Doe" --all
```

**By amount:**
```
$ kl browse --amount 2000

  $2,000 · 2 mentions

  2019-03-03  paid John Doe $2,000 for logo design
  2019-01-15  budget approved: $2,000 for branding work
```

Rules:
- Show 5 results by default, `--all` flag shows everything
- Sorted by date descending — most recent first
- Note type badge in brackets for non-text [article] [image]
- Amount search normalizes: 2000 matches "2k", "$2,000", "2000", "2,000"

#### kl stats

```
$ kl stats

  vault · ~/brain

  total         2,847   notes
  this week        23   ████░░░░░░░░░░░░░░░
  this month       94   ████████░░░░░░░░░░░░

  top tags
  #react           142  ████████████████████
  #go               98  ████████████████░░░░
  #work             87  ██████████████░░░░░
  #finance          34  ██████░░░░░░░░░░░░░░
  #distributed      28  █████░░░░░░░░░░░░░░

  top people
  John Doe          34  mentions
  Sarah Chen        18  mentions
  Robert Kim         6  mentions

  capture streak    12 days
  longest streak    34 days
```

Rules:
- ASCII bar charts — normalized to longest bar = full width
- Top 5 tags, top 3 people — not exhaustive
- Capture streak — motivates consistent use
- No processing stats here — that's khayal status
- Pure SQL, no LLM, runs in milliseconds

#### kl status — lightweight, not admin

```
$ kl status

  ✓ khayal v0.1.0 · http://100.x.x.x:1133

  queue
    processing   1   image
    pending      2
    failed       0

  last capture  14:23 · useEffect cleanup runs after...
```

If server unreachable:
```
$ kl status

  ✗ khayal unreachable · http://100.x.x.x:1133
    → ssh mac-air khayal start
```

Rules:
- 6 lines maximum — this is a quick check, not a dashboard
- No memory, no db size, no pid — that's khayal's job
- Failed > 0 → shown in error color with hint to check khayal status
- Server version shown — helps debug version mismatch issues

#### kl init — Huh wizard

```
$ kl init

  kl setup

  ? Server address
  › http://127.0.0.1:1133
    ──────────────────────────────────────────────────
    Your khayal server address.
    Use Tailscale IP for remote access: http://100.x.x.x:1133

  ? Token
  › ••••••••••••••••••••••••••••••••
    ──────────────────────────────────────────────────
    Find this in ~/.config/khayal/config.yaml on your server.
    Or run: khayal init to regenerate.

  testing connection...  ✓ khayal v0.1.0

  ✓ written to ~/.config/khayal/kl.yaml

  you're ready. try:
    kl "your first thought"
```

Rules:
- Validate connection BEFORE writing config — don't write broken config
- If connection fails: show error + what to check, stay in wizard
- Token input masked with bullets
- End with first command to try — time to first successful capture

#### kl config set

```
$ kl config set host http://100.x.x.x:1133

  ✓ host updated
    ~/.config/khayal/kl.yaml

$ kl config set token abc123def456

  ✓ token updated
    ~/.config/khayal/kl.yaml
```

Silent success — one line. User doesn't need more than that.

### Typography system — both tools

All output uses Rawnaqs theme Lip Gloss styles. Never define colors inline.

```
Primary text      styles.Primary         → GoldLight #E8B86D
Secondary/meta    styles.Muted           → GoldDark  #8B6020
Timestamps/counts styles.Dim             → GoldDim   #3A2E18
Success (✓)       styles.SuccessStyle    → Green     #4A7C59
Error (✗)         styles.ErrorStyle      → Red       #8B3A3A
Processing (⏳)   styles.ProcessingStyle → GoldDark italic
Borders/dividers  styles.Divider()       → GoldDim
Tags              styles.Tag             → Gold bg, dark text
Scores            styles.Dim             → present but not dominant
Panel borders     styles.Panel           → rounded, GoldDim border
```

Rules:
- One color family — gold on dark. Nothing fights for attention.
- Color on metadata (date, score, tag) — never on content text
- Error color only for actual errors — not warnings, not muted info
- Processing style (italic) only for in-progress states

### Error message format — both tools

Never show raw Go errors. Every error tells the user what to do next.

Format:
```
✗ short description of what failed

  → action to try first
  → action to try second
  → where to get more info
```

Examples:

```
# Server unreachable
✗ cannot reach khayal at http://100.x.x.x:1133
  → is khayal running?    ssh mac-air khayal start
  → wrong address?        kl config set host <address>
  → check logs            ssh mac-air khayal logs

# Auth failed
✗ unauthorized · invalid token
  → get token from        ~/.config/khayal/config.yaml on server
  → update token          kl config set token <token>

# Vault write failed
✗ cannot write to vault · ~/brain/khayal/
  → does the path exist?  check vault.path in config
  → permissions ok?       ls -la ~/brain/khayal/

# Ollama unreachable (khayal start)
✗ ollama not running at localhost:11434
  → start ollama          ollama serve
  → wrong port?           edit llm.ollama_host in config
```

### Spinner rules — both tools

Never show a spinner for operations under 200ms. It appears and disappears so fast it's jarring.

Show a spinner for:
- `kl init` connection test (network, variable latency)
- `kl --url` and `kl --image` file upload (variable size)
- `khayal reindex` (replaced by progress bar)
- `khayal start` dep checking (replaced by step-by-step output)
- `kl search` with reranking enabled (potential 2-3s)

Never show a spinner for:
- `kl "thought"` text capture — always <100ms
- `kl status` — always <200ms
- `kl recent` — always <50ms
- `kl stats` — always <50ms
- `khayal stop` — always <500ms (show "stopping..." text instead)

### Command structure convention — both tools

Always: `tool verb [target] [--flags]`

Never: `tool target verb` or `tool noun verb`

```
kl search "query"          ✓ verb then target
kl browse --tag react      ✓ verb then flag-target
kl config set key value    ✓ verb verb key value (set is sub-verb)
kl "thought"               ✓ implicit capture verb
```

### Help format — both tools

Examples first, flags second. Real commands not abstract syntax.

```
$ kl search --help

  Search your vault using keyword and semantic search.

  Usage:
    kl search <query> [flags]

  Examples:
    kl search "paid john money"
    kl search "react hooks" --mode keyword
    kl search "last week finances" --from 2024-03-11 --to 2024-03-18
    kl search "distributed systems" --limit 10

  Flags:
    --mode     hybrid | keyword | semantic  (default: hybrid)
    --limit    number of results            (default: 5, max: 50)
    --from     start date  YYYY-MM-DD
    --to       end date    YYYY-MM-DD
    --verbose  show scores and debug info
```

### Exit codes — both tools

```
0   success
1   user error    (wrong args, note not found, config missing)
2   server error  (unreachable, auth failed, API error)
3   vault error   (write failed, file locked, permission denied)
4   dep error     (Ollama unreachable)
```

Scripts consuming kl or khayal can rely on these codes being consistent.

### Progressive disclosure — both tools

Default output is minimal. Verbose mode always available.

```
kl "thought"            # one line
kl "thought" --verbose  # full processing details

kl search "query"            # clean results
kl search "query" --verbose  # scores, timing, which pipeline matched

khayal start            # step by step but not overwhelming
khayal start --verbose  # every internal detail

khayal status           # dashboard (already rich)
khayal status --json    # machine-readable for scripting
```

### Shell completion — both tools

Cobra generates shell completion scripts automatically from the command tree.

```bash
# kl
kl completion zsh   # prints zsh completion script
kl completion bash  # prints bash completion script
kl completion fish  # prints fish completion script

# khayal
khayal completion zsh
khayal completion bash
khayal completion fish
```

**Install instructions:**

```bash
# zsh
kl completion zsh >> ~/.zshrc && source ~/.zshrc

# bash
kl completion bash >> ~/.bashrc && source ~/.bashrc

# fish
kl completion fish > ~/.config/fish/completions/kl.fish
```

**What gets completed:**

```
kl se<tab>              → kl search
kl search --m<tab>      → --mode
kl search --mode <tab>  → hybrid  keyword  semantic
kl browse --<tab>       → --tag  --person  --amount  --all
kl config set <tab>     → host  token
khayal <tab>            → init  start  stop  restart  status  reindex  version  logs  config  completion
```

### What NOT to show — both tools

Never show:
- Raw Go error messages (dial tcp, unexpected EOF, etc.)
- Internal job IDs unless user needs to poll (capture response shows ID, not kl)
- Stack traces (log them, never print to user)
- Redundant confirmations ("are you sure?" for non-destructive operations)
- "No results found" with no guidance
- Timestamps on one-line outputs like `kl "thought"` — noise
- Version on every command — only on `kl version` and `khayal version`

---

## v1 Scope

### In v1
- Text / quick thought capture
- Image / screenshot capture
- Article / web URL capture
- Keyword + semantic hybrid search
- Search results with relevant excerpt
- PWA (React + Vite, dark only, embedded in binary)
- CLI (`khayal` server + `kl` client)
- macOS + Linux
- Single binary + Docker Compose both supported
- Token auth on every request
- Request logging
- Ollama primary, Groq + OpenAI as fallbacks
- Installer checks dependencies and guides user
- GitHub releases + Homebrew formula
- AGPLv3

### Search Implementation

**Keyword Search:**
- SQLite FTS5 with porter stemming
- BM25 ranking for relevance
- FTS5 triggers on content, title, tags

**Semantic Search:**
- sqlite-vec for vector similarity (MIT/Apache-2 licensed)
- CGo required (using mattn/go-sqlite3)
- Virtual table for embeddings
- 10-20x faster than pure Go

```sql
-- sqlite-vec virtual table
CREATE VIRTUAL TABLE vec_chunks USING vec0(
    chunk_embedding float[768]
);
```

**Sync Strategy:**
- mtime check on search
- Re-index stale files inline (on-demand)

**Hybrid Merge:**
- Reciprocal Rank Fusion (RRF, k=60)
- Combines keyword + semantic rankings

**Date Filter:**
- Pre-filter by date range before both keyword and semantic search
- Applies to GET /v1/search
- Future: POST /v1/ask
- Parameters: `from=2024-03-11&to=2024-03-16` (optional ISO date strings)

### Explicitly Out of v1
- Voice notes
- PDF ingestion
- YouTube / video ingestion
- Browser extension
- Raycast extension
- Mobile app
- iOS Shortcuts
- Graph connections / wikilinks
- Windows support
- Setup wizard UI (non-technical users)
- Multi-user
- Chat over vault (Open WebUI integration)

---

## Proactive Connections

After every capture is processed, khayal automatically finds related notes from the past and surfaces them asynchronously. Never blocks capture.

### Philosophy

- Triggered by capture only — not ambient/background
- Delivered asynchronously — capture response is always instant
- Strict quality gates — max 3 connections per capture, high thresholds
- All connection types individually toggleable in config
- Never surface notes younger than 7 days — surprise comes from temporal distance

### Six Connection Types

#### Type 1 — Semantic Similar (v1.1)

Find notes with high embedding similarity to the new note.

```
Detection: sqlite-vec cosine similarity
Threshold: score > 0.85
Filter:    note age > 7 days, exclude current note
Max:       3 results
Label:     "you thought about this N days/months/years ago"
```

#### Type 2 — Same Person Mentioned (v1.1)

Find all past notes mentioning the same people extracted from the new note.

```
Detection: SQL query against entities table
Filter:    note age > 7 days
Label:     "[Name] also appears in N other notes"
Surface:   most recent + most relevant excerpt
```

#### Type 3 — Same Amount / Financial (v1.1)

Find past notes mentioning the same financial amounts. Amounts normalized to integers (2k → 2000, $2,000 → 2000).

```
Detection: SQL query against entities table (type = 'amount')
Filter:    note age > 7 days
Label:     "you've mentioned [amount] before"
```

#### Type 4 — Contradicting Thoughts (v1.2)

Find notes that semantically conflict with the new note.

```
Detection: two-step
  Step 1: find top-5 similar notes (score > 0.80)
  Step 2: for each, run LLM contradiction check
  Surface: LLM returns "yes" only
Label:     "contradicts something you wrote [date]"
```

#### Type 5 — Follow-ups Never Completed (v1.2)

Find past notes expressing follow-up intent mentioning the same people, with no subsequent record of completion.

```
Detection: three-step
  Step 1: FTS5 query for intent keywords + same person
  Step 2: check date of intent note
  Step 3: check if subsequent notes mention same person
Filter:    intent note age > 14 days
Label:     "you planned to follow up with [name] — no record of this happening"
```

#### Type 6 — Ideas Revisited Over Time (v1.2)

Detect when the new note is part of a recurring pattern — same topic appearing multiple times across months or years.

```
Detection: reuse semantic similar results
           if 3+ similar notes found spanning > 6 months
Label:     "you've returned to this idea [N] times since [earliest date]"
Surface:   oldest + newest note as bookends
```

### Async Delivery

Connections run as a separate job type after ingest completes:

```
POST /v1/capture
→ note saved
→ ingest job pushed to queue (existing)
→ connections job pushed to queue (new, runs after ingest)
→ capture response returned immediately

Client polls GET /v1/queue/:id for connections_job_id
```

### Ranking and Deduplication

```
If more than 3 connections qualify → rank by type priority → take top 3

Type priority (highest to lowest):
  1. contradiction  — most surprising, most valuable
  2. follow_up      — actionable
  3. revisit        — contextual pattern
  4. person         — contextual
  5. amount         — contextual
  6. similar        — most common, lowest priority

Deduplication:
  If same note qualifies for multiple types → show once, use highest priority
```

### Capture Response

```json
{
  "id": "abc123",
  "status": "done",
  "note_path": "khayal/2024-03-16-thought.md",
  "created_at": "2024-03-16T14:23:00Z",
  "connections_job_id": "def456"
}
```

Connections retrieved via `GET /v1/queue/def456`:

```json
{
  "id": "def456",
  "type": "connections",
  "status": "done",
  "result": {
    "connections": [
      {
        "type": "similar",
        "note_path": "khayal/2022-04-10-react-perf.md",
        "excerpt": "useEffect with wrong deps causes infinite loops",
        "score": 0.91,
        "label": "you thought about this 2 years ago"
      }
    ]
  }
}
```

### Config

```yaml
connections:
  enabled: true
  min_age_days: 7
  max_per_capture: 3
  similarity_threshold: 0.85
  contradiction_threshold: 0.80
  types:
    similar:      true
    person:       true
    amount:       true
    contradiction: true
    follow_up:    true
    revisit:      true
```

### Performance

```
semantic similar:    ~5ms   (sqlite-vec)
person + amount:    ~2ms   (SQL entity lookup)
revisit:             ~10ms  (reuses semantic search)
follow-up:           ~10ms  (SQL + date comparison)
contradiction:        ~3s    (similarity + LLM check per candidate)

Total v1.1 types:    ~20ms  (imperceptible async)
Total v1.2 types:    ~3-4s  (async — never blocks capture)
```

### New Package

```
internal/connections/
├── engine.go        ← orchestrates all types, ranking, dedup
├── similar.go       ← semantic similarity
├── entity.go        ← person + amount
├── revisit.go       ← revisit detection
├── followup.go     ← follow-up detection
└── contradiction.go ← LLM-based contradiction detection
```

---

## v1.1 Scope

### 1. Chunking

Replace whole-note embeddings with chunk-level embeddings for better search granularity.

**Chunk Specification:**
- Target: 150-200 words per chunk
- Overlap: 30-50 words between consecutive chunks
- Split on paragraph boundaries — never mid-sentence
- Minimum chunk size: 50 words — don't embed tiny fragments

**Database Changes:**
- New `chunks` table replaces `embeddings` table

```sql
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding BLOB NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX idx_chunks_note ON chunks(note_path);
```

**Search Changes:**
- Semantic search queries `chunks` table
- Returns parent note, not the chunk
- Deduplicate: if multiple chunks from same note match, return note once with best-scoring chunk as excerpt

**New CLI Command:**
```bash
khayal reindex  # Rebuild all chunk embeddings from vault markdown files
                 # Required when upgrading from v1 to v1.1
                 # Safe to run anytime — non-destructive to vault
```

---

### 2. Entity Extraction

Add to ingest pipeline for all capture types. After tags/summary, extract structured entities:

**Frontmatter:**
```yaml
entities:
  people:  ["John Doe"]
  amounts: ["2000", "2k"]
  dates:   ["2019-03-03"]
  places:  []
  urls:    []
  orgs:    []
```

**Database Table:**
```sql
CREATE TABLE entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER,
    entity_type TEXT NOT NULL, -- person, amount, date, place, org, url
    entity_value TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX idx_entities_note ON entities(note_path);
CREATE INDEX idx_entities_type ON entities(entity_type);
```

### 3. Proactive Connections (v1.1)

Auto-discover related notes after capture. Delivered asynchronously.

**v1.1 Connection Types:**
- Semantic similar (sqlite-vec similarity > 0.85)
- Same person mentioned (entity lookup)
- Same amount/financial (entity lookup)

**Flow:**
```
POST /v1/capture → capture response immediately
                    → connections job queued
                    → poll GET /v1/queue/:id for connections
```

**Config:**
```yaml
connections:
  enabled: true
  min_age_days: 7
  max_per_capture: 3
  types:
    similar: true
    person:  true
    amount:  true
```

**See "Proactive Connections" section above for full details.**

---

### 4. Search Pipeline (v1.1)

*Note: sqlite-vec is already included in v1. This section adds advanced features.*

**Full pipeline in order:

```
User query
    │
    ▼
1. Temporal detection    → extract date filter, clean query
    │
    ▼
2. Query rewriting       → LLM expands to search-optimized terms
    │
    ▼
3. HyDE                  → LLM generates hypothetical answer, embed it
    │
    ├──────────────────────────────┐
    ▼                              ▼
4a. FTS5 keyword search       4b. sqlite-vec vector search
    (original + rewritten)          (HyDE embedding)
    date filter applied             date filter applied
    top-20                         top-20
    │                              │
    └──────────────┬───────────────┘
                   ▼
5. RRF merge       → combine results (k=60)
                   → deduplicate by note_path
                   → top-20 unique chunks
                   │
                   ▼
6. Cross-encoder rerank  → score each (query, chunk) pair
                         → sort by relevance
                         → top-5
                   │
                   ▼
7. Return results  → note_path, excerpt, score, source
```

**Temporal detection** — detect time references, auto-inject date range:
```
"last week"   → from: -7d,  to: now
"last 5 days" → from: -5d,  to: now
"yesterday"   → from: -1d,  to: -1d
"in 2022"     → from: 2022-01-01, to: 2022-12-31
```
Temporal words stripped from query before search.

**Query rewriting** — LLM rewrites natural language into search-optimized keywords. Both original and rewritten query searched. Results merged via RRF.

**HyDE** — LLM generates a hypothetical answer, that gets embedded instead of the raw query. Falls back to direct query embedding if LLM call fails.

**Cross-encoder reranking** — small reranker model (e.g. `bge-reranker-v2-m3`, 270MB) scores each (query, chunk) pair after initial retrieval. Optional — if model not configured, skip reranking.

**Config:**
```yaml
search:
  retrieval_top_k:  20                   # candidates before reranking
  return_top_k:     5                    # final results returned
  reranker_model:   bge-reranker-v2-m3 # blank = skip reranking
  hyde_enabled:     true
  rewrite_enabled: true
  temporal_enabled: true
```

**Technical Changes:**
- Uses `mattn/go-sqlite3` (CGO) from v1
- Add gcc to goreleaser CI build environment
- Virtual table for vectors, regular table for content/metadata

```sql
-- sqlite-vec virtual table (v1.1)
CREATE VIRTUAL TABLE chunks_vec USING vec0(
    embedding FLOAT[768],
    note_id TEXT,
    chunk_id TEXT
);
```

**v1.1 Changes:**
- Add sqlite-vec extension for faster similarity search
- Switch to virtual table for embeddings

---

## Phases After v1

```
v1.1  → Chunking + Entity extraction + connections (similar, person, amount) + backup
v1.2  → connections (contradiction, follow_up, revisit) + voice notes + PDF
v1.3  → YouTube / video ingestion
v1.4  → Browser extension (github.com/rawnaqs/khayal-browser)
v1.5  → Raycast extension (github.com/rawnaqs/khayal-raycast)
v1.6  → iOS Shortcuts (github.com/rawnaqs/khayal-ios)
v2.0  → Setup wizard UI for non-technical users
v2.1  → Graph connections, backlinks
v2.2  → Windows support
v2.3  → Mobile app (github.com/rawnaqs/khayal-mobile)
```
v1.4  → Browser extension (github.com/rawnaqs/khayal-browser)
v1.5  → Raycast extension (github.com/rawnaqs/khayal-raycast)
v1.6  → iOS Shortcuts (github.com/rawnaqs/khayal-ios)
v2.0  → Setup wizard UI for non-technical users
v2.1  → Graph connections, backlinks
v2.2  → Windows support
v2.3  → Mobile app (github.com/rawnaqs/khayal-mobile)
```

**v1.1 Details:**
- Chunk-level embeddings (150-200 words, 30-50 overlap)
- Entity extraction (people, amounts, dates, places, orgs, urls)
- Advanced search: temporal detection, query rewriting, HyDE, reranking

---

## Architecture

### Capture Interface Philosophy — Lego Model

All capture interfaces are independent, pluggable clients. They speak HTTP to the server. No interface implements capture logic — that lives exclusively in the server.

```
┌──────────────────────────────────────────────────────┐
│                 Capture Interfaces                   │
│  kl CLI  │  PWA  │  Browser  │  Raycast  │  Mobile  │
│          │ React │  Ext(v2)  │  Ext(v2)  │  (v3)   │
└──────────┴───────┴───────────┴───────────┴──────────┘
                    │ HTTP + X-Khayal-Token
                    │ POST /v1/capture
                    │ GET  /v1/search
                    │ GET  /v1/queue/:id
                    ▼
┌──────────────────────────────────────────────────────┐
│                  khayal server                       │
│   API layer → capture core → worker → ingest        │
└──────────────────────────────────────────────────────┘
```

### Interface Repos

```
github.com/rawnaqs/theme               ← shared design system (colors, typography, styles)
github.com/rawnaqs/khayal              ← core server + kl CLI + PWA (v1)
github.com/rawnaqs/khayal-browser      ← browser extension (v1.3)
github.com/rawnaqs/khayal-raycast      ← Raycast extension (v1.4)
github.com/rawnaqs/khayal-ios          ← iOS Shortcuts (v1.5)
github.com/rawnaqs/khayal-mobile       ← mobile app (v2.3)
```

---

## Project Structure

```
khayal/
├── cmd/
│   └── khayal/
│       └── main.go              ← single binary entry point
├── internal/
│   ├── api/
│   │   ├── server.go            ← HTTP server, middleware, auth, logger
│   │   ├── capture.go           ← POST /v1/capture
│   │   ├── search.go            ← GET /v1/search
│   │   ├── health.go            ← GET /v1/health
│   │   └── queue.go             ← GET /v1/queue, GET /v1/queue/:id
│   ├── worker/
│   │   └── worker.go            ← job processor, configurable concurrency, exponential backoff
│   ├── ingest/
│   │   ├── text.go              ← text processing
│   │   ├── image.go             ← LLaVA/moondream + OCR
│   │   └── article.go           ← scrape + summarize
│   ├── llm/
│   │   ├── interface.go         ← LLM interface definition
│   │   ├── ollama.go            ← Ollama client (primary)
│   │   ├── groq.go              ← Groq fallback
│   │   └── openai.go            ← OpenAI fallback
│   ├── vault/
│   │   └── writer.go            ← markdown writer, vault-agnostic
│   ├── queue/
│   │   └── queue.go             ← SQLite job queue
│   ├── search/
│   │   ├── keyword.go           ← full-text search
│   │   └── semantic.go          ← vector similarity search
│   └── config/
│       └── config.go            ← config loader, fail hard on error
├── cli/
│   ├── root.go                  ← Cobra root command
│   ├── capture.go               ← kl "thought", --url, --image
│   ├── search.go                ← kl search (Glamour rendering)
│   ├── recent.go                ← kl recent
│   ├── browse.go                ← kl browse
│   ├── stats.go                 ← kl stats
│   ├── status.go                ← kl status (lightweight, read-only)
│   ├── init.go                  ← kl init (Huh wizard)
│   └── config.go                ← kl config
├── cmd/
│   └── khayal/                ← khayal server admin commands
│       ├── main.go
│       └── vault.go             ← khayal vault subcommands
├── ui/
│   ├── react/                      ← Vite + React source
│   │   ├── src/
│   │   │   ├── App.tsx
│   │   │   ├── components/
│   │   │   └── lib/
│   │   └── package.json
│   └── static/                     ← Built output (embedded in binary)
├── install/
│   └── check.go                 ← dependency checker + guidance
├── .github/
│   └── workflows/
│       ├── ci.yml               ← test + vet + lint on every PR
│       └── release.yml          ← goreleaser on tag push
├── docker-compose.yml
├── .goreleaser.yml
├── config.example.yaml          ← safe to commit, no secrets
├── .gitignore
├── LICENSE                      ← AGPLv3
├── README.md
└── CONTRIBUTING.md
```

---

## Design System

Khayal uses `github.com/rawnaqs/theme` — the shared Rawnaqs design system. Never define colors or typography directly in Khayal code.

```
Theme dependency flow:

rawnaqs/theme/tokens/tokens.json   ← source of truth
        │
        ▼ generate.py
rawnaqs/theme/generated/
  ├── go/theme.go        ← imported in CLI (Lip Gloss constants)
  ├── css/variables.css  ← imported in ui/static/style.css
  └── python/theme.py    ← imported in any Python tooling

rawnaqs/theme/custom/
  └── go/styles.go       ← pre-built Lip Gloss styles, used in cli/
```

### CLI color usage

```go
import (
    theme  "github.com/rawnaqs/theme/generated/go"
    styles "github.com/rawnaqs/theme/custom/go"
)

// Use pre-built styles
fmt.Println(styles.SuccessStyle.Render("✓ saved"))
fmt.Println(styles.CaptureOK("saved", "#react", "3ms"))
fmt.Println(styles.CaptureQueued("image", "abc123"))
```

### PWA CSS usage

```css
/* ui/static/style.css */
@import url('https://raw.githubusercontent.com/rawnaqs/theme/main/generated/css/variables.css');

body {
  background: var(--bg-base);
  color: var(--text-primary);
  font-family: var(--font-mono);
}
```

---

## Binary

Single binary — `khayal` — does everything:

```bash
khayal init       # first-run setup, generates config + token
khayal start      # starts server + worker
khayal version    # prints version
```

CLI client — `kl` — is a separate subpackage compiled into the same binary or as standalone:

```bash
kl "thought"
kl --url https://...
kl --image ~/file.png
kl search "query"
kl status
kl init
kl config set token abc123
kl config set host http://100.x.x.x:1133
```

---

## Data Directory

```
~/.config/khayal/
├── config.yaml          ← main config (permissions: 600)
├── khayal.db            ← SQLite: job queue + embeddings
└── logs/
    └── khayal.log       ← request + system logs
```

Vault lives wherever the user points it — completely separate from `~/.config/khayal/`.

---

## Config

Format: YAML only. Behavior on missing/malformed: fail hard with actionable error message.

```yaml
# config.yaml — never commit this file
# Copy from config.example.yaml and fill in

vault:
  path: ~/Documents/brain              # required — where markdown notes are written
   inbox_dir: khayal                     # relative to vault root
  media:
    default_dir: khayal/media           # fallback for unspecified types
    strategy:
      image: vault                     # saved inside vault, linked relatively
      pdf: vault                       # saved inside vault
      audio: config                    # saved in ~/.config/khayal/media/
      video: config                    # transcript goes to vault, raw file stays here

server:
  host: 127.0.0.1                      # never 0.0.0.0 by default
  port: 1133
  token: ""                            # auto-generated on first run if empty
  log_format: text                     # text | json
  log_file: ~/.config/khayal/logs/khayal.log

llm:
  provider: ollama                     # ollama | groq | openai
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: llama3.2:3b
  vision_model: moondream
  fallback_provider: ""                # groq | openai | "" (none)
  fallback_api_key: ""

worker:
  max_workers: 1                       # configurable concurrency
  max_retries: 3                       # then mark permanently failed
  retry_backoff: exponential           # immediate | fixed | exponential

db:
  path: ~/.config/khayal/khayal.db
```

### Reserved Directories

The following directories are reserved by khayal and must not be modified by users:
- `.khayal-trash/` — soft-delete location for failed notes and media
- `.khayal-pending/` — temporary writes before atomic rename

These directories are automatically created in the vault root and should be excluded from Obsidian/graph views. See "Obsidian Integration" section for recommended settings.

### First Run

```bash
$ khayal start
✗ Config file not found at ~/.config/khayal/config.yaml
  Run 'khayal init' to generate a default config

$ khayal init
Created ~/.config/khayal/config.yaml (permissions: 600)
Created ~/.config/khayal/logs/
Generated token: a3f9c2e1d7b4... (save this — shown once)

Edit ~/.config/khayal/config.yaml to set your vault path
Then run 'khayal start'
```

### Malformed Config

```bash
$ khayal start
✗ Config error: vault.path is required
  Line 4 in ~/.config/khayal/config.yaml
  Fix the error and restart
```

---

## Security Model

| Layer | Rule |
|---|---|
| Default bind | `127.0.0.1` — never `0.0.0.0` unless explicitly set |
| Token | Auto-generated 32-byte hex on first run, printed once, stored in config.yaml |
| Auth | `X-Khayal-Token` header required on every request |
| Logging | Timestamp, method, path, status, latency — never logs token or request body |
| Config permissions | Written as `600` on creation |
| `.gitignore` | `config.yaml`, `khayal.db`, `*.log` auto-ignored |
| Tailscale | User's responsibility — documented in README |
| Media outside vault | Raw audio/video stored in `~/.config/khayal/media/` — never in vault |

---

## Logging

Uses Go stdlib `log/slog` (Go 1.21+). Two formats, configured via `log_format` in config:

```yaml
server:
  log_format: text   # text (default) | json (machine readable)
  log_file: ~/.config/khayal/logs/khayal.log
```

**Text format (default) — human readable:**

```
2024-03-16T14:23:01Z INFO  POST /v1/capture 200 47ms type=text job=abc123
2024-03-16T14:23:04Z INFO  worker: processed text abc123 3.2s
2024-03-16T14:23:18Z ERROR vault write failed path=khayal/note.md err=permission denied code=VAULT_002
```

**JSON format — machine readable, for jq:**

```json
{"time":"2024-03-16T14:23:01Z","level":"INFO","method":"POST","path":"/v1/capture","status":200,"latency_ms":47,"type":"text","job_id":"abc123"}
{"time":"2024-03-16T14:23:04Z","level":"INFO","msg":"worker processed","type":"text","job_id":"abc123","duration_ms":3200}
{"time":"2024-03-16T14:23:18Z","level":"ERROR","msg":"vault write failed","path":"khayal/note.md","error":"permission denied","code":"VAULT_002"}
```

**Fields logged:**

| Field | Always | Notes |
|-------|--------|-------|
| time | ✓ | ISO 8601 |
| level | ✓ | INFO, WARN, ERROR |
| method | ✓ (requests) | HTTP method |
| path | ✓ (requests) | endpoint path |
| status | ✓ (requests) | HTTP status |
| latency_ms | ✓ (requests) | request duration |
| type | capture | text, image, article |
| job_id | capture | job identifier |
| query | search | truncated 50 chars |
| results_count | search | number of results |
| code | errors | error code from taxonomy |

**Never logged:**

- Token (X-Khayal-Token header)
- Request body (capture content)
- Note content

**khayal logs command:**

```bash
khayal logs           # tail human readable log
khayal logs --json    # tail JSON log, pipe to jq
khayal logs --level error  # only show errors
khayal logs --since 1h     # last hour only
```

**jq examples:**

```bash
# average search latency
khayal logs --json | jq 'select(.path=="/v1/search") | .latency_ms' | awk '{sum+=$1; n++} END {print sum/n}'

# all errors in last hour
khayal logs --json --since 1h | jq 'select(.level=="ERROR")'

# capture count by type today
khayal logs --json | jq 'select(.path=="/v1/capture") | .type' | sort | uniq -c
```

---

## API — v1

Base: `/v1/`
Auth: `X-Khayal-Token: <token>` on every request

### Endpoints

```
POST   /v1/capture          → capture anything
GET    /v1/search           → keyword + semantic search
GET    /v1/health           → dependency status + queue counts
GET    /v1/queue            → job list with pagination
GET    /v1/queue/:id        → single job status
```

Queue retry + delete → v2.

---

### POST /v1/capture

**Text / URL — JSON**

```json
POST /v1/capture
Content-Type: application/json

{ "type": "text", "content": "useEffect cleanup runs after every render" }
{ "type": "url",  "content": "https://blog.example.com/post" }
```

**Image — Multipart**

```
POST /v1/capture
Content-Type: multipart/form-data

type=image
file=<binary>
note="optional context"         ← becomes frontmatter context + first paragraph
```

**Response — text (queued, processed by worker)**

```json
{
  "id": "abc123",
  "type": "text",
  "status": "pending",
  "note_path": "",
  "created_at": "2024-03-16T14:23:00Z"
}
```

**Response — url (queued as article)**

```json
{
  "id": "def456",
  "type": "article",
  "status": "pending",
  "note_path": "",
  "created_at": "2024-03-16T14:23:00Z"
}
```

**Response — image (queued)**

```json
{
  "id": "ghi789",
  "type": "image",
  "status": "pending",
  "note_path": "khayal/2024-03-16-image.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

---

### GET /v1/search

```
GET /v1/search?q=distributed+systems&limit=10&mode=hybrid&excerpt_length=200
```

Parameters:

| Param | Required | Default | Options |
|---|---|---|---|
| `q` | yes | — | any string |
| `limit` | no | 10 | max 50 |
| `mode` | no | hybrid | hybrid \| keyword \| semantic |
| `excerpt_length` | no | 200 | max 500 chars |

Response:

```json
{
  "query": "distributed systems",
  "mode": "hybrid",
  "results": [
    {
      "id": "abc123",
      "note_path": "khayal/2024-03-10-cap-theorem.md",
      "title": "CAP Theorem Notes",
      "excerpt": "...consistency and availability cannot both be guaranteed...",
      "score": 0.94,
      "type": "text",
      "created_at": "2024-03-10T09:00:00Z"
    }
  ],
  "total": 3,
  "took_ms": 42
}
```

---

### GET /v1/health

```json
{
  "status": "ok",
  "version": "0.1.0",
  "dependencies": {
    "ollama": { "status": "ok", "host": "http://localhost:11434" },
    "vault":  { "status": "ok", "path": "~/Documents/brain" },
    "db":     { "status": "ok", "path": "~/.config/khayal/khayal.db" }
  },
  "queue": {
    "pending":    2,
    "processing": 1,
    "done":       147,
    "failed":     0
  }
}
```

---

### GET /v1/queue

```
GET /v1/queue?status=pending&limit=20&offset=0
```

Parameters:

| Param | Default | Options |
|---|---|---|
| `status` | all | all \| pending \| processing \| done \| failed |
| `limit` | 20 | max 100 |
| `offset` | 0 | — |

Response:

```json
{
  "total": 3,
  "limit": 20,
  "offset": 0,
  "jobs": [
    {
      "id": "abc123",
      "type": "image",
      "status": "processing",
      "note_path": "khayal/2024-03-16-image.md",
      "created_at": "2024-03-16T14:23:00Z",
      "processed_at": null,
      "error": null
    }
  ]
}
```

---

### GET /v1/queue/:id

```json
{
  "id": "abc123",
  "type": "image",
  "status": "done",
  "note_path": "khayal/2024-03-16-image.md",
  "created_at": "2024-03-16T14:23:00Z",
  "processed_at": "2024-03-16T14:23:12Z",
  "error": null
}
```

---

### Error Responses

```json
400 { "error": "missing required field: content", "code": "CAPTURE_004" }
401 { "error": "invalid token",                    "code": "AUTH_001" }
500 { "error": "failed to write note to vault",   "code": "VAULT_002" }
```

See Error Taxonomy section for full code list.

---

## Error Taxonomy

All API errors return consistent JSON with `code` for machine parsing and `hint` for actionable guidance:

```json
{
  "error": "human readable message",
  "code": "VAULT_002",
  "hint": "check permissions: ls -la ~/brain/khayal/"
}
```

`hint` is optional — present only when a specific action is obvious. Codes are stable across versions — never change existing codes, only add new ones.

### Capture errors

| Code | Meaning | Hint |
|------|---------|------|
| `CAPTURE_001` | content too long (>50,000 chars) | — |
| `CAPTURE_002` | file too large (>20MB) | — |
| `CAPTURE_003` | unsupported file type | — |
| `CAPTURE_004` | missing required field | — |

### Vault errors

| Code | Meaning | Hint |
|------|---------|------|
| `VAULT_001` | vault path not found | check vault.path in config |
| `VAULT_002` | vault write failed | check permissions: `ls -la <vault>/khayal/` |
| `VAULT_003` | file modified externally | mtime check failed, user edits protected |
| `VAULT_004` | filename collision | could not generate unique filename |

### LLM errors

| Code | Meaning | Hint |
|------|---------|------|
| `LLM_001` | ollama unreachable | start ollama: `ollama serve` |
| `LLM_002` | model not found | run: `ollama pull <model>` |
| `LLM_003` | context length exceeded | content too long for model |
| `LLM_004` | fallback exhausted | ollama down, no fallback configured |

### Queue errors

| Code | Meaning | Hint |
|------|---------|------|
| `QUEUE_001` | queue full | backpressure limit reached |
| `QUEUE_002` | job not found | invalid job ID |

### Auth errors

| Code | Meaning | Hint |
|------|---------|------|
| `AUTH_001` | invalid token | get from: `~/.config/khayal/config.yaml` |
| `AUTH_002` | token missing | include X-Khayal-Token header |

### Search errors

| Code | Meaning | Hint |
|------|---------|------|
| `SEARCH_001` | query too short | minimum 2 characters |
| `SEARCH_002` | invalid date range | `from` must be before `to` |
| `SEARCH_003` | invalid mode | must be hybrid, keyword, or semantic |

### System errors

| Code | Meaning | Hint |
|------|---------|------|
| `SYS_001` | database error | khayal.db unreachable or corrupted |
| `SYS_002` | config error | config.yaml missing or malformed |
| `SYS_003` | dep missing | required dependency not installed |

### CLI error mapping

```go
switch apiErr.Code {
case "LLM_001":
    fmt.Println("✗ ollama not running")
    fmt.Println("  → start ollama: ollama serve")
case "AUTH_001":
    fmt.Println("✗ invalid token")
    fmt.Println("  → get token: cat ~/.config/khayal/config.yaml")
case "VAULT_002":
    fmt.Println("✗ cannot write to vault")
    fmt.Println("  → check permissions: ls -la " + vaultPath)
// ... etc
}
```

---

## Worker

- Configurable concurrency via `worker.max_workers` in config
- Single goroutine per worker, jobs processed serially within each worker
- Exponential backoff on failure
- Max 3 retries then permanently failed
- On permanent failure: note moved to `.khayal-trash/`, media file moved to trash, job marked failed in DB
- On startup: reset any jobs stuck in `processing` state (crash recovery)

### Processing Times (M2 Mac Air)

| Type | Time |
|---|---|
| Text | ~3s |
| Image | ~10s |
| Article / URL | ~15s |

---

## Note Structures

All frontmatter keys: `snake_case`

### Text — done

```markdown
---
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:04
type: text
status: done
tags:
  - react
  - performance
entities:
  people:  []
  amounts: []
  dates:   []
  places:  []
  urls:    []
  orgs:    []
related:
  - khayal/2022-04-10-react-perf.md
  - khayal/2023-01-05-hooks.md
history:
  - at: 2024-03-16T14:23:04
    event: processed
---

# useEffect cleanup runs after every render

## Summary
A brief summary of the thought...

## Key Ideas
- useEffect cleanup prevents memory leaks
- Dependency array controls when effect runs

## Raw
useEffect cleanup runs after every render
```

### Article / URL — done

```markdown
---
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:18
type: article
status: done
source_url: "https://blog.example.com/post"
tags:
  - distributed-systems
entities:
  people:  ["John Doe"]
  amounts: ["2000"]
  dates:   ["2019-03-03"]
  places:  []
  urls:    ["https://blog.example.com/post"]
  orgs:    ["ACME Corp"]
history:
  - at: 2024-03-16T14:23:18
    event: processed
---

# Article Title

## Summary
A concise summary of the article...

## Key Ideas
- First key idea
- Second key idea

## Source
https://blog.example.com/post
```

### Image — processing (before worker)

```markdown
---
created: 2024-03-16T14:23:00
type: image
status: processing
source_file: "media/2024-03-16-image.png"
user_context: "optional note user attached"
---

# Image — 2024-03-16

optional note user attached

![image](media/2024-03-16-image.png)

_Processing — content will appear here shortly_
```

### Image — done (after worker)

```markdown
---
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:12
type: image
status: done
source_file: "media/2024-03-16-image.png"
user_context: "optional note user attached"
tags:
  - system-design
history:
  - at: 2024-03-16T14:23:12
    event: processed
---

# Image — 2024-03-16

optional note user attached

![image](media/2024-03-16-image.png)

## Description
LLM generated description of the image...

## Extracted Text
OCR output here...
```

### Permanent failure

Failed notes and media moved to `.khayal-trash/` (soft-delete). Failure tracked in `khayal.db` only, visible via `GET /v1/queue?status=failed`. User can restore manually if needed.

---

## LLM Interface

```go
type LLM interface {
    Embed(text string) ([]float32, error)
    Generate(prompt string) (string, error)
    DescribeImage(path string) (string, error)
}
```

Implementations: `OllamaClient` (primary), `GroqClient` (fallback), `OpenAIClient` (fallback).

Fallback activates if Ollama is unreachable. If no fallback configured and Ollama is down — raw note is saved, job queued and retried when Ollama recovers.

### Default Models (Ollama)

| Task | Model | Size |
|---|---|---|
| Embeddings | nomic-embed-text | 274MB |
| Text extraction / tagging | llama3.2:3b | 2GB |
| Vision / image description | moondream | 1.8GB |

---

## CLI — kl

### Libraries

| Library | Purpose |
|---|---|
| Cobra | Command structure, flags, help |
| Lip Gloss | Output styling — **use `rawnaqs/theme/custom/go/styles.go`, never define colors directly** |
| Glamour | Markdown rendering in terminal |
| Huh | `kl init` wizard, `kl config set` prompts |
| rawnaqs/theme | Shared design system — colors, typography, pre-built styles |

### Commands

```bash
kl "thought"                     # capture text → ✓ saved · #tag · 3ms
kl --url https://...             # capture URL  → ⏳ queued · article · id: abc123
kl --image ~/screenshot.png      # capture image → ⏳ queued · image · id: def456
kl search "distributed systems"  # search → Glamour renders excerpts
kl status                        # lightweight — server reachable, queue health
kl recent                        # show recent captures
kl browse                        # browse by tag, person, amount
kl stats                        # show vault statistics
kl init                          # Huh wizard → writes ~/.config/khayal/kl.yaml
kl config set token abc123       # update single config value
kl config set host http://...
```

**Note:** khayal is designed to work alongside Obsidian. These CLI commands complement Obsidian's reading interface — use khayal to capture and find, Obsidian to read and navigate.

### kl.yaml

```yaml
# ~/.config/khayal/kl.yaml
host: http://127.0.0.1:1133
token: your-token-here
```

---

## PWA

**Scope:** Capture + Search only. No timeline view, no entity browsing. Use CLI or Obsidian for those features.

- Stack: React + Vite, embedded in binary via `embed.FS`
- Theme: `rawnaqs/theme` — `generated/css/variables.css` imported in `ui/static/style.css`
- No Node.js, no build pipeline, no npm
- Served at `http://<host>:<port>/`

### Features

- Text input
- URL input
- Image upload
- Camera capture
- Offline queue (IndexedDB, ~50 lines JS)
- Search with excerpts

### Offline Behavior

```
No connection → thought saved to IndexedDB
               → UI shows "N items pending sync"
Connection restored → auto-flushes queue silently
```

---

## Dependency Checker

Runs on `khayal start`. Missing dependencies degrade gracefully — text capture always works.

```
$ khayal start

Checking dependencies...
  ✓ Ollama        found at localhost:11434
  ✓ ffmpeg        found at /usr/local/bin/ffmpeg
  ✗ yt-dlp        not found
    → Install: brew install yt-dlp
    → Or: pip install yt-dlp
    → Video ingestion unavailable until installed
    → Continue anyway? [Y/n]
```

---

## CI / CD

### On every PR

```yaml
- go test ./...
- go vet ./...
- staticcheck ./...
- verify generated/ is in sync with tokens.json   ← theme integrity check
```

### On tag push (v*)

```yaml
- goreleaser
  - build for: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
  - create GitHub release with binaries
  - update Homebrew tap (github.com/rawnaqs/homebrew-tap)
  - publish Docker image to ghcr.io/rawnaqs/khayal
```

### Versioning

Semantic versioning: `v0.1.0`

---

## Distribution

```bash
# Homebrew
brew install rawnaqs/tap/khayal

# Direct download
curl -fsSL https://github.com/rawnaqs/khayal/releases/latest/download/install.sh | sh

# Docker
docker pull ghcr.io/rawnaqs/khayal
docker compose up
```

---

## Org Repos Required Before v1 Launch

```
github.com/rawnaqs/theme            ← must exist, khayal depends on it
github.com/rawnaqs/.github          ← org profile (avatar, banner, README)
github.com/rawnaqs/homebrew-tap     ← for brew install to work
github.com/rawnaqs/khayal           ← this repo
```

---

## Vault Compatibility

khayal is designed to work alongside Obsidian. The vault is plain markdown with rich frontmatter. Obsidian reads it natively. Use khayal to capture and find. Use Obsidian to read and navigate.

**Tested compatible with:**

- Obsidian
- Logseq
- Foam
- Any text editor

**Frontmatter fields:**
```yaml
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:04
type: text
status: done
tags:
  - react
entities:
  people:  []
  amounts: []
  dates:   []
  places:  []
  urls:    []
  orgs:    []
related:
  - khayal/2022-04-10-react-perf.md
  - khayal/2023-01-05-hooks.md
history:
  - at: 2024-03-16T14:23:04
    event: processed
```

The `related` field lists paths to semantically related notes (from proactive connections). This feeds Obsidian's graph view.

---

## Obsidian Integration

To use khayal alongside Obsidian, add these paths to your Obsidian vault's **Settings → Files & Links → Excluded Files**:

```
.khayal-trash/
.khayal-pending/
```

This ensures:
- Deleted/failed notes are hidden from Obsidian's graph
- Temporary pending files don't appear in navigation
- Clean separation between khayal's operational files and your notes

---

## What We Are Not Building in v1

To be explicit — these are out of scope and will not be reconsidered for v1:

- No ambient/always-on voice capture
- No Telegram bot (transits third-party servers)
- No iCloud sync (Apple in the middle of private data)
- No Open WebUI integration
- No graph database (Neo4j, SurrealDB)
- No multi-user support
- No Windows support
- No browser extension
- No custom color/font definitions in Khayal — always import from rawnaqs/theme
