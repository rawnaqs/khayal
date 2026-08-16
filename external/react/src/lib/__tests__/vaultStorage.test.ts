import { describe, it, expect } from 'vitest'
import {
  getVaultRecord,
  saveVaultRecord,
  deleteVaultRecord,
  getAllOfflineItems,
  putOfflineItem,
  deleteOfflineItem,
} from '../vaultStorage'

describe('vaultStorage', () => {
  it('vault record CRUD', async () => {
    expect(await getVaultRecord()).toBeNull()

    await saveVaultRecord({
      id: 'vault',
      mode: 'prf',
      salt: 'abc',
      encryptedToken: 'xyz',
    })
    const record = await getVaultRecord()
    expect(record).toEqual({
      id: 'vault',
      mode: 'prf',
      salt: 'abc',
      encryptedToken: 'xyz',
    })

    await deleteVaultRecord()
    expect(await getVaultRecord()).toBeNull()
  })

  it('offline item CRUD', async () => {
    await putOfflineItem({
      id: 'o1',
      request: { type: 'text', content: 'hi' },
      token: 'tok',
      timestamp: 1,
    })
    await putOfflineItem({ id: 'o2', cipher: 'abc', timestamp: 2 })

    const items = await getAllOfflineItems()
    expect(items).toHaveLength(2)

    await deleteOfflineItem('o1')
    const after = await getAllOfflineItems()
    expect(after).toHaveLength(1)
    expect(after[0].id).toBe('o2')
  })
})
