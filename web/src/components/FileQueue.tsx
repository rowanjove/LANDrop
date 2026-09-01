import { useState, useRef } from 'preact/hooks'
import { Button } from './Button.tsx'
import { IconButton } from './IconButton.tsx'
import { ProgressBar } from './ProgressBar.tsx'
import { IconCheck, IconClose, IconCopy, IconFile, IconPlus, IconQr, IconSend } from './Icons.tsx'
import { QRDialog } from './QRDialog.tsx'
import { formatSize, formatSpeed, formatEta } from '../utils/format.ts'
import { sendFiles, getShareUrl, UploadTask } from '../api/transfers.ts'
import { transfersState, PendingFile } from '../state/transfers.ts'
import { toast } from './Toast.tsx'
import { t } from '../locales/index.ts'

export interface FileQueueProps {
  pendingFiles: PendingFile[]
  onAddMore: () => void
}

export function FileQueue({ pendingFiles, onAddMore }: FileQueueProps) {
  const [uploading, setUploading] = useState(false)
  const [uploadFailed, setUploadFailed] = useState(false)
  const [progress, setProgress] = useState(0)
  const [loadedBytes, setLoadedBytes] = useState(0)
  const [totalBytes, setTotalBytes] = useState(0)
  const [speed, setSpeed] = useState(0)
  const [eta, setEta] = useState(0)
  const [readyToken, setReadyToken] = useState<string | null>(null)
  const [readyName, setReadyName] = useState('')
  const [readySize, setReadySize] = useState(0)
  const [qrOpen, setQrOpen] = useState(false)
  const [copied, setCopied] = useState(false)

  const lastProgressRef = useRef<{ time: number; loaded: number }>({ time: 0, loaded: 0 })
  const uploadRef = useRef<UploadTask | null>(null)

  const totalQueueSize = pendingFiles.reduce((acc, f) => acc + f.size, 0)

  const handleSend = async () => {
    if (pendingFiles.length === 0 || uploading) return

    setUploading(true)
    setUploadFailed(false)
    setProgress(0)
    setLoadedBytes(0)
    setTotalBytes(totalQueueSize)
    setSpeed(0)
    setEta(0)
    lastProgressRef.current = { time: Date.now(), loaded: 0 }

    try {
      const files = pendingFiles.map((pf) => pf.file)
      const upload = sendFiles(files, (pct, loaded, total) => {
        setProgress(pct)
        setLoadedBytes(loaded)
        setTotalBytes(total)

        const now = Date.now()
        const elapsed = (now - lastProgressRef.current.time) / 1000
        if (elapsed >= 0.3) {
          const deltaBytes = loaded - lastProgressRef.current.loaded
          const curSpeed = deltaBytes / elapsed
          setSpeed(curSpeed)
          const remaining = total - loaded
          setEta(curSpeed > 0 ? remaining / curSpeed : 0)
          lastProgressRef.current = { time: now, loaded }
        }
      })
      uploadRef.current = upload
      const res = await upload

      setReadyToken(res.token)
      setReadyName(res.name)
      setReadySize(res.size)
      setUploadFailed(false)
      transfersState.clearPendingFiles()
      transfersState.refreshHistory(10)
      toast.success(t('send.uploadSuccess'))
    } catch (err: any) {
      if (err?.message !== 'Upload aborted') {
        setUploadFailed(true)
        toast.error(err.message || t('send.uploadFailed'))
      }
    } finally {
      uploadRef.current = null
      setUploading(false)
    }
  }

  const handleCancel = () => {
    uploadRef.current?.abort()
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

  // Ready State View
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
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
            <div style={{ color: 'var(--success)' }}>
              <IconCheck size={20} />
            </div>
            <div style={{ minWidth: 0 }}>
              <div style={{ fontSize: '13px', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {readyName}
              </div>
              <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
                {formatSize(readySize)} · {t('send.fileReady')}
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
          }}
          style={{ alignSelf: 'flex-start' }}
        >
          {t('send.sendAnotherFile')}
        </Button>

        <QRDialog
          open={qrOpen}
          onClose={() => setQrOpen(false)}
          deviceName={readyName}
          address={shareUrl}
        />
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {/* Summary Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '12px', color: 'var(--text-secondary)' }}>
        <span>{t('send.filesSummary', { count: pendingFiles.length, size: formatSize(totalQueueSize) })}</span>
        {!uploading && (
          <button
            onClick={() => transfersState.clearPendingFiles()}
            style={{ color: 'var(--danger)', fontSize: '12px', cursor: 'pointer' }}
          >
            {t('send.clearQueue')}
          </button>
        )}
      </div>

      {/* File List */}
      <div
        style={{
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-md)',
          maxHeight: '200px',
          overflowY: 'auto',
          background: 'var(--surface)',
        }}
      >
        {pendingFiles.map((item) => (
          <div
            key={item.id}
            className="item-row"
            style={{ padding: '8px 12px', gap: '10px' }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: 1 }}>
              <div style={{ color: 'var(--text-secondary)', flexShrink: 0 }}>
                <IconFile size={16} />
              </div>
              <span
                style={{
                  fontSize: '13px',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {item.name}
              </span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexShrink: 0 }}>
              <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{formatSize(item.size)}</span>
              {!uploading && (
                <IconButton
                  icon={<IconClose size={14} />}
                  aria-label={t('accessibility.deleteFile')}
                  onClick={() => transfersState.removeFile(item.id)}
                  size="sm"
                  variant="ghost"
                />
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Uploading Progress */}
      {uploading && (
        <div style={{ padding: '8px 0' }}>
          <ProgressBar
            percent={progress}
            transferredText={`${formatSize(loadedBytes)} / ${formatSize(totalBytes)} (${progress}%)`}
            speedText={speed > 0 ? formatSpeed(speed) : undefined}
            etaText={eta > 0 ? formatEta(eta) : undefined}
          />
        </div>
      )}

      {/* Actions */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '4px', gap: '8px' }}>
        <Button
          variant="secondary"
          size="sm"
          icon={<IconPlus size={14} />}
          onClick={onAddMore}
          disabled={uploading}
        >
          {t('send.addMoreFiles')}
        </Button>
        {uploading ? (
          <Button variant="danger" onClick={handleCancel}>
            {t('dialogs.cancel')}
          </Button>
        ) : (
          <Button
            variant="primary"
            icon={<IconSend size={15} />}
            onClick={handleSend}
            disabled={pendingFiles.length === 0}
          >
            {uploadFailed ? t('send.retryButton') : t('send.sendButton')}
          </Button>
        )}
      </div>
    </div>
  )
}
