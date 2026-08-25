import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AIAnswer, AIAnswerCTA } from '../AIAnswer'
import { useAIAnswer } from '@/hooks/useAIAnswer'

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    button: ({ children, ...props }: any) => (
      <button {...props} onClick={props.onClick}>
        {children}
      </button>
    ),
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}))

vi.mock('@/hooks/useVaultLock', () => ({
  useVaultLock: () => ({ token: 'test-token' }),
}))

describe('AIAnswerCTA', () => {
  it('renders nothing when not visible', () => {
    const { container } = render(<AIAnswerCTA visible={false} onClick={() => {}} />)
    expect(container.querySelector('[data-testid="ai-answer-cta"]')).toBeNull()
  })

  it('fires onClick exactly once per click — never automatic', () => {
    const onClick = vi.fn()
    render(<AIAnswerCTA visible={true} onClick={onClick} />)
    fireEvent.click(screen.getByTestId('ai-answer-cta'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})

describe('AIAnswer states', () => {
  it('shows skeleton with shimmer lines while loading', () => {
    render(<AIAnswer state="loading" overview={null} onCitationClick={() => {}} onRetry={() => {}} onClose={() => {}} />)
    expect(screen.getByTestId('ai-answer-skeleton')).toBeTruthy()
    expect(document.querySelectorAll('.ai-skel').length).toBeGreaterThanOrEqual(3)
  })

  it('renders answer text with clickable citations', () => {
    render(
      <AIAnswer
        state="ready"
        overview={{ text: 'Bob owes money [1] and Alice agrees [2].', citations: [0, 1] }}
        onCitationClick={(n) => citations.push(n)}
        onRetry={() => {}}
        onClose={() => {}}
      />,
    )
    expect(screen.getByTestId('ai-answer')).toBeTruthy()
    const cites = screen.getAllByText(/^\[\d+\]$/)
    expect(cites.length).toBe(2)
  })

  it('error state offers retry without touching results', () => {
    const onRetry = vi.fn()
    render(<AIAnswer state="error" overview={null} onCitationClick={() => {}} onRetry={onRetry} onClose={() => {}} />)
    expect(screen.getByTestId('ai-answer-error')).toBeTruthy()
    fireEvent.click(screen.getByTitle('try again'))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('citation click passes zero-based index', () => {
    const clicked: number[] = []
    render(
      <AIAnswer
        state="ready"
        overview={{ text: 'one [1]', citations: [0] }}
        onCitationClick={(n) => clicked.push(n)}
        onRetry={() => {}}
        onClose={() => {}}
      />,
    )
    fireEvent.click(screen.getByText('[1]'))
    expect(clicked).toEqual([0])
  })
})

// keep import used
void useAIAnswer

const citations: number[] = []
