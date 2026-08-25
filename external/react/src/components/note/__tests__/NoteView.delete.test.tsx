import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { NoteView } from '../NoteView'

vi.mock('@/hooks/useNote', () => ({
  useNote: () => ({
    note: {
      note_path: 'khayal/victim.md',
      title: 'Victim',
      type: 'text',
      created_at: '2026-08-25T10:00:00Z',
    },
    loading: false,
    error: null,
  }),
}))

vi.mock('@/hooks/useVaultLock', () => ({
  useVaultLock: () => ({ token: 'test-token' }),
}))

const toastMock = vi.fn()
vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}))

const deleteMock = vi.fn()
vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    createClient: () => ({
      deleteNote: deleteMock,
    }),
  }
})

// Sheet renders children directly for testing
vi.mock('@/components/ui/sheet', () => ({
  Sheet: ({ children }: any) => <>{children}</>,
  SheetContent: ({ children }: any) => <div>{children}</div>,
}))

describe('NoteView delete affordance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows trash trigger only after note loads', () => {
    render(<NoteView notePath="khayal/victim.md" onClose={() => {}} />)
    expect(screen.getByTestId('note-delete-trigger')).toBeTruthy()
  })

  it('two-step confirm: trash click asks, cancel backs out without API call', () => {
    render(<NoteView notePath="khayal/victim.md" onClose={() => {}} />)
    fireEvent.click(screen.getByTestId('note-delete-trigger'))
    expect(screen.getByTestId('note-delete-confirm')).toBeTruthy()
    expect(deleteMock).not.toHaveBeenCalled()
    fireEvent.click(screen.getByTestId('note-delete-cancel'))
    expect(screen.queryByTestId('note-delete-confirm')).toBeNull()
  })

  it('confirm deletes, toasts, fires onDeleted and closes', async () => {
    const onClose = vi.fn()
    const onDeleted = vi.fn()
    deleteMock.mockResolvedValueOnce({ deleted: true, trash_path: '.khayal-trash/victim.md.123' })

    render(<NoteView notePath="khayal/victim.md" onClose={onClose} onDeleted={onDeleted} />)
    fireEvent.click(screen.getByTestId('note-delete-trigger'))
    fireEvent.click(screen.getByTestId('note-delete-go'))

    await waitFor(() => {
      expect(deleteMock).toHaveBeenCalledWith('khayal/victim.md')
      expect(onDeleted).toHaveBeenCalledWith('khayal/victim.md')
      expect(onClose).toHaveBeenCalled()
      expect(toastMock).toHaveBeenCalled()
    })
  })

  it('failed delete keeps the sheet open with destructive toast', async () => {
    const onClose = vi.fn()
    deleteMock.mockRejectedValueOnce(new Error('boom'))

    render(<NoteView notePath="khayal/victim.md" onClose={onClose} />)
    fireEvent.click(screen.getByTestId('note-delete-trigger'))
    fireEvent.click(screen.getByTestId('note-delete-go'))

    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(expect.objectContaining({ title: 'Delete failed' }))
    })
    expect(onClose).not.toHaveBeenCalled()
  })
})
