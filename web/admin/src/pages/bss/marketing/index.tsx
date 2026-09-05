// 营销与积分规则页(2028 Q2 交付「营销规则配置」):五个规则域 Tab。
// 契约:api/openapi/admin/promotion.yaml、loy.yaml;接口门禁 menu:userdata。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { TabBar } from '../../../components/business'
import CouponTemplatesTab from './coupons'
import GiftRulesTab from './gift'
import EarnRuleTab from './earn'
import LevelsTab from './levels'
import TasksTab from './tasks'

type TabKey = 'coupons' | 'gift' | 'earn' | 'levels' | 'tasks'

export default function MarketingRulesPage() {
  const t = useT()
  const m = t.pages.marketing
  const [tab, setTab] = useState<TabKey>('coupons')
  return (
    <div>
      <TabBar<TabKey>
        value={tab}
        onChange={setTab}
        tabs={[
          { key: 'coupons', label: m.tabCoupons },
          { key: 'gift', label: m.tabGift },
          { key: 'earn', label: m.tabEarn },
          { key: 'levels', label: m.tabLevels },
          { key: 'tasks', label: m.tabTasks },
        ]}
      />
      {tab === 'coupons' && <CouponTemplatesTab />}
      {tab === 'gift' && <GiftRulesTab />}
      {tab === 'earn' && <EarnRuleTab />}
      {tab === 'levels' && <LevelsTab />}
      {tab === 'tasks' && <TasksTab />}
    </div>
  )
}
