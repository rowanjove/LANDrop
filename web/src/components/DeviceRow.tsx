import { DeviceItem } from '../api/devices.ts'
import { formatTime } from '../utils/format.ts'
import { IconDevice, IconGlobe, IconMobile } from './Icons.tsx'
import { t } from '../locales/index.ts'

export interface DeviceRowProps {
  device: DeviceItem
}

function getDeviceIcon(os?: string, version?: string) {
  const lower = (os || '').toLowerCase()
  if (lower === 'ios' || lower === 'android') {
    return <IconMobile size={15} />
  }
  if (version === 'web' || lower === 'web') {
    return <IconGlobe size={15} />
  }
  return <IconDevice size={15} />
}

export function DeviceRow({ device }: DeviceRowProps) {
  const isWebClient = device.version === 'web'
  const handleClick = () => {
    if (isWebClient) return
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
        cursor: isWebClient ? 'default' : 'pointer',
        opacity: device.online ? 1 : 0.6,
        padding: '10px 14px',
      }}
      title={isWebClient ? undefined : t('devices.openDevice')}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
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
            color: device.online ? 'var(--accent)' : 'var(--text-muted)',
            flexShrink: 0,
            position: 'relative',
          }}
        >
          {getDeviceIcon(device.os, device.version)}
          <span
            className={`status-dot ${device.online ? 'online' : 'offline'}`}
            style={{
              position: 'absolute',
              bottom: '-2px',
              right: '-2px',
              border: '2px solid var(--surface)',
            }}
          />
        </div>
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
