import { apiGet, apiPost } from './client.ts'

export interface SettingsData {
  device_name?: string
  default_expiry?: number
  theme?: string
  language?: string
}

export function fetchSettings(): Promise<SettingsData> {
  return apiGet<SettingsData>('/settings')
}

export function updateSettings(data: SettingsData): Promise<SettingsData> {
  return apiPost<SettingsData>('/settings', data)
}
