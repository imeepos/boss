// 代客下单可装性徽标(T14-2):选定地址后自动判定,内联五态展示;失败不阻断下单。
// 五态:可装(绿)/待覆盖(蓝)/不可装(红)/评估中(等待)/无法评估(灰,附解释);失败=轻提示可复制+重试。
import { useEffect, useRef, useState } from 'react'
import { Badge } from '../../../components/ui/badge'
import { CopyButton, Spinner } from '../../../components/business/feedback'
import { useT } from '../../../i18n'
import { resolveServability, type Servability } from './servability'

type Phase = { s: 'loading' } | { s: 'done'; v: Servability } | { s: 'fail'; msg: string }

function statusVariant(s: string): 'success' | 'info' | 'danger' | 'default' {
  return s === 'SERVED' ? 'success' : s === 'PENDING' ? 'info' : s === 'UNSERVED' ? 'danger' : 'default'
}

export function OrderServabilityBadge({ addressId }: { addressId: number }) {
  const t = useT()
  const o = t.pages.orderPage
  const [phase, setPhase] = useState<Phase>({ s: 'loading' })
  const seqRef = useRef(0)

  useEffect(() => {
    if (addressId <= 0) return
    const id = ++seqRef.current // 地址变更竞态守卫:只接受最后一次判定结果
    setPhase({ s: 'loading' })
    resolveServability(addressId)
      .then((v) => { if (seqRef.current === id) setPhase({ s: 'done', v }) })
      .catch((e) => { if (seqRef.current === id) setPhase({ s: 'fail', msg: e instanceof Error ? e.message : o.badgeFail }) })
  }, [addressId]) // eslint-disable-line react-hooks/exhaustive-deps

  if (addressId <= 0) return null
  const retry = () => {
    const id = ++seqRef.current
    setPhase({ s: 'loading' })
    resolveServability(addressId)
      .then((v) => { if (seqRef.current === id) setPhase({ s: 'done', v }) })
      .catch((e) => { if (seqRef.current === id) setPhase({ s: 'fail', msg: e instanceof Error ? e.message : o.badgeFail }) })
  }
  return (
    <div className="mt-1 flex flex-wrap items-center gap-2" data-testid="order-servability">
      {phase.s === 'loading' && (
        <Badge variant="default" className="gap-1"><Spinner size={12} /> {o.badgeEvaluating}</Badge>
      )}
      {phase.s === 'done' && phase.v.kind === 'no-coords' && (
        <span className="flex flex-col">
          <Badge variant="default">{o.badgeNoCoords}</Badge>
          <span className="mt-0.5 text-[12px] text-[var(--shell-group-title)]">{o.badgeNoCoordsHint}</span>
        </span>
      )}
      {phase.s === 'done' && phase.v.kind !== 'no-coords' && (
        <span className="flex items-center gap-2">
          <Badge variant={statusVariant(phase.v.status)}>
            {phase.v.status === 'SERVED' ? o.badgeServable : phase.v.status === 'PENDING' ? o.badgePending : phase.v.status === 'UNSERVED' ? o.badgeUnservable : phase.v.status}
          </Badge>
          {phase.v.kind === 'proximity' && (
            <span className="text-[12px] text-[var(--shell-group-title)]">{t.pages.odn.distance}: {Math.round(phase.v.distanceM)}m</span>
          )}
        </span>
      )}
      {phase.s === 'fail' && (
        <span className="flex flex-wrap items-center gap-2">
          <span className="text-[12px] text-[var(--color-danger)]">{o.badgeFail}: {phase.msg}</span>
          <CopyButton text={phase.msg} />
          <button type="button"
            className="h-6 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]"
            onClick={retry}>{o.badgeRetry}</button>
        </span>
      )}
    </div>
  )
}