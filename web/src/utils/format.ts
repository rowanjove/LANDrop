export function formatSize(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const idx = Math.min(i, sizes.length - 1)
  const val = bytes / Math.pow(k, idx)
  return `${val.toFixed(idx === 0 ? 0 : 1)} ${sizes[idx]}`
}

export function formatTime(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

export function formatDate(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  return date.toLocaleDateString()
}

export function formatDateTime(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
}

export function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec <= 0) return ''
  return `${formatSize(bytesPerSec)}/s`
}

export function formatEta(seconds: number): string {
  if (seconds <= 0 || !isFinite(seconds)) return ''
  if (seconds < 60) {
    const value = Math.round(seconds)
    return getLanguage() === 'en' ? `about ${value}s` : `约 ${value} 秒`
  }
  const minutes = Math.floor(seconds / 60)
  const remainingSecs = Math.round(seconds % 60)
  return getLanguage() === 'en' ? `about ${minutes}m ${remainingSecs}s` : `约 ${minutes} 分 ${remainingSecs} 秒`
}
import { getLanguage } from '../locales/index.ts'
