import { AppInfo, fetchAppInfo } from '../api/info.ts'
import { fetchSettings, updateSettings } from '../api/settings.ts'
import { getLanguage, Language, setLanguage, subscribeLanguage } from '../locales/index.ts'

export type Theme = 'system' | 'light' | 'dark'

export interface AppState {
  info: AppInfo | null
  connected: boolean
  theme: Theme
  language: Language
}

let state: AppState = {
  info: null,
  connected: false,
  theme: (localStorage.getItem('landrop_theme') as Theme) || 'system',
  language: getLanguage(),
}

type Listener = (s: AppState) => void
const listeners = new Set<Listener>()

function notify() {
  listeners.forEach((fn) => fn({ ...state }))
}

function applyTheme(theme: Theme) {
  const root = document.documentElement
  if (theme === 'system') {
    root.removeAttribute('data-theme')
  } else {
    root.setAttribute('data-theme', theme)
  }
}

function persistSettings(settings: Parameters<typeof updateSettings>[0]) {
  void updateSettings(settings).catch(() => undefined)
}

// Initial theme apply
applyTheme(state.theme)

export const appState = {
  get(): AppState {
    return { ...state }
  },
  subscribe(fn: Listener): () => void {
    listeners.add(fn)
    fn({ ...state })
    return () => listeners.delete(fn)
  },
  setConnected(connected: boolean) {
    if (state.connected !== connected) {
      state.connected = connected
      notify()
    }
  },
  setInfo(info: AppInfo) {
    state.info = info
    notify()
  },
  setTheme(theme: Theme) {
    state.theme = theme
    localStorage.setItem('landrop_theme', theme)
    applyTheme(theme)
    persistSettings({ theme })
    notify()
  },
  setLanguage(lang: Language) {
    setLanguage(lang)
    state.language = lang
    persistSettings({ language: lang })
    notify()
  },
  async loadInfo() {
    try {
      const info = await fetchAppInfo()
      state.info = info
      state.connected = true
      notify()
    } catch {
      state.connected = false
      notify()
    }
    try {
      const settings = await fetchSettings()
      if (settings.theme === 'system' || settings.theme === 'light' || settings.theme === 'dark') {
        state.theme = settings.theme
        localStorage.setItem('landrop_theme', settings.theme)
        applyTheme(settings.theme)
      }
      if (settings.language === 'zh-CN' || settings.language === 'en') {
        state.language = settings.language
        setLanguage(settings.language)
      }
      notify()
    } catch {
      // Settings are optional for older peers; local preferences remain active.
    }
  },
}

// Synchronize language events
subscribeLanguage((lang) => {
  if (state.language !== lang) {
    state.language = lang
    notify()
  }
})
