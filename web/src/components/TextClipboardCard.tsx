import { useEffect, useRef, useState } from 'preact/hooks'
import { Button } from './Button.tsx'
import { IconButton } from './IconButton.tsx'
import { IconCheck, IconCopy, IconQr, IconRefresh, IconSend, IconText } from './Icons.tsx'
import { QRDialog } from './QRDialog.tsx'
import { formatSize } from '../utils/format.ts'
import { copyText } from '../utils/clipboard.ts'
import { fetchClipboard, getShareUrl, pushClipboard, sendText } from '../api/transfers.ts'
import { transfersState } from '../state/transfers.ts'
import { toast } from './Toast.tsx'
import { t } from '../locales/index.ts'

const isShowcase = typeof window !== 'undefined' && window.location.search.includes('showcase=true')

export function TextClipboardCard() {
  const [content, setContent] = useState(isShowcase ? 'https://github.com/rowanjove/LANDrop' : '')
  const [loadingServer, setLoadingServer] = useState(false)
  const [sendingText, setSendingText] = useState(false)
  const [readyToken, setReadyToken] = useState<string | null>(null)
  const [qrOpen, setQrOpen] = useState(false)
  const [copied, setCopied] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const byteSize = new Blob([content]).size
  const isUrl = /^(https?:\/\/[^\s]+)$/i.test(content.trim())

  const loadServerClipboard = async () => {
    setLoadingServer(true)
    try {
      const res = await fetchClipboard()
      if (res.content) {
        setContent(res.content)
        toast.info(t('clipboard.readSuccess'))
      } else {
        toast.info(t('clipboard.empty'))
      }
    } catch {
      // ignore
    } finally {
      setLoadingServer(false)
    }
  }

  const handleSendText = async () => {
    if (!content.trim() || sendingText) return
    setSendingText(true)
    try {
      const res = await sendText(content)
      setReadyToken(res.token)
      // In parallel, sync to LAN clipboard so both channels are updated seamlessly
      pushClipboard(content).catch(() => {})
      transfersState.refreshHistory(10)
      toast.success(t('send.uploadSuccess'))
    } catch (err: any) {
      toast.error(err.message || t('send.uploadFailed'))
    } finally {
      setSendingText(false)
    }
  }

  const handleCopyLink = async () => {
    if (!readyToken) return
    const url = getShareUrl(readyToken)
    const ok = await copyText(url)
    if (ok) {
      setCopied(true)
      toast.success(t('send.copiedLink'))
      setTimeout(() => setCopied(false), 2000)
    } else {
      toast.error(t('send.copyFailed'))
    }
  }

  useEffect(() => {
    if (isShowcase) return
    // Optionally load server clipboard initially if available
    fetchClipboard().then((res) => {
      if (res.content && !content) {
        setContent(res.content)
      }
    }).catch(() => {})
  }, [])

  return (
    <section className="panel-section text-clipboard-card">
      <div className="panel-header">
        <div className="panel-title">
          <IconText size={16} />
          <span>{t('send.textAndClipboard')}</span>
        </div>
        <div style={{ display: 'flex', gap: '6px' }}>
          <Button
            variant="ghost"
            size="sm"
            icon={<IconRefresh size={14} />}
            onClick={loadServerClipboard}
            loading={loadingServer}
            title={t('clipboard.refresh')}
          >
            {t('clipboard.refresh')}
          </Button>
        </div>
      </div>

      <div className="panel-body">
        {readyToken ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            <div
              style={{
                padding: '12px 14px',
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
                setContent('')
              }}
              style={{ alignSelf: 'flex-start' }}
            >
              {t('send.sendAnotherFile')}
            </Button>

            <QRDialog
              open={qrOpen}
              onClose={() => setQrOpen(false)}
              deviceName={t('transfer.textMessage')}
              address={getShareUrl(readyToken)}
            />
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', height: '100%' }}>
            <textarea
              ref={textareaRef}
              className="textarea"
              placeholder={t('text.placeholder')}
              value={content}
              onInput={(e) => setContent((e.target as HTMLTextAreaElement).value)}
              rows={3}
              style={{ minHeight: '74px', resize: 'vertical' }}
            />

            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                flexWrap: 'wrap',
                gap: '8px',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
                  {formatSize(byteSize)}
                </span>
                {isUrl && (
                  <span
                    className="status-badge"
                    style={{ background: 'var(--accent-subtle)', color: 'var(--accent)', fontSize: '11px' }}
                  >
                    {t('text.urlDetected')}
                  </span>
                )}
              </div>

              <div>
                <Button
                  variant="primary"
                  size="sm"
                  icon={<IconSend size={14} />}
                  onClick={handleSendText}
                  disabled={!content.trim() || sendingText}
                  loading={sendingText}
                >
                  {t('text.sendText')}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}
