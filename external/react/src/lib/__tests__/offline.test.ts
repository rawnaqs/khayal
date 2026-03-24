import { describe, it, expect, vi, beforeEach } from 'vitest'
import { saveOffline, getOfflineQueue, removeOfflineItem, flushOfflineQueue, getOfflineCount } from '../offline'
import { get, set, del, keys } from 'idb-keyval'
import type { CaptureRequest } from '../api'

const mockCapture: CaptureRequest = {
  type: 'text',
  content: 'test capture',
}

describe('offline.ts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([])
    ;(get as ReturnType<typeof vi.fn>).mockResolvedValue(undefined)
    ;(set as ReturnType<typeof vi.fn>).mockResolvedValue(undefined)
    ;(del as ReturnType<typeof vi.fn>).mockResolvedValue(undefined)
  })

  describe('saveOffline', () => {
    it('should save a capture to offline queue', async () => {
      const id = await saveOffline(mockCapture)

      expect(id).toMatch(/^offline-\d+-[a-z0-9]+$/)
      expect(set).toHaveBeenCalledWith(
        `khayal-offline-${id}`,
        expect.objectContaining({
          id,
          request: mockCapture,
          timestamp: expect.any(Number),
        })
      )
    })
  })

  describe('getOfflineQueue', () => {
    it('should return empty array when no items', async () => {
      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([])

      const queue = await getOfflineQueue()

      expect(queue).toEqual([])
    })

    it('should return items sorted by timestamp', async () => {
      const mockItems = [
        { id: 'offline-2', request: mockCapture, timestamp: 2000 },
        { id: 'offline-1', request: mockCapture, timestamp: 1000 },
      ]

      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([
        'khayal-offline-offline-2',
        'khayal-offline-offline-1',
      ])
      ;(get as ReturnType<typeof vi.fn>)
        .mockResolvedValueOnce(mockItems[0])
        .mockResolvedValueOnce(mockItems[1])

      const queue = await getOfflineQueue()

      expect(queue).toHaveLength(2)
      expect(queue[0].id).toBe('offline-1')
      expect(queue[1].id).toBe('offline-2')
    })

    it('should filter out non-offline keys', async () => {
      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([
        'khayal-offline-offline-1',
        'other-key',
        'khayal-offline-offline-2',
      ])
      ;(get as ReturnType<typeof vi.fn>).mockResolvedValue({
        id: 'offline-1',
        request: mockCapture,
        timestamp: 1000,
      })

      const queue = await getOfflineQueue()

      expect(queue).toHaveLength(2)
    })
  })

  describe('removeOfflineItem', () => {
    it('should delete item from offline queue', async () => {
      await removeOfflineItem('offline-1')

      expect(del).toHaveBeenCalledWith('khayal-offline-offline-1')
    })
  })

  describe('flushOfflineQueue', () => {
    it('should capture each item and remove on success', async () => {
      const mockClient = {
        capture: vi.fn().mockResolvedValue({}),
      }

      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue(['khayal-offline-offline-1'])
      ;(get as ReturnType<typeof vi.fn>).mockResolvedValue({
        id: 'offline-1',
        request: mockCapture,
        timestamp: 1000,
      })

      await flushOfflineQueue(mockClient as any)

      expect(mockClient.capture).toHaveBeenCalledWith(mockCapture)
      expect(del).toHaveBeenCalledWith('khayal-offline-offline-1')
    })

    it('should stop flush on first failure', async () => {
      const mockClient = {
        capture: vi.fn().mockRejectedValueOnce(new Error('Network error')),
      }

      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([
        'khayal-offline-offline-1',
        'khayal-offline-offline-2',
      ])
      ;(get as ReturnType<typeof vi.fn>)
        .mockResolvedValueOnce({ id: 'offline-1', request: mockCapture, timestamp: 1000 })
        .mockResolvedValueOnce({ id: 'offline-2', request: mockCapture, timestamp: 2000 })

      await flushOfflineQueue(mockClient as any)

      expect(mockClient.capture).toHaveBeenCalledTimes(1)
      expect(del).not.toHaveBeenCalled()
    })
  })

  describe('getOfflineCount', () => {
    it('should return count of offline items', async () => {
      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([
        'khayal-offline-offline-1',
        'khayal-offline-offline-2',
      ])
      ;(get as ReturnType<typeof vi.fn>).mockResolvedValue({
        id: 'offline-1',
        request: mockCapture,
        timestamp: 1000,
      })

      const count = await getOfflineCount()

      expect(count).toBe(2)
    })

    it('should return 0 when no items', async () => {
      ;(keys as ReturnType<typeof vi.fn>).mockResolvedValue([])

      const count = await getOfflineCount()

      expect(count).toBe(0)
    })
  })
})
