import { describe, it, expect, vi } from 'vitest'
import {
  saveOffline,
  getOfflineQueue,
  removeOfflineItem,
  flushOfflineQueue,
  getOfflineCount,
} from '../offline'
import type { VaultSession } from '../offline'
import type { CaptureRequest } from '../api'

const mockCapture: CaptureRequest = {
  type: 'text',
  content: 'test capture',
}

async function makeAesKey(): Promise<CryptoKey> {
  return crypto.subtle.generateKey(
    { name: 'AES-GCM', length: 256 },
    true,
    ['encrypt', 'decrypt'],
  )
}

describe('offline.ts', () => {
  it('saves plaintext when no session', async () => {
    const id = await saveOffline(mockCapture)
    expect(id).toMatch(/^offline-\d+-[a-z0-9]+$/)

    const queue = await getOfflineQueue()
    expect(queue).toHaveLength(1)
    expect(queue[0].request).toEqual(mockCapture)
  })

  it('encrypts when a locked session is provided', async () => {
    const key = await makeAesKey()
    const session: VaultSession = { mode: 'prf', key, token: 'tok' }

    await saveOffline(mockCapture, session)

    // Without a session, ciphered items are invisible (still encrypted at rest)
    expect(await getOfflineQueue()).toHaveLength(0)

    const queue = await getOfflineQueue(session)
    expect(queue).toHaveLength(1)
    expect(queue[0].request).toEqual(mockCapture)
    expect(queue[0].token).toBe('tok')
  })

  it('flushes items in order and removes them', async () => {
    await saveOffline(mockCapture)
    const mockClient = {
      capture: vi.fn().mockResolvedValue({}),
    }

    await flushOfflineQueue(mockClient as any)

    expect(mockClient.capture).toHaveBeenCalledWith(mockCapture)
    expect(await getOfflineQueue()).toHaveLength(0)
  })

  it('removeOfflineItem and getOfflineCount', async () => {
    const id = await saveOffline(mockCapture)
    expect(await getOfflineCount()).toBe(1)

    await removeOfflineItem(id)
    expect(await getOfflineCount()).toBe(0)
  })
})
