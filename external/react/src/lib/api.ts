import { STORAGE_KEYS } from './constants'

export interface CaptureRequest {
  type: 'text' | 'url' | 'image'
  content: string
}

export interface CaptureResponse {
  id: string
  status: string
  type: string
  note_path?: string
  error?: string
  code?: string
}

export interface SearchOptions {
  mode?: 'keyword' | 'semantic' | 'hybrid'
  limit?: number
  excerpt_length?: number
  from?: string
  to?: string
  connections?: boolean
}

export interface SearchResult {
  id: string
  note_path: string
  title: string
  excerpt: string
  score: number
  type: string
  created_at: string
  tags: string[]
}

export interface SearchResponse {
  query: string
  mode: string
  results: SearchResult[] | null
  total: number
  took_ms: number
}

export interface HealthResponse {
  status: string
  version: string
  update?: {
    available: boolean
    latest: string
    server_version: string
  }
  dependencies: {
    db: { status: string }
    vault: { status: string }
    llm: { status: string; host?: string }
  }
}

export interface QueueOptions {
  status?: string
  limit?: number
  offset?: number
}

export interface QueueJob {
  id: string
  type: string
  status: string
  note_path?: string
  created_at: string
  processed_at?: string
  error?: string
}

export interface QueueResponse {
  total: number
  limit: number
  offset: number
  jobs: QueueJob[]
}

export interface StatsResponse {
  streak: {
    current: number
    best: number
    next_milestone: number
    days_to_milestone: number
    this_week: boolean[]
  }
  today: {
    count: number
    by_hour: number[]
    avg_per_day: number
  }
  vault: {
    total_notes: number
    today_delta: number
    last_capture_at: string
    last_7_days: number[]
  }
}

export class KhayalClient {
  private host: string
  private token: string

  constructor(host: string, token: string) {
    this.host = host.replace(/\/$/, '')
    this.token = token
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const response = await fetch(`${this.host}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'X-Khayal-Token': this.token,
      },
      body: body ? JSON.stringify(body) : undefined,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new Error(error.error || `Request failed: ${response.status}`)
    }

    return response.json()
  }

  async capture(req: CaptureRequest): Promise<CaptureResponse> {
    return this.request<CaptureResponse>('POST', '/v1/capture', req)
  }

  async uploadImage(file: File, note?: string): Promise<CaptureResponse> {
    const formData = new FormData()
    formData.append('file', file)
    if (note) formData.append('note', note)

    const response = await fetch(`${this.host}/v1/capture`, {
      method: 'POST',
      headers: {
        'X-Khayal-Token': this.token,
      },
      body: formData,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }))
      throw new Error(error.error || 'Upload failed')
    }

    return response.json()
  }

  async search(query: string, opts: SearchOptions = {}): Promise<SearchResponse> {
    const params = new URLSearchParams()
    params.set('q', query)
    if (opts.mode) params.set('mode', opts.mode)
    if (opts.limit) params.set('limit', opts.limit.toString())
    if (opts.excerpt_length) params.set('excerpt_length', opts.excerpt_length.toString())
    if (opts.from) params.set('from', opts.from)
    if (opts.to) params.set('to', opts.to)
    if (opts.connections) params.set('connections', 'true')

    return this.request<SearchResponse>('GET', `/v1/search?${params.toString()}`)
  }

  async health(): Promise<HealthResponse> {
    return this.request<HealthResponse>('GET', '/v1/health')
  }

  async queue(opts: QueueOptions = {}): Promise<QueueResponse> {
    const params = new URLSearchParams()
    if (opts.status) params.set('status', opts.status)
    if (opts.limit) params.set('limit', opts.limit.toString())
    if (opts.offset) params.set('offset', opts.offset.toString())

    return this.request<QueueResponse>('GET', `/v1/queue?${params.toString()}`)
  }

  async retryJob(id: string): Promise<void> {
    await this.request('POST', `/v1/queue/${id}/retry`)
  }

  async discardJob(id: string): Promise<void> {
    await this.request('POST', `/v1/queue/${id}/discard`)
  }

  async stats(): Promise<StatsResponse> {
    return this.request<StatsResponse>('GET', '/v1/stats')
  }
}

export function createClient(): KhayalClient {
  // Auto-detect: same origin in production, proxy in dev
  const host = localStorage.getItem(STORAGE_KEYS.HOST) || window.location.origin
  const token = localStorage.getItem(STORAGE_KEYS.TOKEN) || ''
  return new KhayalClient(host, token)
}
