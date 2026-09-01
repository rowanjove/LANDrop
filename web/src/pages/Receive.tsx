import { useEffect, useState } from 'preact/hooks'
import { Button } from '../components/Button.tsx'
import { EmptyState } from '../components/EmptyState.tsx'
import { IconCheck, IconDownload, IconFile, IconCopy } from '../components/Icons.tsx'
import { fetchShareInfo, getDownloadUrl, getItemDownloadUrl, ShareData } from '../api/transfers.ts'
import { ApiException } from '../api/client.ts'
import { formatSize } from '../utils/format.ts'
import { toast } from '../components/Toast.tsx'
import { t } from '../locales/index.ts'

export interface ReceiveProps {
  token: string
}

export function Receive({ token }: ReceiveProps) {
  const [data, setData] = useState<ShareData | null>(null)
  const [error, setError] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let cancelled = false
    setData(null)
    setError(false)
    fetchShareInfo(token)
      .then((result) => {
        if (!cancelled) setData(result)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        setError(true)
        if (!(err instanceof ApiException && (err.status === 404 || err.status === 410))) {
          toast.error(t('receive.loadError'))
        }
      })
    return () => {
      cancelled = true
    }
  }, [token])

  const handleCopy = async () => {
    if (!data?.content) return
    try {
      await navigator.clipboard.writeText(data.content)
      setCopied(true)
      toast.success(t('receive.copied'))
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('receive.loadError'))
    }
  }

  if (error) {
    return <EmptyState title={t('receive.expired')} description={t('receive.loadError')} />
  }

  if (!data) {
    return <EmptyState title={t('receive.title')} description="…" />
  }

  const isText = data.type === 'text'
  const name = data.name || t('receive.textReady')
  const items = data.items || []
  return (
    <div style={{ maxWidth: '640px', margin: '0 auto' }}>
      <section className="panel-section">
        <div className="panel-header">
          <div className="panel-title">
            {isText ? <IconCopy size={17} /> : <IconFile size={17} />}
            {t('receive.title')}
          </div>
          {data.one_time && <span className="status-badge success">{t('receive.fileReady')}</span>}
        </div>
        <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div style={{ color: 'var(--success)' }}><IconCheck size={22} /></div>
            <div style={{ minWidth: 0 }}>
              <div style={{ fontSize: '15px', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{name}</div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{t(isText ? 'receive.textReady' : 'receive.fileReady')} · {t('receive.size')}: {formatSize(data.size)}</div>
            </div>
          </div>

          {isText ? (
            <>
              <pre style={{ margin: 0, padding: '14px', borderRadius: 'var(--radius-md)', background: 'var(--surface-subtle)', border: '1px solid var(--border)', whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', maxHeight: '50vh', overflow: 'auto', fontFamily: 'inherit', fontSize: '13px', lineHeight: 1.6 }}>{data.content}</pre>
              <Button variant="primary" icon={copied ? <IconCheck size={15} /> : <IconCopy size={15} />} onClick={handleCopy} style={{ width: '100%' }}>
                {copied ? t('receive.copied') : t('receive.copy')}
              </Button>
            </>
          ) : items.length > 0 ? (
            <>
              <div style={{ display: 'flex', flexDirection: 'column', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }}>
                {items.map((item) => (
                  <div key={item.id} className="item-row" style={{ gap: '10px' }}>
                    <span style={{ minWidth: 0, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.name}</span>
                    <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>{formatSize(item.size)}</span>
                    <a className="btn btn-secondary btn-sm" href={getItemDownloadUrl(token, item.id)} style={{ textDecoration: 'none' }}>{t('receive.download')}</a>
                  </div>
                ))}
              </div>
              <a className="btn btn-primary" href={getDownloadUrl(token)} style={{ width: '100%', boxSizing: 'border-box', textDecoration: 'none' }}>
                <span className="btn-icon-wrapper"><IconDownload size={15} /></span>
                <span>{t('receive.download')} ({items.length})</span>
              </a>
            </>
          ) : (
            <a className="btn btn-primary" href={getDownloadUrl(token)} style={{ width: '100%', boxSizing: 'border-box', textDecoration: 'none' }}>
              <span className="btn-icon-wrapper"><IconDownload size={15} /></span>
              <span>{t('receive.download')}</span>
            </a>
          )}
        </div>
      </section>
    </div>
  )
}
