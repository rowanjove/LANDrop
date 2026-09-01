import { useEffect, useState } from 'preact/hooks'
import { Header } from './components/Header.tsx'
import { ToastContainer } from './components/Toast.tsx'
import { Home } from './pages/Home.tsx'
import { Activity } from './pages/Activity.tsx'
import { Settings } from './pages/Settings.tsx'
import { Receive } from './pages/Receive.tsx'
import { IconHistory, IconHome, IconSettings } from './components/Icons.tsx'
import { useEvents } from './hooks/useEvents.ts'
import { t } from './locales/index.ts'

function receiveTokenFromPath(path = window.location.pathname): string | null {
  const match = path.match(/^\/r\/([^/]+)\/?$/)
  if (!match) return null
  try {
    return decodeURIComponent(match[1])
  } catch {
    return null
  }
}

function getNormalizedPath(): string {
  const path = window.location.pathname
  const token = receiveTokenFromPath(path)
  if (token) return `/r/${encodeURIComponent(token)}`
  if (path.endsWith('/activity') || path.endsWith('/activity/')) return '/activity'
  if (path.endsWith('/settings') || path.endsWith('/settings/')) return '/settings'
  return '/'
}

export function App() {
  const [currentPath, setCurrentPath] = useState<string>(getNormalizedPath())

  // Activate SSE connection & event dispatcher
  useEvents()

  useEffect(() => {
    const handlePopState = () => {
      setCurrentPath(getNormalizedPath())
    }

    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  const navigate = (path: string) => {
    if (path !== currentPath) {
      window.history.pushState({}, '', path)
      setCurrentPath(path)
      window.scrollTo(0, 0)
    }
  }

  return (
    <div className="app-container">
      <Header currentPath={currentPath} onNavigate={navigate} />

      <main className="main-wrapper">
        {currentPath === '/' && <Home onNavigate={navigate} />}
        {currentPath === '/activity' && <Activity />}
        {currentPath === '/settings' && <Settings />}
        {currentPath.startsWith('/r/') && <Receive token={decodeURIComponent(currentPath.slice(3))} />}
      </main>

      {/* Mobile Bottom Navigation (<768px) */}
      <nav className="mobile-nav" aria-label={t('accessibility.mobileNav')}>
        <button
          className={`mobile-nav-item ${currentPath === '/' ? 'active' : ''}`}
          onClick={() => navigate('/')}
        >
          <IconHome size={18} />
          <span>{t('nav.home')}</span>
        </button>

        <button
          className={`mobile-nav-item ${currentPath === '/activity' ? 'active' : ''}`}
          onClick={() => navigate('/activity')}
        >
          <IconHistory size={18} />
          <span>{t('nav.activity')}</span>
        </button>

        <button
          className={`mobile-nav-item ${currentPath === '/settings' ? 'active' : ''}`}
          onClick={() => navigate('/settings')}
        >
          <IconSettings size={18} />
          <span>{t('nav.settings')}</span>
        </button>
      </nav>

      <ToastContainer />
    </div>
  )
}
