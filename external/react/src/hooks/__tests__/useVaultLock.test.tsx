import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { VaultLockProvider, useVaultLock } from '../useVaultLock'
import { STORAGE_KEYS } from '@/lib/constants'

function Probe() {
  const { lockMode, locked, configured, token } = useVaultLock()
  return (
    <div data-testid="probe">
      {lockMode}|{String(locked)}|{String(configured)}|{token ?? ''}
    </div>
  )
}

function OnboardingProbe({ persist }: { persist: boolean }) {
  const { completeOnboarding, configured, token } = useVaultLock()
  return (
    <div>
      <span data-testid="configured">{String(configured)}</span>
      <span data-testid="token">{token ?? ''}</span>
      <button onClick={() => completeOnboarding('new-token', persist)}>
        complete
      </button>
    </div>
  )
}

describe('useVaultLock', () => {
  it('initializes to none + configured when token is in localStorage', async () => {
    const { getItem } = localStorage
    ;(getItem as ReturnType<typeof vi.fn>).mockImplementation((key: string) => {
      if (key === STORAGE_KEYS.TOKEN) return 'tok'
      if (key === STORAGE_KEYS.HOST) return 'http://localhost'
      return null
    })

    render(
      <VaultLockProvider>
        <Probe />
      </VaultLockProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('probe').textContent).toBe('none|false|true|tok')
    })
  })

  it('completeOnboarding with persist stores the token', async () => {
    const { setItem } = localStorage
    const user = userEvent.setup()
    render(
      <VaultLockProvider>
        <OnboardingProbe persist={true} />
      </VaultLockProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('configured').textContent).toBe('false')
    })

    await user.click(screen.getByRole('button', { name: 'complete' }))

    expect(setItem).toHaveBeenCalledWith(STORAGE_KEYS.TOKEN, 'new-token')
    expect(setItem).toHaveBeenCalledWith(STORAGE_KEYS.LOCK_SETUP_DECIDED, '1')
    expect(screen.getByTestId('configured').textContent).toBe('true')
    expect(screen.getByTestId('token').textContent).toBe('new-token')
  })

  it('completeOnboarding without persist does not store the token', async () => {
    const { setItem, removeItem } = localStorage
    const user = userEvent.setup()
    render(
      <VaultLockProvider>
        <OnboardingProbe persist={false} />
      </VaultLockProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('configured').textContent).toBe('false')
    })

    await user.click(screen.getByRole('button', { name: 'complete' }))

    expect(setItem).not.toHaveBeenCalledWith(STORAGE_KEYS.TOKEN, 'new-token')
    expect(removeItem).toHaveBeenCalledWith(STORAGE_KEYS.TOKEN)
    expect(setItem).toHaveBeenCalledWith(STORAGE_KEYS.LOCK_SETUP_DECIDED, '1')
    expect(screen.getByTestId('token').textContent).toBe('new-token')
  })
})
