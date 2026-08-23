# Q4 开放平台与互操作执行计划

> 对应三年路线图 2028 Q3「Q4 开放平台与互操作」。目标:把稳定的内部能力包装为
> 安全、可版本化、可测试的外部集成能力。本文是该季度的执行分解,进度以 git 提交为准。

## 边界(先读)

- **明确不做**:不以拆微服务作为交付目标;外部系统不得直接写入订单、账务等核心事实表。
- 开放面一律**只读或经内部服务校验写入**,开放路由不直连核心表。
- 契约以 `docs/contract/{terms,fields,domain-map}.md` 为准;新增端点同步 `api/openapi/open.yaml`。

## 里程碑

| 里程碑 | 交付 | 内容 | 状态 |
| --- | --- | --- | --- |
| M1 | 开放 API v1 骨架 | `/api/open/v1` 版本化路由;open_app 凭证(AppId+Secret, HMAC-SHA256 签名);时间戳防重放窗口;每应用 RPM 限流 + 日配额;只读样例端点(订单状态查询) | 已合并(000122) |
| M2 | Webhook 订阅中心 | 事件订阅 CRUD、投递 outbox、HMAC 负载签名、指数退避重试、幂等去重 | 已合并(000125;业务事件挂接见 M5) |
| M3 | 供应商适配器规范 | 支付/短信/实名/税务/地图/设备/AAA 适配器接口规范文档 + 现有 Stripe/SMS 归位对齐 | 已合并(docs/contract/vendor-adapter-spec.md;sandbox fixture 归 M4) |
| M4 | 沙箱环境 | 沙箱 app 凭证隔离、样例数据集、回放样例(请求/响应 fixture) | 已合并(sandbox 双向隔离 + SBX 样例 + Webhook 回放样例 + selftest 脚本) |
| M5 | 集成测试与回放工具 | 契约测试跑 open.yaml、流量回放脚本、集成方自助验收 checklist 工具 | 已合并(selftest 脚本 5 项验收 + openplat-replay.mjs 回放工具与 fixture + 同步守护测试;业务事件 order.stage.done 已挂接) |
| M6 | 开发者门户 + 部署评估 | admin 内开发者门户页面(凭证/订阅/用量)、基于 102 真实负载的独立部署评估报告 | 部分:评估报告已合并(docs/ops/open-platform-deployment-assessment.md,结论不拆+再评估触发线);门户页面未开始 |

## M1 设计要点(本季第一刀)

### 认证:AppId + Secret + HMAC-SHA256 请求签名

- 管理端创建 `open_app`,生成 `AppId`(公开标识)与 `Secret`(仅创建时返回一次)。
- 请求头:`X-BOSS-AppId` / `X-BOSS-Timestamp`(Unix 秒)/ `X-BOSS-Nonce` / `X-BOSS-Signature`。
- 签名串:`appId\nMETHOD\npath\ntimestamp\nnonce\nsha256(body)`,HMAC-SHA256 后 hex。
- 防重放:时间戳偏差 >5 分钟拒绝。v1 不做 nonce 去重存储(单实例窗口内风险可接受,M4 沙箱阶段补)。
- **Secret 必须原文落库**(与内部 api_key 只存哈希不同:HMAC 验签需要服务端持有原文)。
  不可逆决策,见 `docs/notes/adopted/2026-08-22-open-platform-secret.md`。

### 限流与配额

- 限流:每应用 RPM 令牌桶(进程内存,单实例部署下即全局限别);超限 HTTP 429。
- 配额:日调用次数落 `open_usage_day`(按 app×天 upsert 自增);超限 HTTP 429 + `X-BOSS-Quota-Exceeded`。

### 版本化

- 路由前缀 `/api/open/v1`;不兼容变更发 v2,旧版本至少保留两个季度。
- OpenAPI 契约文件 `api/openapi/open.yaml`,纳入 check-contract-sync A 项对账。

## 验收口径(季度出口)

- 外部调用具备认证、授权、限流、审计(last_used/用量)与版本兼容策略。
- Webhook 可重试且不重复执行业务动作(投递幂等键)。
- 至少两类外部系统真实联调(已有:Stripe 支付;短信通道)。
- 集成方可按文档独立完成沙箱验收。
