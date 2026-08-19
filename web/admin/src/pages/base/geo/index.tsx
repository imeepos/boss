// 国家与行政区划维护页:PageContainer 惯例(页头 + 页签)+ 卡片化面板。
// 字段口径 docs/contract/fields.md 1.5.1;样式 tailwind 原子类。
// 页签/搜索/分页状态均由 URL search 初始化,刷新页面状态不变。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { CountryPanel } from './CountryPanel'
import { SubdivisionPanel } from './SubdivisionPanel'

const TAB_BASE = 'mb-[-1px] cursor-pointer border-b-2 border-transparent bg-none px-0.5 pt-2.5 pb-3 text-sm text-[var(--shell-content-text)] hover:text-[var(--shell-heading)]'
const TAB_ACTIVE = 'font-semibold text-[var(--shell-heading)] border-b-[var(--color-brand-gold-500)]'

export default function GeoPage() {
  const t = useT()
  const [urlTab, setUrlTab] = useQueryState('tab', 'country')
  const [tab, setTab] = useState<'country' | 'subdiv'>(
    urlTab === 'subdiv' ? 'subdiv' : 'country',
  )

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{t.pages.geo.title}</h2>
          <p className="mt-1 text-xs text-[var(--shell-crumb-text)]">ISO 3166-1 / ISO 3166-2 · CLDR · UN M49</p>
        </div>
      </div>
      <nav className="mb-4 flex gap-8 border-b border-[var(--shell-side-border)]" role="tablist">
        {(
          [
            ['country', t.pages.geo.tabCountry],
            ['subdiv', t.pages.geo.tabSubdiv],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            className={tab === key ? `${TAB_BASE} ${TAB_ACTIVE}` : TAB_BASE}
            onClick={() => { setTab(key); setUrlTab(key) }}
          >
            {label}
          </button>
        ))}
      </nav>
      {tab === 'subdiv' ? <SubdivisionPanel /> : <CountryPanel />}
    </div>
  )
}
