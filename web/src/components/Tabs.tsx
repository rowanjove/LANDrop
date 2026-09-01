import { ComponentChildren } from 'preact'

export interface TabItem<T extends string = string> {
  id: T
  label: string
  icon?: ComponentChildren
  badge?: number | string
}

export interface TabsProps<T extends string = string> {
  tabs: TabItem<T>[]
  activeTab: T
  onChange: (tabId: T) => void
  className?: string
}

export function Tabs<T extends string = string>({ tabs, activeTab, onChange, className = '' }: TabsProps<T>) {
  return (
    <div className={`tabs-header ${className}`.trim()} role="tablist">
      {tabs.map((tab) => {
        const isActive = tab.id === activeTab
        return (
          <button
            key={tab.id}
            role="tab"
            aria-selected={isActive}
            className={`tab-btn ${isActive ? 'active' : ''}`}
            onClick={() => onChange(tab.id)}
          >
            {tab.icon && <span style={{ display: 'flex', alignItems: 'center' }}>{tab.icon}</span>}
            <span>{tab.label}</span>
            {tab.badge !== undefined && (
              <span className="status-badge" style={{ marginLeft: '2px', background: isActive ? 'var(--accent-subtle)' : 'var(--border)' }}>
                {tab.badge}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
