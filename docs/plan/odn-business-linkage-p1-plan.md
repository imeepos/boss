# ODN 业务关联 P1 覆盖关联——开发计划

> 决策依据：docs/notes/adopted/2026-09-06-odn-business-linkage.md。
> 范围：可查可判（覆盖关联 + 可装性查询 + admin 展示），不做下单硬校验（P2）。
> 任务编号接续 .devloop/loop-state.json 既有 T1-T3（P5 已验收），本期为 T4-T7。

## 背景证据（2026-09-06 实测）

- 102 库：odn_grid/facility/cable_segment/fiber/device 全 0 行；region_code 82、city_code 119（种子）；odn_site 1（疑似验收残留，T7 顺带核查）。
- 代码：order/provision/resource 三域 rg -i odn 零引用；PON 链路反查（/ports/:id/path）走逻辑资源层自有 SPL/OLT 实体，与 odn_* 无关。
- 结论：ODN 是登记型孤岛，本计划把它变成订单链路可消费的判据。

## 设计（P1）

### 数据（迁移 1 个，查两处定号）

- 新表 `address_coverage`：`id`、`address_id`（FK addresses，UQ）、`facility_code`（FK odn_facility，可空）、`device_id`（FK odn_device，可空）、`status`（SERVED/PENDING/UNSERVED）、`note`、审计列。
- 约束：facility_code 与 device_id 至少其一非空（CHECK）；status=UNSERVED 时两者可空。
- 迁移号：动工前按规则查两处（`ls migrations | tail` + `git for-each-ref refs/heads` 逐分支 `git ls-tree`），main 水位 000195 起，feat/aaa-* 三分支占号必须核对。

### 后端（internal/domain/odn 内新增，不新开域）

- coverage.go / pg_coverage.go：Upsert（建档关联，menu:odn 门禁）、GetByAddress、Resolve（按 addressId 或 lat/lng KNN 就近设施命中，返回可装状态 + 服务设施/设备摘要）。
- API：POST /odn/coverage、GET /odn/coverage?addressId=、GET /odn/coverage/resolve?lat=&lng=；契约写 api/openapi/admin/oss.yaml。
- 失败路径留痕：Upsert/Resolve 失败输出 `[odn-coverage] ... FAILED` 级日志（红线）。

### 前端（web/admin，不新增菜单/权限）

- /oss/odn 新增"覆盖关联"页签：按地址检索、录入/编辑关联（选设施/设备）、列表。
- 订单相关地址展示位（开户工作台 /bss/onboarding 地址步骤 + 客户档案地址区）只读"可装性"徽标：SERVED=可装 / PENDING=待覆盖 / UNSERVED=不可装，仅展示不阻断。

## 任务分解（账本 T4-T7）

| id | 任务 | 关键交付 | acceptanceCommand |
|---|---|---|---|
| T4 | 迁移与契约同步 | address_coverage 迁移 + fields.md/data-relations/openapi 同步 | make check |
| T5 | coverage 后端域 | Upsert/Get/Resolve + 单测 + FAILED 留痕 | go build ./... && go test ./internal/domain/odn/... |
| T6 | admin 前端 | 覆盖页签 + 可装性徽标（双主题） | make web-admin-check |
| T7 | 102 端到端验收 | scripts/e2e/verify-odn-coverage-e2e.sh（造数断言自清理，仿 verify-oss-pon-path-e2e.sh；含 odn_site 残留核查） | bash scripts/e2e/verify-odn-coverage-e2e.sh |

## 纪律

- worktree 开发（feat/odn-coverage-p1），合前 merge main + make check，收尾四步（push gitea → 主树 ff-only → remove worktree → 删分支）。
- 中央注册类改动（menu.def/fields.md/i18n types+locale）压独立小提交。
- P2/P3 不在本期：P2 端口状态机（复用 2026-08-28 预占裁定）、P3 绑定表与 GIS 反查（届时 Amended 2026-08-25 note 的"点位点击不查详情"）。
