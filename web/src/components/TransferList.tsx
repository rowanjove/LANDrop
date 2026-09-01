import { useEffect, useState } from 'preact/hooks'
import { TransferRow } from './TransferRow.tsx'
import { Button } from './Button.tsx'
import { EmptyState } from './EmptyState.tsx'
import { IconHistory } from './Icons.tsx'
import { transfersState, TransfersState } from '../state/transfers.ts'
import { t } from '../locales/index.ts'

export interface TransferListProps {
  onViewAll?: () => void
}

export function TransferList({ onViewAll }: TransferListProps) {
  const [state, setState] = useState<TransfersState>(transfersState.get())

  useEffect(() => {
    return transfersState.subscribe((s) => setState(s))
  }, [])

  // Limit to 5 records for homepage summary
  const recentRecords = state.recentHistory.slice(0, 5)

  return (
    <section className="panel-section">
      <div className="panel-header">
        <div className="panel-title">
          <IconHistory size={16} />
          <span>{t('transfer.recent')}</span>
        </div>
        {onViewAll && state.recentHistory.length > 0 && (
          <Button variant="ghost" size="sm" onClick={onViewAll}>
            {t('transfer.viewAll')}
          </Button>
        )}
      </div>

      <div>
        {recentRecords.length === 0 ? (
          <EmptyState
            icon={<IconHistory size={24} />}
            title={t('transfer.emptyTitle')}
            description={t('transfer.emptyHint')}
          />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {recentRecords.map((record, idx) => (
              <TransferRow key={`${record.timestamp}-${idx}`} record={record} />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
