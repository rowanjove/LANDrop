import { fetchHistory, HistoryRecord } from '../api/history.ts'

export interface PendingFile {
  id: string
  file: File
  name: string
  size: number
}

export interface ActiveTransfer {
  id: string
  type: 'file' | 'text'
  name?: string
  size: number
  status: 'pending' | 'uploading' | 'ready' | 'completed' | 'failed'
  progress: number
  loaded: number
  total: number
  token?: string
  downloadUrl?: string
  error?: string
}

export interface TransfersState {
  pendingFiles: PendingFile[]
  activeTransfer: ActiveTransfer | null
  recentHistory: HistoryRecord[]
  loadingHistory: boolean
}

let state: TransfersState = {
  pendingFiles: [],
  activeTransfer: null,
  recentHistory: [],
  loadingHistory: false,
}

type Listener = (s: TransfersState) => void
const listeners = new Set<Listener>()

function notify() {
  listeners.forEach((fn) => fn({ ...state }))
}

export const transfersState = {
  get(): TransfersState {
    return { ...state }
  },
  subscribe(fn: Listener): () => void {
    listeners.add(fn)
    fn({ ...state })
    return () => listeners.delete(fn)
  },
  addFiles(files: FileList | File[]) {
    const newItems: PendingFile[] = Array.from(files).map((f) => ({
      id: Math.random().toString(36).slice(2, 9),
      file: f,
      name: f.name,
      size: f.size,
    }))
    state.pendingFiles = [...state.pendingFiles, ...newItems]
    notify()
  },
  removeFile(id: string) {
    state.pendingFiles = state.pendingFiles.filter((f) => f.id !== id)
    notify()
  },
  clearPendingFiles() {
    state.pendingFiles = []
    notify()
  },
  setActiveTransfer(transfer: ActiveTransfer | null) {
    state.activeTransfer = transfer
    notify()
  },
  updateActiveTransfer(partial: Partial<ActiveTransfer>) {
    if (state.activeTransfer) {
      state.activeTransfer = { ...state.activeTransfer, ...partial }
      notify()
    }
  },
  setRecentHistory(history: HistoryRecord[]) {
    state.recentHistory = history
    notify()
  },
  async refreshHistory(limit = 10) {
    state.loadingHistory = true
    notify()
    try {
      const records = await fetchHistory({ limit })
      state.recentHistory = records
    } catch {
      // ignore
    } finally {
      state.loadingHistory = false
      notify()
    }
  },
}
