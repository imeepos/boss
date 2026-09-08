// 营销规则启用态徽标:营销域枚举 ENABLED/DISABLED 映射 StatusTag userdata 注册域
// (active/disabled,三语标签已登记),消除状态列裸枚举;不改 components/ 注册表。
import { StatusTag } from '../../../components/StatusTag'

export function RuleStatus({ status }: { status: string }) {
  return <StatusTag domain="userdata" value={status === 'ENABLED' ? 'active' : 'disabled'} />
}
