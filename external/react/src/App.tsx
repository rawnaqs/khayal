import { useState, useCallback } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { Toaster } from '@/components/ui/toaster'
import { Header } from '@/components/layout/Header'
import { BottomNav } from '@/components/layout/BottomNav'
import { CaptureView } from '@/components/capture/CaptureView'
import { SearchView } from '@/components/search/SearchView'
import { QueueView } from '@/components/queue/QueueView'
import { NoteView } from '@/components/note/NoteView'
import { Onboarding } from '@/components/Onboarding'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { STORAGE_KEYS } from '@/lib/constants'

export type Tab = 'capture' | 'search' | 'queue'

const pageVariants = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -12 },
}

export default function App() {
  const [activeTab, setActiveTab] = useState<Tab>('capture')
  const [captureQuery, setCaptureQuery] = useState<string | undefined>(undefined)
  const [selectedNote, setSelectedNote] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [isConfigured, setIsConfigured] = useState(() => {
    return !!localStorage.getItem(STORAGE_KEYS.TOKEN) && !!localStorage.getItem(STORAGE_KEYS.HOST)
  })

  const handleCaptureQuery = useCallback((query: string) => {
    setCaptureQuery(query)
    setActiveTab('capture')
  }, [])

  const handleCaptureQueryConsumed = useCallback(() => {
    setCaptureQuery(undefined)
  }, [])

  const handleNoteSelect = useCallback((notePath: string, query?: string) => {
    setSelectedNote(notePath)
    setSearchQuery(query || '')
  }, [])

  const handleBackToSearch = useCallback(() => {
    setSelectedNote(null)
    setSearchQuery('')
  }, [])

  if (!isConfigured) {
    return (
      <ErrorBoundary>
        <Onboarding onComplete={() => setIsConfigured(true)} />
      </ErrorBoundary>
    )
  }

  const renderView = () => {
    switch (activeTab) {
      case 'capture':
        return <CaptureView captureQuery={captureQuery} onCaptureQueryConsumed={handleCaptureQueryConsumed} />
      case 'search':
        return <SearchView onCaptureQuery={handleCaptureQuery} onNoteSelect={handleNoteSelect} />
      case 'queue':
        return <QueueView />
      default:
        return <CaptureView captureQuery={captureQuery} onCaptureQueryConsumed={handleCaptureQueryConsumed} />
    }
  }

  return (
    <ErrorBoundary>
      <div className="flex flex-col h-screen overflow-hidden" style={{ background: '#070707' }}>
        <Header />
        <main className="flex-1 overflow-hidden">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              variants={pageVariants}
              initial="initial"
              animate="animate"
              exit="exit"
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="h-full overflow-y-auto"
              style={{ paddingBottom: '1rem' }}
            >
              {renderView()}
            </motion.div>
          </AnimatePresence>
        </main>
        <BottomNav activeTab={activeTab} onTabChange={setActiveTab} />
        <Toaster />
        <NoteView
          notePath={selectedNote}
          query={searchQuery || undefined}
          onClose={handleBackToSearch}
        />
      </div>
    </ErrorBoundary>
  )
}
