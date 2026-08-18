// 数据导入中心:地址层级 + ISO 地理数据两个 JSON 导入面板(menu:importer;geo 面板另需 menu:geo)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import '../geo/geo.css'

export default function ImporterPage() {
  const t = useT()
  const im = t.pages.importer
  return (
    <div className="geo-card">
      <h2 className="geo-section-title">{im.title}</h2>
      <ImportPanel title={im.addrTitle} hint={im.addrHint} endpoint="/addresses/import" text={im} kind="array" />
      <ImportPanel title={im.geoTitle} hint={im.geoHint} endpoint="/geo/import" text={im} kind="object" />
    </div>
  )
}

// ImportPanel 单个导入面板:textarea + 导入按钮 + 结果。
function ImportPanel({ title, hint, endpoint, text, kind }: {
  title: string
  hint: string
  endpoint: string
  text: Translations['pages']['importer']
  kind: 'array' | 'object'
}) {
  const [payload, setPayload] = useState('')
  const [result, setResult] = useState('')
  const [error, setError] = useState('')

  const run = async () => {
    setResult('')
    setError('')
    let parsed: unknown
    try {
      parsed = JSON.parse(payload)
    } catch {
      setError(text.parseFail)
      return
    }
    const ok = kind === 'array' ? Array.isArray(parsed) : typeof parsed === 'object' && parsed !== null
    if (!ok) {
      setError(text.parseFail)
      return
    }
    try {
      const res = await apiFetch<{ imported?: number }>(endpoint, {
        method: 'POST',
        body: parsed as Record<string, unknown>,
      })
      setResult(text.imported.replace('{count}', String(res?.imported ?? 0)))
    } catch {
      setError(text.loadFail)
    }
  }

  return (
    <div className="geo-field full" style={{ margin: '0 16px 20px' }}>
      <label style={{ fontSize: 14 }}>{title}</label>
      <p className="hint" style={{ margin: '4px 0 8px', fontSize: 12 }}>{hint}</p>
      <textarea className="geo-input addr-import-area" value={payload}
        placeholder='[{"path":"gz","name":"广州市","countryCode":"CN"}]'
        onChange={(e) => setPayload(e.target.value)} />
      <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginTop: 8 }}>
        <button className="geo-btn geo-btn-primary" disabled={!payload.trim()} onClick={run}>
          {text.importBtn}
        </button>
        {result && <span className="geo-tag geo-tag-on">{result}</span>}
        {error && <span className="geo-error" style={{ margin: 0 }}>{error}</span>}
      </div>
    </div>
  )
}
