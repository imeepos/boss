# 2026-09-07 ODN 资产化转固模型（P-INFRA-1 W8）

## 裁定

1. **odn×asset 关联=桥表软引用，不加跨域 FK，不动两侧既有 schema**：
   新表 odn_asset_registrations（资产化凭证），一行=一次转固事实：
   entity_kind(FACILITY/DEVICE) + facility_code/device_id 二选一（CHECK 兜底，
   与 odn_facility 单列主键 / odn_device 数值主键的物理形态一一对应）→ asset_id（软引用
   assets，跨域不加 FK，先例 construction_projects.contractor_id 2026-09-07）。
   部分唯一索引保证：一对象同时至多一张 ACTIVE 凭证、一资产同时至多挂一张 ACTIVE 凭证。
2. **资产状态枚举扩五态**：assets.status 增 IN_TRANSIT（出库在途，000216 放宽 DB CHECK）：
   出库单 OPEN 备出库不动状态；CONFIRMED 同事务置 IN_TRANSIT；转固 ACTIVE 凭证生效即 DEPLOYED；
   出库单取消（含已出库退库）回 IN_STOCK。全程 asset_lifecycles 留轨迹。
3. **转固=凭证事件，不是列复制**：价值（value_amount）、来源（PROCUREMENT/CONSTRUCTION/DIRECT）、
   项目溯源（construction_project_id）、采购溯源（batch_id 自 assets 快照）都落在凭证行上；
   冲销 REVERSED 终态留历史不删（同结算单 VOIDED 先例），冲销后对象/资产可重新登记。
4. **跨域表访问口径**：odn 域 SQL 写 assets/asset_lifecycles（先例 quadlink/pg_scan.go 000185
   装机联动同事务置 DEPLOYED），Go 层两域零 import；asset 域盘点（scope='ODN'）SQL 只读
   桥表解析快照范围，同样零 Go import。域边界靠 Go import 断言，SQL 共享是本仓既成现实。
5. **盘点口径**：stocktakes.scope 保留值 ODN=网络资产专项盘点，快照范围=有 ACTIVE 凭证的资产；
   差异处置/关单守卫复用既有 S10 流程，零新状态机。
6. **出库单边界**：W8 只管台账连续性（哪台资产在哪个项目工地、什么状态）；材料成本归集
   （采购-施工金额链）归 W9 项目领料，不越界（infra-buildout-plan 顺序六）。

## 为什么

- **桥表 vs 外键/软引用列**：facility 与 device 主键形态不同（VARCHAR8 vs BIGSERIAL），
  两侧各自加列都要改契约表；桥表把「关联+凭证+价值+溯源」合成一个可审计事实，
  冲销/重登记天然留历史；跨域 FK 会让迁移/revert 连锁（同 W1 弃 FK 理由）。
- **IN_TRANSIT 新状态 vs 台账事实不动状态**：出库后仍 IN_STOCK 会被库存盘点误捕，
  记 DEPLOYED 则虚增装网口径；「在途」是资产客观状态而非台账视图，枚举扩态是契约级
  最小正确表达（terms.md §4 同步修订）。
- **CONSTRUCTION 来源强制 ACCEPTED**：竣工才转固，与 AcceptProject 设施批量 IN_SERVICE
  的既有口径对齐，防在建项目提前形成固定资产。

## 放弃了什么

- **assets 侧反向指针列**：换取 asset 域 schema 零改动；代价是资产→凭证反查走桥表查询
  （idx_odn_asset_reg_asset 已建）而非行内列。
- **site（局点）转固对象**：局点是场所/土建，本期仅 FACILITY/DEVICE 入册；后续需要时
  桥表加 entity_kind 第三值即可，不用改结构。
- **出库单成本列**：换取与 W9 边界干净；材料成本核算在 W9 项目领料里接采购金额链。
- **散料（光缆米数）出库**：assets 台账本就逐台，散料不入 assets；W9 若需要再建散料台账。
