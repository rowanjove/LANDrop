import { zhCN, TranslationType } from './zh-CN.ts'
import { en } from './en.ts'

export type Language = 'zh-CN' | 'en'

const translations: Record<Language, TranslationType> = {
  'zh-CN': zhCN,
  en,
}

let currentLang: Language = (localStorage.getItem('landrop_lang') as Language) ||
  (navigator.language.startsWith('zh') ? 'zh-CN' : 'en')

type Listener = (lang: Language) => void
const listeners = new Set<Listener>()

export function getLanguage(): Language {
  return currentLang
}

export function setLanguage(lang: Language) {
  currentLang = lang
  localStorage.setItem('landrop_lang', lang)
  listeners.forEach((fn) => fn(lang))
}

export function subscribeLanguage(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function t(path: string, params?: Record<string, string | number>): string {
  const dict = translations[currentLang] || translations['zh-CN']
  const keys = path.split('.')
  let current: any = dict

  for (const k of keys) {
    if (current && typeof current === 'object' && k in current) {
      current = current[k]
    } else {
      return path
    }
  }

  if (typeof current !== 'string') {
    return path
  }

  let text = current
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v))
    })
  }

  return text
}
