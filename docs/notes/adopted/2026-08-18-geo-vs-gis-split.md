# geo 与 gis 分立两个域

日期：2026-08-18（来源提交 04:39 国际地理基础数据落地）

## 决策

`internal/domain/geo`（地理基础数据：国家/行政区划 CRUD、PSGC 内置、批量导入）与 `internal/domain/gis`（阶段 8：asset.changed 消费、实体空间同步）保持两个独立域，不合并。

## why

geo 是低频维护的纯台账（admin 维护页 + 全量内置数据），gis 是事件驱动的联动服务（Kafka 消费、空间同步），写侧/读侧/部署物（gis 有独立 cmd）完全不同。合并会让低 churn 的基础数据与高 churn 的联动逻辑互相穿刺。

## 放弃了什么

- 合并为单一 geo 域：阶段 8 未开工，现在合并等于提前猜测 gis 的形态（07-design/03：先长后分）。
- geo 也做空间联动：明确否决，geo 只管行政区划字典，不含业务实体空间状态。

## 关联

- ADR-002（地址层级 ltree 权威）——geo 的 subdivisions 是其国际版扩展
- 待 gis 域（阶段 8）真正开工后回头看本决策，必要时 amend
