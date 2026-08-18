import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles.css'
import './theme/tokens.css'
import { LocaleProvider } from './i18n'
import { ThemeProvider } from './theme/context'
import { applyUrlPrefs } from './lib/urlPrefs'

applyUrlPrefs()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeProvider>
      <LocaleProvider>
        <App />
      </LocaleProvider>
    </ThemeProvider>
  </React.StrictMode>,
)