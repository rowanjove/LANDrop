import { useState } from 'preact/hooks'
import { Dialog } from './Dialog.tsx'
import { Button } from './Button.tsx'
import { IconCheck, IconCopy } from './Icons.tsx'
import { getQRUrl } from '../api/transfers.ts'
import { toast } from './Toast.tsx'
import { t } from '../locales/index.ts'

export interface QRDialogProps {
  open: boolean
  onClose: () => void
  deviceName?: string
  address?: string
}

export function QRDialog({ open, onClose, deviceName, address }: QRDialogProps) {
  const [copied, setCopied] = useState(false)
  const fullAddress = address ? (address.startsWith('http') ? address : `http://${address}`) : window.location.origin

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(fullAddress)
      setCopied(true)
      toast.success(t('header.copiedAddress'))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('clipboard.pushFailed'))
    }
  }

  return (
    <Dialog open={open} onClose={onClose} title={t('dialogs.qrTitle')} maxWidth={380}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: '14px' }}>
        <div style={{ fontSize: '14px', fontWeight: 600 }}>{deviceName || 'LANDrop'}</div>
        <div
          style={{
            padding: '12px',
            background: '#ffffff',
            borderRadius: 'var(--radius-md)',
            border: '1px solid var(--border)',
            display: 'inline-block',
          }}
        >
          <img
            src={getQRUrl(220, fullAddress)}
            alt="Connection QR"
            width={220}
            height={220}
            style={{ display: 'block', imageRendering: 'pixelated' }}
          />
        </div>
        <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{t('dialogs.qrScanHint')}</div>
        <div
          className="font-mono"
          style={{
            fontSize: '12px',
            background: 'var(--surface-subtle)',
            padding: '6px 12px',
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--border)',
            wordBreak: 'break-all',
            maxWidth: '100%',
          }}
        >
          {fullAddress}
        </div>
        <Button
          variant="secondary"
          icon={copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
          onClick={handleCopy}
          style={{ width: '100%' }}
        >
          {copied ? t('header.copiedAddress') : t('header.copyAddress')}
        </Button>
      </div>
    </Dialog>
  )
}
