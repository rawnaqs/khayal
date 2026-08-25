import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DoneItem } from '../DoneItem'
import { Header } from '@/components/layout/Header'

const baseJob = {
  id: 'j1',
  type: 'text',
  status: 'done',
  note_path: 'khayal/note-a.md',
  created_at: new Date().toISOString(),
}

describe('DoneItem flare + open', () => {
  it('shows connection-count chip when flare present', () => {
    render(<DoneItem job={baseJob} flare={{ connections: 3 }} />)
    expect(screen.getByTitle('3 connected notes')).toBeTruthy()
    expect(screen.getByText('3')).toBeTruthy()
  })

  it('hides chip for zero connections', () => {
    render(<DoneItem job={baseJob} flare={{ connections: 0 }} />)
    expect(screen.queryByTitle('0 connected notes')).toBeNull()
  })

  it('clicking opens the note via onSelect', () => {
    const onSelect = vi.fn()
    const container = render(<DoneItem job={baseJob} onSelect={onSelect} />)
    fireEvent.click(container.getByTestId('done-item'))
    expect(onSelect).toHaveBeenCalledWith('khayal/note-a.md')
  })

  it('not clickable without note_path or handler', () => {
    const onSelect = vi.fn()
    const container = render(<DoneItem job={{ ...baseJob, note_path: undefined }} onSelect={onSelect} />)
    fireEvent.click(container.getByTestId('done-item'))
    expect(onSelect).not.toHaveBeenCalled()
  })

  it('enriched marker appears when result is set', () => {
    render(<DoneItem job={{ ...baseJob, result: { summary: 'x' } }} />)
    expect(screen.getByTitle('enriched with memory context')).toBeTruthy()
  })
})

vi.mock('@/hooks/useServerStatus', () => ({
  useServerStatus: () => ({ status: 'ok', health: null }),
}))

vi.mock('@/hooks/useVaultLock', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/hooks/useVaultLock')>()
  return {
    ...actual,
    useVaultLock: () => ({
      lockMode: 'prf',
      lock: lockMock,
      locked: false,
      configured: true,
      session: null,
      token: 't',
    }),
  }
})

vi.mock('@/components/settings/SecuritySheet', () => ({
  SecuritySheet: () => null,
}))

import { lockMock } from './Header.logout.test-helpers'

describe('Header logout two-step', () => {
  it('confirm flow locks the session', () => {
    render(<Header />)
    expect(screen.queryByTestId('logout-confirm')).toBeNull()
    fireEvent.click(screen.getByTestId('logout-trigger'))
    expect(screen.getByTestId('logout-confirm')).toBeTruthy()
    expect(lockMock).not.toHaveBeenCalled()
    fireEvent.click(screen.getByTestId('logout-go'))
    expect(lockMock).toHaveBeenCalledTimes(1)
  })

  it('cancel backs out without locking', () => {
    render(<Header />)
    fireEvent.click(screen.getByTestId('logout-trigger'))
    fireEvent.click(screen.getByTestId('logout-cancel'))
    expect(screen.queryByTestId('logout-confirm')).toBeNull()
    expect(lockMock).not.toHaveBeenCalled()
  })
})
