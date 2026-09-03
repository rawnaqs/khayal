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
  SheetTitle: ({ children }: any) => <div>{children}</div>,
  SheetDescription: ({ children }: any) => <div>{children}</div>,
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

describe('NoteView linked-notes chips', () => {
  it('renders related links as clickable chips and switches note on click', async () => {
    vi.resetModules()
    const onOpenNote = vi.fn()
    vi.doMock('@/hooks/useNote', () => ({
      useNote: () => ({
        note: {
          note_path: 'khayal/hates.md',
          title: 'Hates',
          type: 'text',
          related_links: [
            { note_path: 'khayal/2026-08-26-bob-loves-note-abc123.md', title: 'Bob loves the note' },
          ],
        },
        loading: false,
        error: null,
      }),
    }))
    const { NoteView: NV } = await import('../NoteView')
    const { render: r, screen: s2, fireEvent: fe } = await import('@testing-library/react')
    r(<NV notePath="khayal/hates.md" onClose={() => {}} onOpenNote={onOpenNote} />)
    const chip = s2.getAllByTestId('note-link-chip')[0]
    expect(chip.textContent).toContain('Bob loves the note')
    fe.click(chip)
    expect(onOpenNote).toHaveBeenCalledWith('khayal/2026-08-26-bob-loves-note-abc123.md')
  })
})
