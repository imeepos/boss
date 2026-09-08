// 业务参数页(A1):列名与交互照抄 docs/admin/settings.html 原型。
// 契约: GET /params、PUT /params/{key}(sys.yaml;已上线,失败展示错误占位)。
// 行内编辑 + 批量保存:值列 input,dirty 行标"已修改",保存时逐项 PUT。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { filterParams, paramLabel, type BizParam } from './logic'

interface PartialSaveReport { ok: number; fail: number; reason: string }

export default function ParamsPage() {
  const t = useT()
  const [origin, setOrigin] = useState<BizParam[]>([])
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('')
  const [detail, setDetail] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveState, setSaveState] = useState<SubmitState>('idle')
  const [saveReport, setSaveReport] = useState<PartialSaveReport | null>(null)

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

  // 批量保存:P1 裁定——allSettled 聚合口径,部分失败必须失败呈现(成功 N/失败 M+原因),
  // 只回写成功项;禁止一把成功色。
  const save = async () => {
    if (!dirty.length || saving) return
    setSaving(true)
    setSaveReport(null)
    setSaveState('loading')
    try {
      const results = await Promise.allSettled(dirty.map((p) => apiFetch(`/params/${encodeURIComponent(p.key)}`, {
        method: 'PUT',
        body: { value: draft[p.key] },
      })))
      const failed = results.filter((r) => r.status === 'rejected')
      const ok = results.length - failed.length
      if (failed.length === 0) {
        setSaveState('success')
        toast.success(t.pages.params.saved.replace('{count}', String(ok)))
        setSaveReport(null)
        setTimeout(() => setSaveState((s) => (s === 'success' ? 'idle' : s)), 1500)
      } else {
        const first = failed[0]
        const reason = first.reason instanceof Error ? first.reason.message : t.pages.params.saveFail
        setSaveState('failed')
        toast.error(t.pages.params.savePartial
          .replace('{ok}', String(ok))
          .replace('{fail}', String(failed.length))
          .replace('{reason}', reason))
        setSaveReport({ ok, fail: failed.length, reason })
        setTimeout(() => setSaveState((s) => (s === 'failed' ? 'idle' : s)), 2500)
      }
      if (ok > 0) {
        const okKeys = new Set(dirty.filter((_, i) => results[i].status === 'fulfilled').map((p) => p.key))
        setOrigin(origin.map((p) => (okKeys.has(p.key) ? { ...p, value: draft[p.key] ?? p.value } : p)))
      }
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
        <Dropdown
          value={status}
          options={[
            { value: '', label: t.pages.params.allStatus },
            { value: 'changed', label: t.pages.params.statusChanged },
            { value: 'origin', label: t.pages.params.statusOrigin },
          ]}
          onChange={(v) => setStatus(v)}
          ariaLabel={t.pages.params.allStatus}
        />
        <div className="flex-1" />
        <ToolbarButton onClick={load}>{t.pages.params.refresh}</ToolbarButton>
      </div>
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="p-4">
        <div className="mb-3 font-semibold text-[var(--shell-heading)]">
          {t.pages.params.cardTitle}
          <span className="ml-2 text-xs font-normal text-[var(--shell-crumb-text)]">{t.pages.params.hotUpdate}</span>
        </div>
        {saveReport && (
          <div className="mb-3">
            <ErrorBanner message={
              t.pages.params.savePartial
                .replace('{ok}', String(saveReport.ok))
                .replace('{fail}', String(saveReport.fail))
                .replace('{reason}', saveReport.reason)
            } />
          </div>
        )}
        {error ? null : (
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
          <SubmitButton
            state={saveState}
            labels={{
              idle: t.pages.params.save,
              loading: t.pages.params.saving,
              success: t.pages.params.saved.replace('{count}', String(dirty.length)),
              failed: t.pages.params.saveFail,
            }}
            disabled={saving || !dirty.length}
            onClick={save}
          />
          <ToolbarButton onClick={load}>{t.pages.params.resetForm}</ToolbarButton>
        </div>
      </Card>
      <Dialog open={detailRow !== null} onOpenChange={(v) => { if (!v) setDetail(null) }}>
        <DialogContent className="w-95">
          <DialogHeader>
            <DialogTitle>{t.pages.params.detailTitle}</DialogTitle>
          </DialogHeader>
          {detailRow && (
            <dl className="grid grid-cols-[80px_1fr] gap-x-3 gap-y-2 text-[13px]">
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.colName}</dt><dd className="m-0">{paramLabel(detailRow)}</dd>
              <dt className="text-[var(--shell-crumb-text)]">Key</dt><dd className="m-0">{detailRow.key}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.currentValue}</dt><dd className="m-0">{draft[detailRow.key]}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.originValue}</dt><dd className="m-0">{detailRow.value}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.params.colDesc}</dt><dd className="m-0">{detailRow.desc}</dd>
            </dl>
          )}
          <ToolbarButton onClick={() => setDetail(null)}>{t.pages.params.close}</ToolbarButton>
        </DialogContent>
      </Dialog>
    </div>
  )
}
