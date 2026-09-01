import { useEffect, useState } from 'preact/hooks'
import { TransferRow } from '../components/TransferRow.tsx'
import { Button } from '../components/Button.tsx'
import { Dialog } from '../components/Dialog.tsx'
import { EmptyState } from '../components/EmptyState.tsx'
import { IconHistory, IconSearch, IconTrash } from '../components/Icons.tsx'
import { clearHistory, HistoryRecord } from '../api/history.ts'
import { transfersState, TransfersState } from '../state/transfers.ts'
import { formatDate } from '../utils/format.ts'
import { toast } from '../components/Toast.tsx'
import { t } from '../locales/index.ts'

export function Activity() {
  const [state, setState] = useState<TransfersState>(transfersState.get())
  const [search, setSearch] = useState('')
  const [showFilter, setShowFilter] = useState(false)
  const [filterType, setFilterType] = useState<'all' | 'file' | 'text'>('all')
  const [filterDirection, setFilterDirection] = useState<'all' | 'send' | 'recv'>('all')
  const [filterStatus, setFilterStatus] = useState<'all' | 'success' | 'failed' | 'interrupted'>('all')
  const [clearDialogOpen, setClearDialogOpen] = useState(false)
  const [clearing, setClearing] = useState(false)

  useEffect(() => {
    transfersState.refreshHistory(200)
    return transfersState.subscribe((s) => setState(s))
  }, [])

  const handleClearHistory = async () => {
    setClearing(true)
    try {
      await clearHistory()
      transfersState.setRecentHistory([])
      toast.success(t('activity.clearedSuccess'))
      setClearDialogOpen(false)
    } catch {
      toast.error(t('activity.clearFailed'))
    } finally {
      setClearing(false)
    }
  }

  // Filter and search records
  const filteredRecords = state.recentHistory.filter((r) => {
    if (filterType !== 'all' && r.type !== filterType) return false
    if (filterDirection !== 'all' && r.direction !== filterDirection) return false
    if (filterStatus !== 'all' && r.status !== filterStatus) return false

    if (search.trim()) {
      const q = search.toLowerCase().trim()
      const matchName = r.name && r.name.toLowerCase().includes(q)
      const matchPeer = r.peer && r.peer.toLowerCase().includes(q)
      return matchName || matchPeer
    }

    return true
  })

  // Group by Today, Yesterday, or Date string
  const today = new Date().toDateString()
  const yesterday = new Date(Date.now() - 86400000).toDateString()

  const groups: Record<string, HistoryRecord[]> = {}
  filteredRecords.forEach((r) => {
    const recordDate = new Date(r.timestamp * 1000).toDateString()
    let groupKey = formatDate(r.timestamp)
    if (recordDate === today) {
      groupKey = t('activity.today')
    } else if (recordDate === yesterday) {
      groupKey = t('activity.yesterday')
    }
    if (!groups[groupKey]) {
      groups[groupKey] = []
    }
    groups[groupKey].push(r)
  })

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* Page Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 style={{ fontSize: '1.2rem', fontWeight: 600 }}>{t('activity.title')}</h1>
        {state.recentHistory.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            icon={<IconTrash size={14} />}
            onClick={() => setClearDialogOpen(true)}
          >
            {t('activity.clearHistory')}
          </Button>
        )}
      </div>

      {/* Search & Filter Bar (Only show if there are records) */}
      {state.recentHistory.length > 0 && (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '10px',
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            padding: '12px',
          }}
        >
          <div style={{ display: 'flex', gap: '8px' }}>
            <div style={{ position: 'relative', flex: 1 }}>
              <div
                style={{
                  position: 'absolute',
                  left: '10px',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  color: 'var(--text-muted)',
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <IconSearch size={15} />
              </div>
              <input
                className="input"
                style={{ paddingLeft: '32px' }}
                placeholder={t('activity.searchPlaceholder')}
                value={search}
                onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
              />
            </div>
            <Button
              variant={showFilter ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setShowFilter(!showFilter)}
            >
              {t('activity.filter')}
            </Button>
          </div>

          {/* Filter Options */}
          {showFilter && (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))',
                gap: '8px',
                paddingTop: '8px',
                borderTop: '1px solid var(--border)',
              }}
            >
              <select
                className="select"
                value={filterType}
                onChange={(e) => setFilterType((e.target as HTMLSelectElement).value as any)}
              >
                <option value="all">{t('activity.allTypes')}</option>
                <option value="file">{t('activity.typeFile')}</option>
                <option value="text">{t('activity.typeText')}</option>
              </select>

              <select
                className="select"
                value={filterDirection}
                onChange={(e) => setFilterDirection((e.target as HTMLSelectElement).value as any)}
              >
                <option value="all">{t('activity.allDirections')}</option>
                <option value="send">{t('activity.dirSend')}</option>
                <option value="recv">{t('activity.dirRecv')}</option>
              </select>

              <select
                className="select"
                value={filterStatus}
                onChange={(e) => setFilterStatus((e.target as HTMLSelectElement).value as any)}
              >
                <option value="all">{t('activity.allStatuses')}</option>
                <option value="success">{t('activity.statusSuccess')}</option>
                <option value="failed">{t('activity.statusFailed')}</option>
                <option value="interrupted">{t('activity.statusInterrupted')}</option>
              </select>
            </div>
          )}
        </div>
      )}

      {/* History Records List */}
      {state.recentHistory.length === 0 ? (
        <EmptyState
          icon={<IconHistory size={32} />}
          title={t('activity.empty')}
          description={t('activity.emptyHint')}
        />
      ) : filteredRecords.length === 0 ? (
        <EmptyState
          icon={<IconSearch size={28} />}
          title={t('activity.noMatch')}
          description={t('activity.noMatchHint')}
        />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {Object.entries(groups).map(([groupTitle, records]) => (
            <div key={groupTitle} className="panel-section">
              <div
                style={{
                  padding: '8px 14px',
                  fontSize: '12px',
                  fontWeight: 600,
                  color: 'var(--text-secondary)',
                  background: 'var(--surface-subtle)',
                  borderBottom: '1px solid var(--border)',
                }}
              >
                {groupTitle}
              </div>
              <div style={{ display: 'flex', flexDirection: 'column' }}>
                {records.map((r, idx) => (
                  <TransferRow key={`${r.timestamp}-${idx}`} record={r} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Clear Confirmation Modal */}
      <Dialog
        open={clearDialogOpen}
        onClose={() => setClearDialogOpen(false)}
        title={t('dialogs.clearHistoryTitle')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setClearDialogOpen(false)} disabled={clearing}>
              {t('dialogs.cancel')}
            </Button>
            <Button variant="danger" onClick={handleClearHistory} loading={clearing}>
              {t('dialogs.confirm')}
            </Button>
          </>
        }
      >
        <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
          {t('dialogs.clearHistoryConfirm')}
        </p>
      </Dialog>
    </div>
  )
}
