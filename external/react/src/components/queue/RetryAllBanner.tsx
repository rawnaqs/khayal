import { RotateCcw } from 'lucide-react'

interface RetryAllBannerProps {
  count: number
  onRetryAll: () => void
}

export function RetryAllBanner({ count, onRetryAll }: RetryAllBannerProps) {
  return (
    <div className="retry-all" onClick={onRetryAll}>
      <div className="ra-left">
        <span className="ra-ct">{count}</span>
        <span className="ra-txt">jobs failed</span>
      </div>
      <div className="ra-btn">
        <RotateCcw className="ra-icon" />
        retry all
      </div>
    </div>
  )
}
