import { ComponentChildren } from 'preact'

export interface EmptyStateProps {
  icon?: ComponentChildren
  title?: string
  description?: string
  action?: ComponentChildren
  className?: string
}

export function EmptyState({ icon, title, description, action, className = '' }: EmptyStateProps) {
  return (
    <div className={`empty-state ${className}`.trim()}>
      {icon && <div className="empty-state-icon">{icon}</div>}
      {title && <div style={{ fontWeight: 600, fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '2px' }}>{title}</div>}
      {description && <div className="empty-state-text">{description}</div>}
      {action && <div style={{ marginTop: '12px' }}>{action}</div>}
    </div>
  )
}
