# admin 契约对齐复核 · Agent 提示词

> 用法：把下面整段发给执行 Agent（或新会话）。它自包含全部事实源与验收口径，不依赖本会话上下文。

---

【能力域】SYS【Agent】AG-01【internal 包】user【阶段】1【页面】admin 契约复核

## 角色与任务

你是 BOSS 项目的契约审计员。任务：对管理后台 REST 契约（`api/openapi/admin.yaml` + `api/openapi/admin/*.yaml`）与 Go 后端实现（`internal/app/http_*.go`）做一次**独立、从零开始**的全量对齐复核。不信任任何既有文档的"已对齐"声明，一切以代码为准。

## 单一事实源（只读不猜，冲突时以代码为准）

1. Go 路由实现：`internal/app/http.go`（总装 + auth）+ `http_{org,order,billing,customer,resource,scan,asset,aaa,device,gis,provision,quadlink,worker}.go`
2. 响应 envelope：`internal/app/http.go` 的 `respond/respondErr` + `pkg/apitypes`（错误码）
3. 字段与枚举：`docs/contract/{terms,fields,domain-map}.md` + Go 枚举定义
4. 契约文件：`api/openapi/admin.yaml`（聚合）+ `api/openapi/admin/*.yaml`（17 个域文件）

## 已知基线（供核对，若与代码不符以代码为准并报告）

- 前缀 `/api/v1`（三端共用）；envelope `{code,msg,data}`，HTTP 恒 200，code=0 成功
- `/auth/me` 返回 roleCode（无菜单树）；`/auth/logout`、`/menu-perms`、sys 域 accounts/roles/params/audit-logs/import-tasks 后端未实现
- 契约应已覆盖 Go 全部 75 个路径；inline schema 描述 data 负载；列表 data 为裸数组

## 复核步骤（按序执行，每步留证据）

### S1 路由清单提取
从 `internal/app/http*.go` 提取全部路由注册（注意 `g.Group("/orders")` + `ord.GET("")` 的空后缀拼接、`requirePerm` 的 permCode），产出 `METHOD 路径 permCode` 三列清单。

### S2 契约路径提取
解析 `api/openapi/admin.yaml` 聚合 paths（149 条预期），并列出每个 path 的 methods 与 operationId。

### S3 三向差异比对
- Go 有、契约无 → 补契约（含 summary 标注"对齐 Go 实现 + permCode"）
- 契约有、Go 无 → 确认是否已标注"未实现"；未标注则补注
- 双方都有 → 核对：路径参数名/类型（:id 形式 vs {xx}）、HTTP method、query 参数名（对照 handler 里 `c.Query(...)`）

### S4 响应形状抽查（已实现端点）
对每个已实现 GET 端点，读 handler 的 `respond(c, CodeOK, X)` 确认 X 是裸数组、对象还是分页结构；与契约 inline schema/`List`/`Item` 引用比对。重点：分页参数（page/pageSize?）是否在契约声明。

### S5 引用与语法完整性
- 全部 YAML `yaml.safe_load` 通过
- 聚合文件每条 `$ref` 指向的域文件 path 存在（注意 `~1`/`~0` 转义）
- `schemas.yaml` 共享 responses/parameters 被 domain 文件引用的名称全部存在
- operationId 全局唯一

### S6 字段级抽查（5 个代表端点）
`/orders`、`/bills`、`/assets`、`/ports`、`/cdrs`：handler 返回 struct 的 json tag 逐字段与契约 schema 比对，命名/类型/枚举不一致即改契约。

## 修正规则

- 永远改契约对齐实现，不改实现迁就契约（除非实现明显违反 `docs/contract/terms.md`，此时只报告不改）
- 单文件 ≤300 行；summary 含 `: ` 必须加引号（存量曾因此解析失败）
- 未实现端点标注格式：summary/description 加"未实现:Go 后端暂未提供"
- 改聚合文件 ref 时同步改域文件，反之亦然

## 验收（全部满足才算完成）

1. S1~S6 每步有命令输出证据（不是口头声称）
2. Go 全部路径 100% 在契约中，且 method/参数名一致
3. 全部 YAML 解析通过 + 全部 $ref 可解析 + operationId 无重复
4. 抽查的 5 个端点字段零差异，或差异已修正并列出清单
5. 产出复核报告：差异清单（修正前→后）、未实现端点清单、遗留风险

## 输出格式

```
## 复核结论（通过/有差异）
## S1 Go 路由清单（N 条）
## 差异与修正（逐条：文件/位置/修正前/修正后）
## 未实现端点清单
## 验证证据（命令 + 输出摘要）
## 遗留风险与建议
```
