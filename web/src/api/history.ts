import { apiDelete, apiGet } from './client.ts'

export interface HistoryRecord {
  timestamp: number
  direction: 'send' | 'recv'
  name: string
  size: number
  type: 'file' | 'text'
  status: 'success' | 'failed' | 'interrupted'
  peer: string
}

interface HistoryResponse {
  history: HistoryRecord[]
}

interface ClearHistoryResponse {
  cleared: number
}

export interface HistoryQueryParams {
  limit?: number
  peer?: string
  date?: string
}

export async function fetchHistory(params?: HistoryQueryParams): Promise<HistoryRecord[]> {
  const data = await apiGet<HistoryResponse>('/history', params as Record<string, string | number | undefined>)
  return data.history || []
}

export async function clearHistory(): Promise<number> {
  const data = await apiDelete<ClearHistoryResponse>('/history')
  return data.cleared || 0
}
