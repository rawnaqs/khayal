import { useServerStatus } from '@/hooks/useServerStatus'

export function Header() {
  const { status } = useServerStatus()

  const onlineColor = status === 'ok' ? '#3ddc84' : status === 'degraded' ? '#ffb340' : '#ff4d4d'

  return (
    <header className="hdr">
      <div className="brand">
        <img src="/icon.svg" alt="khayal" className="mark" />
        <span className="bname">khayal</span>
      </div>
      <div className="flex items-center gap-2">
        <span className="ver">v0.1.0</span>
        <div className="online" style={{ 
          background: onlineColor,
          boxShadow: status === 'ok' ? `0 0 8px ${onlineColor}` : 'none'
        }} />
      </div>
    </header>
  )
}
