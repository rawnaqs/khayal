import '@testing-library/jest-dom'
import 'fake-indexeddb/auto'
import { IDBFactory } from 'fake-indexeddb'

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
}

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// Mock navigator.onLine
Object.defineProperty(navigator, 'onLine', {
  writable: true,
  value: true,
})

// Mock fetch
global.fetch = vi.fn()

// Mock performance.now
if (!window.performance) {
  Object.defineProperty(window, 'performance', {
    value: {
      now: () => Date.now(),
    },
  })
}

// Mock idb-keyval (legacy store, still used by the one-time migration)
vi.mock('idb-keyval', () => {
  const store = new Map()
  return {
    get: vi.fn((key) => Promise.resolve(store.get(String(key)))),
    set: vi.fn((key, value) => {
      store.set(String(key), value)
      return Promise.resolve()
    }),
    del: vi.fn((key) => {
      store.delete(String(key))
      return Promise.resolve()
    }),
    keys: vi.fn(() => Promise.resolve([...store.keys()])),
    clear: vi.fn(() => {
      store.clear()
      return Promise.resolve()
    }),
  }
})

// Reset mocks and IndexedDB before each test
beforeEach(() => {
  vi.clearAllMocks()
  localStorageMock.getItem.mockReturnValue(null)
  globalThis.indexedDB = new IDBFactory()
})
