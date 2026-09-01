import { useEffect, useState } from 'preact/hooks'
import { appState, AppState } from '../state/app.ts'
import { Button } from './Button.tsx'
import { IconDevice, IconHistory, IconLogo, IconQr, IconSettings } from './Icons.tsx'
import { QRDialog } from './QRDialog.tsx'
import { Dialog } from './Dialog.tsx'
import { toast } from './Toast.tsx'
import { t } from '../locales/index.ts'

export interface HeaderProps {
  currentPath: string
  onNavigate: (path: string) => void
}

export function Header({ currentPath, onNavigate }: HeaderProps) {
  const [state, setState] = useState<AppState>(appState.get())
  const [qrOpen, setQrOpen] = useState(false)
  const [deviceInfoOpen, setDeviceInfoOpen] = useState(false)

  useEffect(() => {
    return appState.subscribe((s) => setState(s))
  }, [])

  const deviceName = state.info?.name || 'LANDrop'
  const isOnline = state.connected
  const addr = state.info?.addr || window.location.host
  const addressUrl = addr.startsWith('http://') || addr.startsWith('https://')
    ? addr
    : `${window.location.protocol}//${addr}`

  const handleCopyAddr = async () => {
    try {
      await navigator.clipboard.writeText(addressUrl)
      toast.success(t('header.copiedAddress'))
    } catch {
      toast.error(t('send.copyFailed'))
    }
  }

  return (
    <header className="app-header">
      <div className="header-inner">
        <div
          className="header-brand"
          style={{ cursor: 'pointer' }}
          onClick={() => onNavigate('/')}
        >
          <span className="header-brand-icon" style={{ color: 'var(--accent)' }}>
            <IconLogo size={20} />
          </span>
          <span>{t('app.title')}</span>
        </div>

        <nav className="header-nav">
          {/* Current Device Pill */}
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setDeviceInfoOpen(true)}
            icon={<IconDevice size={14} />}
            style={{ maxWidth: '46vw' }}
          >
            <span style={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>{deviceName}</span>
            <span className={`status-dot ${isOnline ? 'online' : 'offline'}`} style={{ marginLeft: '4px' }} />
          </Button>

          {/* Desktop Navigation Items */}
          <div className="desktop-nav-items">
            <Button
              variant={currentPath === '/activity' ? 'secondary' : 'ghost'}
              size="sm"
              icon={<IconHistory size={15} />}
              onClick={() => onNavigate('/activity')}
            >
              {t('nav.activity')}
            </Button>
            <Button
              variant={currentPath === '/settings' ? 'secondary' : 'ghost'}
              size="sm"
              icon={<IconSettings size={15} />}
              onClick={() => onNavigate('/settings')}
            >
              {t('nav.settings')}
            </Button>
          </div>
        </nav>
      </div>

      {/* Device Info Modal (Section 5.2) */}
      <Dialog
        open={deviceInfoOpen}
        onClose={() => setDeviceInfoOpen(false)}
        title={deviceName}
        maxWidth={360}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div
              style={{
                width: '36px',
                height: '36px',
                borderRadius: 'var(--radius-md)',
                background: 'var(--surface-subtle)',
                border: '1px solid var(--border)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--accent)',
              }}
            >
              <IconDevice size={20} />
            </div>
            <div>
              <div style={{ fontWeight: 600 }}>{deviceName}</div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                {state.info?.os || 'System'} · {addr}
              </div>
            </div>
          </div>

          <div
            style={{
              padding: '8px 12px',
              borderRadius: 'var(--radius-md)',
              background: isOnline ? 'var(--success-subtle)' : 'var(--danger-subtle)',
              color: isOnline ? 'var(--success)' : 'var(--danger)',
              fontSize: '12px',
              fontWeight: 500,
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <span className={`status-dot ${isOnline ? 'online' : 'offline'}`} />
            <span>{isOnline ? t('app.networkReady') : t('app.networkDisconnected')}</span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '6px' }}>
            <Button variant="secondary" onClick={handleCopyAddr} style={{ width: '100%', justifyContent: 'center' }}>
              {t('header.copyAddress')}
            </Button>
            <Button
              variant="secondary"
              icon={<IconQr size={15} />}
              onClick={() => {
                setDeviceInfoOpen(false)
                setQrOpen(true)
              }}
              style={{ width: '100%', justifyContent: 'center' }}
            >
              {t('header.qrConnect')}
            </Button>
          </div>
        </div>
      </Dialog>

      {/* QR Dialog */}
      <QRDialog
        open={qrOpen}
        onClose={() => setQrOpen(false)}
        deviceName={deviceName}
        address={addressUrl}
      />
    </header>
  )
}
