import { useEffect, useState } from 'preact/hooks'
import { DeviceRow } from './DeviceRow.tsx'
import { Button } from './Button.tsx'
import { EmptyState } from './EmptyState.tsx'
import { ManualConnectDialog } from './ManualConnectDialog.tsx'
import { IconDevice, IconPlus, IconRefresh } from './Icons.tsx'
import { devicesState, DevicesState } from '../state/devices.ts'
import { t } from '../locales/index.ts'

export function DeviceList() {
  const [state, setState] = useState<DevicesState>(devicesState.get())
  const [manualOpen, setManualOpen] = useState(false)

  useEffect(() => {
    return devicesState.subscribe((s) => setState(s))
  }, [])

  return (
    <section className="panel-section">
      <div className="panel-header">
        <div className="panel-title">
          <IconDevice size={16} />
          <span>{t('devices.title')}</span>
          {state.devices.length > 0 && (
            <span
              className="status-badge"
              style={{ background: 'var(--surface-subtle)', color: 'var(--text-secondary)' }}
            >
              {state.devices.length}
            </span>
          )}
        </div>
        <div style={{ display: 'flex', gap: '4px' }}>
          <Button
            variant="ghost"
            size="sm"
            icon={<IconRefresh size={14} />}
            onClick={() => devicesState.refresh()}
            loading={state.loading}
            aria-label={t('accessibility.refreshDevices')}
          />
          <Button
            variant="ghost"
            size="sm"
            icon={<IconPlus size={14} />}
            onClick={() => setManualOpen(true)}
          >
            {t('devices.manualConnect')}
          </Button>
        </div>
      </div>

      <div style={{ minHeight: '120px' }}>
        {state.loading && state.devices.length === 0 ? (
          <div style={{ padding: '24px 16px', textAlign: 'center', fontSize: '12px', color: 'var(--text-muted)' }}>
            {t('devices.scanning')}
          </div>
        ) : state.devices.length === 0 ? (
          <EmptyState
            icon={<IconDevice size={24} />}
            title={t('devices.empty')}
            description={t('devices.emptyHint')}
          />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {state.devices.map((device) => (
              <DeviceRow key={device.addr} device={device} />
            ))}
          </div>
        )}
      </div>

      <ManualConnectDialog open={manualOpen} onClose={() => setManualOpen(false)} />
    </section>
  )
}
