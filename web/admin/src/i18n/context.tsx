// 语言上下文:LocaleProvider + useT() + useLocale() + useLang()。
import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import type { Locale, Translations } from './types'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'
import msMY from './locales/ms-MY'

const LOCALE_KEY = 'boss.locale'

const LOCALES: Record<Locale, Translations> = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ms-MY': msMY,
}

const LOCALE_LABELS: Record<Locale, string> = {
  'zh-CN': '中文',
  'en-US': 'English',
  'ms-MY': 'Bahasa Malaysia',
}

interface LocaleCtx {
  locale: Locale
  setLocale: (l: Locale) => void
  t: Translations
}

const Ctx = createContext<LocaleCtx | null>(null)

function readStoredLocale(): Locale {
  if (typeof localStorage === 'undefined') return 'zh-CN'
  const v = localStorage.getItem(LOCALE_KEY)
  if (v === 'en-US' || v === 'ms-MY') return v
  return 'zh-CN'
}

export function LocaleProvider({ children, localeOverride }: { children: ReactNode; localeOverride?: Locale }) {
  const [locale, setLocaleState] = useState<Locale>(() => localeOverride ?? readStoredLocale())

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l)
    try { localStorage.setItem(LOCALE_KEY, l) } catch { /* quota */ }
  }, [])

  // 同步 html lang 属性
  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  return (
    <Ctx.Provider value={{ locale, setLocale, t: LOCALES[locale] }}>
      {children}
    </Ctx.Provider>
  )
}

/** 获取翻译对象(类型安全,按路径访问)。 */
export function useT(): Translations {
  const c = useContext(Ctx)
  if (!c) throw new Error('useT 必须在 LocaleProvider 内使用')
  return c.t
}

/** 获取当前语言代码。 */
export function useLocale(): Locale {
  const c = useContext(Ctx)
  if (!c) throw new Error('useLocale 必须在 LocaleProvider 内使用')
  return c.locale
}

/** 语言切换函数 + 当前语言代码。 */
export function useLang(): { locale: Locale; setLocale: (l: Locale) => void } {
  const c = useContext(Ctx)
  if (!c) throw new Error('useLang 必须在 LocaleProvider 内使用')
  return { locale: c.locale, setLocale: c.setLocale }
}

/** 所有支持的语言列表(供下拉选择器)。 */
export function localeOptions(): Array<{ value: Locale; label: string }> {
  return (Object.entries(LOCALE_LABELS) as Array<[Locale, string]>).map(([value, label]) => ({ value, label }))
}