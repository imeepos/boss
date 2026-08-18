// 业务参数页(A1):列名与交互照抄 docs/admin/settings.html 原型。
// 契约: GET /params、PUT /params/{key}(sys.yaml;后端 planned,失败展示错误占位)。
// 行内编辑 + 批量保存:值列 input,dirty 行标"已修改",保存时逐项 PUT。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { filterParams, type BizParam } from './logic'
import './params.css'

export default function ParamsPage() {
  const t = useT()
  const [origin, setOrigin] = useState<BizParam[]>([])
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('')
  const [detail, setDetail] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState('')

  const load = () => {
    setError('')
    apiFetch<BizParam[]>('/params')
      .then((d) => {
        setOrigin(d ?? [])
        setDraft(Object.fromEntries((d ?? []).map((p) => [p.key, p.value])))
      })
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.params.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const rows = useMemo(
    () => filterParams(origin, draft, keyword, status),
    [origin, draft, keyword, status],
  )

  const dirty = origin.filter((p) => draft[p.key] !== p.value)

  const save = async () => {
    if (!dirty.length || saving) return
    setSaving(true)
    try {
      await Promise.all(dirty.map((p) => apiFetch(`/params/${encodeURIComponent(p.key)}`, {
        method: 'PUT',
        body: { value: draft[p.key] },
      })))
      setOrigin(origin.map((p) => ({ ...p, value: draft[p.key] ?? p.value })))
      setToast(t.pages.params.saved.replace('{count}', String(dirty.length)))
    } catch (e) {
      setToast((e instanceof Error ? e.message : t.pages.params.saveFail))
    } finally {
      setSaving(false)
    }
  }

  const detailRow = detail ? origin.find((p) => p.key === detail) : null

  return (
    <div className="params-page">
      <div className="page-head">
        <h2>{t.pages.params.title}</h2>
        <p className="page-desc">{t.pages.params.desc}</p>
      </div>
      <div className="params-toolbar">
        <input
          className="ctl"
          placeholder={t.pages.params.searchPlaceholder}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
        <select className="ctl" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">{t.pages.params.allStatus}</option>
          <option value="changed">{t.pages.params.statusChanged}</option>
          <option value="origin">{t.pages.params.statusOrigin}</option>
        </select>
        <span className="spacer" />
        <button className="btn" onClick={load}>{t.pages.params.refresh}</button>
      </div>
      <div className="card">
        <div className="card-title">
          {t.pages.params.cardTitle}
          <span className="extra">{t.pages.params.hotUpdate}</span>
        </div>
        {error ? <div className="error">{error}</div> : (
          <table className="tbl">
            <thead>
              <tr><th style={{ width: 220 }}>{t.pages.params.colName}</th><th>{t.pages.params.colValue}</th><th>{t.pages.params.colDesc}</th><th>{t.pages.params.colOp}</th></tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.key}>
                  <td>{r.label}</td>
                  <td>
                    <input
                      className="ctl"
                      style={{ width: 140 }}
                      value={draft[r.key] ?? ''}
                      onChange={(e) => setDraft({ ...draft, [r.key]: e.target.value })}
                    />
                    {draft[r.key] !== r.value && <span className="tag tag-orange">{t.pages.params.modified}</span>}
                  </td>
                  <td>{r.desc}</td>
                  <td><a onClick={() => setDetail(r.key)}>{t.pages.params.detail}</a></td>
                </tr>
              ))}
              {!rows.length && <tr><td colSpan={4} className="empty">{t.pages.params.empty}</td></tr>}
            </tbody>
          </table>
        )}
        <div className="params-actions">
          <button className="btn btn-primary" disabled={saving || !dirty.length} onClick={save}>
            {saving ? t.pages.params.saving : t.pages.params.save}
          </button>
          <button className="btn" onClick={load}>{t.pages.params.resetForm}</button>
          {toast && <span className="toast">{toast}</span>}
        </div>
      </div>
      {detailRow && (
        <div className="modal-mask" onClick={() => setDetail(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>{t.pages.params.detailTitle}</h3>
            <dl>
              <dt>{t.pages.params.colName}</dt><dd>{detailRow.label}</dd>
              <dt>Key</dt><dd>{detailRow.key}</dd>
              <dt>{t.pages.params.currentValue}</dt><dd>{draft[detailRow.key]}</dd>
              <dt>{t.pages.params.originValue}</dt><dd>{detailRow.value}</dd>
              <dt>{t.pages.params.colDesc}</dt><dd>{detailRow.desc}</dd>
            </dl>
            <button className="btn btn-primary" onClick={() => setDetail(null)}>{t.pages.params.close}</button>
          </div>
        </div>
      )}
    </div>
  )
}
