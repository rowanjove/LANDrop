import { DeviceItem, fetchDevices } from '../api/devices.ts'

export interface DevicesState {
  devices: DeviceItem[]
  loading: boolean
  lastSync: number
}

let state: DevicesState = {
  devices: [],
  loading: false,
  lastSync: 0,
}

type Listener = (s: DevicesState) => void
const listeners = new Set<Listener>()

function sortDevices(list: DeviceItem[]): DeviceItem[] {
  return [...list].sort((a, b) => {
    if (a.online !== b.online) {
      return a.online ? -1 : 1
    }
    const aTime = a.last_seen || 0
    const bTime = b.last_seen || 0
    if (aTime !== bTime) {
      return bTime - aTime
    }
    return a.name.localeCompare(b.name)
  })
}

function notify() {
  listeners.forEach((fn) => fn({ ...state }))
}

export const devicesState = {
  get(): DevicesState {
    return { ...state }
  },
  subscribe(fn: Listener): () => void {
    listeners.add(fn)
    fn({ ...state })
    return () => listeners.delete(fn)
  },
  setDevices(list: DeviceItem[]) {
    state.devices = sortDevices(list)
    state.lastSync = Date.now()
    notify()
  },
  upsertDevice(device: DeviceItem) {
    const existing = state.devices.findIndex((d) => d.addr === device.addr || (d.name === device.name && d.os === device.os))
    if (existing >= 0) {
      state.devices[existing] = { ...state.devices[existing], ...device }
    } else {
      state.devices.push(device)
    }
    state.devices = sortDevices(state.devices)
    state.lastSync = Date.now()
    notify()
  },
  markOffline(addr: string) {
    state.devices = state.devices.map((d) => {
      if (d.addr === addr) {
        return { ...d, online: false, last_seen: Math.floor(Date.now() / 1000) }
      }
      return d
    })
    state.devices = sortDevices(state.devices)
    notify()
  },
  async refresh() {
    state.loading = true
    notify()
    try {
      const list = await fetchDevices()
      state.devices = sortDevices(list)
      state.lastSync = Date.now()
    } catch {
      // ignore
    } finally {
      state.loading = false
      notify()
    }
  },
}
