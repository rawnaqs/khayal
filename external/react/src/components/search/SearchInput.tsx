import { useState, useRef, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Search } from 'lucide-react'

interface SearchInputProps {
  onSearch: (query: string) => void
  loading: boolean
}

export function SearchInput({ onSearch, loading }: SearchInputProps) {
  const [query, setQuery] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      onSearch(query.trim())
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (query.trim()) {
        onSearch(query.trim())
      }
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex gap-2">
      <Input
        ref={inputRef}
        placeholder="search notes..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        disabled={loading}
        className="flex-1 glass input-glow"
      />
      <Button
        type="submit"
        disabled={loading || !query.trim()}
        className="btn-gradient h-auto px-5"
      >
        <Search className="w-5 h-5" />
      </Button>
    </form>
  )
}
