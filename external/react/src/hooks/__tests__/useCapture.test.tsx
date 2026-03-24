import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useCapture } from '../useCapture'

// Mock the api module
vi.mock('@/lib/api', () => ({
  createClient: vi.fn(() => ({
    capture: vi.fn(),
    uploadImage: vi.fn(),
  })),
}))

// Mock the offline module
vi.mock('@/lib/offline', () => ({
  saveOffline: vi.fn().mockResolvedValue('offline-123'),
}))

describe('useCapture', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(global.fetch as ReturnType<typeof vi.fn>).mockReset()
    Object.defineProperty(navigator, 'onLine', { value: true, writable: true })
  })

  it('should start with default state', () => {
    const { result } = renderHook(() => useCapture())

    expect(result.current.loading).toBe(false)
    expect(result.current.result).toBeNull()
    expect(result.current.error).toBeNull()
    expect(result.current.isOffline).toBe(false)
  })

  it('should capture text successfully when online', async () => {
    const { createClient } = await import('@/lib/api')
    const mockCapture = vi.fn().mockResolvedValue({
      id: '123',
      status: 'done',
      type: 'text',
      note_path: 'khayal/2024-01-01/1234.md',
    })
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ capture: mockCapture })

    const { result } = renderHook(() => useCapture())

    await act(async () => {
      await result.current.capture('text', 'test content')
    })

    expect(mockCapture).toHaveBeenCalledWith({ type: 'text', content: 'test content' })
    expect(result.current.result).toEqual({
      id: '123',
      status: 'done',
      type: 'text',
      note_path: 'khayal/2024-01-01/1234.md',
    })
    expect(result.current.loading).toBe(false)
  })

  it('should save offline when offline', async () => {
    Object.defineProperty(navigator, 'onLine', { value: false, writable: true })
    const { saveOffline } = await import('@/lib/offline')

    const { result } = renderHook(() => useCapture())

    await act(async () => {
      await result.current.capture('text', 'test content')
    })

    expect(saveOffline).toHaveBeenCalledWith({ type: 'text', content: 'test content' })
    expect(result.current.isOffline).toBe(true)
  })

  it('should save offline on network error', async () => {
    const { createClient } = await import('@/lib/api')
    const mockCapture = vi.fn().mockRejectedValue(new Error('fetch failed'))
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ capture: mockCapture })
    const { saveOffline } = await import('@/lib/offline')

    const { result } = renderHook(() => useCapture())

    await act(async () => {
      await result.current.capture('text', 'test content')
    })

    expect(saveOffline).toHaveBeenCalledWith({ type: 'text', content: 'test content' })
    expect(result.current.isOffline).toBe(true)
  })

  it('should set error on non-network error', async () => {
    const { createClient } = await import('@/lib/api')
    const mockCapture = vi.fn().mockRejectedValue(new Error('Server error'))
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ capture: mockCapture })

    const { result } = renderHook(() => useCapture())

    await act(async () => {
      await result.current.capture('text', 'test content')
    })

    expect(result.current.error).toBe('Server error')
    expect(result.current.isOffline).toBe(false)
  })

  it('should track processing time', async () => {
    const { createClient } = await import('@/lib/api')
    const mockCapture = vi.fn().mockResolvedValue({ id: '123', status: 'done', type: 'text' })
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ capture: mockCapture })

    const { result } = renderHook(() => useCapture())

    await act(async () => {
      await result.current.capture('text', 'test content')
    })

    expect(result.current.processingTime).toBeGreaterThanOrEqual(0)
  })

  it('should set loading state during capture', async () => {
    const { createClient } = await import('@/lib/api')
    let resolveCapture: (value: any) => void
    const mockCapture = vi.fn().mockImplementation(
      () => new Promise((resolve) => { resolveCapture = resolve })
    )
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ capture: mockCapture })

    const { result } = renderHook(() => useCapture())

    act(() => {
      result.current.capture('text', 'test content')
    })

    expect(result.current.loading).toBe(true)

    await act(async () => {
      resolveCapture!({ id: '123', status: 'done', type: 'text' })
    })

    expect(result.current.loading).toBe(false)
  })

  it('should clear state on clear()', async () => {
    const { createClient } = await import('@/lib/api')
    const mockCapture = vi.fn().mockResolvedValue({ id: '123', status: 'done', type: 'text' })
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ capture: mockCapture })

    const { result } = renderHook(() => useCapture())

    await act(async () => {
      await result.current.capture('text', 'test content')
    })

    expect(result.current.result).not.toBeNull()

    act(() => {
      result.current.clear()
    })

    expect(result.current.result).toBeNull()
    expect(result.current.error).toBeNull()
    expect(result.current.isOffline).toBe(false)
  })

  it('should reject image upload when offline', async () => {
    Object.defineProperty(navigator, 'onLine', { value: false, writable: true })

    const { result } = renderHook(() => useCapture())
    const file = new File(['test'], 'test.png', { type: 'image/png' })

    await act(async () => {
      await result.current.uploadImage(file, 'test note')
    })

    expect(result.current.error).toBe('Image upload requires connection')
  })
})
