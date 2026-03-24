import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useSearch } from '../useSearch'

// Mock the api module
vi.mock('@/lib/api', () => ({
  createClient: vi.fn(() => ({
    search: vi.fn(),
  })),
}))

describe('useSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should start with default state', () => {
    const { result } = renderHook(() => useSearch())

    expect(result.current.loading).toBe(false)
    expect(result.current.results).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('should search successfully', async () => {
    const mockSearch = vi.fn().mockResolvedValue({
      query: 'test',
      mode: 'hybrid',
      results: [
        {
          id: '1',
          note_path: 'khayal/2024-01-01/test.md',
          title: 'Test Note',
          excerpt: 'Test content',
          score: 0.95,
          type: 'text',
          created_at: '2024-01-01T10:00:00Z',
          tags: ['test'],
        },
      ],
      total: 1,
      took_ms: 100,
    })

    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    await act(async () => {
      await result.current.search('test query')
    })

    expect(mockSearch).toHaveBeenCalledWith('test query', expect.objectContaining({ mode: 'hybrid' }))
    expect(result.current.results).toEqual(
      expect.objectContaining({
        query: 'test',
        results: expect.arrayContaining([expect.objectContaining({ title: 'Test Note' })]),
      })
    )
  })

  it('should not search with empty query', async () => {
    const mockSearch = vi.fn()
    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    await act(async () => {
      await result.current.search('')
    })

    expect(mockSearch).not.toHaveBeenCalled()
    expect(result.current.results).toBeNull()
  })

  it('should not search with whitespace-only query', async () => {
    const mockSearch = vi.fn()
    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    await act(async () => {
      await result.current.search('   ')
    })

    expect(mockSearch).not.toHaveBeenCalled()
  })

  it('should set error on search failure', async () => {
    const mockSearch = vi.fn().mockRejectedValue(new Error('Network error'))
    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    await act(async () => {
      await result.current.search('test')
    })

    expect(result.current.error).toBe('Network error')
  })

  it('should set loading state during search', async () => {
    let resolveSearch: (value: any) => void
    const mockSearch = vi.fn().mockImplementation(
      () => new Promise((resolve) => { resolveSearch = resolve })
    )
    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    act(() => {
      result.current.search('test')
    })

    expect(result.current.loading).toBe(true)

    await act(async () => {
      resolveSearch!({ query: 'test', mode: 'hybrid', results: [], total: 0, took_ms: 0 })
    })

    expect(result.current.loading).toBe(false)
  })

  it('should clear results on clear()', async () => {
    const mockSearch = vi.fn().mockResolvedValue({
      query: 'test',
      mode: 'hybrid',
      results: [{ id: '1', title: 'Test' }],
      total: 1,
      took_ms: 100,
    })
    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    await act(async () => {
      await result.current.search('test')
    })

    expect(result.current.results).not.toBeNull()

    act(() => {
      result.current.clear()
    })

    expect(result.current.results).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('should use custom search options', async () => {
    const mockSearch = vi.fn().mockResolvedValue({
      query: 'test',
      mode: 'semantic',
      results: [],
      total: 0,
      took_ms: 50,
    })
    const { createClient } = await import('@/lib/api')
    ;(createClient as ReturnType<typeof vi.fn>).mockReturnValue({ search: mockSearch })

    const { result } = renderHook(() => useSearch())

    await act(async () => {
      await result.current.search('test', { mode: 'semantic', limit: 5 })
    })

    expect(mockSearch).toHaveBeenCalledWith(
      'test',
      expect.objectContaining({
        mode: 'semantic',
        limit: 5,
      })
    )
  })
})
