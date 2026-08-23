// 消息中心页:师傅站内消息 + 师傅端公告 + 后台提醒,三个页签(契约 worker.yaml Worker tag,perm menu:dispatch)。
// 页签状态经 URL 持久(useQueryState);设计系统 §2.1 TabBar + PageHead(page-patterns.md §1 头部)。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { PageHead } from '../../../components/business/page-head'
import { TabBar } from '../../../components/business/tab-bar'
import { AdminNotifsTab } from './AdminNotifsTab'
import { NoticesTab } from './NoticesTab'
import { WorkerMessagesTab } from './WorkerMessagesTab'

type TabKey = 'worker' | 'notice' | 'admin'

export default function MessageCenterPage() {
  const t = useT()
  const m = t.pages.message
  const [urlTab, setUrlTab] = useQueryState('tab', 'worker')
  const [tab, setTab] = useState<TabKey>(
    urlTab === 'notice' ? 'notice' : urlTab === 'admin' ? 'admin' : 'worker',
  )
  const tabs: Array<{ key: TabKey; label: string }> = [
    { key: 'worker', label: m.tabWorker },
    { key: 'notice', label: m.tabNotice },
    { key: 'admin', label: m.tabAdmin },
  ]
  return (
    <div>
      <PageHead title={m.title} desc={m.desc} />
      <TabBar tabs={tabs} value={tab} onChange={(k) => { setTab(k); setUrlTab(k) }} />
      {tab === 'admin' ? <AdminNotifsTab /> : tab === 'notice' ? <NoticesTab t={m} /> : <WorkerMessagesTab t={m} />}
    </div>
  )
}
