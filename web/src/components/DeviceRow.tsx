import { DeviceItem } from '../api/devices.ts'
import { formatTime } from '../utils/format.ts'
import { t } from '../locales/index.ts'

export interface DeviceRowProps {
  device: DeviceItem
}

export function DeviceRow({ device }: DeviceRowProps) {
  const handleClick = () => {
    let url = device.addr
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      const fallbackScheme = window.location.protocol.replace(':', '') || 'http'
      url = `${device.scheme || fallbackScheme}://${url}`
    }
    window.location.href = url
  }

  return (
    <div
      className="item-row"
      onClick={handleClick}
      style={{
        cursor: 'pointer',
        opacity: device.online ? 1 : 0.6,
        padding: '10px 14px',
      }}
      title={t('devices.openDevice')}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
        <span
          className={`status-dot ${device.online ? 'online' : 'offline'}`}
          style={{ flexShrink: 0 }}
        />
        <div style={{ minWidth: 0 }}>
          <div
            style={{
              fontSize: '13px',
              fontWeight: 500,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {device.name}
          </div>
          <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
            {device.os || 'LAN'} · {device.addr}
          </div>
        </div>
      </div>

      <div style={{ flexShrink: 0, textAlign: 'right' }}>
        {device.online ? (
          <span style={{ fontSize: '11px', color: 'var(--success)' }}>{t('devices.online')}</span>
        ) : (
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
            {device.last_seen ? t('devices.lastSeen', { time: formatTime(device.last_seen) }) : t('devices.offline')}
          </span>
        )}
      </div>
    </div>
  )
}
