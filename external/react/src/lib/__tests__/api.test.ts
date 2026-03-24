import { describe, it, expect, vi, beforeEach } from 'vitest'
import { KhayalClient, createClient } from '../api'

describe('api.ts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(global.fetch as ReturnType<typeof vi.fn>).mockReset()
  })

  describe('KhayalClient', () => {
    const client = new KhayalClient('http://localhost:1133', 'test-token')

    describe('capture', () => {
      it('should send POST request to /v1/capture', async () => {
        const mockResponse = {
          id: '123',
          status: 'queued',
          type: 'text',
          note_path: 'khayal/2024-01-01/1234.md',
        }
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockResponse),
        })

        const result = await client.capture({ type: 'text', content: 'test' })

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/capture',
          expect.objectContaining({
            method: 'POST',
            headers: expect.objectContaining({
              'Content-Type': 'application/json',
              'X-Khayal-Token': 'test-token',
            }),
            body: JSON.stringify({ type: 'text', content: 'test' }),
          })
        )
        expect(result).toEqual(mockResponse)
      })

      it('should throw error on failed request', async () => {
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: false,
          json: () => Promise.resolve({ error: 'Invalid token' }),
        })

        await expect(
          client.capture({ type: 'text', content: 'test' })
        ).rejects.toThrow('Invalid token')
      })
    })

    describe('search', () => {
      it('should send GET request to /v1/search with query params', async () => {
        const mockResponse = {
          query: 'test',
          mode: 'hybrid',
          results: [],
          total: 0,
          took_ms: 100,
        }
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockResponse),
        })

        const result = await client.search('test', { mode: 'keyword', limit: 10 })

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/search?q=test&mode=keyword&limit=10',
          expect.objectContaining({
            method: 'GET',
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('health', () => {
      it('should send GET request to /v1/health', async () => {
        const mockResponse = {
          status: 'ok',
          version: '0.1.0',
          dependencies: {
            db: { status: 'ok' },
            vault: { status: 'ok' },
            llm: { status: 'ok', host: 'http://localhost:11434' },
          },
          queue: {
            pending: 0,
            queued: 0,
            processing: 0,
            done: 10,
            failed: 0,
          },
        }
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockResponse),
        })

        const result = await client.health()

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/health',
          expect.objectContaining({ method: 'GET' })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('queue', () => {
      it('should send GET request to /v1/queue with options', async () => {
        const mockResponse = {
          total: 5,
          limit: 10,
          offset: 0,
          jobs: [],
        }
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockResponse),
        })

        const result = await client.queue({ status: 'failed', limit: 10 })

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/queue?status=failed&limit=10',
          expect.objectContaining({ method: 'GET' })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('stats', () => {
      it('should send GET request to /v1/stats', async () => {
        const mockResponse = {
          streak: { current: 5, best: 10, next_milestone: 7, days_to_milestone: 2, this_week: [true, true, true, false, false, false, false] },
          today: { count: 3, by_hour: [], avg_per_day: 2.5 },
          vault: { total_notes: 100, today_delta: 3, last_capture_at: '2024-01-01T10:00:00Z', last_7_days: [1, 2, 3, 4, 5, 6, 7] },
        }
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockResponse),
        })

        const result = await client.stats()

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/stats',
          expect.objectContaining({ method: 'GET' })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('retryJob', () => {
      it('should send POST request to /v1/queue/:id/retry', async () => {
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({}),
        })

        await client.retryJob('job-123')

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/queue/job-123/retry',
          expect.objectContaining({ method: 'POST' })
        )
      })
    })

    describe('discardJob', () => {
      it('should send POST request to /v1/queue/:id/discard', async () => {
        ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({}),
        })

        await client.discardJob('job-123')

        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:1133/v1/queue/job-123/discard',
          expect.objectContaining({ method: 'POST' })
        )
      })
    })
  })

  describe('createClient', () => {
    it('should create client with host and token from localStorage', () => {
      const { getItem } = localStorage
      ;(getItem as ReturnType<typeof vi.fn>)
        .mockReturnValueOnce('http://localhost:1133')
        .mockReturnValueOnce('test-token')

      const client = createClient()

      expect(client).toBeInstanceOf(KhayalClient)
    })

    it('should use window.location.origin as fallback host', () => {
      const { getItem } = localStorage
      ;(getItem as ReturnType<typeof vi.fn>).mockReturnValue(null)

      const client = createClient()

      expect(client).toBeInstanceOf(KhayalClient)
    })
  })
})
