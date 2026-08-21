# 后台提醒中心:广播+读回执、30s 轮询、handler 层接入

## 日期

2026-08-25

## 决策

admin 侧提醒中心(notify 域,迁移 000090)三项不可逆裁定:

1. **广播 + 每账号读回执**,不做逐人投递箱:`admin_notifications` 面向角色广播(`target_role` 空=全员),`admin_notification_reads` 按账号记已读。
2. **30s 轮询拉取**(未读数 + 铃铛下拉),不建 SSE/WebSocket 长连接。
3. **来源域在 httpapi/app 层接入**(Emit/Resolve 调用点放 handler 与 patrol/daemon 接线处),域之间不互相 import notify;provision daemon 经 `OnDone` 回调注入。

## why

1. 待办是角色职责不是私人信件,一人已读不应让全员消失;后台账号量小(数十级),投递扇出的存储与一致性成本不值得。
2. 后端无长连接基建;提醒实时性要求低(分钟级),轮询实现面最小、随连接断开自动降级。
3. 依赖倒置(ADR-001):域间只经接口依赖。notify 是横切读模型,让 6 个来源域各自 import notify 会造成耦合扇入;handler 层已知终态上下文(结果数、操作人),接入点天然在那里。

## 放弃了什么(被否决项)

- 逐人投递箱(delivery 行/账号):被否,理由见上;如未来需要定向通知再增 `target_account_id` 列演进。
- SSE/WebSocket 推送:被否,后端无长连接基建,引 gin SSE 需要新的连接治理;轮询 30s 的 QPS(后台在线账号×2/min)可忽略。
- 领域服务内直接 Emit:被否,保持域纯净;代价是只有经 handler/编排层的路径才发通知(绕过 handler 的内部直调不发),可接受。

## 关联

- 设计:docs/plan/admin-notify-center.md
- 契约:docs/contract/fields.md §7.4 admin_notifications;docs/contract/domain-map.md NOT 行
