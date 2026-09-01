import { HistoryRecord } from '../api/history.ts'
import { formatSize, formatTime } from '../utils/format.ts'
import { IconDownload, IconFile, IconText, IconUpload } from './Icons.tsx'
import { t } from '../locales/index.ts'

export interface TransferRowProps {
  record: HistoryRecord
}

export function TransferRow({ record }: TransferRowProps) {
  const isSend = record.direction === 'send'
  const isText = record.type === 'text'

  const statusClass =
    record.status === 'success'
      ? 'success'
      : record.status === 'failed'
      ? 'failed'
      : 'interrupted'

  const statusLabel =
    record.status === 'success'
      ? t('activity.statusSuccess')
      : record.status === 'failed'
      ? t('activity.statusFailed')
      : t('activity.statusInterrupted')

  return (
    <div className="item-row" style={{ padding: '9px 12px', gap: '10px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: 1 }}>
        <div
          style={{
            width: '28px',
            height: '28px',
            borderRadius: 'var(--radius-sm)',
            background: 'var(--surface-subtle)',
            border: '1px solid var(--border)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--text-secondary)',
            flexShrink: 0,
          }}
        >
          {isText ? <IconText size={15} /> : <IconFile size={15} />}
        </div>
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              fontSize: '13px',
              fontWeight: 500,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {record.name || (isText ? t('transfer.textMessage') : t('transfer.unknownFile'))}
          </div>
          <div style={{ fontSize: '11px', color: 'var(--text-secondary)', display: 'flex', gap: '6px', alignItems: 'center' }}>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: '2px' }}>
              {isSend ? <IconUpload size={11} /> : <IconDownload size={11} />}
              {isSend ? t('activity.dirSend') : t('activity.dirRecv')}
            </span>
            <span>·</span>
            <span>{formatSize(record.size)}</span>
            {record.peer && (
              <>
                <span>·</span>
                <span>{record.peer}</span>
              </>
            )}
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '2px', flexShrink: 0 }}>
        <span className={`status-badge ${statusClass}`}>{statusLabel}</span>
        <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
          {formatTime(record.timestamp)}
        </span>
      </div>
    </div>
  )
}
