import { apiGet } from './client.ts'

export interface AppInfo {
  name: string
  version: string
  os: string
  addr: string
  one_time: boolean
}

export function fetchAppInfo(): Promise<AppInfo> {
  return apiGet<AppInfo>('/info')
}
