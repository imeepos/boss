// 业务参数页(A1):列名与交互照抄 docs/admin/settings.html 原型。
// 契约: GET /params、PUT /params/{key}(sys.yaml;已上线,失败展示错误占位)。
// 行内编辑 + 批量保存:值列 input,dirty 行标"已修改",保存时逐项 PUT。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { filterParams, paramLabel, type BizParam } from './logic'

const SELECT_CLS = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-xs text-[var(--shell-input-text)] outline-none focus:border-[var(--shell-input-border-focus)]'

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
    apiFetch<{ items: BizParam[] }>('/params')
      .then((d) => {
        const items = d?.items ?? []
        setOrigin(items)
        setDraft(Object.fromEntries(items.map((p) => [p.key, p.value])))
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
    <div>
      <PageHead title={t.pages.params.title} desc={t.pages.params.desc} />
      <div className="mb-3 flex items-center gap-2">
        <Input
          className="w-52"
          placeholder={t.pages.params.searchPlaceholder}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
        <select className={SELECT_CLS} value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">{t.pages.params.allStatus}</option>
          <option value="changed">{t.pages.params.statusChanged}</option>
          <option value="origin">{t.pages.params.statusOrigin}</option>
        </select>
        <div className="flex-1" />
        <ToolbarButton onClick={load}>{t.pages.params.refresh}</ToolbarButton>
      </div>
      <Card className="p-4">
        <div className="mb-3 font-semibold text-[var(--shell-heading)]">
          {t.pages.params.cardTitle}
          <span className="ml-2 text-xs font-normal text-[var(--shell-crumb-text)]">{t.pages.params.hotUpdate}</span>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-55">{t.pages.params.colName}</TableHead>
                <TableHead>{t.pages.params.colValue}</TableHead>
                <TableHead>{t.pages.params.colDesc}</TableHead>
                <TableHead>{t.pages.params.colOp}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={r.key}>
                  <TableCell>{paramLabel(r)}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1.5">
                      <Input
                        className="w-35"
                        value={draft[r.key] ?? ''}
                        onChange={(e) => setDraft({ ...draft, [r.key]: e.target.value })}
                      />
                      {draft[r.key] !== r.value && <Badge variant="warning">{t.pages.params.modified}</Badge>}
                    </span>
                  </TableCell>
                  <TableCell>{r.desc}</TableCell>
                  <TableCell>
                    <button className="border-none bg-none px-0 text-xs text-[var(--color-text-link)] cursor-pointer hover:underline" onClick={() => setDetail(r.key)}>
                      {t.pages.params.detail}
                    </button>
                  </TableCell>
                </TableRow>
              ))}
              {!rows.length && <TableRow><TableCell colSpan={4}><EmptyState text={t.pages.params.empty} /></TableCell></TableRow>}
            </TableBody>
          </Table>
        )}
        <div className="mt-3.5 flex items-center gap-2">
          <ToolbarButton primary disabled={saving || !dirty.length} onClick={save}>
            {saving ? t.pages.params.saving : t.pages.params.save}
          </ToolbarButton>
          <ToolbarButton onClick={load}>{t.pages.params.resetForm}</ToolbarButton>
          {toast && <span className="text-xs text-[var(--color-success)]">{toast}</span>}
        </div>
      </Card>
      {detailRow && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45" onClick={() => setDetail(null)}>
          <div className="w-95 rounded-md bg-[var(--shell-card-bg)] p-5 shadow-[var(--shadow-panel)]" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-3 text-base font-semibold text-[var(--shell-heading)]">{t.pages.params.detailTitle}</h3>
            <dl className="mb-4 grid grid-cols-[80px_1fr] gap-x-3 gap-y-2 text-[13px]">
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.colName}</dt><dd className="m-0">{paramLabel(detailRow)}</dd>
              <dt className="text-[var(--shell-crumb-text)]">Key</dt><dd className="m-0">{detailRow.key}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.currentValue}</dt><dd className="m-0">{draft[detailRow.key]}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.originValue}</dt><dd className="m-0">{detailRow.value}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.colDesc}</dt><dd className="m-0">{detailRow.desc}</dd>
            </dl>
            <ToolbarButton primary onClick={() => setDetail(null)}>{t.pages.params.close}</ToolbarButton>
          </div>
        </div>
      )}
    </div>
  )
}
