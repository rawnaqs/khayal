import { motion } from 'framer-motion'
import { PenLine, Search, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Tab } from '@/App'

interface BottomNavProps {
  activeTab: Tab
  onTabChange: (tab: Tab) => void
}

const tabs = [
  { id: 'capture' as Tab, label: 'capture', icon: PenLine },
  { id: 'search' as Tab, label: 'search', icon: Search },
  { id: 'queue' as Tab, label: 'queue', icon: Clock },
]

export function BottomNav({ activeTab, onTabChange }: BottomNavProps) {
  return (
    <nav className="nav" style={{ paddingBottom: 'max(env(safe-area-inset-bottom), 26px)' }}>
      {tabs.map((tab) => {
        const Icon = tab.icon
        const isActive = activeTab === tab.id

        return (
          <div
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            className={cn('nt', isActive && 'on')}
          >
            <Icon />
            {isActive ? (
              <motion.div
                layoutId="navIndicator"
                className="w-5 h-0.5 rounded-full"
                style={{ background: '#C9933A' }}
                transition={{ type: 'spring', stiffness: 380, damping: 30 }}
              />
            ) : (
              <div className="w-5 h-0.5" />
            )}
            <span className="nt-l">{tab.label}</span>
          </div>
        )
      })}
    </nav>
  )
}
