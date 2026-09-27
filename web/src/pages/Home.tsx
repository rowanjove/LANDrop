import { FileSendCard } from '../components/FileSendCard.tsx'
import { TextClipboardCard } from '../components/TextClipboardCard.tsx'
import { DeviceList } from '../components/DeviceList.tsx'
import { TransferList } from '../components/TransferList.tsx'

export interface HomeProps {
  onNavigate: (path: string) => void
}

export function Home({ onNavigate }: HomeProps) {
  return (
    <div className="home-grid">
      {/* Primary Column: File Transfer & Text/Clipboard */}
      <div className="home-col">
        <FileSendCard />
        <TextClipboardCard />
      </div>

      {/* Secondary Column: Nearby Devices & Current Transfers */}
      <div className="home-col">
        <DeviceList />
        <TransferList onViewAll={() => onNavigate('/activity')} />
      </div>
    </div>
  )
}
