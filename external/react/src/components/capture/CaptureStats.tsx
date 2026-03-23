import type { StatsResponse } from '@/lib/api'

interface CaptureStatsProps {
  stats: StatsResponse | null
  loading: boolean
}

function timeAgo(dateStr: string) {
  if (!dateStr) return ''
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
    if (diff < 60) return `${diff}s ago`
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
    return `${Math.floor(diff / 86400)}d ago`
  } catch {
    return ''
  }
}

function StreakTile({ stats }: { stats: StatsResponse }) {
  const { current, next_milestone, days_to_milestone, this_week } = stats.streak

  // Arc progress
  const circumference = 2 * Math.PI * 23
  const progress = next_milestone > 0 ? current / next_milestone : 1
  const dashOffset = circumference * (1 - Math.min(progress, 1))

  // Today index for week dots (0=Mon, 6=Sun)
  const dayOfWeek = new Date().getDay()
  const todayIndex = dayOfWeek === 0 ? 6 : dayOfWeek - 1

  return (
    <div className="bt bt-streak">
      <div className="lbl">streak</div>
      <div className="streak-body">
        <div className="arc">
          <svg viewBox="0 0 58 58">
            <circle
              fill="none"
              stroke="rgba(255,255,255,0.05)"
              strokeWidth="5"
              strokeLinecap="round"
              cx="29"
              cy="29"
              r="23"
              transform="rotate(-90 29 29)"
            />
            <circle
              fill="none"
              stroke="url(#streakGrad)"
              strokeWidth="5"
              strokeLinecap="round"
              cx="29"
              cy="29"
              r="23"
              strokeDasharray={circumference}
              strokeDashoffset={dashOffset}
              transform="rotate(-90 29 29)"
            />
            <defs>
              <linearGradient id="streakGrad" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#C9933A" />
                <stop offset="100%" stopColor="#E8B86D" />
              </linearGradient>
            </defs>
          </svg>
          <div className="arc-center">
            <span className="arc-n">{current}</span>
            <span className="arc-u">days</span>
          </div>
        </div>
        <div className="streak-right">
          <div className="streak-num">{current}</div>
          <div className="streak-unit">day streak</div>
          {days_to_milestone > 0 && (
            <div className="streak-goal">
              {days_to_milestone} days to {next_milestone}
            </div>
          )}
        </div>
      </div>
      <div className="week-dots">
        {this_week.map((on, i) => (
          <div
            key={i}
            className={`wd ${i === todayIndex ? (on ? 'today' : 'off') : on ? 'on' : 'off'}`}
          />
        ))}
      </div>
    </div>
  )
}

function TodayTile({ stats }: { stats: StatsResponse }) {
  const { count, by_hour, avg_per_day } = stats.today
  const currentHour = new Date().getHours()

  const maxCount = Math.max(...by_hour, 1)

  return (
    <div className="bt">
      <div className="lbl">today</div>
      <div className="today-num">{count}</div>
      <div className="today-sub">captures</div>
      <div className="hours">
        {by_hour.map((c, i) => {
          const height = c > 0 ? Math.max((c / maxCount) * 100, 8) : 8
          const isCurrent = i === currentHour
          const isHigh = c > 0 && c === maxCount
          const isEmpty = c === 0

          let className = 'hb'
          if (isEmpty) className += ' empty'
          else if (isCurrent) className += ' now'
          else if (isHigh) className += ' hi'

          return (
            <div
              key={i}
              className={className}
              style={{ height: `${height}%` }}
            />
          )
        })}
      </div>
      <div className="today-footer">
        <span className="tf-stat">avg <span>{avg_per_day.toFixed(1)}/day</span></span>
        <span className="tf-stat">last <span>{timeAgo(stats.vault.last_capture_at)}</span></span>
      </div>
    </div>
  )
}

function VaultTile({ stats }: { stats: StatsResponse }) {
  const { total_notes, today_delta, last_7_days } = stats.vault

  const maxDay = Math.max(...last_7_days, 1)

  return (
    <div className="bt wide">
      <div className="lbl">vault</div>
      <div className="vault-inner">
        <div>
          <div className="vault-num">{total_notes.toLocaleString()}</div>
          <div className="vault-unit">notes</div>
          {today_delta > 0 && (
            <div className="vault-delta">+{today_delta} today</div>
          )}
        </div>
        <div className="vault-center">
          <div className="vc-stat">last 7d<span>{last_7_days.reduce((a, b) => a + b, 0)}</span></div>
        </div>
        <div className="spark">
          {last_7_days.map((c, i) => {
            const height = c > 0 ? Math.max((c / maxDay) * 100, 8) : 8
            const isToday = i === 6
            return (
              <div
                key={i}
                className={`sb-bar ${isToday ? 'today' : 'prev'}`}
                style={{ height: `${height}%` }}
              />
            )
          })}
        </div>
      </div>
    </div>
  )
}

export function CaptureStats({ stats, loading }: CaptureStatsProps) {
  if (loading) {
    return (
      <div className="bento">
        <div className="bt animate-shimmer" style={{ height: 140 }} />
        <div className="bt animate-shimmer" style={{ height: 140 }} />
        <div className="bt wide animate-shimmer" style={{ height: 80 }} />
      </div>
    )
  }

  if (!stats) {
    return (
      <div className="bento">
        <div className="bt">
          <div className="lbl">streak</div>
          <div className="streak-num">0</div>
          <div className="streak-unit">day streak</div>
        </div>
        <div className="bt">
          <div className="lbl">today</div>
          <div className="today-num">0</div>
          <div className="today-sub">captures</div>
        </div>
        <div className="bt wide">
          <div className="lbl">vault</div>
          <div className="vault-num">0</div>
          <div className="vault-unit">notes</div>
        </div>
      </div>
    )
  }

  return (
    <div className="bento">
      <StreakTile stats={stats} />
      <TodayTile stats={stats} />
      <VaultTile stats={stats} />
    </div>
  )
}
