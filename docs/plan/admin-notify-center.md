# 后台提醒中心设计(admin 通知 + 待办)

> 状态:设计定稿,待实现。2026-08-21 与需求方确认:6 个事件来源全接;TopBar 铃铛与消息中心页签同批交付。
> 契约对齐:docs/contract/terms.md(消息 level 枚举)、docs/contract/fields.md(补"后台提醒"节)、docs/contract/domain-map.md(NOT 域补 admin 投影)。

## 1. 背景与边界

现状:

- `web/admin` 消息中心页(`/boss/message`,`src/pages/boss/message/`)只覆盖**师傅侧**消息与公告,是"给师傅发消息"的编辑视角。
- TopBar 有一个纯装饰通知按钮(`t.shell.notifications`),无数据。
- 后端有 Kafka 事件底座(`internal/pkg/events`、report 域快照推送),无任何 admin 侧通知/待办实体。迁移已到 000089。

本设计新增 **notify 域**(internal/domain/notify):面向后台账号的"收提醒"视角。两类载荷,一张表:

- **todo(待办)**:需 admin 处理的事项,带跳转 link,来源域完成后标记 resolved。
- **task(任务结果)**:后台异步任务终态通知,成功 INFO、失败 WARN/URGENT。

不做(一期明确排除):

- 站内信编辑(现有 worker 消息页承担);邮件/IM 外发(Kafka 信封预留,消费方二期自建)。
- 逐人定向投递、订阅配置、通知清理策略(量级小,二期 retention)。
- WebSocket/SSE 长连接推送(后端无基建,30s 轮询够用)。

## 2. 数据模型(PG,迁移 000090)

```sql
-- 通知主体(面向角色广播,不逐账号复制)
CREATE TABLE admin_notifications (
    id           BIGSERIAL PRIMARY KEY,
    category     TEXT NOT NULL,              -- todo | task
    level        TEXT NOT NULL,              -- INFO | WARN | URGENT(复用 terms.md 枚举)
    title        TEXT NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    link         TEXT NOT NULL DEFAULT '',   -- 前端路由,如 /boss/worker-reg
    ref_type     TEXT NOT NULL DEFAULT '',   -- 来源域标识,见 §4 表
    ref_id       TEXT NOT NULL DEFAULT '',   -- 来源域主键(字符串,跨域兼容)
    target_role  TEXT NOT NULL DEFAULT '',   -- 空=全部后台角色;否则 RoleCode
    resolved     BOOLEAN NOT NULL DEFAULT FALSE, -- todo 专用
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_admin_notif_feed ON admin_notifications (target_role, created_at DESC);
CREATE INDEX idx_admin_notif_ref ON admin_notifications (ref_type, ref_id);

-- 每账号读状态(独立表,广播通知不被一人已读全员消失)
CREATE TABLE admin_notification_reads (
    notification_id BIGINT NOT NULL REFERENCES admin_notifications(id) ON DELETE CASCADE,
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (notification_id, account_id)
);
```

设计裁定(写 adopted note,见 §8):

1. **广播 + 读回执,不做逐人投递箱**:待办是角色职责不是私人信件;后台账号量小,投递扇出不值。
2. **resolved 与 read 分离**:resolved 是 todo 的生命周期(来源域驱动),read 是人的视角(账号驱动)。前端默认隐藏 resolved 未读。
3. **幂等**:生产者以 `(ref_type, ref_id, category)` 查重,任务类通知只发一次;todo 重复 Emit 为幂等 no-op。

## 3. API(internal/httpapi/admin/notify.go)

前缀 `/api/admin/v1`,信封契约照旧(HTTP 恒 200,code=0 取 data):

```
GET  /notifications?category=&level=&unread=&page=&pageSize= → { items, total }
GET  /notifications/unread-count                              → { count }
POST /notifications/read        { ids: number[] }             // ids=[] = 全部已读
```

- 列表/未读数按 `target_role IN ('', 当前账号角色)` 过滤。
- items 字段:`id/category/level/title/content/link/refType/refId/resolved/createdAt/read`。
- 写入侧不走 HTTP:领域服务同进程调 `notify.Service.Emit(ctx, Input)`;todo 终结调 `notify.Service.Resolve(ctx, refType, refID)`(把匹配未 resolved 行批量置 true)。

## 4. 事件来源(6 个,一期全接)

| 来源 | 时机 | category | level | ref_type | link |
|---|---|---|---|---|---|
| importer 导入任务 | 终态回写(失败 WARN/成功 INFO) | task | 按结果 | `importer` | `/base/importer` |
| 师傅注册审核 | 新注册待审 | todo | WARN | `worker_reg` | `/boss/worker-reg` |
| provision 下发 | 批量下发终态 | task | 按结果 | `provision` | `/provision/provlog` |
| report 快照 | Push 成功后 | task | INFO | `report` | `/intel/report` |
| billing 出账 | 出账批次完成/失败 | task | 按结果 | `billing` | `/billing/billing` |
| 报障工单超时 | 定时巡检 | todo | URGENT | `complaint` | `/boss/complaint` |

接入方式:领域服务直调 `notify.Service`(事务一致,简单);Kafka 仅旁路留痕(对齐 report 域"留痕为权威、推送尽力而为")。todo 的 resolved 回调:

- `worker_reg`:审核通过/驳回时 Resolve。
- `complaint`:工单关闭时 Resolve。

## 5. 前端设计(web/admin)

入口 A:TopBar 铃铛(现有装饰按钮激活)

- `src/lib/useNotifications.ts`:登录后 30s 轮询 `unread-count`;URGENT 未读时铃铛红点徽标。
- 点击展开下拉(复用 `Dropdown.tsx` 浮层骨架:点击外部收起):最近 10 条(level 色点 + 标题 + 相对时间),底部"查看全部 / 全部已读"。
- 点条目 → 标记已读 + navigate(link)。

入口 B:消息中心页第三页签「后台提醒」

- `src/pages/boss/message/AdminNotifsTab.tsx`:列表 + category/level/unread 筛选 + 分页复用 `components/Pagination.tsx` + `useQueryState`(刷新条件不变,与 geo 页同款)。
- todo 行操作「去处理」跳 link;resolved 行灰显。
- 顺手迁移本页存量 inline style/硬编码色(#1677ff、#888)为 tailwind + `--shell-*` 令牌(红线:禁止硬编码色)。

硬性约定:

- 文案全走 i18n,加 key 同步 4 处(types.ts + zh-CN/en-US/ms-MY 三份 locale)。
- level 色点用组件级 `[data-theme]` 双令牌块(参照 geo.css 模式),引用任何 var(--x) 前 grep tokens.css/styles.css。
- 相对时间格式化沉淀 `src/lib/format.ts`。

## 6. 权限与菜单

- 复用现有 `menu:dispatch` 权限(消息中心页现挂它);铃铛与页签同权限,**无需新迁移授权**(sysadmin 无隐式全权,新菜单才需要 permissions+role_permissions 两步)。
- 后续如拆独立权限 `menu:notif`,再走新迁移(两步缺一不可)。

## 7. 提交拆分(revert 可行硬约束)

1. `feat(notify)`: 迁移 000090 + internal/domain/notify(Service/pg/测试,内存实现供单测)
2. `feat(api)`: httpapi/admin/notify.go 三端点 + e2e
3. `feat(notify)`: 6 个来源域接入 Emit/Resolve(可按域拆子提交)
4. `feat(web)`: 铃铛 + useNotifications + AdminNotifsTab + 存量样式迁移 + i18n 三语言
5. `docs(contract)`: fields.md/domain-map.md/terms.md(如需)同步 + adopted note

每步门禁:后端 `go test ./...`;前端 `pnpm typecheck && pnpm test && pnpm build`;UI 交付前双主题截图(cdp-capture)+ 真实 102 环境 DOM 断言。

## 8. 决策记录(落地当天写 dated note 到 docs/notes/adopted/)

- 广播+读回执 vs 逐人投递箱 → 选广播+读回执,放弃投递扇出(why:待办是角色职责;账号量小)。
- 轮询 vs SSE/WebSocket → 选 30s 轮询,放弃长连接(why:后端无长连接基建,提醒实时性要求低)。
- 通知保留策略 → 一期不做,二期 retention(放弃自动清理,接受量级可控的暂时膨胀)。
