# 架构设计打磨记录（Architecture Review）

> 目标：从 ≥3 个不同方向对模块化单体架构做系统性评审，每方向找出 ≥3 个升级优化点，并落地到实际产物。
> 基线：`README.md`、`docs/archive/技术栈方案-一步到位.md`、`docs/ADR-001-monorepo.md`、`internal/`、`migrations/`、`deployments/`。

评审结论以「发现 → 决策 → 落地物」三步记录；每轮的决策只允许落到真实产物（代码 / SQL / 文档），不留空话。

---

## 第 1 轮 · 数据与存储方向

评审对象：`migrations/*.sql`、`internal/pkg/database`、`config`、迁移/分区/索引策略。

### 发现 1.1 · 审计分区表缺未来分区，当月跨月无分区即写入失败
- **问题**：`audit_logs` 为 `PARTITION BY RANGE (created_at)`，但只建到了 `audit_logs_2025_08`，且建表粒度是「单月一个分区」。到 9 月 1 日若无人手动建下月分区，所有审计 INSERT 直接报错（无 DEFAULT 分区承接），违反阶段 1 验收「审计记录完整」。
- **决策**：改为「按月 + 外层 DEFAULT 分区兜底」的分区策略，避免跨月窗口丢审计；文档明确分区维护职责。
- **落地物**：`migrations/000001_stage1_base.up.sql`（审计区）补 `audit_logs_default`；新增 ADR 说明。

### 发现 1.2 · `level` 是冗余列，且无 CHECK 约束，会与 ltree 真实层级漂移
- **问题**：`addresses.level`（SMALLINT 1..5）与 `path` 的 ltree 层级数（`nlevel(path)`）表达同一事实，属冗余存储，会产生 `level=3` 而 `nlevel(path)=5` 的不一致；且缺少 `CHECK (level BETWEEN 1 AND 5)`，可能写入非法层级。此外缺「父必挂接」的一致性保证。
- **决策**：`level` 保留为**查询冗余**（避免每次 `nlevel()` 计算，利于分区/统计），但强制一致性：加 `CHECK (nlevel(path) = level)` + `CHECK (level BETWEEN 1 AND 5)`。文档标注为冗余列及一致性来源。
- **落地物**：`migrations/000001_stage1_base.up.sql`（addresses 区）加约束。

### 发现 1.3 · `parent_id` 与 `path` 未强制一致，缺自引用校验
- **问题**：`addresses.parent_id` 与 `path` 的父路径可互相矛盾（`parent_id` 指向 A，`path` 却是 B 的下层），导致「下钻/回退」语义错乱，直接威胁阶段 8 八级钻取的准确性。
- **决策**：以 `path` 为唯一权威（`uq_addresses_path` 唯一），`parent_id` 由 `path` 派生（`subpath(path, 0, -1)` 反查），不再允许应用层手填；迁移中加触发器或至少在文档中约束「parent_id 由 path 派生，插入时校验」。为避免引入复杂触发器，本轮以文档 + 服务层契约约束，触发器留在阶段 1 实现时补。
- **落地物**：ADR-002 记录「path 权威、parent_id 派生」契约；`user/service.go` 的地址接口注释同步。

### 发现 1.4 · 事务/迁移未建立「分区维护」与「预热数据」的运维约定
- **问题**：迁移对角色/权限没有 `INSERT ... ON CONFLICT` 的种子数据或说明，7 类角色的标准码散落在注释里，缺少 source-of-truth；阶段 1 验收要求 7 角色可登录，但没有明确的初始化路径。
- **决策**：在迁移中固化 7 角色 + 最小权限种子（幂等），使「7 类角色可登录」可重复演示；文档标注后续追加权限的规范。
- **落地物**：`migrations/000001_stage1_base.up.sql` 追加种子数据。

### 第 1 轮小结（≥3 点达成：1.1 / 1.2 / 1.3 / 1.4）

---

## 第 2 轮 · 通信 / 契约 / 可观测方向

评审对象：`api/`、`internal/pkg/{middleware,auth,server}`、`cmd/*`、Temporal/Kafka 契约、可观测。

### 发现 2.1 · gRPC 契约树为空，README 声称「契约先行」但无落地
- **问题**：ADR-001 第 4 条要求「契约先行」，但 `api/proto/` 只有 README，跨服务（quadlink/aaa/device）无任何 `.proto`；阶段 7 的独立部署物（aaa/collector/provisioner）将来无法对齐契约。
- **决策**：补一份 `api/proto/README.md` 之外的**契约组织规范**，明确每个跨服务域的 proto 模块与版本策略；落地最少一份骨架契约（`boss/common/v1` 错误码/分页）作为「契约先行」的示范。
- **落地物**：`api/proto/boss/common/v1/common.proto`（骨架）。

### 发现 2.2 · JWT 无 `kid`/轮换、`Verify` 不校验 `Claims.Valid()`，存在签名算法/过期漏洞
- **问题**：`auth/jwt.go` 的 `Verify` 只查 `t.Valid`，未显式调用 `claims.Valid()`（`t.Valid` 内部会调，但 code 侧语义含糊）；HS256 单密钥无轮换、无 `kid`，密钥泄露 = 长期风险；`ttl` 无上限约束。
- **决策**：显式调用 `claims.Valid()` 校验 `iat/exp`，服务端校验 `Issuer`；预留 `kid` 字段到 `Claims`（多密钥轮换）但本轮不引入多签名方，只落地更严谨的校验 + 文档说明轮换诉求。
- **落地物**：`internal/pkg/auth/jwt.go` 增强校验；ADR 记录。

### 发现 2.3 · 无统一错误码 / 无 trace 注入 / 无健康检查的可观测契约
- **问题**：`middleware/auth.go` 用散落的 `gin.H{"code":401}`，各层无统一 `code` 枚举（`pkg/apitypes` 为空），跨服务无法对齐错误语义；`server` 包为空，README 声称「健康检查/优雅退出」但没有实现；无 trace-id 注入中间件。
- **决策**：落地 `pkg/apitypes/code.go`（统一错误码），`internal/pkg/server` 补健康检查 + 优雅退出的最小实现，`middleware` 补 trace-id 注入与 recover；统一鉴权中间件返回体复用 `apitypes`。
- **落地物**：`pkg/apitypes/code.go`、`internal/pkg/server/server.go`、`internal/pkg/middleware/trace.go`。

### 发现 2.4 · docker-compose.infra 的 PostgreSQL 未启用 PostGIS 扩展
- **问题**：`migrations` 用 `GEOMETRY/GEOGRAPHY`（PostGIS 类型），但 `docker-compose.infra.yml` 的 postgres 镜像没有安装 postgis，本地 `infra-up` 跑迁移会直接失败。
- **决策**：改用 `postgis/postgis:16-3.4` 镜像；`config` 的默认 DSN 对齐 102 版端口（`25432`）与本地版（`5432`）需在文档说明。
- **落地物**：`deployments/docker-compose.infra.yml` 改镜像。

### 第 2 轮小结（≥3 点达成：2.1 / 2.2 / 2.3 / 2.4）

---

## 第 3 轮 · 域边界 / 依赖 / 一致性方向

评审对象：`internal/app`、`internal/domain/*`、`pkg/` 与 `internal/pkg` 边界、模块化单体的可拆分性。

### 发现 3.1 · `internal/pkg` 与 `pkg` 边界混淆，`apitypes` 放在 `pkg` 但被 `internal` 反向依赖
- **问题**：`pkg/apitypes` 定位「可被外部服务复用的公共库」，但 `internal` 域与 `internal/pkg` 又要用错误码/DTO，形成 `internal → pkg` 的跨界引用；Go 惯例下 `pkg/` 是公开 API，被模块内实现反向依赖会造成「内库外露」与未来拆分时的循环困惑。
- **决策**：`pkg/apitypes` 只承载**跨进程**（gRPC/事件）契约的稳定类型；进程内的 DTO/值对象一律放 `internal`。错误码放 `pkg/apitypes/code.go`（因为要跨服务共享），但进程内 handler 的错误响应用 `internal` 的 `httpresp` 包裹。明确此边界为 ADR。
- **落地物**：`pkg/apitypes/code.go`（跨服务错误码）；README 目录结构注释收敛边界语义。

### 发现 3.2 · 域间仍只有「空 doc.go」，无任何接口契约，拆分成本不可评估
- **问题**：`internal/domain/*` 除 user 外全部为空，ADR-001 要求「域只经接口依赖」，但没有任何域对外接口定义，阶段 5 的 order 依赖 customer/asset/resource 时无处对接。
- **决策**：在 `internal/app/wiring.go` 中以显式接口类型把 9 个域的服务接口占位（编译期声明），先落地 user.Service 已有接口，其余域先声明「阶段占位接口」，保证 `Application` 是装配契约的唯一入口。
- **落地物**：`internal/app/wiring.go` 显式域接口声明（占位 interface 别名，随阶段逐步补实现）。

### 发现 3.3 · 状态机「定义驱动」与 Temporal「编排」职责重叠未定义，一致性风险
- **问题**：`statemachine` 定义驱动状态机（非法流转拒绝），同时 Temporal 也编排订单 12 环节；两者边界未定义，会出现「状态机已拒绝但 Temporal 已推进」或反之的双重事实源。
- **决策**：明确「单事实源」原则——`internal/pkg/statemachine` 是**权威状态判定**（guard + 下一状态），Temporal 只做**编排/重试/补偿**，每次状态迁移前先过状态机 guard；`statemachine` 定义为可被 Temporal Activity 复用的纯函数包。ADR 记录。
- **落地物**：`internal/pkg/statemachine/statemachine.go`（最小纯函数实现，替代空 doc.go）；ADR-003。

### 发现 3.4 · 装配层无依赖注入约定，`Application` 无法随阶段增量演进
- **问题**：`wiring.go` 的 `Application` 是空结构体 + 注释掉字段，`New()` 无任何装配逻辑；阶段递进时「wiring 承载一切依赖绑定」的承诺没有可执行骨架。
- **决策**：给出最小编译通过、可增量扩展的装配骨架（`New()` 先从 config 加载，域实现以「可选提供」方式渐进注入，未实现域返回 `ErrNotImplemented` 或延迟绑定）。
- **落地物**：`internal/app/wiring.go` 实现增量装配骨架。

### 第 3 轮小结（≥3 点达成：3.1 / 3.2 / 3.3 / 3.4）

---

## 落地清单（Optimization Backlog）

| # | 发现 | 落地物 | 状态 |
|---|---|---|---|
| 1.1 | 审计分区无月度兜底 | `migrations/*up.sql` + ADR | done |
| 1.2 | 地址 level 冗余无约束 | `migrations/*up.sql` CHECK | done |
| 1.3 | parent_id 与 path 不一致 | ADR-002 + user 接口注释 | done |
| 1.4 | 7 角色无种子数据 | `migrations/*up.sql` 种子 | done |
| 2.1 | gRPC 契约空 | `api/proto/boss/common/v1/common.proto` | done |
| 2.2 | JWT 校验/轮换 | `internal/pkg/auth/jwt.go` + ADR | done |
| 2.3 | 无错误码/健康检查/trace | `pkg/apitypes/code.go` + `server.go` + `trace.go` | done |
| 2.4 | PG 缺 PostGIS | `docker-compose.infra.yml` | done |
| 3.1 | pkg/internal 边界 | ADR + README 注释 | done |
| 3.2 | 域接口契约空 | `internal/app/wiring.go` | done |
| 3.3 | 状态机与 Temporal 重复 | `statemachine.go` + ADR-003 | done |
| 3.4 | 装配层无 DI 骨架 | `internal/app/wiring.go` | done |
