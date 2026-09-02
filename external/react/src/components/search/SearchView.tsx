import { useCallback, useEffect, useState, useMemo, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Search, X, AlertCircle } from 'lucide-react'
import { ResultHero } from './ResultHero'
import { ResultCompact } from './ResultCompact'
import { useSearch } from '@/hooks/useSearch'
import { useAIAnswer } from '@/hooks/useAIAnswer'
import { AIAnswerRow } from './AIAnswer'
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'
import { STORAGE_KEYS, SEARCH_SUGGESTIONS, LIMITS, TYPE_FILTERS, SEARCH_MODES } from '@/lib/constants'

type SearchMode = typeof SEARCH_MODES[number]
type TypeFilter = typeof TYPE_FILTERS[number]

const RECENT_KEY = STORAGE_KEYS.RECENT_SEARCHES
const MAX_RECENT = LIMITS.RECENT_SEARCHES

const SUGGESTIONS = SEARCH_SUGGESTIONS

function getRecentSearches(): string[] {
  try {
    const stored = localStorage.getItem(RECENT_KEY)
    return stored ? JSON.parse(stored) : []
  } catch {
    return []
  }
}

function saveRecentSearch(query: string) {
  try {
    const recent = getRecentSearches()
    const filtered = recent.filter(q => q.toLowerCase() !== query.toLowerCase())
    const updated = [query, ...filtered].slice(0, MAX_RECENT)
    localStorage.setItem(RECENT_KEY, JSON.stringify(updated))
  } catch {
    // localStorage not available
  }
}

function removeRecentSearch(query: string) {
  try {
    const recent = getRecentSearches()
    const updated = recent.filter(q => q.toLowerCase() !== query.toLowerCase())
    localStorage.setItem(RECENT_KEY, JSON.stringify(updated))
  } catch {
    // localStorage not available
  }
}

interface SearchViewProps {
  onCaptureQuery?: (query: string) => void;
  onNoteSelect?: (notePath: string, query?: string) => void;
  deletedPaths?: string[];
  initialQuery?: string;
  onInitialQueryConsumed?: () => void;
}

export function SearchView({ onCaptureQuery, onNoteSelect, deletedPaths, initialQuery, onInitialQueryConsumed }: SearchViewProps = {}) {
  const [query, setQuery] = useState('')
  const [searchedQuery, setSearchedQuery] = useState("")
  const [mode, setMode] = useState<SearchMode>('hybrid')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')
  const [recentSearches, setRecentSearches] = useState<string[]>(getRecentSearches)
  const { loading, results, error, search } = useSearch()
  const ai = useAIAnswer()
  const [aiExpanded, setAiExpanded] = useState(false)
  const { toast } = useToast()

  useEffect(() => {
    if (error) {
      toast({ title: 'Search failed', description: error, variant: 'destructive' })
    }
  }, [error, toast])

  const handleSearch = useCallback((searchQuery: string, searchMode?: SearchMode) => {
    const q = searchQuery.trim()
    if (!q) return
    setQuery(q)
    setSearchedQuery(q)
    ai.reset()
    setAiExpanded(false)
    setMode(searchMode || mode)
    search(q, { mode: searchMode || mode })
    saveRecentSearch(q)
    setRecentSearches(getRecentSearches())
  }, [search, mode])

  // Entity-chip entry: fire the pending search once on mount
  const consumedInitialRef = useRef<string | undefined>(undefined)
  useEffect(() => {
    if (initialQuery && consumedInitialRef.current !== initialQuery) {
      consumedInitialRef.current = initialQuery
      handleSearch(initialQuery)
      onInitialQueryConsumed?.()
    }
  }, [initialQuery, handleSearch, onInitialQueryConsumed])

  const handleClear = useCallback(() => {
    setQuery('')
    setSearchedQuery("")
    setTypeFilter('all')
    ai.reset()
    setAiExpanded(false)
    search('')
  }, [search, ai])

  const handleModeChange = useCallback((newMode: SearchMode) => {
    setMode(newMode)
    const q = query.trim();
    if (q) {
      setSearchedQuery(q)
      setQuery(q)
      search(q, { mode: newMode })
    }
  }, [query, search])

  const handleRemoveRecent = useCallback((q: string, e: React.MouseEvent) => {
    e.stopPropagation()
    removeRecentSearch(q)
    setRecentSearches(getRecentSearches())
  }, [])

  const handleNoteSelectLocal = useCallback((notePath: string) => {
    onNoteSelect?.(notePath, searchedQuery)
  }, [onNoteSelect, searchedQuery])

  const handleAskAI = useCallback(() => {
    if (!hasResults || !searchedQuery.trim()) return
    ai.ask(searchedQuery, mode)
    setAiExpanded(true)
  }, [ai, mode, searchedQuery])

  const handleCloseAI = useCallback(() => {
    ai.reset()
    setAiExpanded(false)
  }, [ai])

  const handleCitationClick = useCallback((index: number) => {
    document.getElementById(`result-${index}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [])

  const filteredResults = useMemo(() => {
    if (!results?.results) return null
    let list = results.results
    if (deletedPaths && deletedPaths.length > 0) {
      list = list.filter(r => !deletedPaths.includes(r.note_path))
    }
    if (typeFilter === 'all') return list
    return list.filter(r => r.type === typeFilter)
  }, [results, typeFilter, deletedPaths])

  const hasSearched = searchedQuery.length > 0
  const hasTyped = query.trim().length > 0
  const hasResults = filteredResults && filteredResults.length > 0
  const noResultsAtAll = !loading && hasSearched && results && results.results && results.results.length === 0
  const noResultsForFilter = !loading && hasSearched && results && results.results && results.results.length > 0 && filteredResults && filteredResults.length === 0

  return (
    <div className="flex flex-col h-full">
      {/* Search bar */}
      <div className="srch-area">
        <div className={cn('srch-bar', hasTyped && 'active')}>
          <Search />
          <input
            type="text"
            placeholder='Search your vault...'
            value={query}
            onChange={(e) => e.target.value ? setQuery(e.target.value) : handleClear()}
            onKeyDown={(e) => {
              const q = query.trim()
              if (e.key === 'Enter' && q) {
                handleSearch(query.trim())
              }
            }}
            className="srch-val bg-transparent outline-none"
            style={{ fontFamily: "'Bricolage Grotesque', sans-serif" }}
          />
          {hasTyped ? (
            <div className="srch-clear" onClick={handleClear}>
              <X className="w-2.5 h-2.5" />
            </div>
          ) : null}
        </div>
        <div className="modes">
          <span className={cn('mc', mode === 'hybrid' && 'on')} onClick={() => handleModeChange('hybrid')}>hybrid</span>
          <span className={cn('mc', mode === 'keyword' && 'on')} onClick={() => handleModeChange('keyword')}>keyword</span>
          <span className={cn('mc', mode === 'semantic' && 'on')} onClick={() => handleModeChange('semantic')}>semantic</span>
        </div>
      </div>

      {/* Content */}
      <AnimatePresence mode="wait">
        {/* Loading */}
        {loading && (
          <motion.div key="loading" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="results">
            {[1, 2, 3].map(i => <div key={i} className="animate-shimmer rounded-lg h-24" />)}
          </motion.div>
        )}

        {/* No results at all */}
        {noResultsAtAll && (
          <motion.div key="no-results" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="no-results">
            <div className="nr-icon">
              <AlertCircle className="w-5 h-5" style={{ color: 'rgba(245,245,245,0.3)' }} />
            </div>
            <div className="nr-title">nothing found</div>
            <div className="nr-sub">no notes match<br />&ldquo;{searchedQuery}&rdquo;</div>
            <div className="nr-suggestions">
              <div className="recent-lbl">try instead</div>
              {mode !== 'keyword' && (
                <div className="nr-sug" onClick={() => handleSearch(query, 'keyword')}>
                  <Search className="w-3.5 h-3.5" style={{ color: 'rgba(245,245,245,0.2)' }} />
                  <span className="nr-sug-txt">{searchedQuery}</span>
                  <span className="nr-sug-mode">keyword</span>
                </div>
              )}
              {mode !== 'semantic' && (
                <div className="nr-sug" onClick={() => handleSearch(query, 'semantic')}>
                  <Search className="w-3.5 h-3.5" style={{ color: 'rgba(245,245,245,0.2)' }} />
                  <span className="nr-sug-txt">{searchedQuery}</span>
                  <span className="nr-sug-mode">semantic</span>
                </div>
              )}
              <div className="nr-sug capture" onClick={() => {
                onCaptureQuery?.(searchedQuery)
              }}>
                <span style={{ fontSize: 14, color: '#C9933A' }}>+</span>
                <span className="nr-sug-txt">capture a note about this</span>
              </div>
            </div>
          </motion.div>
        )}

        {/* No results for filter */}
        {noResultsForFilter && results && (
          <motion.div key="filter-empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
            <div className="results-header">
              <div className="rh-row">
                <span className="rh-count">0 results</span>
                <span className="rh-ms">{results.took_ms}ms</span>
              </div>
              <div className="filter-chips">
                <span className={cn('fc', typeFilter === 'all' && 'on')} onClick={() => setTypeFilter('all')}>all</span>
                <span className={cn('fc', typeFilter === 'text' && 'on')} onClick={() => setTypeFilter('text')}>text</span>
                <span className={cn('fc', typeFilter === 'article' && 'on')} onClick={() => setTypeFilter('article')}>article</span>
                <span className={cn('fc', typeFilter === 'image' && 'on')} onClick={() => setTypeFilter('image')}>image</span>
              </div>
            </div>
            <div className="no-results" style={{ paddingTop: 40 }}>
              <div className="nr-title">no {typeFilter} results</div>
              <div className="nr-sub">try a different filter<br />or reset to &ldquo;all&rdquo;</div>
            </div>
          </motion.div>
        )}

        {/* Results */}
        {!loading && hasResults && results && (
          <motion.div key="results" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
            <div className="results-header">
              <div className="rh-row">
                <span className="rh-count">{filteredResults.length} results</span>
                <span className="rh-ms">{results.took_ms}ms</span>
              </div>
              <div className="filter-chips">
                <span className={cn('fc', typeFilter === 'all' && 'on')} onClick={() => setTypeFilter('all')}>all</span>
                <span className={cn('fc', typeFilter === 'text' && 'on')} onClick={() => setTypeFilter('text')}>text</span>
                <span className={cn('fc', typeFilter === 'article' && 'on')} onClick={() => setTypeFilter('article')}>article</span>
                <span className={cn('fc', typeFilter === 'image' && 'on')} onClick={() => setTypeFilter('image')}>image</span>
              </div>
            </div>
            <div className="results">
              <AIAnswerRow
                state={ai.state}
                expanded={aiExpanded}
                overview={ai.overview}
                onAsk={handleAskAI}
                onToggle={() => setAiExpanded(v => !v)}
                onRetry={handleAskAI}
                onClose={handleCloseAI}
                onCitationClick={handleCitationClick}
              />
              {filteredResults.map((result, index) => (
                <div key={result.id} id={`result-${index}`}>
                  {index === 0 && result.score > 0.9 ? (
                    <ResultHero result={result} query={searchedQuery} onSelect={handleNoteSelectLocal} />
                  ) : (
                    <ResultCompact result={result} rank={index + 1} query={searchedQuery} onSelect={handleNoteSelectLocal} />
                  )}
                </div>
              ))}
            </div>
          </motion.div>
        )}

        {/* Idle/empty state */}
        {!loading && !hasSearched && !results && (
          <motion.div key="idle" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="search-empty">
            {/* Recent searches */}
            {recentSearches.length > 0 && (
              <>
                <div className="recent-lbl">recent searches</div>
                {recentSearches.map((q, i) => (
                  <div key={i} className="recent-item" onClick={() => handleSearch(q)}>
                    <div className="ri-icon t">
                      <Search className="w-3 h-3" style={{ color: 'rgba(245,245,245,0.3)' }} />
                    </div>
                    <span className="ri-text">{q}</span>
                    <div className="srch-clear" onClick={(e) => handleRemoveRecent(q, e)}>
                      <X className="w-2 h-2" />
                    </div>
                  </div>
                ))}
              </>
            )}

            {/* Suggestions */}
            <div className="suggestions-lbl">try searching for</div>
            <div className="sug-chips">
              {SUGGESTIONS.map(s => (
                <span key={s} className="sc" onClick={() => handleSearch(s)}>{s}</span>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
