import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles.css'
import './theme/tokens.css'
import { LocaleProvider } from './i18n'
import { ThemeProvider } from './theme/context'
import { applyUrlPrefs } from './lib/urlPrefs'
import { Toaster } from './components/ui/sonner'

applyUrlPrefs()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeProvider>
      <LocaleProvider>
        <App />
        <Toaster position="top-center" />
      </LocaleProvider>
    </ThemeProvider>
  </React.StrictMode>,
)