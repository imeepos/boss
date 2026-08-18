// 国家与行政区划维护页:PageContainer 惯例(页头 + 页签)+ 卡片化面板。
// 字段口径 docs/contract/fields.md 1.5.1;主题走 geo.css 令牌。
// 页签/搜索/分页状态均由 URL search 初始化,刷新页面状态不变。
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { CountryPanel } from './CountryPanel'
import { SubdivisionPanel } from './SubdivisionPanel'
import './geo.css'

export default function GeoPage() {
  const t = useT()
  const [tab, setTab] = useQueryState('tab', 'country')

  return (
    <div>
      <div className="geo-page-head">
        <div>
          <h2 className="geo-page-title">{t.pages.geo.title}</h2>
          <p className="geo-page-desc">ISO 3166-1 / ISO 3166-2 · CLDR · UN M49</p>
        </div>
      </div>
      <nav className="geo-tabs" role="tablist">
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
            className={`geo-tab${tab === key ? ' active' : ''}`}
            onClick={() => setTab(key)}
          >
            {label}
          </button>
        ))}
      </nav>
      {tab === 'subdiv' ? <SubdivisionPanel /> : <CountryPanel />}
    </div>
  )
}
