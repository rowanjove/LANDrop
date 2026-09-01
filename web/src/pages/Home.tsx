import { SendPanel } from '../components/SendPanel.tsx'
import { DeviceList } from '../components/DeviceList.tsx'
import { TransferList } from '../components/TransferList.tsx'

export interface HomeProps {
  onNavigate: (path: string) => void
}

export function Home({ onNavigate }: HomeProps) {
  return (
    <div className="home-grid">
      {/* Primary Column: Send Panel */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
        <SendPanel />
      </div>

      {/* Secondary Column: Nearby Devices & Recent Transfers */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
        <DeviceList />
        <TransferList onViewAll={() => onNavigate('/activity')} />
      </div>
    </div>
  )
}
