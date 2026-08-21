// 消息中心页:师傅站内消息 + 师傅端公告 + 后台提醒,三个页签(契约 worker.yaml Worker tag,perm menu:dispatch)。
// 页签状态经 URL 持久(useQueryState);样式 tailwind + shell-* 令牌(双主题)。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { AdminNotifsTab } from './AdminNotifsTab'
import { NoticesTab } from './NoticesTab'
import { WorkerMessagesTab } from './WorkerMessagesTab'

const TAB_BASE = 'mb-[-1px] cursor-pointer border-b-2 border-transparent bg-none px-0.5 pt-2.5 pb-3 text-sm text-[var(--shell-content-text)] hover:text-[var(--shell-heading)]'
const TAB_ACTIVE = 'font-semibold text-[var(--shell-heading)] border-b-[var(--color-brand-gold-500)]'

type TabKey = 'worker' | 'notice' | 'admin'

export default function MessageCenterPage() {
  const t = useT()
  const m = t.pages.message
  const [urlTab, setUrlTab] = useQueryState('tab', 'worker')
  const [tab, setTab] = useState<TabKey>(
    urlTab === 'notice' ? 'notice' : urlTab === 'admin' ? 'admin' : 'worker',
  )
  const tabs: Array<[TabKey, string]> = [
    ['worker', m.tabWorker],
    ['notice', m.tabNotice],
    ['admin', m.tabAdmin],
  ]
  return (
    <div>
      <h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{m.title}</h2>
      <p className="mt-1 mb-3 text-xs text-[var(--shell-crumb-text)]">{m.desc}</p>
      <nav className="mb-4 flex gap-8 border-b border-[var(--shell-side-border)]" role="tablist">
        {tabs.map(([key, label]) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            className={tab === key ? `${TAB_BASE} ${TAB_ACTIVE}` : TAB_BASE}
            onClick={() => { setTab(key); setUrlTab(key) }}
          >
            {label}
          </button>
        ))}
      </nav>
      {tab === 'admin' ? <AdminNotifsTab /> : tab === 'notice' ? <NoticesTab t={m} /> : <WorkerMessagesTab t={m} />}
    </div>
  )
}
