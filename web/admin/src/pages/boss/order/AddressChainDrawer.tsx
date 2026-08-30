// 内联建址弹层:订单抽屉之上逐级先搜后建,全链 5 级(市/区/街道/小区/楼栋)。
// 契约 fields.md §1.5.0b:POST /orders/address {customerId, city..building 五级平铺必填, backfillCustomer}
//   → {addressId, fullPath, fullPathNames, legalEntityId, regionPath, fallback, needsReview:[{id,level,name}], backfilled}。
// 后端逐级 lookup-miss-then-create:选中已有节点与本地名都只发 name,复用语义由服务端保证。
// 零阻塞:搜索不可用时仍可输入名称逐级新建;兜底归属/并发重名不拦提交,警示条+待治理黄标承接。
import { useEffect, useRef, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { ApiError } from '../../../api/envelope'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { Button } from '../../../components/ui/button'
import type { AddressRow } from '../../base/address/AddressGeoDrawer'
import { ChainCrumb, ChainSummary, OwnerWarningBar, SiblingHint, type ChainStage } from './AddressChainParts'

interface ChainHit { node: AddressRow; ancestors: AddressRow[] }
interface ChainOpt { id: number; name: string; ancestors: string[] }

// fields.md §1.5.0b 契约(出入只改这里)。40900=UNIQUE(path) 撞库→前端引导复用;42200=参数缺失。
interface OrderAddressResp {
  addressId: number
  fullPath: string
  fullPathNames?: string
  legalEntityId: number
  regionPath?: string
  fallback?: boolean
  needsReview?: { id: number; level: number; name: string }[]
  backfilled?: boolean
}

export interface ChainPickResult { addressId: number; fullPath: string }

const LEVEL_COUNT = 5

export function AddressChainDrawer({ customerId, customerAddressId, onDone, onClose }: {
  customerId: string
  customerAddressId: number
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

  // 命名参照:上级已定(或有 id)时加载同层已有节点,输入侧防「3栋/3号楼」并存;数据源既有 /addresses?parentId=。
  const parentId = active === 0 ? 0 : stages[active - 1]?.id ?? -1
  const [siblings, setSiblings] = useState<AddressRow[]>([])
  useEffect(() => {
    if (result || parentId < 0) { setSiblings([]); return }
    let alive = true
    apiFetch<AddressRow[]>(`/addresses?parentId=${parentId}`)
      .then((d) => { if (alive) setSiblings(d ?? []) })
      .catch(() => { if (alive) setSiblings([]) })
    return () => { alive = false }
  }, [parentId, result])

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

  const submit = async (withBackfill: boolean) => {
    if (busy || !allDone || !Number(customerId)) return // customerId=0 会被 binding required 拒为 42200,入口已防呆,此处兜底。
    setBusy(true); setError('')
    try {
      // §1.5.0b:五级平铺必填 name(后端 lookup-miss-then-create,复用语义服务端保证);发 id 会 422。
      // 首建恒不带回填;档案更新走结果区征询后 withBackfill=true 重发(同链幂等复用)。
      const [city, district, street, compound, building] = stages.map((s) => s!.name)
      const resp = await apiFetch<OrderAddressResp>('/orders/address', {
        method: 'POST',
        body: { customerId: Number(customerId), city, district, street, compound, building, backfillCustomer: withBackfill },
      })
      if (!resp) {
        setError(o.chainFail)
        return
      }
      const marks = new Set((resp.needsReview ?? []).map((n) => n.level))
      setStages(stages.map((s, idx) => ({ ...s!, needsReview: marks.has(idx + 1) })))
      setResult(resp)
    } catch (e) {
      // 40900=同父同名 UNIQUE(path) 撞库:软提示引导复用已有节点,不作为终止分支。
      if (e instanceof ApiError && e.code === 40900) {
        setError(o.chainDuplicate)
        return
      }
      // 统一失败提示(可重试、层级保留)+ 原始细节并置,客服不必理解 HTTP 码。
      setError(o.chainFail + (e instanceof Error && e.message ? `(${e.message})` : ''))
    } finally { setBusy(false) }
  }

  const confirm = () => {
    if (!result) return
    onDone({ addressId: result.addressId, fullPath: result.fullPathNames ?? result.fullPath })
  }

  // 征询同意后带 backfillCustomer=true 重发:同链幂等复用同一 addressId,仅档案被覆盖更新。
  const refill = () => submit(true)

  const key = kw.trim()
  const options: DropdownOption[] = [
    ...opts.map((x) => ({ value: `id:${x.id}`, label: `${x.name} · ${[...x.ancestors, x.name].join(' / ')}` })),
    ...(key ? [{ value: `new:${key}`, label: o.chainNewOption.replace('{name}', key) }] : []),
  ]

  if (result) {
    const entity = `ID:${result.legalEntityId}`
    const fallback = result.fallback === true
    return (
      <Drawer title={o.chainTitle} onClose={onClose}
        footer={
          <>
            <Button variant="outline" size="sm" onClick={onClose}>{t.pages.company.cancel}</Button>
            <Button size="sm" disabled={busy} onClick={confirm}>{o.chainConfirm}</Button>
          </>
        }>
        <div className="flex flex-col gap-3">
          <div className="text-[13px] text-[var(--shell-content-text)]">{o.chainDone}</div>
          <ChainSummary stages={stages.filter(Boolean) as ChainStage[]} fullPath={result.fullPathNames ?? result.fullPath} reviewText={o.chainNeedsReview} />
          <OwnerWarningBar entity={entity} fallback={fallback}
            ownerText={o.chainOwner} fallbackText={o.chainOwnerFallback} fallbackHint={o.chainOwnerFallbackHint} />
          {result.backfilled === true && (
            <div className="text-[12px] text-[var(--shell-group-title)]">{o.chainBackfilled}</div>
          )}
          {result.backfilled !== true && customerAddressId > 0 && customerAddressId !== result.addressId && (
            <div className="flex items-center justify-between gap-2 rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-3 py-2">
              <span className="text-[12px] text-[var(--shell-content-text)]">{o.chainAskBackfill}</span>
              <Button variant="outline" size="sm" onClick={refill}>{busy ? o.chainRefilling : o.chainAskBackfillConfirm}</Button>
            </div>
          )}
        </div>
      </Drawer>
    )
  }

  return (
    <Drawer title={o.chainTitle} onClose={onClose}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <Button variant="outline" size="sm" onClick={onClose}>{t.pages.company.cancel}</Button>
          <Button size="sm" disabled={busy || !allDone} onClick={() => submit(false)}>
            {busy ? o.chainCreating : o.chainCreate}
          </Button>
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
            <SiblingHint nodes={siblings} pickText={o.chainSiblings} moreText={o.chainSiblingsMore}
              onPick={(n) => commit({ id: n.id, name: n.name })} />
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

