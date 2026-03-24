import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SearchView } from '../SearchView'

// Mock framer-motion
vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}))

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Search: () => <div data-testid="search-icon" />,
  X: () => <div data-testid="x-icon" />,
  AlertCircle: () => <div data-testid="alert-icon" />,
}))

// Mock hooks
vi.mock('@/hooks/useSearch', () => ({
  useSearch: () => ({
    loading: false,
    results: null,
    error: null,
    search: vi.fn(),
    clear: vi.fn(),
  }),
}))

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({
    toast: vi.fn(),
  }),
}))

// Mock child components
vi.mock('../ResultHero', () => ({
  ResultHero: ({ result, query }: any) => (
    <div data-testid="result-hero">
      <span>{result.title}</span>
      <span>Query: {query}</span>
    </div>
  ),
}))

vi.mock('../ResultCompact', () => ({
  ResultCompact: ({ result, rank, query }: any) => (
    <div data-testid="result-compact">
      <span>#{rank}</span>
      <span>{result.title}</span>
    </div>
  ),
}))

describe('SearchView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem.mockReturnValue(null)
  })

  it('should render with search bar and suggestions', () => {
    render(<SearchView />)

    expect(screen.getByPlaceholderText('Search your vault...')).toBeInTheDocument()
    expect(screen.getByText('hybrid')).toBeInTheDocument()
    expect(screen.getByText('keyword')).toBeInTheDocument()
    expect(screen.getByText('semantic')).toBeInTheDocument()
    expect(screen.getByText('try searching for')).toBeInTheDocument()
  })

  it('should show recent searches if available', () => {
    localStorage.getItem.mockReturnValue(JSON.stringify(['react', 'go', 'python']))

    render(<SearchView />)

    expect(screen.getByText('recent searches')).toBeInTheDocument()
    expect(screen.getByText('react')).toBeInTheDocument()
    expect(screen.getByText('go')).toBeInTheDocument()
    expect(screen.getByText('python')).toBeInTheDocument()
  })

  it('should not show recent searches if none', () => {
    render(<SearchView />)

    expect(screen.queryByText('recent searches')).not.toBeInTheDocument()
  })

  it('should show mode chips', () => {
    render(<SearchView />)

    expect(screen.getByText('hybrid')).toBeInTheDocument()
    expect(screen.getByText('keyword')).toBeInTheDocument()
    expect(screen.getByText('semantic')).toBeInTheDocument()
  })

  it('should show suggestion chips', () => {
    render(<SearchView />)

    expect(screen.getByText('people')).toBeInTheDocument()
    expect(screen.getByText('payments')).toBeInTheDocument()
    expect(screen.getByText('this week')).toBeInTheDocument()
  })

  it('should render search input', () => {
    render(<SearchView />)

    const input = screen.getByPlaceholderText('Search your vault...')
    expect(input).toBeInTheDocument()
  })

  it('should handle onCaptureQuery prop', () => {
    const onCaptureQuery = vi.fn()
    render(<SearchView onCaptureQuery={onCaptureQuery} />)

    // Component should render without errors
    expect(screen.getByPlaceholderText('Search your vault...')).toBeInTheDocument()
  })

  it('should update query on input change', () => {
    render(<SearchView />)

    const input = screen.getByPlaceholderText('Search your vault...')
    fireEvent.change(input, { target: { value: 'test query' } })

    expect(input).toHaveValue('test query')
  })
})
