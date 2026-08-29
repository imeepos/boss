// 内联建址弹层:订单抽屉之上逐级先搜后建,全链 5 级(市/区/街道/小区/楼栋)。
// 契约:POST /orders/address {customerId, backfillCustomer, levels:[{id}|{name}]×5}
//   → {addressId, fullPath, legalEntityId, needsReview[]}(联调适配点集中在本文件类型区)。
// 零阻塞:搜索不可用时仍可输入名称逐级新建;兜底归属/并发重名不拦提交,警示条+待治理黄标承接。
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import type { AddressRow } from '../../base/address/AddressGeoDrawer'
import { ChainCrumb, ChainSummary, OwnerWarningBar, type ChainStage } from './AddressChainParts'

interface ChainHit { node: AddressRow; ancestors: AddressRow[] }
interface ChainOpt { id: number; name: string; ancestors: string[] }

// POST /orders/address 契约(以陈默合入 fields.md 为准,出入只改这里)。
// needsReview:待治理层级 level(1..5)列表;1-3 级客服所建节点强制在内。
interface OrderAddressResp {
  addressId: number
  fullPath: string
  legalEntityId: number
  legalEntityName?: string
  needsReview?: number[]
}

export interface ChainPickResult { addressId: number; fullPath: string }

const LEVEL_COUNT = 5

export function AddressChainDrawer({ customerId, backfillCustomer, onDone, onClose }: {
  customerId: string
  backfillCustomer: boolean
  onDone: (r: ChainPickResult) => void
  onClose: () => void
}) {
  const t = useT()
  const o = t.pages.orderPage
  const [stages, setStages] = useState<(ChainStage | null)[]>(Array(LEVEL_COUNT).fill(null))
  const [active, setActive] = useState(0)
  const [kw, setKw] = useState('')
  const [opts, setOpts] = useState<ChainOpt[]>([])
  const [searching, setSearching] = useState(false)
  const [searchDown, setSearchDown] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<OrderAddressResp | null>(null)
  const seqRef = useRef(0)

  const labels = o.chainLevels
  const allDone = stages.every(Boolean)

  // 先搜后建:关键字防抖走 /addresses/search,候选限定「已选前缀之下」的本级节点;失败降级只影响候选可见性,新建路径不受阻。
  useEffect(() => {
    if (result || active >= LEVEL_COUNT) return
    const key = kw.trim()
    if (!key) { setOpts([]); return }
    const timer = setTimeout(() => {
      const seq = ++seqRef.current
      setSearching(true)
      apiFetch<ChainHit[]>('/addresses/search', { query: { q: key } })
        .then((hits) => {
          if (seq !== seqRef.current) return
          setOpts(collect(hits ?? [], active, stages))
          setSearchDown(false)
        })
        .catch(() => { if (seq === seqRef.current) { setOpts([]); setSearchDown(true) } })
        .finally(() => { if (seq === seqRef.current) setSearching(false) })
    }, 300)
    return () => clearTimeout(timer)
  }, [kw, active, result]) // eslint-disable-line react-hooks/exhaustive-deps

  const commit = (s: ChainStage) => {
    const next = [...stages]
    next[active] = s
    setStages(next)
    if (active < LEVEL_COUNT - 1) {
      setActive(active + 1)
      setKw(''); setOpts([]); setSearchDown(false)
    }
  }

  const pick = (value: string) => {
    if (value.startsWith('new:')) {
      commit({ name: value.slice(4) })
      return
    }
    const opt = opts.find((x) => x.id === Number(value.slice(3)))
    if (opt) commit({ id: opt.id, name: opt.name })
  }

  // 回退到某级:保留 0..i-1,丢弃 i 及其后;当前级输入态一并重置。
  const jump = (i: number) => {
    setStages(stages.map((s, idx) => (idx < i ? s : null)))
    setActive(i)
    setKw(''); setOpts([]); setSearchDown(false); setError('')
  }

  const submit = async () => {
    if (busy || !allDone) return
    setBusy(true); setError('')
    try {
      const resp = await apiFetch<OrderAddressResp>('/orders/address', {
        method: 'POST',
        body: {
          customerId: Number(customerId),
          backfillCustomer,
          levels: stages.map((s) => (s!.id != null ? { id: s!.id } : { name: s!.name })),
        },
      })
      if (!resp) {
        setError(o.chainFail)
        return
      }
      const marks = new Set(resp.needsReview ?? [])
      setStages(stages.map((s, idx) => ({ ...s!, needsReview: marks.has(idx + 1) })))
      setResult(resp)
    } catch (e) {
      // 统一失败提示(可重试、层级保留)+ 原始细节并置,客服不必理解 HTTP 码。
      setError(o.chainFail + (e instanceof Error && e.message ? `(${e.message})` : ''))
    } finally { setBusy(false) }
  }

  const confirm = () => {
    if (!result) return
    onDone({ addressId: result.addressId, fullPath: result.fullPath })
  }

  const key = kw.trim()
  const options: DropdownOption[] = [
    ...opts.map((x) => ({ value: `id:${x.id}`, label: `${x.name} · ${[...x.ancestors, x.name].join(' / ')}` })),
    ...(key ? [{ value: `new:${key}`, label: o.chainNewOption.replace('{name}', key) }] : []),
  ]

  if (result) {
    const entity = result.legalEntityName ?? `#${result.legalEntityId}`
    const fallback = (result.needsReview ?? []).length > 0
    return (
      <Drawer title={o.chainTitle} onClose={onClose}
        footer={
          <>
            <ToolbarishButton onClick={onClose}>{t.pages.company.cancel}</ToolbarishButton>
            <PrimaryishButton disabled={busy} onClick={confirm}>{o.chainConfirm}</PrimaryishButton>
          </>
        }>
        <div className="flex flex-col gap-3">
          <div className="text-[13px] text-[var(--shell-content-text)]">{o.chainDone}</div>
          <ChainSummary stages={stages.filter(Boolean) as ChainStage[]} fullPath={result.fullPath} reviewText={o.chainNeedsReview} />
          <OwnerWarningBar entity={entity} fallback={fallback}
            ownerText={o.chainOwner} fallbackText={o.chainOwnerFallback} />
        </div>
      </Drawer>
    )
  }

  return (
    <Drawer title={o.chainTitle} onClose={onClose}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarishButton onClick={onClose}>{t.pages.company.cancel}</ToolbarishButton>
          <PrimaryishButton disabled={busy || !allDone} onClick={submit}>
            {busy ? o.chainCreating : o.chainCreate}
          </PrimaryishButton>
        </>
      }>
      <div className="flex flex-col gap-3">
        <ChainCrumb stages={stages} active={active} labels={labels} reviewText={o.chainNeedsReview} onJump={jump} />
        {searchDown && <DownHint text={o.chainSearchDown} />}
        {active < LEVEL_COUNT && (
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] text-[var(--shell-group-title)]">
              {o.chainStepLabel.replace('{level}', labels[active])}
              {searching ? ` · ${o.chainSearching}` : ''}
            </label>
            <Dropdown value="" options={options}
              onChange={pick} ariaLabel={labels[active]} searchable remote
              onKeywordChange={setKw} searchPlaceholder={o.chainPick}
              triggerStyle={{ width: '100%' }} />
          </div>
        )}
        {allDone && <div className="text-[12px] text-[var(--shell-group-title)]">{o.chainReady}</div>}
      </div>
    </Drawer>
  )
}

// collect:搜索命中限定为「已选前缀之下」的本级节点;前缀含新建名(无 id)时已有节点不可能落在其下,返回空只走新建。
function collect(hits: ChainHit[], stage: number, stages: (ChainStage | null)[]): ChainOpt[] {
  if (stages.slice(0, stage).some((s) => s && s.id == null)) return []
  const out: ChainOpt[] = []
  const seen = new Set<number>()
  for (const h of hits) {
    const chain = [...h.ancestors, h.node]
    const cand = chain.find((n) => n.level === stage + 1)
    if (!cand || seen.has(cand.id)) continue
    let under = true
    for (let i = 0; i < stage; i++) {
      if (chain[i]?.id !== stages[i]!.id) { under = false; break }
    }
    if (!under) continue
    seen.add(cand.id)
    out.push({ id: cand.id, name: cand.name, ancestors: chain.slice(0, stage).map((n) => n.name) })
  }
  return out
}

function DownHint({ text }: { text: string }) {
  return (
    <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-warning)_40%,transparent)] bg-[color-mix(in_srgb,var(--color-warning)_10%,transparent)] px-3 py-2 text-[12px] text-[var(--color-warning)]">
      {text}
    </div>
  )
}

function ToolbarishButton({ onClick, children }: { onClick: () => void; children: ReactNode }) {
  return (
    <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={onClick}>{children}</button>
  )
}

function PrimaryishButton({ disabled, onClick, children }: { disabled?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50"
      disabled={disabled} onClick={onClick}>{children}</button>
  )
}
