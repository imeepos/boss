// 数据导入中心:地址层级 + ISO 地理数据两个 JSON 导入面板 + 导入任务历史(menu:importer;geo 面板另需 menu:geo)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { ToolbarButton } from '../../../components/business/page-head'
import { Badge } from '../../../components/ui/badge'
import { ImportTaskList } from './TaskList'
import { CARD } from '../geo/styles'

const AREA_CLS = 'min-h-35 resize-y rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-2 font-mono text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]'

export default function ImporterPage() {
  const t = useT()
  const im = t.pages.importer
  return (
    <div className={CARD}>
      <h2 className="mx-4 mt-4 mb-3 text-base text-[var(--shell-content-text)]">{im.title}</h2>
      <ImportPanel title={im.addrTitle} hint={im.addrHint} endpoint="/addresses/import" text={im} kind="array" />
      <ImportPanel title={im.geoTitle} hint={im.geoHint} endpoint="/geo/import" text={im} kind="object" />
      <ImportTaskList />
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
    <div className="col-span-full mx-4 mb-5 flex flex-col gap-1.5">
      <label className="text-sm text-[var(--shell-content-text)]">{title}</label>
      <p className="m-0 mb-1 text-xs text-[var(--shell-input-placeholder)]">{hint}</p>
      <textarea className={AREA_CLS} value={payload}
        placeholder='[{"path":"gz","name":"广州市","countryCode":"CN"}]'
        onChange={(e) => setPayload(e.target.value)} />
      <div className="mt-2 flex items-center gap-3">
        <ToolbarButton primary disabled={!payload.trim()} onClick={run}>
          {text.importBtn}
        </ToolbarButton>
        {result && <Badge variant="success">{result}</Badge>}
        {error && <span className="text-xs text-[var(--color-danger)]">{error}</span>}
      </div>
    </div>
  )
}
