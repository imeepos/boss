// 工作台占位(A0 不做业务页;A1 起按 admin-a0-plan 迁移 46 页)。
import type { Profile } from '../../api/auth'

export default function DashboardPage({ profile }: { profile: Profile }) {
  return (
    <div>
      <h2>工作台</h2>
      <p>
        欢迎,{profile.realName}({profile.roleCode})。业务页面自 A1 批次起接入。
      </p>
    </div>
  )
}
