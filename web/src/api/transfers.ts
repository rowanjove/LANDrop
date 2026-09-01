import { apiGet, apiPost } from './client.ts'

export interface UploadResult {
  token: string
  name: string
  size: number
  expires_at?: number
}

export interface TextSendResult {
  token: string
  length: number
}

export interface PreviewData {
  type: 'file' | 'text'
  name?: string
  size: number
  preview?: string
  one_time?: boolean
}

export interface ShareData {
  type: 'file' | 'text'
  name?: string
  size: number
  content?: string
  expires_at?: number
  one_time?: boolean
  status?: string
  items?: ShareItem[]
}

export interface ShareItem {
  id: string
  name: string
  size: number
  mime_type?: string
}

export type UploadTask = Promise<UploadResult> & { abort: () => void }

export function sendFiles(
  files: File[],
  onProgress?: (percent: number, loaded: number, total: number) => void
): UploadTask {
  let xhr: XMLHttpRequest | null = null
  const promise = new Promise<UploadResult>((resolve, reject) => {
    const request = new XMLHttpRequest()
    xhr = request
    request.open('POST', '/send/file')

    if (request.upload && onProgress) {
      request.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          const percent = Math.round((e.loaded / e.total) * 100)
          onProgress(percent, e.loaded, e.total)
        }
      }
    }

    request.onload = () => {
      if (request.status >= 200 && request.status < 300) {
        try {
          const res = JSON.parse(request.responseText)
          resolve(res)
        } catch {
          reject(new Error('Invalid server response'))
        }
      } else {
        let msg = `Upload failed (${request.status})`
        try {
          const err = JSON.parse(request.responseText)
          if (err.error) msg = err.error
        } catch {
          // ignore
        }
        reject(new Error(msg))
      }
    }

    request.onerror = () => reject(new Error('Network error during upload'))
    request.onabort = () => reject(new Error('Upload aborted'))

    const formData = new FormData()
    for (const f of files) {
      formData.append('file', f, f.name)
    }

    request.send(formData)
  })
  const task = promise as UploadTask
  task.abort = () => xhr?.abort()
  return task
}

export function sendText(content: string): Promise<TextSendResult> {
  return apiPost<TextSendResult>('/send/text', { content })
}

export function fetchPreview(token: string): Promise<PreviewData> {
  return apiGet<PreviewData>(`/preview/${token}`)
}

export function fetchShareInfo(token: string): Promise<ShareData> {
  return apiGet<ShareData>(`/api/v2/share/${encodeURIComponent(token)}`)
}

export function fetchClipboard(): Promise<{ content: string }> {
  return apiGet<{ content: string }>('/clipboard')
}

export function pushClipboard(content: string): Promise<{ pushed_to: number }> {
  return apiPost<{ pushed_to: number }>('/clipboard/push', { content })
}

export function getDownloadUrl(token: string): string {
  return `${window.location.origin}/recv/${encodeURIComponent(token)}`
}

export function getItemDownloadUrl(token: string, itemID: string): string {
  return `${window.location.origin}/api/v2/share/${encodeURIComponent(token)}/items/${encodeURIComponent(itemID)}`
}

export function getShareUrl(token: string): string {
  return `${window.location.origin}/r/${encodeURIComponent(token)}`
}

export function getQRUrl(size = 200, content?: string): string {
  const params = new URLSearchParams({ size: String(size), t: String(Date.now()) })
  if (content) params.set('target', content)
  return `/qr?${params.toString()}`
}
