import { useState } from 'preact/hooks'
import { Dialog } from './Dialog.tsx'
import { Button } from './Button.tsx'
import { Input } from './Input.tsx'
import { t } from '../locales/index.ts'

export interface ManualConnectDialogProps {
  open: boolean
  onClose: () => void
}

export function ManualConnectDialog({ open, onClose }: ManualConnectDialogProps) {
  const [addr, setAddr] = useState('')
  const [error, setError] = useState('')

  const handleConnect = () => {
    let clean = addr.trim()
    if (!clean) {
      setError(t('dialogs.invalidAddress'))
      return
    }

    const lower = clean.toLowerCase()
    if (lower.startsWith('javascript:') || lower.startsWith('data:') || lower.startsWith('file:')) {
      setError(t('dialogs.invalidAddress'))
      return
    }

    if (!clean.startsWith('http://') && !clean.startsWith('https://')) {
      const fallbackScheme = window.location.protocol.replace(':', '') || 'http'
      clean = `${fallbackScheme}://${clean}`
    }

    try {
      const url = new URL(clean)
      if (!url.hostname) {
        setError(t('dialogs.invalidAddress'))
        return
      }
      // Navigate in same tab as specified by Section 10.4
      window.location.href = url.href
    } catch {
      setError(t('dialogs.invalidAddress'))
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('dialogs.manualConnectTitle')}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            {t('dialogs.cancel')}
          </Button>
          <Button variant="primary" onClick={handleConnect}>
            {t('dialogs.connect')}
          </Button>
        </>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <Input
          placeholder="192.168.1.32:53217"
          value={addr}
          onInput={(e) => {
            setAddr((e.target as HTMLInputElement).value)
            setError('')
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleConnect()
          }}
          error={error}
          hint={t('dialogs.enterAddress')}
          autoFocus
        />
      </div>
    </Dialog>
  )
}
