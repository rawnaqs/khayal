import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueueView } from '../QueueView'

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}))

const queueState = {
  loading: false,
  jobs: [
    { id: 't1', type: 'text', status: 'done', note_path: 'khayal/a.md', created_at: new Date().toISOString() },
    { id: 'c1', type: 'connections', status: 'done', created_at: new Date().toISOString() },
    { id: 'm1', type: 'memory', status: 'done', created_at: new Date().toISOString() },
    { id: 'p1', type: 'text', status: 'pending', created_at: new Date().toISOString() },
  ],
  total: 4,
  error: null,
  flares: {},
  doneExpanded: false,
  doneLoadingMore: false,
  loadMoreDone: vi.fn(),
  setDoneExpanded: vi.fn(),
  fetchQueue: vi.fn(),
  retryJob: vi.fn(),
  discardJob: vi.fn(),
}

vi.mock('@/hooks/useQueue', () => ({
  useQueue: () => queueState,
}))

vi.mock('@/hooks/useVaultLock', () => ({
  useVaultLock: () => ({ session: null }),
}))

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}))

vi.mock('@/lib/offline', () => ({
  getOfflineQueue: async () => [],
}))

describe('QueueView internal job filtering', () => {
  it('never renders connections/memory entries; counts exclude them', () => {
    render(<QueueView />)
    expect(screen.queryByText(/pending \(1\)/)).toBeTruthy()
    expect(screen.queryByText(/done \(1\)/)).toBeTruthy()
    // internal entries absent even though their jobs exist
    expect(screen.queryByText(/connections/)).toBeNull()
    expect(screen.queryByText(/memory/)).toBeNull()
    expect(screen.getByText('khayal/a.md')).toBeTruthy()
  })
})
