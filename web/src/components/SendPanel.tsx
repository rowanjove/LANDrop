import { useEffect, useRef, useState } from 'preact/hooks'
import { Tabs } from './Tabs.tsx'
import { FilePicker } from './FilePicker.tsx'
import { FileQueue } from './FileQueue.tsx'
import { TextEditor } from './TextEditor.tsx'
import { ClipboardPanel } from './ClipboardPanel.tsx'
import { IconClipboard, IconFile, IconText } from './Icons.tsx'
import { transfersState, TransfersState } from '../state/transfers.ts'
import { t } from '../locales/index.ts'

export type SendTab = 'file' | 'text' | 'clipboard'

export function SendPanel() {
  const [activeTab, setActiveTab] = useState<SendTab>('file')
  const [transfers, setTransfers] = useState<TransfersState>(transfersState.get())
  const hiddenInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    return transfersState.subscribe((s) => setTransfers(s))
  }, [])

  const tabs = [
    { id: 'file' as SendTab, label: t('send.tabFile'), icon: <IconFile size={15} />, badge: transfers.pendingFiles.length || undefined },
    { id: 'text' as SendTab, label: t('send.tabText'), icon: <IconText size={15} /> },
    { id: 'clipboard' as SendTab, label: t('send.tabClipboard'), icon: <IconClipboard size={15} /> },
  ]

  const handleFilesAdded = (files: FileList | File[]) => {
    transfersState.addFiles(files)
  }

  return (
    <section className="panel-section">
      <Tabs tabs={tabs} activeTab={activeTab} onChange={(id) => setActiveTab(id)} />

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

        {activeTab === 'file' && (
          transfers.pendingFiles.length === 0 ? (
            <FilePicker onFilesSelected={handleFilesAdded} />
          ) : (
            <FileQueue
              pendingFiles={transfers.pendingFiles}
              onAddMore={() => hiddenInputRef.current?.click()}
            />
          )
        )}

        {activeTab === 'text' && <TextEditor />}

        {activeTab === 'clipboard' && <ClipboardPanel />}
      </div>
    </section>
  )
}
