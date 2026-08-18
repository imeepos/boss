// 国家维护面板:卡片化列表 + 工具栏(搜索/主操作) + 抽屉式新建编辑与详情。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { CountryForm, type CountryRow, EMPTY } from './CountryForm'
import { CountryDetail, type CountryDetailData } from './CountryDetail'
import './geo.css'

export function CountryPanel() {
  const t = useT()
  const g = t.pages.geo
  const [rows, setRows] = useState<CountryRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [form, setForm] = useState<CountryRow | null>(null)
  const [editing, setEditing] = useState(false)
  const [detail, setDetail] = useState<CountryDetailData | null>(null)

  const load = useCallback(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setRows(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [g])

  useEffect(load, [load])

  const openDetail = async (alpha2: string) => {
    const d = await apiFetch<CountryDetailData>(`/geo/countries/${alpha2}`).catch(() => null)
    setDetail(d ?? null)
    if (!d) setError(g.loadFail)
  }

  const toggle = async (row: CountryRow) => {
    await apiFetch(`/geo/countries/${row.alpha2}/active`, {
      method: 'PUT',
      body: { active: !row.isActive },
    }).catch(() => setError(g.saveFail))
    load()
  }

  const kw = keyword.trim().toLowerCase()
  const filtered = kw
    ? rows.filter((r) =>
        [r.alpha2, r.alpha3, r.shortName, r.displayName].some((s) => s.toLowerCase().includes(kw)))
    : rows

  return (
    <div className="geo-card">
      {error && <div className="geo-error" role="alert">{error}</div>}
      <div className="geo-toolbar">
        <input
          className="geo-input"
          style={{ width: 240 }}
          placeholder={g.searchPlaceholder}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
        <div className="spacer" />
        <button className="geo-btn geo-btn-primary" onClick={() => { setForm({ ...EMPTY }); setEditing(false) }}>
          + {g.add}
        </button>
      </div>
      <CountryTable rows={filtered} onEdit={(r) => { setForm({ ...r }); setEditing(true) }}
        onToggle={toggle} onDetail={openDetail} />
      <div className="geo-footer">{g.total.replace('{count}', String(filtered.length))}</div>
      {form && <CountryForm initial={form} editing={editing}
        onDone={() => { setForm(null); load() }} onCancel={() => setForm(null)} />}
      {detail && (
        <CountryDetail
          data={detail}
          onChanged={() => openDetail(detail.alpha2)}
          onClose={() => setDetail(null)}
        />
      )}
    </div>
  )
}

// CountryTable 国家列表;启停按语义 Tag 呈现(design-spec §4.2 表格规格)。
function CountryTable({ rows, onEdit, onToggle, onDetail }: {
  rows: CountryRow[]
  onEdit: (r: CountryRow) => void
  onToggle: (r: CountryRow) => void
  onDetail: (alpha2: string) => void
}) {
  const g = useT().pages.geo
  if (!rows.length) return <div className="geo-empty">{g.empty}</div>
  return (
    <div className="geo-table-wrap">
      <table className="geo-table">
        <thead>
          <tr>{g.countryColumns.map((c) => <th key={c}>{c}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.alpha2}>
              <td className="num">{r.alpha2}</td>
              <td>{r.alpha3}</td>
              <td>{r.numericCode}</td>
              <td>{r.displayName}</td>
              <td>{r.continentCode}</td>
              <td><StatusTag on={r.isActive} /></td>
              <td>
                <div className="geo-act">
                  <button onClick={() => onEdit(r)}>{g.edit}</button><span className="sep">|</span>
                  <button onClick={() => onToggle(r)}>{r.isActive ? g.disable : g.enable}</button><span className="sep">|</span>
                  <button onClick={() => onDetail(r.alpha2)}>{g.detail}</button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// StatusTag 启用/停用语义标签。
export function StatusTag({ on }: { on: boolean }) {
  const g = useT().pages.geo
  return <span className={`geo-tag ${on ? 'geo-tag-on' : 'geo-tag-off'}`}>{on ? g.active : g.inactive}</span>
}
