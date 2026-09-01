import { useRef, useState } from 'preact/hooks'
import { IconUpload } from './Icons.tsx'
import { t } from '../locales/index.ts'

export interface FilePickerProps {
  onFilesSelected: (files: FileList | File[]) => void
}

export function FilePicker({ onFilesSelected }: FilePickerProps) {
  const [isDragOver, setIsDragOver] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleDragOver = (e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(true)
  }

  const handleDragLeave = (e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(false)
  }

  const handleDrop = (e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(false)
    if (e.dataTransfer && e.dataTransfer.files.length > 0) {
      onFilesSelected(e.dataTransfer.files)
    }
  }

  return (
    <div
      className={`dropzone ${isDragOver ? 'dragover' : ''}`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onClick={() => inputRef.current?.click()}
    >
      <input
        ref={inputRef}
        type="file"
        multiple
        style={{ display: 'none' }}
        onChange={(e) => {
          const files = (e.target as HTMLInputElement).files
          if (files && files.length > 0) {
            onFilesSelected(files)
            // reset input value so selecting the same file again triggers onChange
            ;(e.target as HTMLInputElement).value = ''
          }
        }}
      />
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
        <div style={{ color: 'var(--accent)', opacity: 0.85 }}>
          <IconUpload size={28} />
        </div>
        <div style={{ fontSize: '13px', fontWeight: 500 }}>{t('send.dropHint')}</div>
        <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
          {t('send.multiFileHint')}
        </div>
      </div>
    </div>
  )
}
