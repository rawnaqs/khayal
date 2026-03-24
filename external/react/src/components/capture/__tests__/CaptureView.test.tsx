import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CaptureView } from '../CaptureView'

// Mock framer-motion
vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}))

// Mock hooks
vi.mock('@/hooks/useCapture', () => ({
  useCapture: () => ({
    loading: false,
    result: null,
    error: null,
    errorCode: undefined,
    isOffline: false,
    processingTime: undefined,
    capture: vi.fn(),
    uploadImage: vi.fn(),
    clear: vi.fn(),
  }),
}))

vi.mock('@/hooks/useStats', () => ({
  useStats: () => ({
    stats: null,
    loading: false,
    refresh: vi.fn(),
  }),
}))

describe('CaptureView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should render with text mode by default', () => {
    render(<CaptureView />)

    // Should show the compose area with mode pills
    expect(screen.getByText('txt')).toBeInTheDocument()
    expect(screen.getByText('url')).toBeInTheDocument()
    expect(screen.getByText('img')).toBeInTheDocument()
  })

  it('should show greeting', () => {
    render(<CaptureView />)

    // Should show some greeting text
    const greeting = document.querySelector('.cap-greeting')
    expect(greeting).toBeInTheDocument()
  })

  it('should show hint for text mode', () => {
    render(<CaptureView />)

    expect(screen.getByText('cmd+enter to capture')).toBeInTheDocument()
  })

  it('should render with compose area', () => {
    render(<CaptureView />)

    expect(document.querySelector('.compose')).toBeInTheDocument()
  })

  it('should have send button', () => {
    render(<CaptureView />)

    expect(document.querySelector('.send')).toBeInTheDocument()
  })

  it('should handle captureQuery prop', () => {
    const onCaptureQueryConsumed = vi.fn()
    render(<CaptureView captureQuery="test query" onCaptureQueryConsumed={onCaptureQueryConsumed} />)

    // Should call onCaptureQueryConsumed
    expect(onCaptureQueryConsumed).toHaveBeenCalled()
  })

  it('should show stats component', () => {
    render(<CaptureView />)

    // Stats should be rendered (even with null stats)
    expect(document.querySelector('.bento')).toBeInTheDocument()
  })

  it('should show type pills', () => {
    render(<CaptureView />)

    const pills = document.querySelector('.pills')
    expect(pills).toBeInTheDocument()
    expect(screen.getByText('txt')).toBeInTheDocument()
    expect(screen.getByText('url')).toBeInTheDocument()
    expect(screen.getByText('img')).toBeInTheDocument()
  })
})
