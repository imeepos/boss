# 2026-09-07 承包商与工程结算模型（P-INFRA-1 W1）

## 裁定

1. **承包商=软引用 + 名称快照，不加跨域 FK**：`construction_projects.contractor_id`（→ procurement_suppliers.id）
   与 `construction_settlements.contractor_id` 均为软引用，落名称快照列；指定时由 httpapi 层组合采购域
   （校验 CONSTRUCTION 类型 + ENABLED）与 ODN 域（落快照），两域零 import。
2. **金额=STORED 生成列**：`construction_items.amount GENERATED ALWAYS AS (round(quantity*unit_price,2)) STORED`，
   「金额一律后端计算」由数据库兜底，任何写路径（含未来新代码）无法直写金额；前端仅展示。
3. **结算状态机**：PENDING→SETTLED（确认）；PENDING/SETTLED→VOIDED（原因必填）；VOIDED 终态。
   同项目同时最多一张有效结算单（部分唯一索引 `uq_construction_settlements_project_active`）。
4. **作废与重开**：作废不删单（VOIDED 保留为历史与审计锚点）；「重开」= 项目解除锁定后发起新结算单，
   不复用/重置原单。资金纠错（SETTLED 后作废）允许，需原因入 `void_reason` 并审计。
5. **发起前置**：项目 ACCEPTED 且已指定承包商（应付对象缺失即拒绝）；结算发起后承包商与清单金额均锁定，
   明细锁定沿用 ACCEPTED 既有口径，故应付金额不漂移。
6. **迁移让号**：预分配 000203/000204；kaihu 线 `000203_vlan_columns_integer` 先行合入 main，
   按让号规则顺延为 **000205/000206**（000204 留空可复用）。W2 预留号 000205 被本次占用，
   W2 落库时应重新占号核查（其任务书已含该门禁）。

## 为什么

- 软引用：ODN 与采购域是两个能力域（domain-map 规则 2：跨域只经契约/事件）；
  参照 `odn_port.order_id` 软引用先例，避免 DDL 层跨域耦合导致迁移/revert 连锁。
  应付对象完整性靠发起前置校验（40900 + reason）保证，而非 FK。
- 生成列而非应用层计算：应用层计算仍可能被新写路径绕过或漂移；生成列把口径固化为物理事实，
  结算单 SUM 读取即权威。PG 16 支持 STORED（102 实测 16.4）。
- 部分唯一而非「作废即删」：结算单是财务凭证，作废是业务事件不可抹除；
  「同项目单有效」用部分唯一索引在并发下机械保证。
- 重开以新单表达：避免状态机出现 VOIDED→PENDING 回退（回退会让「已作废」语义丢失），
  新单携带新审计链，两单经 project_id 关联可回放。

## 放弃了什么

- **跨域 FK**：换取域边界干净与迁移可 revert；代价是供应商被删时快照悬挂（供应商无删除端点，仅停用，风险可控）。
- **结算单改单/重开原单**：换取状态机无回退边；代价是纠错产生新单据号（财务对账需按 project_id 聚合）。
- **金额应用层持久化**：换取物理不可篡改；代价是 up/down 迁移对生成列的显式 DROP（已配对）。
- **结算驳回/部分结算/分期**：本期不做，SETTLED 即终确认；如需再扩状态机。
- **供应商承建类型的 admin 表单列**：W1 约束禁触 i18n 中央登记文件，类型维度经 API 承载，
  施工页承包商下拉按 CONSTRUCTION 过滤即完整闭环；采购页 UI 不动（既有语义不变）。