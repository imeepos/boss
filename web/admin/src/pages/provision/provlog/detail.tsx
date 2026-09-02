// 下发日志详情抽屉:GET /provision-logs/{id} 的多维上下文 + 完整指令/设备应答原文。
// 关联数据已删时后端回空串/0,各字段统一回退"—";FAILED 结果以 danger 令牌高亮。
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import type { ProvisionLogDetail } from '../types'

function KV({ k, v, danger }: { k: string; v: string; danger?: boolean }) {
  return (
    <div className="flex gap-3 text-xs">
      <span className="w-24 flex-none text-[var(--shell-group-title)]">{k}</span>
      <span className={`break-all ${danger ? 'text-[var(--color-danger)]' : 'text-[var(--shell-content-text)]'}`}>{v || '—'}</span>
    </div>
  )
}

function RawSection({ title, text }: { title: string; text?: string }) {
  return (
    <section className="mb-5">
      <h4 className="mb-2 mt-0 text-[13px] font-semibold text-[var(--shell-heading)]">{title}</h4>
      {text ? (
        <pre className="m-0 overflow-x-auto whitespace-pre rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-3 font-mono text-xs leading-5 text-[var(--shell-content-text)]">{text}</pre>
      ) : (
        <div className="text-xs text-[var(--shell-content-text)]">—</div>
      )}
    </section>
  )
}

export function ProvisionLogDetailDrawer({ detail, onClose }: { detail: ProvisionLogDetail; onClose: () => void }) {
  const t = useT()
  const p = t.pages.provlogPage
  const { log, task, order, template } = detail
  const templateText = [
    log.templateCode || template.code,
    template.name,
    template.version ? `v${template.version}` : '',
    template.status,
  ].filter(Boolean).join(' · ')
  return (
    <Drawer
      title={p.detailTitle}
      width={560}
      onClose={onClose}
      footer={
        <button className="h-8 cursor-pointer rounded-sm border-none px-4 text-xs text-[var(--shell-fab-icon)] bg-[var(--shell-fab-bg)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>
          {t.pages.company.cancel}
        </button>
      }
    >
      <div className="flex flex-col gap-2.5">
        <KV k={p.labelTask} v={task.taskNo || `#${log.taskId}`} />
        <KV k={p.labelOrder} v={order.orderNo || (task.orderId ? `#${task.orderId}` : '')} />
        <KV k={p.labelOffer} v={order.offerName} />
        <KV k={p.labelTemplate} v={templateText} />
        <KV k={p.labelResult} v={log.result} danger={log.result === 'FAILED'} />
        <KV k={p.labelRetries} v={String(log.retries)} />
        <KV k={p.labelTime} v={fmtTime(log.createdAt)} />
      </div>
      <RawSection title={p.sectionCommand} text={log.commands} />
      <RawSection title={p.sectionResponse} text={log.deviceResponse} />
    </Drawer>
  )
}
