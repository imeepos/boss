// 券积分对账报表页(2028 Q2 交付):券对账(模板三角)与积分对账(账本 vs 流水)两 Tab。
// 契约:GET /coupon-recon、GET /loy/points-recon(diff=drift 只看差异行)。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { TabBar } from '../../../components/business'
import CouponReconTab from './recon-coupons'
import PointsReconTab from './recon-points'

type TabKey = 'coupons' | 'points'

export default function MarketingReconPage() {
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
          { key: 'points', label: m.tabPoints },
        ]}
      />
      {tab === 'coupons' ? <CouponReconTab /> : <PointsReconTab />}
    </div>
  )
}
