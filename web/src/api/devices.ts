import { apiGet } from './client.ts'

export interface DeviceItem {
  name: string
  os: string
  addr: string
  scheme?: 'http' | 'https'
  online: boolean
  last_seen?: number
}

interface DevicesResponse {
  devices: DeviceItem[]
}

export async function fetchDevices(): Promise<DeviceItem[]> {
  const data = await apiGet<DevicesResponse>('/devices')
  return data.devices || []
}
