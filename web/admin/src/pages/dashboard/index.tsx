// 工作台占位(A0 不做业务页;A1 起按 admin-a0-plan 迁移 46 页),文本走 i18n。
import type { Profile } from '../../api/auth'
import { useT } from '../../i18n'

export default function DashboardPage({ profile }: { profile: Profile }) {
  const t = useT()
  const welcome = t.pages.dashboard.welcome
    .replace('{name}', profile.realName)
    .replace('{role}', profile.roleCode)
  return (
    <div>
      <h2>{t.pages.dashboard.title}</h2>
      <p>{welcome}</p>
    </div>
  )
}