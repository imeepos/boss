# Q1 契约基线冻结清单

> 冻结日期:2026-08-23 | 基线 commit:`188d9d4`(main)
> 定位:Q1"冻结订单 12 环节、状态、字段、错误码和三端契约"的裁定文书。
> 本文件不改已冻结内容;任何变更走 §5 冻结规则。

## 1. 冻结对象与权威源

| 冻结对象 | 权威源 | 规模(基线) |
|----------|--------|------------|
| 订单 12 环节 | docs/contract/terms.md §1(V1.0) | 12 环节,序号/名称/标识/前置/执行方固定,禁止增删改序 |
| 订单状态枚举 | terms.md §3 | PENDING/RESERVED/INSTALLING/DONE/CANCELLED(与环节正交) |
| 通用状态枚举 | terms.md §4 | 30+ 域状态码集(端口/资产/四码/工单/缴费/发票等) |
| 页面字段命名 | docs/contract/fields.md | 页面列名↔字段名↔状态枚举三列对齐,全局强制 |
| 能力域命名 | docs/contract/domain-map.md | 能力域/Agent/internal包/admin页面四套对齐 |
| 错误码 | pkg/apitypes/code.go | 11 个跨进程错误码(§3) |
| Admin OpenAPI | api/openapi/admin/*.yaml(25 文件) | 275 operationId |
| User OpenAPI | api/openapi/user/*.yaml(10 文件) | 81 operationId |
| Worker OpenAPI | api/openapi/worker/*.yaml(7 文件) | 60 operationId |
| gRPC/事件 | api/proto/, internal/pkg/events | 跨进程消息 envelope |

## 2. 主链路端点冻结(验收脚本依赖)

| 环节 | 端点(admin) | 状态 |
|------|-------------|------|
| 1 下单 | POST /orders(可选 requestId,000116) | 冻结 |
| 2 核查 | POST /orders/:orderNo/check-resource | 冻结 |
| 3 预占 | POST /orders/:orderNo/reserve | 冻结 |
| 4-8 | POST /orders/:orderNo/charge(自动段5-8) | 冻结 |
| 8 指派 | POST /dispatch/pool/:ticketNo/assign | 冻结 |
| 9 扫码 | POST /tickets/:ticketNo/scan-bind | 冻结 |
| 10-12 | POST /tickets/:ticketNo/activate(自动段11-12) | 冻结 |

## 3. 错误码冻结(append-only)

| 码 | 常量 | 语义 |
|----|------|------|
| 0 | CodeOK | 成功 |
| 40100 | CodeUnauthorized | 未认证或凭证无效 |
| 40300 | CodeForbidden | 无权限 |
| 40400 | CodeNotFound | 资源不存在 |
| 40900 | CodeConflict | 资源冲突 |
| 40910 | CodeStateInvalid | 状态机非法流转 |
| 40920 | CodeScanMismatch | 四码合一扫码不一致 |
| 42200 | CodeInvalidParam | 参数非法 |
| 42300 | CodeResourceBusy | 资源占用(端口预占冲突等) |
| 50000 | CodeInternal | 内部错误 |
| 50200 | CodeDownstreamErr | 下游依赖错误 |

**规则**:已冻结码的数值与语义不可变;新增码 append-only 且须更新本表与
code.go 同提交;禁止复用废弃码。

## 4. 冻结基线验证命令

```bash
make contract-sync                 # 契约↔实现同步门禁(路由登记/撞号/行数)
node scripts/gen-bossctl-routes.mjs  # openapi -> bossctl 路由目录生成
bossctl -server http://192.168.0.102:28080 --api-key <KEY> routes  # 线上 390 路由对账
```

## 5. 冻结规则

1. 基线内容变更 = 契约变更,须独立提交,带 fields.md/terms.md 同步 +
   alignment-audit.md 记 D/A/B/E/G# finding。
2. 兼容性只允许:新增可选字段、新增端点、新增错误码(append-only);
   破坏性变更(rename/删除/语义反转)一律走版本化讨论并写 adopted note。
3. check-contract-sync 门禁红的基线豁免仅限已登记存量(见脚本)。
4. 基线随季度滚动复审:Q2 计划期允许在新的 adopted note 保障下演进。

## 6. 基线证据

- 主链路验收: docs/acceptance/2026-08-23-mainchain-102.md(10/10)
- 幂等审计: docs/acceptance/2026-08-23-write-idempotency-audit.md(10/10)
- 回滚/备份演练: docs/acceptance/2026-08-23-rollback-backup-drill.md
- 线上路由实测: 390 条(bossctl routes @102,2026-08-23)
