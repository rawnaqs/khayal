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
│   │   └── Header.tsx          ← minimal top bar
│   ├── ui/                      ← shadcn/ui components
│   │   └── sheet.tsx            ← slide-over panel (note detail)
│   ├── Onboarding.tsx           ← first-run setup
│   └── ErrorBoundary.tsx        ← error catching
├── hooks/
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
}

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
No settings page in v1 — just the three views
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
