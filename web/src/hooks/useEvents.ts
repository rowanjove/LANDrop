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
          transfersState.refreshActiveTransfers()
          transfersState.refreshHistory(10)
        }

        eventSource.onerror = () => {
          appState.setConnected(false)
        }

        eventSource.addEventListener('file_ready', (e) => {
          try {
            const data = JSON.parse(e.data)
            transfersState.refreshActiveTransfers()
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
          transfersState.refreshActiveTransfers()
          transfersState.refreshHistory(10)
        })

        eventSource.addEventListener('transfer_cancelled', () => {
          transfersState.refreshActiveTransfers()
          transfersState.refreshHistory(10)
        })
      } catch {
        appState.setConnected(false)
      }
    }

    // Initial load
    appState.loadInfo()
    devicesState.refresh()
    transfersState.refreshActiveTransfers()
    transfersState.refreshHistory(10)
    connect()

    // Refresh immediately when returning to the tab or focusing window
    function handleActive() {
      if (typeof document !== 'undefined' && document.visibilityState === 'visible') {
        devicesState.refresh()
        transfersState.refreshActiveTransfers()
        transfersState.refreshHistory(10)
        appState.loadInfo()
        if (!eventSource || eventSource.readyState === EventSource.CLOSED) {
          connect()
        }
      }
    }

    document.addEventListener('visibilitychange', handleActive)
    window.addEventListener('focus', handleActive)

    return () => {
      isMounted = false
      document.removeEventListener('visibilitychange', handleActive)
      window.removeEventListener('focus', handleActive)
      if (eventSource) {
        eventSource.close()
      }
      if (fallbackInterval) {
        clearInterval(fallbackInterval)
      }
    }
  }, [])
}
