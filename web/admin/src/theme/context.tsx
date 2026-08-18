// 主题上下文:light/dark 双主题,落 localStorage,经 documentElement[data-theme] 驱动 tokens.css。
import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react'

export type Theme = 'light' | 'dark'

const THEME_KEY = 'boss.theme'

interface ThemeCtx {
  theme: Theme
  setTheme: (t: Theme) => void
  toggleTheme: () => void
}

const Ctx = createContext<ThemeCtx | null>(null)

function readStoredTheme(): Theme {
  if (typeof localStorage === 'undefined') return 'light'
  if (localStorage.getItem(THEME_KEY) === 'dark') return 'dark'
  return 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(readStoredTheme)

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t)
    try { localStorage.setItem(THEME_KEY, t) } catch { /* quota */ }
  }, [])

  const toggleTheme = useCallback(() => {
    setTheme(theme === 'dark' ? 'light' : 'dark')
  }, [theme, setTheme])

  useEffect(() => {
    document.documentElement.dataset.theme = theme
  }, [theme])

  return <Ctx.Provider value={{ theme, setTheme, toggleTheme }}>{children}</Ctx.Provider>
}

/** 当前主题 + 切换函数(仅 ThemeProvider 内使用)。 */
export function useTheme(): ThemeCtx {
  const c = useContext(Ctx)
  if (!c) throw new Error('useTheme 必须在 ThemeProvider 内使用')
  return c
}
