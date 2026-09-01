import { useEffect, useState } from 'preact/hooks'
import { Button } from './Button.tsx'
import { IconClipboard, IconRefresh, IconSend } from './Icons.tsx'
import { fetchClipboard, pushClipboard } from '../api/transfers.ts'
import { toast } from './Toast.tsx'
import { t } from '../locales/index.ts'

export function ClipboardPanel() {
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [pushing, setPushing] = useState(false)

  const loadServerClipboard = async () => {
    setLoading(true)
    try {
      const res = await fetchClipboard()
      setContent(res.content || '')
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  const readLocalClipboard = async () => {
    try {
      const text = await navigator.clipboard.readText()
      if (text) {
        setContent(text)
        toast.info(t('clipboard.readSuccess'))
      } else {
        toast.info(t('clipboard.empty'))
      }
    } catch {
      toast.error(t('clipboard.readError'))
    }
  }

  const handlePush = async () => {
    if (!content.trim() || pushing) return
    setPushing(true)
    try {
      await pushClipboard(content)
      toast.success(t('clipboard.syncSuccess'))
    } catch (err: any) {
      toast.error(err.message || t('clipboard.pushFailed'))
    } finally {
      setPushing(false)
    }
  }

  useEffect(() => {
    loadServerClipboard()
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ fontSize: '12px', fontWeight: 500, color: 'var(--text-secondary)' }}>
          {t('clipboard.title')}
        </div>
        <div style={{ display: 'flex', gap: '6px' }}>
          <Button variant="ghost" size="sm" icon={<IconClipboard size={14} />} onClick={readLocalClipboard}>
            {t('clipboard.readLocal')}
          </Button>
          <Button variant="ghost" size="sm" icon={<IconRefresh size={14} />} onClick={loadServerClipboard} loading={loading}>
            {t('clipboard.refresh')}
          </Button>
        </div>
      </div>

      <textarea
        className="textarea"
        placeholder={t('clipboard.empty')}
        value={content}
        onInput={(e) => setContent((e.target as HTMLTextAreaElement).value)}
        rows={4}
      />

      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          variant="primary"
          icon={<IconSend size={15} />}
          onClick={handlePush}
          disabled={!content.trim() || pushing}
          loading={pushing}
        >
          {t('clipboard.sendClipboard')}
        </Button>
      </div>
    </div>
  )
}
