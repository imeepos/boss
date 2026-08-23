# Q2 渠道与伙伴协同验收矩阵

> 范围：伙伴入驻、企业工作台、渠道订单、佣金结算、受限 API key、渠道审计与基础反作弊。
> 明确不做：跨运营商批发结算、复杂渠道融资。

## 交付矩阵

| 交付项 | 实现证据 | 自动化证据 | 状态 |
|---|---|---|---|
| 伙伴入驻审核流程 | `internal/domain/partner/pg_apply.go`；`POST /partner/applications`；`/approve`、`/reject` | `TestPartnerApprovalIntegration`（真实 PostgreSQL） | PASS |
| 企业工作台 | `internal/httpapi/admin/partner_handlers.go`；`/partner/me`、`/staff`、`/orders` | `partner_handlers_test.go` | PASS |
| 区域权限 | `internal/domain/partner/pg_region.go`；`GET/PUT /partner/region-scope`；LTREE 过滤订单/员工 | `go test ./internal/domain/partner ./internal/httpapi/admin` | PASS |
| 渠道订单接口 | `partner_order_handlers.go`；`POST /partner/orders` | `partner_order_handlers_test.go`；直营订单 E2E 12 环节 | PASS |
| 12 环节、资源预占、计费复用 | `internal/domain/order/pg.go`、`pg_workflow.go` | `TestE2E_OrderLifecycle_Integration` | PASS |
| 佣金结算台账 | `internal/domain/partner/pg_commission.go`；`GET /partner/commissions`、`/settle` | `TestPartnerCommissionLedgerIntegration`（真实 PostgreSQL） | PASS |
| 佣金自动计提 | `UpdateMap` 完成后对 `AGENT` 订单计提；比例读取 `biz_params` | 佣金单测及台账 E2E | PASS |
| API key 权限模板 | `internal/domain/apikey`；`internal/pkg/middleware/auth.go`；`partner-orders-read` | `TestAuthzRestrictedPartnerTemplate` | PASS |
| 渠道审计报表 | `internal/domain/partner/pg_audit.go`；`GET /partner/audit-report` | handler 回归 | PASS |
| 异常订单拦截/反作弊 | `pg_partner_fraud.go`；每日上限、客户冷却、客户/产品租户校验 | `TestPartnerTenantLockIntegration`（真实 PostgreSQL）及风险单测 | PASS |

## 前端交付证据

伙伴前端页面已纳入 `web/admin` 路由与菜单：

- `/partner/apply`：公开入驻申请
- `/org/partner`：入驻审核
- `/partner/home`：我的企业
- `/partner/staff`：员工管理
- `/partner/orders`：企业订单

实现证据：

- `web/admin/src/api/partner.ts`
- `web/admin/src/pages/partner/apply.tsx`
- `web/admin/src/pages/org/partner/index.tsx`
- `web/admin/src/pages/partner/home/index.tsx`
- `web/admin/src/pages/partner/staff/index.tsx`
- `web/admin/src/pages/partner/orders/index.tsx`
- `web/admin/src/router/menu.def.ts`

前端门禁已通过：

```bash
pnpm --dir web/admin run typecheck
pnpm --dir web/admin run test -- --runInBand
pnpm --dir web/admin run build
```

结果：typecheck 通过；45 个测试文件、247 个测试通过；生产构建通过。文档中的 28080 是后端 API 入口；按 `deployments/docker-compose.102.app.yml`，前端实际入口为 `http://192.168.0.102:5180`。已用 CDP 在 5180 验证 `/login` 与 `/partner/apply` 返回 200，伙伴申请页面加载 `apply` 与 `partner` chunks；表单输入事件也已验证（2 个输入值成功写入）。

## 已执行命令

```bash
export PATH=/opt/homebrew/bin:$PATH
go test ./internal/domain/partner ./internal/domain/order ./internal/pkg/middleware ./internal/httpapi/admin ./internal/app

BOSS_PG_TEST_DSN='host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable' \
  go test ./internal/domain/order -run TestPartnerTenantLockIntegration -v -count=1

BOSS_PG_TEST_DSN='host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable' \
  go test ./internal/domain/partner -run 'TestPartner(Approval|CommissionLedger)Integration' -v -count=1
```

结果：聚焦 Go 测试通过；租户行锁、伙伴审批、佣金台账真实 PostgreSQL 测试通过。

## 尚未纳入本矩阵的增强项

- 伙伴 HTTP/PG 全链路单个测试用例尚未把入驻审批后的新账号登录、区域设置、渠道下单、12 环节推进、自动佣金和审计报表全部串成一次旅程。
- 风控每日上限的并发订单创建尚未做压力级集成测试；已有租户行锁阻塞验收。
- 102 后端 API 根路径 `/` 返回 404 属于预期，因为前端入口按部署编排位于 `:5180`；已完成 `/login` 与 `/partner/apply` 的页面加载及表单输入级 CDP 验证。
- 尚未使用真实伙伴账号完成 5180 登录后的 `/partner/home`、`/partner/staff`、`/partner/orders` 点击验收；需要安全的真实账号/一次性凭证后继续。
