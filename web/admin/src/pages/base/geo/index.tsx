// 国家与行政区划维护页:PageContainer 惯例(页头 + 页签)+ 卡片化面板。
// 字段口径 docs/contract/fields.md 1.5.1;样式 tailwind 原子类。
// 页签/搜索/分页状态均由 URL search 初始化,刷新页面状态不变。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { CountryPanel } from './CountryPanel'
import { SubdivisionPanel } from './SubdivisionPanel'
import { BatchImportEntry } from '../importer/BatchImportEntry'
import { PageHead } from '../../../components/business/page-head'
import { TabBar } from '../../../components/business/tab-bar'

export default function GeoPage() {
  const t = useT()
  const [urlTab, setUrlTab] = useQueryState('tab', 'country')
  const [tab, setTab] = useState<'country' | 'subdiv'>(
    urlTab === 'subdiv' ? 'subdiv' : 'country',
  )
  // 导入完成后 bump rev 重挂当前面板(key remount):面板从 URL search 恢复筛选并重新拉取。
  const [rev, setRev] = useState(0)

  return (
    <div>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <PageHead title={t.pages.geo.title} desc="ISO 3166-1 / ISO 3166-2 · CLDR · UN M49" />
        <BatchImportEntry kind="geo" onImported={() => setRev((v) => v + 1)} />
      </div>
      <TabBar
        tabs={[
          { key: 'country' as const, label: t.pages.geo.tabCountry },
          { key: 'subdiv' as const, label: t.pages.geo.tabSubdiv },
        ]}
        value={tab}
        onChange={(k) => { setTab(k); setUrlTab(k) }}
      />
      {tab === 'subdiv' ? <SubdivisionPanel key={rev} /> : <CountryPanel key={rev} />}
    </div>
  )
}
