import { useState } from 'preact/hooks'
import { Button } from './Button.tsx'
import { IconButton } from './IconButton.tsx'
import { IconCheck, IconClipboard, IconCopy, IconQr, IconSend } from './Icons.tsx'
import { QRDialog } from './QRDialog.tsx'
import { formatSize } from '../utils/format.ts'
import { sendText, getShareUrl } from '../api/transfers.ts'
import { transfersState } from '../state/transfers.ts'
import { toast } from './Toast.tsx'
import { t } from '../locales/index.ts'

export function TextEditor() {
  const [text, setText] = useState('')
  const [sending, setSending] = useState(false)
  const [readyToken, setReadyToken] = useState<string | null>(null)
  const [qrOpen, setQrOpen] = useState(false)
  const [copied, setCopied] = useState(false)

  const byteSize = new Blob([text]).size
  const isUrl = /^(https?:\/\/[^\s]+)$/i.test(text.trim())

  const handlePaste = async () => {
    try {
      const clipText = await navigator.clipboard.readText()
      if (clipText) {
        setText(clipText)
      }
    } catch {
      toast.error(t('clipboard.pasteFailed'))
    }
  }

  const handleSend = async () => {
    if (!text.trim() || sending) return

    setSending(true)
    try {
      const res = await sendText(text)
      setReadyToken(res.token)
      transfersState.refreshHistory(10)
      toast.success(t('send.uploadSuccess'))
    } catch (err: any) {
      toast.error(err.message || t('send.uploadFailed'))
    } finally {
      setSending(false)
    }
  }

  const handleCopyLink = async () => {
    if (!readyToken) return
    const url = getShareUrl(readyToken)
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      toast.success(t('send.copiedLink'))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('send.copyFailed'))
    }
  }

  if (readyToken) {
    const shareUrl = getShareUrl(readyToken)
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
        <div
          style={{
            padding: '14px',
            background: 'var(--surface-subtle)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div style={{ color: 'var(--success)' }}>
              <IconCheck size={20} />
            </div>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 600 }}>{t('receive.textReady')}</div>
              <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
                {formatSize(byteSize)} · {t('receive.textReady')}
              </div>
            </div>
          </div>
          <div style={{ display: 'flex', gap: '6px' }}>
            <Button
              variant="secondary"
              size="sm"
              icon={copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
              onClick={handleCopyLink}
            >
              {copied ? t('send.copiedLink') : t('send.copyLink')}
            </Button>
            <IconButton
              icon={<IconQr size={16} />}
              aria-label={t('send.qrCode')}
              onClick={() => setQrOpen(true)}
              size="sm"
            />
          </div>
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setReadyToken(null)
            setText('')
          }}
          style={{ alignSelf: 'flex-start' }}
        >
          {t('send.sendAnotherFile')}
        </Button>

        <QRDialog
          open={qrOpen}
          onClose={() => setQrOpen(false)}
          deviceName={t('transfer.textMessage')}
          address={shareUrl}
        />
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
      <textarea
        className="textarea"
        placeholder={t('text.placeholder')}
        value={text}
        onInput={(e) => setText((e.target as HTMLTextAreaElement).value)}
        rows={5}
      />

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{formatSize(byteSize)}</span>
          {isUrl && (
            <span
              className="status-badge"
              style={{ background: 'var(--accent-subtle)', color: 'var(--accent)', fontSize: '11px' }}
            >
              {t('text.urlDetected')}
            </span>
          )}
        </div>
        <Button variant="ghost" size="sm" icon={<IconClipboard size={14} />} onClick={handlePaste}>
          {t('text.pasteFromClipboard')}
        </Button>
      </div>

      <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '4px' }}>
        <Button
          variant="primary"
          icon={<IconSend size={15} />}
          onClick={handleSend}
          disabled={!text.trim() || sending}
          loading={sending}
        >
          {t('text.sendText')}
        </Button>
      </div>
    </div>
  )
}
