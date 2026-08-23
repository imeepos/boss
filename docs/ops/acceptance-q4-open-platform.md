# Q4 开放平台与互操作——季度验收报告

> 时间:2026-08-22｜季度:Q4(对应三年路线图 2028 Q3「Q4 开放平台与互操作」)
> 验收人:round 1–7 协作流程｜环境:本地(完整链路)+ 102 http://192.168.0.102:28080(部分)
> 执行计划:`docs/plan/q4-open-platform-plan.md`(六里程碑全部已合并)
> 红线复查:外部系统不得直接写入订单、账务等核心事实表 ✅ 开放面只读;不以拆微服务为交付目标 ✅ 维持单体内第四端。

## 1. 六项季度交付物——逐项核验

| # | 交付物 | 落点 | 证据 | 状态 |
|:-:|:-------|:-----|:-----|:-----|
| 1 | 开放 API v1 | `/api/open/v1`(ping/orders/sandbox)+ HMAC-SHA256 验签 + 每应用 RPM 限流 + 日配额 | 000122 迁移;契约 `api/openapi/open.yaml`(已纳入 checker A 项第四端);7 例中间件测试 | ✅ |
| 2 | Webhook 订阅中心 | outbox 投递 + 30s×2ⁿ 退避(封顶 1h,6 次死信)+ 事件幂等(UNIQUE subscription_id+event_id)+ `order.stage.done` 业务挂接(advance 原语幂等键 orderNo+stage) | 000125 迁移;admin 端 6 端点(列表/requeue/test-event)+ WebhookDispatcher 单测 7 例 | ✅ |
| 3 | 开发者门户 | `/org/openplat`(应用凭证 CRUD + 订阅管理 + 投递视图 + 测试事件 + Secret 一次性 toast) | menu:openplat 权限码(000122);web 247 tests + build + DS adoption 100% | ✅ |
| 4 | 沙箱环境 | sandbox 应用与生产数据双向隔离 + SBX-* 样例订单(四态覆盖)+ Webhook 回放样例(固定 ts 可复算)+ `sandbox/samples` 端点 | `internal/domain/openplat/sandbox.go`;契约 `openapi/open/sandbox.yaml`;4 例样例/隔离测试 | ✅ |
| 5 | 供应商适配器规范 | `docs/contract/vendor-adapter-spec.md`:七原则 + 接口形状 + 错误四分类 + 新增 checklist + 现有 7 类系统对齐评估 | 已在 contract/README.md 索引;plan M3 已合并 | ✅ |
| 6 | 集成测试与回放工具 | `scripts/openplat-selftest.mjs`(零依赖 Node,5 项验收:ping/样例清单/样例查询/隔离 404/Webhook 验答)+ `scripts/openplat-replay.mjs`(6 例 fixture + 同步守护)+ `docs/integration/open-platform.md`(集成方自助文档) | e2e 端到端(httptest + node)双绿;Go/Node HMAC 字节级一致 | ✅ |

## 2. 重点任务——逐条勾选

- ✅ 版本化开放 API:`/api/open/v1` 前缀,checker 第四端对账;v1→v2 不兼容策略写进接入指南
- ✅ Webhook/事件订阅:订阅 CRUD + outbox + 指数退避 + 死信 requeue
- ✅ 签名校验:HMAC-SHA256 请求签名(±5min 时间戳)+ Webhook 负载签名 v1;集成方自助工具复算字节级一致
- ✅ 限流与配额:进程内令牌桶 + 日配额 + 计数(`open_usage_day`)
- ✅ 外部系统对接:现有 Stripe/SMS/RealID/Tax Gateway/SNMP profile 已合规(规范 §4 评估表);开放面与现有供应商隔离(供应商侧规范,集成方接入面)
- ✅ 沙箱与契约测试:见 #4 #6
- ✅ 回放样例与自助验收:`scripts/openplat-replay.mjs` + 6 例 fixture + 同步守护 + 集成方接入指南
- ✅ 独立部署评估:`docs/ops/open-platform-deployment-assessment.md`——基于 102 k6 压测结论**不拆**,四条再评估触发线

## 3. 证据汇总

| 维度 | 来源 | 结果 |
|:-----|:-----|:-----|
| Go 单元/集成 | `go test ./... -race -count=1` | 全绿(具体数随本轮变更) |
| Go 契约对账 | `make contract-sync` | A/B/C/D 四项 OK,426 路由全登记 |
| Web 类型 | `pnpm typecheck` | OK |
| Web 单测 | `pnpm test` | 247 passed(45 文件) |
| Web 构建 | `pnpm build` | OK(产物 ~1.3MB gzipped) |
| 设计系统采用 | `node scripts/check-ds-adoption.js` | 阈值 80%,低于 0(100% DS) |
| 开放面 e2e | `TestSelftestScriptAgainstOpenRouter` | 5/5 PASS |
| 回放工具 e2e | `TestReplayToolAgainstOpenRouter` | 6/6 PASS |
| 沙箱隔离回归 | `TestSelftestScriptRejectsBadSecret` | 必失败(Secret 不符) |
| 样例/脚本同源 | `TestReplayFixturesMatchSandboxSamples` | fixture 与编译期样例一致 |
| 签名跨语言 | `gov1` vs `node_v1` | 字节级一致 |
| 102 端点存活 | `curl /api/open/v1/ping` | 401 missing signature headers(=我们的中间件) |

## 4. 验收结论

季度六项交付物全部落地于主树,本地端到端验证全绿,102 端点探测确认开放面路由已部署。**Q4 开放平台与互操作季度目标达成**。

## 5. 跟进事项(下一发布列车)

1. **102 二进制与迁移的同步**:本轮探测发现 `/api/admin/v1/openplat/apps` POST 在 102 返回旧校验错误(管理面读路径正常),疑似二进制领先于代码的某个中间状态。验证脚本:用 admin JWT 创建 sandbox 应用 → 跑 selftest/replay。若二进制已是最新则应通过;失败则联系发布负责人重部署(`scripts/deploy-cluster.sh`)。该事项不影响季度交付物落点,只影响 102 真实环境验收。
2. 开放面真实负载触发线纳入 patrol 巡检观察项(评估报告 §4)。
3. 接受 Stripe + SMS 两类外部系统真实联调已在 M1 之前就绪,作为本季度"对接外部系统"的实证据。

## 6. 不做清单遵守

- ❌ 未拆微服务:open 端作为单体内第四端,域包 `openplat` 自治
- ❌ 外部系统未直写核心事实表:开放面只读(`/orders/{no}` 投影);Webhook 投递只到应用自有端点
