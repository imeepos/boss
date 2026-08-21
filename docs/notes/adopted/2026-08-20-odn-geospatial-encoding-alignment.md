# ODN 地理空间编码对齐方案（PRV/NodeCode 映射 + 网格分域待建）

日期：2026-08-20（依据《Suniway ODN 地理空间编码规范》V1.0，同日生效）

## 决策

1. **PRV/NodeCode 不替代现有 PSGC 编码**：`geo_subdivision.code = 'PH-' + PSGC 10 位码`（migrations/000041）保持权威不动；PRV（`PHL001`~`PHL083`）与 NodeCode（`MNL001` 等）作为 **ODN 域内的派生映射**，由规划部分配后落映射表（PSGC 省级节点 ↔ PRV、PSGC 市级节点 ↔ 城市简写），新增映射须公告备案，应用层不改 000041 数据。
2. **ODN 无源物理层（电杆/人井/铁塔/接头盒/终端盒/网格分区/光缆段落）为待建新域**，归属 `internal/domain/odn`（与 geo/gis/resource 分立），本期只登记差距，不臆造实体。
3. **端口码前缀冲突先于 ODN 落地解决**：`ports.port_code` 现状 `P-SPLxx-yy` 与电杆编码 `P01001` 前缀撞车，ODN 域开工前须先裁定端口码形态或加域名空间隔离。

## why

规范要求 PRV/NodeCode 锚定 PSA 官方数据且由规划部统一分配（红线 1/6）；系统现有 PSGC 数据同为 PSA 口径、粒度更细（43769 节点），是映射的天然锚点。改 000041 主键格式是全局破坏性变更，收益为零。规范描述的是无源物理网络层，与 BOSS 已建成的逻辑资源层（OLT/分光器/端口/四码）是上下层关系，按 07-design/03「先长后分」，映射先行、实体待需求驱动再建。

## 放弃了什么

- 把 `geo_subdivision.code` 改成 PRV 格式：破坏 68 表外链与 PSA 权威口径，否决。
- 现在就建 odn 域全部实体（电杆/人井/网格/光缆段落）：无真实录入与消费方，属提前猜测形态。
- 端口码立即重命名：现值已被四码链路消费，须先出专门裁定（登记为审计 E 项）。

## 关联

- 审计登记：`docs/contract/alignment-audit.md` §10（E4~E10）
- 姊妹规范：《Suniway ODN 基础设施资源编码规范》（SNW/OLT/ODF/OCC/ODB/SDB/PRT/TBP 编码），同一对齐节奏
- `2026-08-18-psgc-builtin-migration.md`（被映射的权威数据源）
