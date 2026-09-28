import { useEffect, useRef, useState } from 'preact/hooks'
import { FilePicker } from './FilePicker.tsx'
import { FileQueue } from './FileQueue.tsx'
import { IconFile, IconPlus } from './Icons.tsx'
import { Button } from './Button.tsx'
import { transfersState, TransfersState } from '../state/transfers.ts'
import { t } from '../locales/index.ts'

export function FileSendCard() {
  const [transfers, setTransfers] = useState<TransfersState>(transfersState.get())
  const hiddenInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    return transfersState.subscribe((s) => setTransfers(s))
  }, [])

  const handleFilesAdded = (files: FileList | File[]) => {
    transfersState.addFiles(files)
  }

  return (
    <section className="panel-section file-send-card">
      <div className="panel-header">
        <div className="panel-title">
          <IconFile size={16} />
          <span>{t('send.filesTitle')}</span>
          {transfers.pendingFiles.length > 0 && (
            <span
              className="status-badge"
              style={{ background: 'var(--surface-subtle)', color: 'var(--accent)' }}
            >
              {transfers.pendingFiles.length}
            </span>
          )}
        </div>

        {transfers.pendingFiles.length > 0 && (
          <div className="desktop-only-action" style={{ display: 'flex', gap: '6px' }}>
            <Button
              variant="ghost"
              size="sm"
              icon={<IconPlus size={14} />}
              onClick={() => hiddenInputRef.current?.click()}
            >
              {t('send.addMoreFiles')}
            </Button>
          </div>
        )}
      </div>

      <div className="panel-body">
        {/* Hidden input for adding more files in queue */}
        <input
          ref={hiddenInputRef}
          type="file"
          multiple
          style={{ display: 'none' }}
          onChange={(e) => {
            const files = (e.target as HTMLInputElement).files
            if (files && files.length > 0) {
              handleFilesAdded(files)
              ;(e.target as HTMLInputElement).value = ''
            }
          }}
        />

        {transfers.pendingFiles.length === 0 ? (
          <FilePicker onFilesSelected={handleFilesAdded} />
        ) : (
          <FileQueue
            pendingFiles={transfers.pendingFiles}
            onAddMore={() => hiddenInputRef.current?.click()}
          />
        )}
      </div>
    </section>
  )
}
