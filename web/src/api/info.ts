import { apiGet } from './client.ts'

export interface AppInfo {
  name: string
  version: string
  os: string
  addr: string
  one_time: boolean
  started_at?: number
}

export function fetchAppInfo(): Promise<AppInfo> {
  return apiGet<AppInfo>('/info')
}

export interface VersionCheckResult {
  current_version: string
  latest_version?: string
  has_update?: boolean
  release_url?: string
  release_notes?: string
  error?: string
}

export function checkVersion(): Promise<VersionCheckResult> {
  return apiGet<VersionCheckResult>('/api/v2/version/check')
}

