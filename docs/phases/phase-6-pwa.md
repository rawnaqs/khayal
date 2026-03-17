# Phase 6: PWA

> Web interface with React. Updated: 2026-03-17

## Goals

- [ ] Vite + React setup
- [ ] Capture UI
- [ ] Search UI
- [ ] Offline queue (IndexedDB)
- [ ] Go static serving
- [ ] SPA fallback

## Directory Structure

```
ui/
├── react/                    # Separate npm project
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── components/
│   │   │   ├── Capture.tsx
│   │   │   ├── Search.tsx
│   │   │   ├── Queue.tsx
│   │   │   └── Layout.tsx
│   │   ├── lib/
│   │   │   ├── api.ts
│   │   │   ├── offline.ts
│   │   │   └── store.ts
│   │   └── styles/
│   │       └── global.css
│   └── public/
│       └── manifest.json
└── static/                    # Built files (after npm build)
    ├── assets/
    └── index.html
```

## Step 6.1: Vite + React Setup

**File:** `ui/react/package.json`

```json
{
  "name": "khayal-pwa",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.22.0",
    "zustand": "^4.5.0",
    "idb-keyval": "^6.2.1"
  },
  "devDependencies": {
    "@types/react": "^18.2.55",
    "@types/react-dom": "^18.2.19",
    "@vitejs/plugin-react": "^4.2.1",
    "typescript": "^5.3.3",
    "vite": "^5.1.0"
  }
}
```

**File:** `ui/react/vite.config.ts`

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: '../static',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
  },
})
```

## Step 6.2: Core Components

### Layout

**File:** `ui/react/src/components/Layout.tsx`

```tsx
import { ReactNode } from 'react'

export function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="p-4 border-b border-border">
        <h1 className="text-xl font-bold">Khayal</h1>
        <nav className="flex gap-4 mt-2">
          <a href="/" className="text-muted-foreground hover:text-foreground">Capture</a>
          <a href="/search" className="text-muted-foreground hover:text-foreground">Search</a>
          <a href="/queue" className="text-muted-foreground hover:text-foreground">Queue</a>
        </nav>
      </header>
      <main className="p-4">
        {children}
      </main>
    </div>
  )
}
```

### Capture Form

**File:** `ui/react/src/components/Capture.tsx`

```tsx
import { useState } from 'react'
import { capture, uploadImage } from '../lib/api'
import { addToOfflineQueue } from '../lib/offline'

export function Capture() {
  const [mode, setMode] = useState<'text' | 'url' | 'image'>('text')
  const [content, setContent] = useState('')
  const [image, setImage] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setUploading(true)
    
    try {
      if (mode === 'image' && image) {
        const response = await uploadImage(image, content)
        setResult(`⏳ Queued as ${response.type} · ID: ${response.id}`)
      } else {
        const response = await capture(mode, content)
        if (response.status === 'done') {
          setResult('✓ Saved')
        } else {
          setResult(`⏳ Queued as ${response.type} · ID: ${response.id}`)
        }
      }
    } catch (err) {
      // Try offline queue
      if (mode === 'text') {
        await addToOfflineQueue({ type: mode, content })
        setResult('✓ Saved offline (will sync when connected)')
      } else {
        setResult('Error: ' + (err as Error).message)
      }
    }
    
    setUploading(false)
    setContent('')
    setImage(null)
  }

  return (
    <div className="max-w-lg mx-auto">
      <div className="flex gap-2 mb-4">
        <button
          className={`px-4 py-2 rounded ${mode === 'text' ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}
          onClick={() => setMode('text')}
        >
          Text
        </button>
        <button
          className={`px-4 py-2 rounded ${mode === 'url' ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}
          onClick={() => setMode('url')}
        >
          URL
        </button>
        <button
          className={`px-4 py-2 rounded ${mode === 'image' ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}
          onClick={() => setMode('image')}
        >
          Image
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {mode === 'image' ? (
          <>
            <input
              type="file"
              accept="image/*"
              onChange={(e) => setImage(e.target.files?.[0] || null)}
              className="w-full p-2 border rounded"
            />
            <textarea
              placeholder="Optional note..."
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="w-full p-2 border rounded"
              rows={3}
            />
          </>
        ) : (
          <textarea
            placeholder={mode === 'url' ? 'https://...' : 'Your thought...'}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            className="w-full p-2 border rounded"
            rows={5}
            autoFocus
          />
        )}
        
        <button
          type="submit"
          disabled={uploading || !content}
          className="w-full py-2 bg-primary text-primary-foreground rounded hover:opacity-90 disabled:opacity-50"
        >
          {uploading ? 'Capturing...' : 'Capture'}
        </button>
      </form>

      {result && (
        <div className="mt-4 p-3 bg-muted rounded text-center">
          {result}
        </div>
      )}
    </div>
  )
}
```

### Search UI

**File:** `ui/react/src/components/Search.tsx`

```tsx
import { useState } from 'react'
import { search } from '../lib/api'

export function Search() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!query.trim()) return
    
    setLoading(true)
    try {
      const response = await search(query)
      setResults(response.results)
    } catch (err) {
      console.error(err)
    }
    setLoading(false)
  }

  return (
    <div className="max-w-2xl mx-auto">
      <form onSubmit={handleSearch} className="mb-6">
        <input
          type="text"
          placeholder="Search your knowledge..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full p-3 border rounded text-lg"
          autoFocus
        />
      </form>

      {loading && <div className="text-center text-muted">Searching...</div>}

      <div className="space-y-4">
        {results.map((result) => (
          <div key={result.id} className="p-4 border rounded hover:bg-muted/50">
            <h3 className="font-semibold">{result.title}</h3>
            <p className="text-sm text-muted-foreground">{result.note_path}</p>
            <p className="mt-2">{result.excerpt}</p>
            <div className="mt-2 text-xs text-muted">
              {result.type} · {result.score.toFixed(2)} score
            </div>
          </div>
        ))}
      </div>

      {!loading && results.length === 0 && query && (
        <div className="text-center text-muted">No results found</div>
      )}
    </div>
  )
}

interface SearchResult {
  id: string
  note_path: string
  title: string
  excerpt: string
  score: number
  type: string
  created_at: string
}
```

## Step 6.3: Offline Support

**File:** `ui/react/src/lib/offline.ts`

```ts
import { get, set, del, keys } from 'idb-keyval'

interface OfflineJob {
  id: string
  type: string
  content: string
  timestamp: number
}

const OFFLINE_QUEUE_KEY = 'khayal_offline_queue'

export async function addToOfflineQueue(job: OfflineJob): Promise<void> {
  const queue = await get<OfflineJob[]>(OFFLINE_QUEUE_KEY) || []
  queue.push({ ...job, timestamp: Date.now() })
  await set(OFFLINE_QUEUE_KEY, queue)
}

export async function getOfflineQueue(): Promise<OfflineJob[]> {
  return (await get<OfflineJob[]>(OFFLINE_QUEUE_KEY)) || []
}

export async function clearOfflineQueue(): Promise<void> {
  await del(OFFLINE_QUEUE_KEY)
}

export async function syncOfflineQueue(apiHost: string, token: string): Promise<number> {
  const queue = await getOfflineQueue()
  let synced = 0
  
  for (const job of queue) {
    try {
      await fetch(`${apiHost}/v1/capture`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Khayal-Token': token,
        },
        body: JSON.stringify({
          type: job.type,
          content: job.content,
        }),
      })
      synced++
    } catch {
      // Keep in queue for next sync
    }
  }
  
  // Remove synced items
  const remaining = queue.slice(synced)
  await set(OFFLINE_QUEUE_KEY, remaining)
  
  return synced
}

// Check connection and sync
export function setupOfflineSync(apiHost: string, token: string) {
  const sync = async () => {
    if (navigator.onLine) {
      const count = await syncOfflineQueue(apiHost, token)
      if (count > 0) {
        console.log(`Synced ${count} offline items`)
      }
    }
  }
  
  window.addEventListener('online', sync)
  
  // Initial sync
  sync()
}
```

## Step 6.4: API Client

**File:** `ui/react/src/lib/api.ts`

```ts
const API_BASE = import.meta.env.VITE_API_BASE || ''

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('khayal_token')
  
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-Khayal-Token': token || '',
      ...options.headers,
    },
  })
  
  if (!response.ok) {
    throw new Error(`API error: ${response.status}`)
  }
  
  return response.json()
}

export interface CaptureResponse {
  id: string
  type: string
  status: string
  note_path: string
  created_at: string
}

export async function capture(type: string, content: string): Promise<CaptureResponse> {
  return request<CaptureResponse>('/v1/capture', {
    method: 'POST',
    body: JSON.stringify({ type, content }),
  })
}

export async function uploadImage(file: File, note: string = ''): Promise<CaptureResponse> {
  const token = localStorage.getItem('khayal_token')
  const formData = new FormData()
  formData.append('type', 'image')
  formData.append('file', file)
  if (note) formData.append('note', note)
  
  const response = await fetch(`${API_BASE}/v1/capture`, {
    method: 'POST',
    headers: {
      'X-Khayal-Token': token || '',
    },
    body: formData,
  })
  
  return response.json()
}

export interface SearchResponse {
  query: string
  mode: string
  results: SearchResult[]
  total: number
  took_ms: number
}

export interface SearchResult {
  id: string
  note_path: string
  title: string
  excerpt: string
  score: number
  type: string
  created_at: string
}

export async function search(query: string, limit = 10, mode = 'hybrid'): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: query, limit: String(limit), mode })
  return request<SearchResponse>(`/v1/search?${params}`)
}

export interface QueueResponse {
  total: number
  limit: number
  offset: number
  jobs: Job[]
}

export interface Job {
  id: string
  type: string
  status: string
  note_path: string
  created_at: string
  processed_at: string | null
  error: string | null
}

export async function listQueue(status = '', limit = 20, offset = 0): Promise<QueueResponse> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (status) params.set('status', status)
  return request<QueueResponse>(`/v1/queue?${params}`)
}
```

## Step 6.5: Go Static Serving

**File:** `internal/api/static.go`

```go
import (
    "embed"
    "io/fs"
    "net/http"
    "path"
)

//go:embed all:../../ui/static
var staticFiles embed.FS

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
    // Get the filesystem
    fsys, err := fs.Sub(staticFiles, "ui/static")
    if err != nil {
        http.Error(w, "Static files not found", 500)
        return
    }
    
    // Try to serve the file
    filePath := path.Clean(r.URL.Path)
    if filePath == "." {
        filePath = "index.html"
    }
    
    file, err := fsys.Open(filePath)
    if err != nil {
        // SPA fallback: serve index.html
        index, err := fsys.Open("index.html")
        if err != nil {
            http.Error(w, "Not found", 404)
            return
        }
        defer index.Close()
        
        http.ServeContent(w, r, "index.html", time.Time{}, index.(io.ReadSeeker))
        return
    }
    defer file.Close()
    
    stat, _ := file.Stat()
    if stat.IsDir() {
        // Try index.html
        index, err := fsys.Open(path.Join(filePath, "index.html"))
        if err != nil {
            http.Error(w, "Not found", 404)
            return
        }
        defer index.Close()
        
        http.ServeContent(w, r, path.Join(filePath, "index.html"), time.Time{}, index.(io.ReadSeeker))
        return
    }
    
    // Determine content type
    contentType := mime.TypeByExtension(path.Ext(filePath))
    if contentType == "" {
        contentType = "application/octet-stream"
    }
    
    w.Header().Set("Content-Type", contentType)
    http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
}
```

## Step 6.6: Build Integration

In `ui/react/src/main.tsx`:

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import './styles/global.css'

// Setup offline sync
const token = localStorage.getItem('khayal_token')
const host = localStorage.getItem('khayal_host')
if (token && host) {
  import('./lib/offline').then(({ setupOfflineSync }) => {
    setupOfflineSync(host, token)
  })
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

## Testing

- [ ] Component tests (Vitest)
- [ ] E2E tests (Playwright)
- [ ] Offline queue tests

```bash
cd ui/react
npm test
npm run build
```

## Checklist

- [ ] Vite + React setup
- [ ] Theme CSS integration
- [ ] Capture form (text, url, image)
- [ ] Search UI
- [ ] Queue display
- [ ] Offline queue (IndexedDB)
- [ ] Auto-sync on reconnect
- [ ] Go static serving
- [ ] SPA fallback
- [ ] PWA manifest
- [ ] Service worker (optional)
- [ ] Tests passing

## Next Phase

[Phase 7: Polish](phase-7-polish.md)

## Notes

- Use `rawnaqs/theme/theme.css` for styling
- API calls include `X-Khayal-Token` header
- Offline queue stores jobs in IndexedDB
- Auto-sync when connection restored
- Static files embedded at compile time
