import { useEffect, useState } from 'preact/hooks'
import { Button } from '../components/Button.tsx'
import { Input } from '../components/Input.tsx'
import { updateSettings } from '../api/settings.ts'
import { appState, AppState, Theme } from '../state/app.ts'
import { Language } from '../locales/index.ts'
import { toast } from '../components/Toast.tsx'
import { t } from '../locales/index.ts'

export function Settings() {
  const [state, setState] = useState<AppState>(appState.get())
  const [deviceName, setDeviceName] = useState(state.info?.name || '')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    return appState.subscribe((s) => {
      setState(s)
      if (s.info?.name && !deviceName) {
        setDeviceName(s.info.name)
      }
    })
  }, [])

  const handleSaveDeviceName = async () => {
    if (!deviceName.trim()) return

    setSaving(true)
    try {
      const res = await updateSettings({ device_name: deviceName.trim() })
      if (res.device_name) {
        setDeviceName(res.device_name)
      }
      appState.loadInfo()
      toast.success(t('settings.deviceNameSaved'))
    } catch {
      toast.error(t('settings.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const handleThemeChange = (theme: Theme) => {
    appState.setTheme(theme)
  }

  const handleLanguageChange = (lang: Language) => {
    appState.setLanguage(lang)
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', maxWidth: '640px', margin: '0 auto' }}>
      <h1 style={{ fontSize: '1.2rem', fontWeight: 600 }}>{t('settings.title')}</h1>

      {/* General Settings */}
      <section className="panel-section">
        <div className="panel-header">
          <div className="panel-title">{t('settings.general')}</div>
        </div>
        <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <Input
            label={t('settings.deviceName')}
            hint={t('settings.deviceNameHint')}
            value={deviceName}
            onInput={(e) => setDeviceName((e.target as HTMLInputElement).value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleSaveDeviceName()
            }}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              variant="primary"
              size="sm"
              onClick={handleSaveDeviceName}
              loading={saving}
              disabled={!deviceName.trim() || deviceName === state.info?.name}
            >
              {t('settings.save')}
            </Button>
          </div>
        </div>
      </section>

      {/* Appearance Settings */}
      <section className="panel-section">
        <div className="panel-header">
          <div className="panel-title">{t('settings.appearance')}</div>
        </div>
        <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {/* Theme */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 500 }}>{t('settings.theme')}</div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{t('settings.themeHint')}</div>
            </div>
            <select
              className="select"
              style={{ width: 'auto', minWidth: '130px' }}
              value={state.theme}
              onChange={(e) => handleThemeChange((e.target as HTMLSelectElement).value as Theme)}
            >
              <option value="system">{t('settings.themeSystem')}</option>
              <option value="light">{t('settings.themeLight')}</option>
              <option value="dark">{t('settings.themeDark')}</option>
            </select>
          </div>

          <div style={{ height: '1px', background: 'var(--border)' }} />

          {/* Language */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 500 }}>{t('settings.language')}</div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{t('settings.languageHint')}</div>
            </div>
            <select
              className="select"
              style={{ width: 'auto', minWidth: '130px' }}
              value={state.language}
              onChange={(e) => handleLanguageChange((e.target as HTMLSelectElement).value as Language)}
            >
              <option value="zh-CN">{t('settings.languageZh')}</option>
              <option value="en">{t('settings.languageEn')}</option>
            </select>
          </div>
        </div>
      </section>

      {/* About Section */}
      <section className="panel-section">
        <div className="panel-header">
          <div className="panel-title">{t('settings.about')}</div>
        </div>
        <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: '10px', fontSize: '13px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-secondary)' }}>{t('settings.version')}</span>
            <span className="font-mono">v{state.info?.version || '2.0.0'}</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-secondary)' }}>{t('settings.os')}</span>
            <span>{state.info?.os || 'Local System'}</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-secondary)' }}>{t('settings.lanAddress')}</span>
            <span className="font-mono">{state.info?.addr || window.location.host}</span>
          </div>
        </div>
      </section>
    </div>
  )
}
