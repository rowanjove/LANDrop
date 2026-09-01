import { render } from 'preact'
import { App } from './app.tsx'

import './styles/tokens.css'
import './styles/reset.css'
import './styles/base.css'
import './styles/layout.css'
import './styles/components.css'

const root = document.getElementById('app')
if (root) {
  render(<App />, root)
}
