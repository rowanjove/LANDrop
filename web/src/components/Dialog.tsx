import { ComponentChildren } from 'preact'
import { useEffect, useRef } from 'preact/hooks'
import { IconButton } from './IconButton.tsx'
import { IconClose } from './Icons.tsx'
import { t } from '../locales/index.ts'

export interface DialogProps {
  open: boolean
  title: string
  onClose: () => void
  children: ComponentChildren
  footer?: ComponentChildren
  maxWidth?: number | string
}

export function Dialog({ open, title, onClose, children, footer, maxWidth = 460 }: DialogProps) {
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown)

    // Focus first focusable element or dialog card
    setTimeout(() => {
      if (cardRef.current) {
        const focusable = cardRef.current.querySelector<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
        if (focusable) {
          focusable.focus()
        } else {
          cardRef.current.focus()
        }
      }
    }, 50)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="dialog-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose()
        }
      }}
    >
      <div
        ref={cardRef}
        className="dialog-card"
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        style={{ maxWidth }}
      >
        <div className="dialog-header">
          <div className="dialog-title">{title}</div>
          <IconButton icon={<IconClose size={16} />} aria-label={t('accessibility.close')} onClick={onClose} size="sm" />
        </div>
        <div className="dialog-body">{children}</div>
        {footer && <div className="dialog-footer">{footer}</div>}
      </div>
    </div>
  )
}
