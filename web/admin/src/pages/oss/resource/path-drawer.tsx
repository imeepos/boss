// PON 链路反查抽屉(P5-W2):GET /ports/{portId}/path 逐跳展示(类型/编码/状态/占用)。
// 断点行醒目标记;只读派生视图,断链不补链。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import type { PortPath, PortPathHop, PortRow } from '../types'

export function PathDrawer({ port, onClose }: { port: PortRow; onClose: () => void }) {
  const t = useT()
  const r = t.pages.resourcePage
  const [path, setPath] = useState<PortPath | null>(null)
  const [fail, setFail] = useState('')

  useEffect(() => {
    setPath(null)
    setFail('')
    apiFetch<PortPath>('/ports/' + port.portId + '/path')
      .then(setPath)
      .catch((e) => setFail(e instanceof Error ? e.message : r.loadFail))
  }, [port.portId]) // eslint-disable-line react-hooks/exhaustive-deps

  const kindText = (kind: string) => {
    if (kind === 'PORT') return r.hopPort
    if (kind === 'SPLITTER') return r.hopSplitter
    if (kind === 'PON_PORT') return r.hopPonPort
    if (kind === 'OLT') return r.hopOlt
    return kind
  }

  const breakText = (reason: string) => {
    if (reason === 'PORT_SPLITTER_MISSING') return r.breakPortSplitterMissing
    if (reason === 'SPLITTER_NO_PARENT') return r.breakSplitterNoParent
    if (reason === 'PON_PORT_UNASSIGNED') return r.breakPonUnassigned
    if (reason === 'OLT_UNREACHABLE') return r.breakOltUnreachable
    return reason
  }

  const statusCell = (h: PortPathHop) => {
    if (h.missing) {
      return (
        <span className="inline-flex items-center gap-1">
          <span className="inline-block rounded border border-[color-mix(in_srgb,var(--color-danger)_55%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] px-2 text-xs leading-[22px] text-[var(--color-danger)]">
            {r.breakTag}
          </span>
          <span className="text-xs text-[var(--color-danger)]">{breakText(h.reason ?? '')}</span>
        </span>
      )
    }
    if (h.kind === 'PORT') return <StatusTag domain="port" value={h.status} />
    if (h.kind === 'PON_PORT') return <span>{'—'}</span>
    return <StatusTag domain="resource" value={h.status} />
  }

  const completeBadge = (complete: boolean) => (
    <span className={"text-xs font-medium " + (complete ? "text-[var(--color-success)]" : "text-[var(--color-danger)]")}>
      {complete ? r.linkComplete : r.linkIncomplete}
    </span>
  )

  return (
    <Drawer
      title={r.linkViewTitle + ' · ' + port.portCode}
      onClose={onClose}
      footer={
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>
          {t.pages.company.cancel}
        </button>
      }
    >
      <div className="px-4 pb-2">{path ? completeBadge(path.complete) : null}</div>
      {fail ? (
        <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{fail}</div>
      ) : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead>
              <tr>
                {r.linkColumns.map((x) => (
                  <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {(path?.hops ?? []).map((h) => (
                <tr key={h.seq} className={h.missing ? 'bg-[color-mix(in_srgb,var(--color-danger)_6%,transparent)]' : ''}>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{h.seq}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{kindText(h.kind)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                    {h.missing ? '—' : h.code}
                    {!h.missing && h.name ? <span className="ml-1 text-xs text-[var(--shell-group-title)]">{h.name}</span> : null}
                  </td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{statusCell(h)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                    {h.occupiedBy ? '#' + h.occupiedBy.id + (h.occupiedBy.orderNo ? ' · ' + h.occupiedBy.orderNo : '') : '—'}
                  </td>
                </tr>
              ))}
              {path !== null && !path.hops.length && (
                <tr><td colSpan={5} className="h-11 px-3 border-b border-[var(--shell-side-border)]">{r.empty}</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Drawer>
  )
}