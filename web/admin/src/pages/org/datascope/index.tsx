// 数据权限页:账号级数据范围清单。列名对齐原型(账号/角色(功能)/子公司/部门/岗位/数据范围/操作)。
// 数据源: GET /data-scopes(后端已实现,keyword 服务端过滤账号/姓名/角色);空 keyword=全量。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterDataScopes, pageSlice, type ScopeRow } from './filter'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { Input } from '../../../components/ui/input'

export default function DataScopePage() {
  const t = useT()
  const [rows, setRows] = useState<ScopeRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [debounced, setDebounced] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<ScopeRow | null>(null)

  // 搜索语义统一:输入防抖 300ms 后走服务端 keyword;不再依赖手动刷新。
  useEffect(() => {
    const h = setTimeout(() => setDebounced(keyword.trim()), 300)
    return () => clearTimeout(h)
  }, [keyword])

  const load = () => {
    setError('')
    apiFetch<ScopeRow[]>('/data-scopes', { query: { keyword: debounced || undefined } })
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.datascope.loadFail))
  }
  useEffect(load, [debounced]) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => filterDataScopes(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)
  const scopeText = (r: ScopeRow) => r.regionScope || t.pages.datascope.allScope

  return (
    <div>
      <PageHead title={t.pages.datascope.title} desc={t.pages.datascope.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" placeholder={t.pages.datascope.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3"><ErrorBanner message={error} /></div> : (
          <Table>
            <TableHeader>
              <TableRow>
                {t.pages.datascope.columns.map((c) => (
                  <TableHead key={c}>{c}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.username}</TableCell>
                  <TableCell>{r.roleName}</TableCell>
                  <TableCell>{r.legalEntityName || '—'}</TableCell>
                  <TableCell>{r.deptName || '—'}</TableCell>
                  <TableCell>{r.postName || '—'}</TableCell>
                  <TableCell>{scopeText(r)}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center">
                      <button className="cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => setDetail(r)}>{t.pages.datascope.detail}</button>
                    </span>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={7} text={t.pages.datascope.empty} />}
            </TableBody>
          </Table>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.datascope)} />
        </div>
      </Card>
      {detail && (
        <DetailDrawer
          title={t.pages.datascope.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.datascope.columns[0], v: detail.username },
            { k: t.pages.datascope.columns[1], v: detail.roleName },
            { k: t.pages.datascope.columns[2], v: detail.legalEntityName },
            { k: t.pages.datascope.columns[3], v: detail.deptName },
            { k: t.pages.datascope.columns[4], v: detail.postName },
            { k: t.pages.datascope.columns[5], v: scopeText(detail) },
          ]}
        />
      )}
    </div>
  )
}
