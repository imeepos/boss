// 国家与行政区划维护页:双 Tab(国家/区划),字段口径 docs/contract/fields.md 1.5.1。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { CountryPanel } from './CountryPanel'
import { SubdivisionPanel } from './SubdivisionPanel'

export default function GeoPage() {
  const t = useT()
  const [tab, setTab] = useState<'country' | 'subdiv'>('country')

  return (
    <div>
      <h2>{t.pages.geo.title}</h2>
      <div style={{ marginBottom: 12, display: 'flex', gap: 8 }}>
        {(
          [
            ['country', t.pages.geo.tabCountry],
            ['subdiv', t.pages.geo.tabSubdiv],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            style={{
              padding: '6px 14px',
              border: '1px solid #d9d9d9',
              borderRadius: 4,
              background: tab === key ? '#1677ff' : '#fff',
              color: tab === key ? '#fff' : '#333',
              cursor: 'pointer',
            }}
          >
            {label}
          </button>
        ))}
      </div>
      {tab === 'country' ? <CountryPanel /> : <SubdivisionPanel />}
    </div>
  )
}
