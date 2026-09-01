import { useEffect, useState } from 'preact/hooks'
import { IconCheck, IconClose } from './Icons.tsx'
import { t } from '../locales/index.ts'

export interface ToastMessage {
  id: string
  type: 'success' | 'error' | 'info'
  text: string
  duration?: number
}

type ToastListener = (toasts: ToastMessage[]) => void

let currentToasts: ToastMessage[] = []
const listeners = new Set<ToastListener>()

function notifyListeners() {
  listeners.forEach((fn) => fn([...currentToasts]))
}

export const toast = {
  show(text: string, type: 'success' | 'error' | 'info' = 'info', duration = 3000) {
    const id = Math.random().toString(36).slice(2, 9)
    const newToast: ToastMessage = { id, type, text, duration }
    currentToasts = [...currentToasts, newToast]
    notifyListeners()

    if (duration > 0) {
      setTimeout(() => {
        toast.dismiss(id)
      }, duration)
    }
  },
  success(text: string, duration = 3000) {
    toast.show(text, 'success', duration)
  },
  error(text: string, duration = 4000) {
    toast.show(text, 'error', duration)
  },
  info(text: string, duration = 3000) {
    toast.show(text, 'info', duration)
  },
  dismiss(id: string) {
    currentToasts = currentToasts.filter((t) => t.id !== id)
    notifyListeners()
  },
}

export function ToastContainer() {
  const [toasts, setToasts] = useState<ToastMessage[]>([])

  useEffect(() => {
    const handler: ToastListener = (updated) => setToasts(updated)
    listeners.add(handler)
    return () => {
      listeners.delete(handler)
    }
  }, [])

  if (toasts.length === 0) return null

  return (
    <div className="toast-container" aria-live="polite">
      {toasts.map((item) => (
        <div key={item.id} className={`toast toast-${item.type}`} role="alert">
          {item.type === 'success' && <IconCheck size={16} color="var(--success)" />}
          <span style={{ flex: 1 }}>{item.text}</span>
          <button
            style={{ padding: '2px', opacity: 0.6, cursor: 'pointer', display: 'flex' }}
            onClick={() => toast.dismiss(item.id)}
            aria-label={t('accessibility.close')}
          >
            <IconClose size={14} />
          </button>
        </div>
      ))}
    </div>
  )
}
