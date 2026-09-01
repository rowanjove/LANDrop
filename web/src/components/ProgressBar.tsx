export interface ProgressBarProps {
  percent: number
  status?: 'normal' | 'success' | 'danger'
  transferredText?: string
  speedText?: string
  etaText?: string
  className?: string
}

export function ProgressBar({
  percent,
  status = 'normal',
  transferredText,
  speedText,
  etaText,
  className = '',
}: ProgressBarProps) {
  const clamped = Math.min(100, Math.max(0, percent))
  const fillClass = status === 'success' ? 'success' : status === 'danger' ? 'danger' : ''

  return (
    <div className={`progress-container ${className}`.trim()}>
      <div className="progress-track">
        <div className={`progress-fill ${fillClass}`} style={{ width: `${clamped}%` }} />
      </div>
      {(transferredText || speedText || etaText) && (
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--text-secondary)' }}>
          <div>{transferredText || `${Math.round(clamped)}%`}</div>
          <div style={{ display: 'flex', gap: '8px' }}>
            {speedText && <span>{speedText}</span>}
            {etaText && <span>{etaText}</span>}
          </div>
        </div>
      )}
    </div>
  )
}
