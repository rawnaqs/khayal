import { cn } from '@/lib/utils'

interface QueueMetricsProps {
  pending: number
  processing: number
  failed: number
}

export function QueueMetrics({ pending, processing, failed }: QueueMetricsProps) {
  return (
    <>
      <div className="sec">queue</div>
      <div className="stats-row">
        <div className="stat sw">
          <div className={cn('stat-n', pending > 0 && 'warn')}>{pending}</div>
          <div className="stat-l">pending</div>
        </div>
        <div className="stat so">
          <div className={cn('stat-n', processing > 0 && 'ok')}>{processing}</div>
          <div className="stat-l">processing</div>
        </div>
        <div className="stat sb">
          <div className={cn('stat-n', failed > 0 && 'bad')}>{failed}</div>
          <div className="stat-l">failed</div>
        </div>
      </div>
    </>
  )
}
