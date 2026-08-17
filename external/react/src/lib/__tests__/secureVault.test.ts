import { describe, it, expect } from 'vitest'
import {
  toBase64,
  fromBase64,
  toBase64Url,
  fromBase64Url,
  encryptWithKey,
  decryptWithKey,
} from '../secureVault'

async function makeAesKey(): Promise<CryptoKey> {
  return crypto.subtle.generateKey(
    { name: 'AES-GCM', length: 256 },
    true,
    ['encrypt', 'decrypt'],
  )
}

describe('secureVault', () => {
  it('base64 round trip', () => {
    const bytes = new Uint8Array([0, 1, 2, 250, 255])
    expect(fromBase64(toBase64(bytes))).toEqual(bytes)
  })

  it('base64url round trip', () => {
    const bytes = new Uint8Array([1, 2, 3, 250, 255])
    const encoded = toBase64Url(bytes)
    expect(encoded).not.toContain('+')
    expect(encoded).not.toContain('/')
    expect(fromBase64Url(encoded)).toEqual(bytes)
  })

  it('encrypt/decrypt round trip', async () => {
    const key = await makeAesKey()
    const ciphertext = await encryptWithKey(key, 'hello world')
    expect(ciphertext).not.toContain('hello world')
    expect(await decryptWithKey(key, ciphertext)).toBe('hello world')
  })

  it('decrypt fails with wrong key', async () => {
    const key1 = await makeAesKey()
    const key2 = await makeAesKey()
    const ciphertext = await encryptWithKey(key1, 'secret')
    await expect(decryptWithKey(key2, ciphertext)).rejects.toThrow()
  })
})
