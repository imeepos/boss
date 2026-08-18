# BOSS 服务端 · 3 个月路线图（一步到位版）

> 版本 V1.1（进度更新）｜配套 `docs/contract/{terms,domain-map,fields}.md` 与 `docs/ADR-00*`
> 定位：未来 12 周的**唯一执行参照**。每周开工先读本文件对应周次，避免返工。
> 基线事实：领域数据层（12 域 PGStore + 迁移 000001~000030 + 全量装配）已交付。

---

## 进度快照（截至 V1.1）

| 项 | 状态 | 说明 |
|---|---|---|
| 领域数据层（12 域 PGStore + 迁移 + 装配） | ✅ 完成 | 全量 TDD |
| D1–D8 架构基线决策 | ✅ 落地 | JWT/状态机/契约/序列化/端口互斥 |
| W1 传输层（12 域 REST + RBAC + envelope + /auth/me + 地址） | ✅ 完成 | 全量 handler |
| W2 订单 12 环节状态机 + 端口预占互斥 | ✅ 完成 | `advance` 原语 + DB 原子互斥 |
| W3 出账（含区域调价覆盖）+ 审计异步写 | ✅ 完成 | `GenerateBills` + `pkg/audit` |
| **CI 质量门禁** | ✅ 完成 | `.github/workflows/ci.yml` + `make lint/check` |
| D5 可观测（trace/日志/Prometheus/审计） | ⚠️ 部分 | 审计已落地；trace/指标未挂 |
| **W4 端到端集成测试（真实 PG）** | ✅ 完成 | e2e 全流程/取消/端口释放/审计留痕(`internal/app/e2e_pg_integration_test.go`);阶段3/4 台账写侧 handler 补齐(盘点/差异/换新/调拨审批/扩容/手动释放预占,对齐 oss/asset.yaml);PG=192.168.0.102:25432 |
| 二期 W5（四码扫码闭环+gRPC契约） | ✅ 完成 | quadlink 写侧(VerifyScan/UnbindRequireScan/Reconcile/ResolveConflict)+worker 扫码 handler+admin 对账;gRPC 契约 quadlink/aaa/device/provision v1 已生成 |
| 二期 W6（AAA 停复机+话单） | ✅ 完成 | PGAuthorizer(LOID→套餐带宽/QoS)+Suspend/Resume 即时生效(停机在线无网)+话单双写(PG落库+Kafka boss-cdr)+cmd/aaa PG 装配 |
| 二期 W7（下发/采集/告警） | ✅ 完成 | provision Execute/Fail/Retry(留痕计数)+provisioner 守护进程;device Collector(指标入库+丢包越限 CRITICAL 告警)+collector 入口;修复 AppendMetric 漏 collected_at |
| 二期 W8（环节自动化+Kafka 链路） | ✅ 完成 | Automation 编排(6/7/10/11 自动,人工只收费/扫码)+pkg/events Kafka 状态变更事件(boss-order-events)+app 装配;e2e 验证下单→激活全自动(7 条事件);Kafka 实链路 round-trip 已验(boss-cdr/boss-order-events,Hash 分区保序) |
| **二期 W5–W8 验收** | ✅ 达成 | 下单→激活全自动、人工只扫码/装维;停复机即时生效;话单入账;失败可重试留痕;告警实时 |
| 二期债务：gRPC server 落地 | ✅ 完成 | quadlink/aaa/device/provision v1 四契约服务端实现（internal/app/grpc_*.go）+cmd/server 同进程起 gRPC（:9090）；provision_tasks 补 task_no/order_id/stage_event（迁移 000032）；修复 provision_logs.result 过短（迁移 000033）；真实 PG e2e 四服务全链路（扫码 MATCH 推进环节9/停复机即时/指标告警/入队重试） |
| 二期债务：真实协议执行器 | ✅ 完成 | provisioner 换 TCP 行协议 TelnetExecutor（login/password/apply→OK，BOSS_PROVISION_OLT_*）；collector 换 gosnmp v2c SNMPPoller（OID 可配，BOSS_SNMP_*）；daemon+TELNET 真实 PG+loopback OLT 集成（成功 SUCCESS 留痕/失败 FAILED 留痕） |
| 三期 W9–W12（GIS/分析/压测/上线） | ✅ 完成 | W9:GIS 派生聚合域八级下钻/统计/详情+cmd/gis 事件同步+PostGIS geom 实测。W10:analytics 五大指标(可解释)+热力图+维护一张表+report 周期自动报告(000034,幂等)。W11:/metrics RED 指标+k6 压测实测 9829 请求 0 失败、读 p95=8.55ms(阈值 500ms)。W12:Helm chart 落地(lint 过,5 Deployment+Service+HPA)、回滚演练(000034 down→up 全绿,业务零影响)、上线交接清单(docs/plan/launch-checklist.md)、整体回归 20 包全绿 |

**剩余 3 个月焦点（阶段已前置完成，剩余为业务自动化 + 集成 + 闭环验收）**：
1. 一期收尾：真实 PG 集成测试 + 可观测补挂 + OpenAPI 一致性校验。
2. 二期：四码扫码强制 → AAA 实时计费 → 配置下发/OLT 采集 → 环节自动化。
3. 三期：GIS 孪生 → 经营分析 → 压测/可观测收口 → 整体回归上线。

---

## 0. 架构基线决策（先定死，防止后期大改）

| # | 决策 | 结论 | 依据/落地 |
|:-:|------|------|-----------|
| D1 | JWT 单事实源 | `internal/pkg/auth.Manager` 是唯一签发/校验方；`user` 域只返回认证身份（accountID/username/roleCode），**不产 token** | 本轮重构 `user.Login` 返回 `*LoginResult`；app 层用 `Manager.Sign` 签发 |
| D2 | 状态机单事实源 | `internal/pkg/statemachine.Machine` 是「状态迁移合法」唯一判定；订单 12 环节用表驱动定义；Temporal 只做编排、可后置替换 | ADR-003；副作用走 outbox（`order_stages` 时序已具备） |
| D3 | 传输契约驱动 | handler 一律对齐 `api/openapi/{admin,user,worker}`；响应统一 envelope `{code,msg,data}`；错误码走 `pkg/apitypes.Code` | 路由注册在 `internal/app/http*.go` |
| D4 | 编排引擎降级路径 | 一期**不上 Temporal**，用「状态机 + outbox + 后台 worker」闭环；保留接口位 | ADR-003「代价/备选」；避免双轨漂移 |
| D5 | 横切先挂上 | trace（OpenTelemetry）+ 结构化日志 + Prometheus 指标 + 审计异步写，从 W1 起不后补 | `internal/pkg/{middleware,audit}` |
| D6 | 参数定值 | 下表 `[X]` 参数全部给定默认值，存 `biz_params`/Nacos 可调，验收按默认值 | 见 §2 |
| D7 | 序列化约定 | 领域 struct 直接带 `json` 标签对齐 OpenAPI camelCase；模块化单体不另建 DTO 层（避免映射爆炸），未来拆微服务时再引跨进程 DTO | 本轮 org 域已落地 |
| D8 | 端口预占互斥 | 数据库条件更新（`UPDATE ... WHERE status='IDLE'`）是**唯一权威**互斥原语；单表单行原子写无需 redsync；redsync 仅在跨域多步 Saga（订单+端口同事务）时引入，属编排层而非锁层 | 本轮 `ReserveFirstAvailable` 已落地 |

---

## 1. 时间轴（12 周 = 三期）

| 月份 | 主线（全案三期） | 结束可验收 |
|---|---|---|
| 第 1 月（W1–W4） | 一期**收款闭环**（阶段1/2/5） | 下单→激活人工闭环全流程可走通，可收费可查账 |
| 第 2 月（W5–W8） | 二期**服务闭环**（阶段3/4/6/7） | 认证/授权/计费/下发/扫码强制全自动 |
| 第 3 月（W9–W12） | 三期**经营闭环**（阶段8/9） | GIS 孪生 + 五大指标 + 自动报告，整体回归上线 |

---

## 2. 验收参数定值（[X] 全部填死）

| 参数 | 默认值 | 说明 |
|---|---|---|
| 带宽套餐档 | 100M / 200M / 500M / 1000M | 主流家庭宽带四档 |
| 电子标签频段 | UHF | 长距、行业主流 |
| 认证响应 P99 | ≤ 100ms | RADIUS 达标线 |
| 并发认证能力 | ≥ 10 万用户 | 全案验收 |
| 话单完整率 | ≥ 99.9% | 计费合规 |
| 话单入账时延 | ≤ 5 分钟 | 近实时 |
| 复机生效时延 | ≤ 5 分钟 | 停机用户可感知 |
| 四码反查响应 | ≤ 1 秒 | 单表索引 |
| 四码对账冲突处理 | ≤ 4 小时 | 限期清零 |
| 地图资产详情 | ≤ 2 秒 | 点击详情 |
| 地图状态同步 | ≤ 5 分钟 | 变更联动 |
| 大屏/地图首屏 | ≤ 3 秒 | 加载 |
| 报告生成 | ≤ 10 分钟 | 自动报告 |
| 端口预占有效期 | 30 分钟 | 超时自动释放 |
| 欠费停机阈值 | 欠费天数 ≥ 15 天 | 触发停复机 |
| 数据核对周期 | 每日 03:00 | 低峰 |
| 报告周期 | 日 / 周 / 月 / 季 | 自动推送 |
| JWT TTL | 24 小时 | 登录态 |

> 这些默认值写入 `biz_params`（后续补一条种子迁移），运行时经 Nacos/biz_params 可调，代码不写死。

---

## 3. 第 1 月：传输层 + 一期收款闭环（W1–W4）

| 周 | 交付物 | 验收点 |
|---|---|---|
| W1 | JWT 单事实源重构；`cmd/server` 真正启动 HTTP；统一 envelope + 错误码；三端路由骨架 + 登录/组织域 handler（模板） | 登录成功发 token、越权 403、未登录 401；错误码与 OpenAPI 一致 |
| W2 | order 12 环节状态机（表驱动 + guard）+ 补齐 charge/applyTag/createUserProfile/preConfig/dispatch；端口预占 Redis 锁 | 并发预占仅一单成功；环节按序流转，非法跳转拒 |
| W3 | billing/arrears 业务规则（出账周期/欠费停机/缴费复机）落到 handler；审计异步写 | 账单=成交价快照；欠费触发停复机；缴费可对账 |
| W4 | 端到端集成测试（真实 PG）；阶段3/4 台账 CRUD handler 补齐 | 一期验收：全流程可走通/可取消/端口释放/全程留痕 |

---

## 4. 第 2 月：二期服务闭环（W5–W8）

| 周 | 交付物 | 验收点 |
|---|---|---|
| W5 | quadlink 四码 + 扫码强制 + 拆机必扫码；四码对账任务；gRPC 契约（`api/proto`） | 任一码反查四码；扫码不一致拒；不扫码拆机拦 |
| W6 | `cmd/aaa` RADIUS（LOID 认证/授权/QoS）+ 实时话单→Kafka→billing；停复机即时生效 | 认证达标；停机在线无网；话单入账 |
| W7 | `cmd/provisioner` 模板/下发/重试；`cmd/collector` SNMP/Trap；指标/告警 | 装维零手工；失败可重试留痕；告警实时 |
| W8 | 环节 6/7/10/11 自动化升级；Kafka 状态变更链路打通 | 二期验收：下单→激活全自动，人工只扫码/装维 |

---

## 5. 第 3 月：三期经营闭环 + 收尾（W9–W12）

| 周 | 交付物 | 验收点 |
|---|---|---|
| W9 | `cmd/gis` 消费 `asset.changed` 同步实体；八级下钻 + 瓦片（Cesium ion 自托管） | 八级可钻取；地图与业务一致 |
| W10 | `cmd/report` + StarRocks：五大指标/热力图/维护清单/自动报告 | 指标可下钻可解释；清单排序可解释 |
| W11 | k6 压测 + Prometheus/Grafana/Loki/Jaeger 全链路 + 性能调优 | 关键链路达标（§2） |
| W12 | 对照主文档第九章整体回归 + Helm/K8s 部署 + 回滚演练 + 交接 | 三期整体验收通过 |

---

## 6. 横切规则（贯穿全程）

1. **TDD 延续**：接口测试先写（`httptest` + 域 mock + pgxmock）；集成测试单独标记，不阻塞单测。
2. **契约优先**：`api/openapi` 是唯一接口事实源；进程内 DTO 放 `internal/app`，跨进程放 `pkg/apitypes`。
3. **域不横向 import**：域之间只经接口；跨域依赖（customerLookup/resource.Check）由 `internal/app` 装配。
4. **可观测**：trace/日志/指标 W1 挂上；审计异步写按月分区表（`audit_logs` 已建）。
5. **CI 门禁**：`go build/vet/test` + `golangci-lint` + OpenAPI 校验进 CI；集成测试用 `deployments/docker-compose.102.yml`。

---

## 7. 风险与依赖

| 风险 | 应对 |
|---|---|
| 本机/102 PG 端口绑定失败（已复现） | W1 首项解决环境，否则集成测试被跳过、一期无法验收 |
| Redis/Kafka 未接入 | 一期先 Redis 锁 + outbox；Kafka 在 W6 接入（业网融合硬依赖） |
| Temporal 依赖重 | 一期不上，D4 降级路径；如需长时编排 W2 再决策，不影响状态机语义 |
| complaint（客服工单）域边界 | 一期按「order 内闭环」交付，是否独立 CS 域后续定 |

---

## 8. 本轮明确不做（下一轮迭代）

按 `domain-map.md` 表1「待建」域：发票税务、渠道经销商、批发结算、营收保障、防欺诈、结算互连、忠诚度积分、客户门户、消息通知。三期闭环稳定后再排期。
