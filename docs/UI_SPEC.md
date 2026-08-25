---

**PWA — Implementation Instructions**

---

## Stack

```
React + TypeScript
shadcn/ui — New York style
Tailwind CSS
Vite
```

---

## shadcn/ui setup

```bash
npx shadcn@latest init
# choose: New York style, no default color — we override with our own
```

Override `globals.css` — use Khayal tokens not shadcn defaults:

```css
@layer base {
  :root {
    --background: 0 0% 6%;          /* #0f0f0f */
    --foreground: 38 65% 67%;       /* #E8B86D */
    --card: 0 0% 10%;               /* #1A1A1A */
    --card-foreground: 38 65% 67%;
    --popover: 0 0% 14%;            /* #242424 */
    --popover-foreground: 38 65% 67%;
    --primary: 35 55% 51%;          /* #C9933A */
    --primary-foreground: 0 0% 6%;
    --secondary: 0 0% 10%;          /* #1A1A1A */
    --secondary-foreground: 38 65% 67%;
    --muted: 0 0% 14%;
    --muted-foreground: 33 40% 48%; /* #A67830 */
    --accent: 0 0% 14%;
    --accent-foreground: 38 65% 67%;
    --destructive: 0 30% 37%;       /* #8B3A3A */
    --destructive-foreground: 38 65% 67%;
    --border: 0 0% 22%;             /* #3A3A3A */
    --input: 0 0% 16%;
    --ring: 35 55% 51%;             /* #C9933A */
    --radius: 0.375rem;             /* New York uses tighter radius */
  }
}

* {
  border-color: hsl(var(--border));
}

body {
  background-color: hsl(var(--background));
  color: hsl(var(--foreground));
  font-family: "IBM Plex Mono", ui-monospace, monospace;
}
```

---

## Component structure

```
src/
├── components/
│   ├── capture/
│   │   ├── CaptureView.tsx     ← main capture screen
│   │   ├── TextCapture.tsx     ← textarea + submit
│   │   ├── UrlCapture.tsx      ← url input
│   │   ├── ImageCapture.tsx    ← file upload
│   │   ├── CaptureResult.tsx   ← success/queued/offline/error tiles
│   │   └── CaptureStats.tsx    ← bento grid stats
│   ├── search/
│   │   ├── SearchView.tsx      ← search with mode chips, filters
│   │   ├── SearchInput.tsx     ← search bar
│   │   ├── ResultCard.tsx      ← generic result card
│   │   ├── ResultHero.tsx      ← hero result (high score)
│   │   ├── ResultCompact.tsx   ← compact result (rest)
│   │   └── HighlightedText.tsx ← keyword highlighting
│   ├── note/
│   │   ├── NoteView.tsx        ← slide-over note detail panel
│   │   ├── FullNoteView.tsx    ← full note content with sections
│   │   └── ExcerptView.tsx     ← excerpt context with search highlights
│   ├── queue/
│   │   ├── QueueView.tsx       ← queue with metrics
│   │   ├── QueueMetrics.tsx    ← queue stats
│   │   ├── ActiveJobCard.tsx   ← processing job
│   │   ├── FailedJobCard.tsx   ← failed job
│   │   ├── FailedJobExpanded.tsx
│   │   ├── DoneItem.tsx        ← completed job
│   │   ├── OfflineSection.tsx  ← offline queue items
│   │   └── RetryAllBanner.tsx  ← retry all failed
│   ├── layout/
│   │   ├── BottomNav.tsx       ← bottom navigation
│   │   └── Header.tsx          ← minimal top bar (brand + security icon)
│   ├── lock/
│   │   ├── LockScreen.tsx      ← unlock gate (PRF) shown when locked
│   │   └── LockSetupPrompt.tsx ← one-time post-onboarding decision
│   ├── settings/
│   │   └── SecuritySheet.tsx   ← security drawer (enable/disable)
│   ├── ui/                      ← shadcn/ui components
│   │   ├── sheet.tsx            ← slide-over panel (note detail)
│   │   └── switch.tsx           ← toggle (lock on/off)
│   ├── Onboarding.tsx           ← first-run setup
│   └── ErrorBoundary.tsx        ← error catching
├── hooks/
│   ├── useVaultLock.tsx         ← app-lock state + token/key context
│   ├── useCapture.ts            ← capture with offline fallback
│   ├── useSearch.ts             ← search execution
│   ├── useStats.ts              ← polling stats
│   ├── useQueue.ts              ← queue polling
│   ├── useServerStatus.ts       ← health polling
│   ├── useNote.ts               ← fetch and cache note content
│   └── useSubmitLock.ts         ← prevent double-submit
├── lib/
│   ├── api.ts                   ← KhayalClient
│   ├── offline.ts               ← IndexedDB + background sync
│   ├── secureVault.ts           ← WebAuthn PRF + AES-GCM primitives
│   ├── vaultStorage.ts          ← IndexedDB vault record + queue
│   ├── constants.ts             ← shared constants
│   └── utils.ts                 ← utility functions
├── sw.ts                        ← service worker (Workbox + bg sync)
├── test/
│   ├── setup.ts                 ← Vitest setup (mocks)
│   └── utils.tsx                ← render helper
├── App.tsx
└── main.tsx
```

---

## shadcn components to install

```bash
npx shadcn@latest add button
npx shadcn@latest add input
npx shadcn@latest add textarea
npx shadcn@latest add badge
npx shadcn@latest add card
npx shadcn@latest add separator
npx shadcn@latest add toast
npx shadcn@latest add tabs
npx shadcn@latest add skeleton
npx shadcn@latest add sheet
```

---

## Bottom navigation — thumb reachable

```tsx
// BottomNav.tsx
const tabs = [
  { id: 'capture', label: 'capture', icon: PenLine },
  { id: 'search',  label: 'search',  icon: Search },
  { id: 'queue',   label: 'queue',   icon: Clock },
]

// active tab: primary color
// inactive: muted-foreground
// fixed bottom, full width, 60px height
// safe area padding for iOS home indicator
```

```css
.bottom-nav {
  padding-bottom: env(safe-area-inset-bottom);
}
```

---

## Capture view — the most important screen

```tsx
// CaptureView.tsx
// auto-focus textarea on mount
// useEffect(() => { ref.current?.focus() }, [])

// layout:
// - full height between header and bottom nav
// - textarea takes all available space (flex-1)
// - type picker row (text/url/image/camera) above submit
// - submit button full width, 52px height (easy tap target)

// states:
// idle     → textarea focused, ready
// loading  → button shows spinner, textarea disabled
// success  → show CaptureResult, auto-clear after 2s, refocus
// offline  → show "saved offline" variant of CaptureResult
// error    → show error message with hint
```

---

## Capture result — instant feedback

```tsx
// CaptureResult.tsx
// success:
<div className="flex flex-col items-center gap-2 py-6">
  <span className="text-green-500 text-lg font-bold">✓ saved</span>
  <div className="flex gap-2">
    {tags.map(tag => <Badge variant="outline">{tag}</Badge>)}
  </div>
  <span className="text-muted-foreground text-sm">{duration}ms</span>
</div>

// queued:
<div className="flex flex-col items-center gap-2 py-6">
  <span className="text-yellow-500 text-lg">⏳ queued</span>
  <span className="text-muted-foreground text-sm">{type} · id: {id}</span>
</div>

// offline:
<div className="flex flex-col items-center gap-2 py-6">
  <span className="text-muted-foreground text-lg">saved offline</span>
  <span className="text-muted-foreground text-sm">will sync when connected</span>
</div>
```

---

## Search results — use Card

```tsx
// ResultCard.tsx
<Card className="cursor-pointer hover:bg-card/80 transition-colors">
  <CardContent className="p-4">
    <div className="flex justify-between items-start mb-1">
      <span className="font-bold text-foreground text-sm leading-tight">
        {title}
      </span>
      <span className="text-muted-foreground text-xs ml-2 shrink-0">
        {score.toFixed(2)}
      </span>
    </div>
    <div className="flex items-center gap-1 mb-2 flex-wrap">
      <span className="text-muted-foreground text-xs">{date}</span>
      <Badge variant="outline" className="text-xs px-1 py-0">{type}</Badge>
      {tags.slice(0, 3).map(tag =>
        <Badge variant="secondary" className="text-xs px-1 py-0">#{tag}</Badge>
      )}
    </div>
    <p className="text-muted-foreground text-xs leading-relaxed border-l-2 border-border pl-2 italic">
      {excerpt}
    </p>
  </CardContent>
</Card>
```

---

## Note view — slide-over detail panel

When a search result is tapped, a `Sheet` (shadcn/ui) slides over from the right showing the full note content. The `NoteView` component orchestrates the overlay and delegates to `FullNoteView` for content display.

```tsx
// NoteView.tsx
// Sheet component (shadcn/ui)
// Slides in from the right, takes full width on mobile
// Header: back button + title
// Body: FullNoteView or loading skeleton
// Props: notePath, query (for excerpt highlighting), onClose
```

### Full note content

```tsx
// FullNoteView.tsx
// Sections displayed in order:
// - Title (h1)
// - Type badge + date + status
// - Tags (Badge component)
// - Summary section (## Summary content)
// - Key Ideas (## Key Ideas list)
// - Description (## Description for images)
// - Source URL (## Source link)
// - Raw content (## Raw — full original capture)
```

### Search excerpt context

When a `query` is provided (user tapped from search results), the `ExcerptView` highlights the matching section:

```tsx
// ExcerptView.tsx
// Shows excerpt section name + content with keyword highlighting
// Appears at the top of the note, above full content
// Marks the section that matched the search query
```

### useNote hook

```tsx
// hooks/useNote.ts
function useNote(notePath: string | null, query?: string): {
  note: NoteResponse | null
  loading: boolean
  error: string | null
}
```

Fetches note content from `GET /v1/notes/{path}` with optional `?q=` for excerpt context. Resets when `notePath` changes.

---

## Offline queue — IndexedDB

```ts
// lib/offline.ts
const DB_NAME = 'khayal-offline'
const STORE   = 'captures'

export async function saveOffline(capture: CaptureRequest): Promise<void>
export async function getOfflineQueue(): Promise<OfflineCapture[]>
export async function removeOfflineItem(id: string): Promise<void>
export async function flushOfflineQueue(api: KhayalClient): Promise<void>
```

Auto-flush on:
- App focus (`window.addEventListener('focus', flush)`)
- Online event (`window.addEventListener('online', flush)`)

---

## API client

```ts
// lib/api.ts
export class KhayalClient {
  constructor(host: string, token: string)

  capture(req: CaptureRequest): Promise<CaptureResponse>
  search(query: string, opts: SearchOptions): Promise<SearchResponse>
  getNote(notePath: string, query?: string): Promise<NoteResponse>
  health(): Promise<HealthResponse>
  queue(opts: QueueOptions): Promise<QueueResponse>
}

// read host + token from:
// 1. localStorage (set during onboarding)
// 2. env VITE_KHAYAL_HOST / VITE_KHAYAL_TOKEN (for dev)
```

---

## Onboarding — first run

If no host/token in localStorage — show setup screen before anything else:

```tsx
<div className="flex flex-col gap-4 p-6 h-screen justify-center">
  <img src="/icon-192.png" className="w-16 h-16 mx-auto" />
  <h1 className="text-center font-bold text-xl">khayal</h1>
  <Input placeholder="server address — http://100.x.x.x:7766" />
  <Input placeholder="token" type="password" />
  <Button className="w-full" onClick={connect}>connect</Button>
</div>
```

Test connection before saving. Show error if unreachable.

---

## App lock — WebAuthn PRF (Face ID / Touch ID)

An **optional** app-lock gate that encrypts the server token (and the offline
queue) at rest using a key derived from the platform authenticator via the
WebAuthn PRF extension. Client-side only — no server or API changes.

### What it protects against

- **In scope:** casual/opportunistic physical access to an unlocked device.
- **Out of scope:** an attacker with sustained access to an unlocked,
  authenticated device (devtools, browser storage inspection, OS compromise).
  This is a client-side web app — it cannot defend against a fully
  compromised endpoint. Stated plainly in the setup copy; never oversold.

### Lock states

```
none  → token in localStorage, plaintext (default, current behavior)
prf   → token + offline queue encrypted, key derived from WebAuthn PRF
```

There is **no passcode tier**. If the device/browser doesn't support a
platform authenticator, the setup prompt instead offers token-persistence
choices (see below) — never a PIN.

### Setup prompt (once, after onboarding)

Shown immediately after the Onboarding connection test succeeds, and **only
once per device**. The decision is persisted in
`localStorage['khayal-lock-setup-decided']` so the prompt never re-appears.

- **Biometrics available** → "secure this device?" with `set up face id`
  (PRF registration) or `skip for now` (remember the token).
- **Biometrics unavailable** → "face id isn't supported here" with
  `don't remember my token` (token held in memory only; re-enter each open
  via password manager) or `remember my token` (plaintext `localStorage`).

Either way, onboarding always completes — the prompt never blocks the app.

### Registration flow

1. Feature-detect
   `PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()`.
2. If unavailable → fall through to the persistence choice.
3. `navigator.credentials.create({ ... extensions: { prf: {} } })` with
   `residentKey: "required"`, `userVerification: "required"`.
4. If `getClientExtensionResults().prf.enabled` → generate a random 32-byte
   salt, store `{ credentialId, salt, mode: "prf" }` in IndexedDB, encrypt the
   token + offline queue under the PRF-derived key, delete plaintext copies.
5. If PRF is unsupported/cancelled → persistence choice (never PIN).

### Unlock flow (every app open, when `prf`)

- Load the vault record, run `navigator.credentials.get({ prf: { eval: { first: salt } } })`.
- `HKDF(prfOutput)` → AES-GCM-256 key → decrypt token + offline queue.
- Hold the plaintext token in memory only (React context, cleared on reload).
- On failure/cancel: "try again" + "reconnect instead".

### Security drawer

A shield icon in `Header.tsx` opens a bottom `Sheet` ("security"):

- `none` → `set up face id`; on unsupported devices, the same
  `remember / don't remember` choice as setup.
- `prf` → a `Switch` (checked) to disable. Disabling re-authenticates via the
  same PRF ceremony before decrypting and writing the token back to
  `localStorage` and deleting the vault record.

### Recovery

Losing the biometric credential does not lose notes — the vault lives on the
server. `LockScreen` offers "reconnect instead", which clears the vault record
and re-runs the onboarding (host + token entry). This is a re-onboard, not a
restore.

### Background sync

When `prf` is active the service worker's background sync skips (it cannot
decrypt the queue); queued items flush next time the app is opened and
unlocked. When `none`, behavior is unchanged.

### What gets encrypted

1. `token` — the `X-Khayal-Token` value.
2. `offline-queue` — captured note content waiting to sync.

Recent searches and UI prefs stay plaintext (not sensitive enough to justify
the complexity).

---

## PWA manifest

The manifest is generated by `vite-plugin-pwa` (not a standalone file). Configuration is in `vite.config.ts`:

```ts
// vite.config.ts
VitePWA({
  manifest: {
    name: 'Khayal',
    short_name: 'khayal',
    description: 'Personal knowledge vault',
    start_url: '/',
    display: 'standalone',
    orientation: 'portrait',
    background_color: '#070707',
    theme_color: '#C9933A',
    icons: [
      { src: '/icon-192.png', sizes: '192x192', type: 'image/png' },
      { src: '/icon-512.png', sizes: '512x512', type: 'image/png' },
    ],
  },
})
```

Generated files in build output:
- `manifest.webmanifest` — auto-generated
- `registerSW.js` — SW registration
- `sw.js` — service worker

---

## Mobile-specific rules

```
Touch targets:     minimum 44px height on all interactive elements
Safe area:         env(safe-area-inset-bottom) on bottom nav
Keyboard:          auto-focus textarea on capture view mount
Viewport:          <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
Scroll:            no horizontal scroll ever — overflow-x: hidden on body
iOS tap highlight: -webkit-tap-highlight-color: transparent
Font size:         minimum 16px on inputs — prevents iOS auto-zoom
```

---

## Search view — mode chips, recent searches, filters

```tsx
// SearchView.tsx
// Mode chips below search bar
const modes = ['hybrid', 'keyword', 'semantic']
// active mode: gold background
// inactive: border only

// Recent searches (idle state)
// stored in localStorage under 'khayal-recent-searches'
// max 10 entries, newest first, deduplicated
// shown when no active search

// Suggestion chips (idle state)
// static list: ['people', 'payments', 'this week', 'ideas', 'decisions', 'meetings']
// click triggers search

// Filter chips (results state)
// types: all, text, article, image
// client-side filtering (no API change)
// visible even when filter returns empty results

// Keyword highlighting
// .hl class on matched terms in title and excerpt
// gold color (#E8B86D) with subtle background
```

---

## Search results — hero + compact

```tsx
// Hero result (first result, score > 0.9)
// .r1 class — gold gradient top line, ghost number
// title + excerpt with keyword highlighting

// Compact results (rest)
// .rc class — numbered, type badge, tags, score
// hover state: background change

// No results state
// mode suggestions (try keyword/semantic)
// capture link (navigates to capture with query)
```

---

## AI answer row — inline, expanding (v1.1 phase 2.6)

```tsx
// AIAnswerRow.tsx — first item in the results list
// Collapsed: gold sparkles + "AI Answer" + chevron, styled like a result row
// Click (idle) -> fetch with overview=true + expand in place
// Expansion: height animation (~320ms), shimmer skeleton bars while loading
// Ready: grounded answer text; [n] tokens are gold superscript chips that
//   smooth-scroll to the anchored result card
// Collapse keeps the answer cached (re-expand is instant); dismiss resets
// Error: quiet inline retry hint — results are never affected
// Never auto-triggered: the click is the only trigger
```

---

## Capture result tiles — 4 states

```tsx
// Success tile (.tile-ok)
// green border, checkmark icon
// title: "saved", subtitle: "{type} · {processingTime}ms"
// auto-dismiss 3s with drain bar animation

// Queued tile (.tile-q)
// yellow border, spinning Loader2 icon
// title: "queued", subtitle: "{note_path} · {id}"
// step progress dots: done (green) / active (yellow pulsing) / waiting (gray)
// auto-dismiss 4s with drain bar

// Offline tile (.tile-off)
// gold border, Zap icon
// title: "saved offline", subtitle: "will sync when connected"
// auto-dismiss 3.5s with drain bar

// Error tile (.tile-err)
// red border, AlertTriangle icon
// title: "capture failed"
// error box with code + message
// actions: retry + discard buttons
// NO auto-dismiss (stays until dismissed)
```

---

## Bento grid stats — 3 tiles

```tsx
// Streak tile (.bt-streak)
// gold gradient background
// SVG arc progress (current / next_milestone)
// big number + "day streak" + goal text
// week dots bar (7 dots, this_week data)

// Today tile (.bt)
// big number + "captures"
// hourly mini bars (24 bars, by_hour data)
// current hour gets special styling (.hb.now)
// footer: avg/day + last capture time

// Vault tile (.bt wide)
// big number + "notes" + delta badge (+8 today)
// center stat: last 7d total
// 7-day sparkline bars (last_7_days data)
```

---

## Frontend constants

```ts
// lib/constants.ts
export const STORAGE_KEYS = {
  TOKEN: 'khayal_token',
  HOST: 'khayal_host',
  RECENT_SEARCHES: 'khayal-recent-searches',
  LOCK_SETUP_DECIDED: 'khayal-lock-setup-decided',
}

export const VAULT_LOCK = {
  DB_NAME: 'khayal-offline',   // same DB as offline queue
  STORE_OFFLINE: 'offline',
  STORE_VAULT: 'vault',
  DB_VERSION: 2,
  PRF_SALT_BYTES: 32,
}

export type LockMode = 'none' | 'prf'

export const SEARCH_SUGGESTIONS = ['people', 'payments', 'this week', 'ideas', 'decisions', 'meetings']

export const PROCESSING_STEPS = {
  text: ['saved', 'tagging', 'summarizing', 'writing'],
  image: ['saved', 'describing', 'tagging', 'writing'],
  article: ['saved', 'extracting', 'summarizing', 'writing'],
}

export const LIMITS = {
  SEARCH_RESULTS: 20,
  QUEUE_JOBS: 50,
  RECENT_SEARCHES: 10,
  DONE_JOBS_SHOWN: 5,
  TAGS_HERO: 3,
  TAGS_COMPACT: 2,
  HERO_SCORE_THRESHOLD: 0.9,
}

export const TIMEOUTS = {
  CAPTURE_DISMISS: 3500,
  STATS_POLL: 60000,
  SERVER_STATUS_POLL: 30000,
}

export const GREETINGS = [
  { maxHour: 5, text: 'late night thoughts?' },
  { maxHour: 12, text: 'good morning' },
  { maxHour: 17, text: 'good afternoon' },
  { maxHour: 21, text: 'good evening' },
  { maxHour: 24, text: 'late night thoughts?' },
]
```

---

## What NOT to build

```
No page routing — single page, tab switching only
No sidebar — bottom nav only
No modals for capture — inline state changes
No pull to refresh — auto-refresh on focus
No pagination — infinite scroll on search results
No dedicated settings tab — a header icon + security sheet is enough
No passcode/PIN lock — WebAuthn PRF only (persistence choice as fallback)
No inactivity-timeout re-lock — open/closed only
```

---

## Service Worker (Workbox)

The service worker (`src/sw.ts`) uses `vite-plugin-pwa` with Workbox for:

### Precaching
All build assets (JS, CSS, HTML, icons) are precached on install.

### Runtime Caching Strategies

| Asset Type | Strategy | Cache Name | TTL |
|------------|----------|------------|-----|
| App shell (JS/CSS/HTML) | CacheFirst | khayal-shell | 30 days |
| Images | CacheFirst | khayal-shell | 30 days |
| `/v1/health` | NetworkFirst | khayal-health | 1 min |
| `/v1/stats` | StaleWhileRevalidate | khayal-stats | 1 min |
| `/v1/search` | NetworkFirst | khayal-search | 5 min |
| `/v1/queue` | NetworkFirst | khayal-queue | 5 min |
| `/v1/capture` | NetworkOnly | — | — |

### Background Sync
When a capture fails due to network, it's saved to IndexedDB. The service worker registers a background sync event that retries failed captures when connection is restored.

### Push Notifications (Optional)
The service worker has a push event handler for future notification support.

### Configuration

```ts
// vite.config.ts
VitePWA({
  registerType: 'autoUpdate',
  workbox: {
    globPatterns: ['**/*.{js,css,html,ico,png,svg}'],
    runtimeCaching: [
      {
        urlPattern: /^https?:\/\/.*\.(js|css|html|ico|png|svg)$/,
        handler: 'CacheFirst',
        options: { cacheName: 'khayal-shell' }
      },
      // ... other strategies
    ]
  }
})
```

---

## Embedded in khayal binary

The PWA is built with Vite and embedded via Go's `embed.FS`. The build output is embedded at compile time. No separate server, no CDN.

```go
//go:embed ui/static
var uiFiles embed.FS

// serve at /
http.Handle("/", http.FileServer(http.FS(uiFiles)))
```

Vite build output goes to `internal/api/ui/static/` (configured in `vite.config.ts` `outDir`). The Go embed directive picks it up at build time.
