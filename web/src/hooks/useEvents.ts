import { useEffect } from 'preact/hooks'
import { appState } from '../state/app.ts'
import { devicesState } from '../state/devices.ts'
import { transfersState } from '../state/transfers.ts'
import { toast } from '../components/Toast.tsx'
import { t } from '../locales/index.ts'

export function useEvents() {
  useEffect(() => {
    let eventSource: EventSource | null = null
    let fallbackInterval: any = null
    let isMounted = true

    function connect() {
      if (!isMounted) return

      try {
        eventSource = new EventSource('/events')

        eventSource.onopen = () => {
          appState.setConnected(true)
          // Refresh devices and history on reconnection
          devicesState.refresh()
          transfersState.refreshHistory(10)
        }

        eventSource.onerror = () => {
          appState.setConnected(false)
        }

        eventSource.addEventListener('file_ready', (e) => {
          try {
            const data = JSON.parse(e.data)
            // If someone else pushed file_ready or local ready
            transfersState.refreshHistory(10)
            if (data.type === 'file') {
              toast.info(t('events.fileNotice', { name: data.name || t('events.unnamedFile') }))
            } else if (data.type === 'text') {
              toast.info(t('events.textNotice'))
            }
          } catch {
            // ignore
          }
        })

        eventSource.addEventListener('device_found', (e) => {
          try {
            const device = JSON.parse(e.data)
            devicesState.upsertDevice(device)
          } catch {
            // ignore
          }
        })

        eventSource.addEventListener('device_lost', (e) => {
          try {
            const data = JSON.parse(e.data)
            if (data.addr) {
              devicesState.markOffline(data.addr)
            }
          } catch {
            // ignore
          }
        })

        eventSource.addEventListener('clipboard', () => {
          toast.info(t('clipboard.syncSuccess'))
        })

        eventSource.addEventListener('settings_updated', (e) => {
          try {
            const data = JSON.parse(e.data)
            if (data.device_name) {
              appState.loadInfo()
            }
          } catch {
            // ignore
          }
        })

        eventSource.addEventListener('done', () => {
          transfersState.refreshHistory(10)
        })
      } catch {
        appState.setConnected(false)
      }
    }

    // Initial load
    appState.loadInfo()
    devicesState.refresh()
    transfersState.refreshHistory(10)
    connect()

    // 30s low-frequency fallback polling
    fallbackInterval = setInterval(() => {
      devicesState.refresh()
    }, 30000)

    return () => {
      isMounted = false
      if (eventSource) {
        eventSource.close()
      }
      if (fallbackInterval) {
        clearInterval(fallbackInterval)
      }
    }
  }, [])
}
