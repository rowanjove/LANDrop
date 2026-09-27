import { useEffect, useState } from 'preact/hooks'
import { TransferRow } from './TransferRow.tsx'
import { Button } from './Button.tsx'
import { EmptyState } from './EmptyState.tsx'
import { IconDownload, IconExchange, IconFile, IconHistory, IconText } from './Icons.tsx'
import { transfersState, TransfersState } from '../state/transfers.ts'
import { appState, AppState } from '../state/app.ts'
import { ActiveTransferItem, getDownloadUrl, getShareUrl } from '../api/transfers.ts'
import { formatSize } from '../utils/format.ts'
import { t } from '../locales/index.ts'

const SESSION_START_KEY = 'landrop_session_start'
function getSessionStart(): number {
  if (typeof window === 'undefined') return 0
  const stored = sessionStorage.getItem(SESSION_START_KEY)
  if (stored) {
    const parsed = parseInt(stored, 10)
    if (!isNaN(parsed) && parsed > 0) return parsed
  }
  const now = Math.floor(Date.now() / 1000)
  sessionStorage.setItem(SESSION_START_KEY, String(now))
  return now
}

export interface TransferListProps {
  onViewAll?: () => void
}

export function TransferList({ onViewAll }: TransferListProps) {
  const [state, setState] = useState<TransfersState>(transfersState.get())
  const [app, setApp] = useState<AppState>(appState.get())

  useEffect(() => {
    const unsubTransfers = transfersState.subscribe((s) => setState(s))
    const unsubApp = appState.subscribe((a) => setApp(a))
    return () => {
      unsubTransfers()
      unsubApp()
    }
  }, [])

  // Only display transfers that occurred in the current session
  const sessionStart = app.info?.started_at ? (app.info.started_at - 1) : getSessionStart()
  const currentRecords = state.recentHistory.filter((record) => record.timestamp >= sessionStart)
  const readyTransfers = state.activeTransfers || []

  return (
    <section className="panel-section">
      <div className="panel-header">
        <div className="panel-title">
          <IconExchange size={16} />
          <span>{t('transfer.recent')}</span>
          {readyTransfers.length > 0 && (
            <span
              className="status-badge"
              style={{ background: 'var(--surface-subtle)', color: 'var(--accent)' }}
            >
              {readyTransfers.length} {t('transfer.readyBadge')}
            </span>
          )}
        </div>
        {onViewAll && (
          <Button variant="ghost" size="sm" icon={<IconHistory size={14} />} onClick={onViewAll}>
            {t('transfer.viewAll')}
          </Button>
        )}
      </div>

      <div>
        {/* Active ready transfers waiting for download */}
        {readyTransfers.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', marginBottom: currentRecords.length > 0 ? '12px' : '0' }}>
            <div style={{ padding: '4px 12px 6px', fontSize: '11px', fontWeight: 600, color: 'var(--accent)' }}>
              {t('transfer.readyTitle')}
            </div>
            {readyTransfers.map((item) => (
              <ReadyTransferRow key={item.id || item.token} item={item} />
            ))}
          </div>
        )}

        {readyTransfers.length === 0 && currentRecords.length === 0 ? (
          <EmptyState
            icon={<IconHistory size={24} />}
            title={t('transfer.emptyTitle')}
            description={t('transfer.emptyHint')}
          />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {currentRecords.map((record, idx) => (
              <TransferRow key={`${record.timestamp}-${idx}`} record={record} />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function ReadyTransferRow({ item }: { item: ActiveTransferItem }) {
  const isText = item.type === 'text'
  const displayName = item.name || (isText ? t('transfer.textMessage') : t('transfer.unknownFile'))

  return (
    <div
      className="item-row"
      style={{
        padding: '9px 12px',
        gap: '10px',
        border: '1px solid var(--border)',
        background: 'var(--surface-subtle)',
        borderRadius: 'var(--radius-sm)',
        marginBottom: '6px',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: 1 }}>
        <div
          style={{
            width: '28px',
            height: '28px',
            borderRadius: 'var(--radius-sm)',
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--accent)',
            flexShrink: 0,
          }}
        >
          {isText ? <IconText size={15} /> : <IconFile size={15} />}
        </div>
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              fontSize: '13px',
              fontWeight: 600,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {displayName}
          </div>
          <div style={{ fontSize: '11px', color: 'var(--text-secondary)', display: 'flex', gap: '6px', alignItems: 'center' }}>
            <span>{formatSize(item.size)}</span>
            <span>·</span>
            <span style={{ color: 'var(--accent)', fontWeight: 500 }}>{t('transfer.readyBadge')}</span>
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexShrink: 0 }}>
        {isText ? (
          <a
            href={getShareUrl(item.token)}
            className="btn btn-primary btn-sm"
            style={{ textDecoration: 'none' }}
          >
            {t('transfer.viewBtn')}
          </a>
        ) : (
          <a
            href={getDownloadUrl(item.token)}
            download={item.name || true}
            className="btn btn-primary btn-sm"
            style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '4px' }}
          >
            <IconDownload size={13} />
            <span>{t('transfer.downloadBtn')}</span>
          </a>
        )}
      </div>
    </div>
  )
}
