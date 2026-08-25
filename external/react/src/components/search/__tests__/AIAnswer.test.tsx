import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AIAnswerRow } from '../AIAnswer'

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}))

const base = {
  onAsk: vi.fn(),
  onToggle: vi.fn(),
  onRetry: vi.fn(),
  onClose: vi.fn(),
  onCitationClick: vi.fn(),
}

describe('AIAnswerRow', () => {
  it('idle + collapsed: header click asks — never automatic', () => {
    const onAsk = vi.fn()
    render(<AIAnswerRow {...base} state="idle" expanded={false} overview={null} onAsk={onAsk} />)
    fireEvent.click(screen.getByTestId('ai-answer-trigger'))
    expect(onAsk).toHaveBeenCalledTimes(1)
    expect(screen.queryByTestId('ai-answer-skeleton')).toBeNull()
  })

  it('loading + expanded: skeleton shimmer lines show', () => {
    render(<AIAnswerRow {...base} state="loading" expanded={true} overview={null} />)
    expect(screen.getByTestId('ai-answer-skeleton')).toBeTruthy()
    expect(document.querySelectorAll('.ai-skel').length).toBeGreaterThanOrEqual(3)
  })

  it('collapsed while ready: no body, toggle re-expands without refetch', () => {
    const onToggle = vi.fn()
    render(
      <AIAnswerRow
        {...base}
        state="ready"
        expanded={false}
        overview={{ text: 'answer [1]', citations: [0] }}
        onToggle={onToggle}
      />,
    )
    expect(screen.queryByTestId('ai-answer')).toBeNull()
    fireEvent.click(screen.getByTestId('ai-answer-trigger'))
    // ready state toggles, does not ask again
    expect(base.onAsk).not.toHaveBeenCalled()
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it('ready + expanded: citations clickable with zero-based index', () => {
    const onCitationClick = vi.fn()
    render(
      <AIAnswerRow
        {...base}
        state="ready"
        expanded={true}
        overview={{ text: 'one [1] two [2]', citations: [0, 1] }}
        onCitationClick={onCitationClick}
      />,
    )
    expect(screen.getByTestId('ai-answer')).toBeTruthy()
    fireEvent.click(screen.getByText('[2]'))
    expect(onCitationClick).toHaveBeenCalledWith(1)
  })

  it('error + expanded: retry offered, results untouched', () => {
    const onRetry = vi.fn()
    render(<AIAnswerRow {...base} state="error" expanded={true} overview={null} onRetry={onRetry} />)
    expect(screen.getByTestId('ai-answer-error')).toBeTruthy()
    fireEvent.click(screen.getByTitle('try again'))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('dismiss resets fully', () => {
    const onClose = vi.fn()
    render(
      <AIAnswerRow
        {...base}
        state="ready"
        expanded={true}
        overview={{ text: 'x', citations: [] }}
        onClose={onClose}
      />,
    )
    fireEvent.click(screen.getByTitle('dismiss'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
