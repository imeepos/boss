# ODN 全生命周期路线图——设计/施工/运维/下单/售后

> 决策依据:docs/notes/adopted/2026-09-06-odn-business-linkage.md。
> 本文档把"五阶段生命周期"映射为系统内可执行增量;P1 细化设计见 docs/plan/odn-business-linkage-p1-plan.md。

## 对标与出处(2026-09-07 检索)

| 厂商/方案 | 生命周期实践 | 对本系统的启发 |
|---|---|---|
| 3-GIS(telecom) | GIS 上一体化 design→construct→operate | 台账与地图同源,状态随施工推进 |
| VETRO FiberMap | Planning→Design→Operations(施工/维护/**as-built 竣工文档**)+ AddressBook 地址校验 + Mobile 外业 | 竣工回填是设计-施工的闭环件;地址是判据入口 |
| IQGeo | 网络变更多次发生,as-built 数据准确性决定运维质量 | 状态机+变更留痕优于一次性录全 |
| ZTE Fiber Fingerprint | 被动资源(纤芯/端口)指纹化,AI 驱动运维 | 运维前提=物理链路可反查可算影响面 |
| 华为 L3 全光网 | 规划-开通-维护自动化,客户满意度导向 | 下单环节自动分配物理口是价值兑现点 |

## 五阶段 → 系统增量映射

| 生命周期 | 业界模式 | 系统现状 | 增量 |
|---|---|---|---|
| 设计 | 覆盖规划/分光比/路由台账/BOQ | odn 八表台账+GIS 图层(空转) | **P1 覆盖关联**(地址判据);P6 设施状态机(PLANNED 起步) |
| 施工 | 施工单+as-built 竣工回填+验收测试 | 无施工概念,台账直接"已存在" | **P6 施工-竣工**:施工项目单+批量竣工回填(设计态→在网态) |
| 运维 | 拓扑反查/影响面/容量预警 | /ports/:id/path 仅逻辑层;容量 API 已有 | **P7 影响面**:光缆段/设施故障→受影响客户清单;报障单挂 ODN 实体 |
| 下单 | coverage 门控+自动分配物理口 | 订单只挂逻辑资源,不判覆盖 | **P2 端口占用态**+下单硬校验(覆盖 SERVED 才可下单) |
| 售后 | 客户↔路径↔物理全链追溯 | 报障工单不挂物理设施 | **P3 逻辑-物理绑定**:激活写绑定,GIS 点位反查在用订单 |

## 增量顺序与验收

1. **P1 覆盖关联(账本 T4-T7,已立项)**:address_coverage + 可装性查询 + admin 展示。
   验收:make check / go test ./internal/domain/odn/... / make web-admin-check / verify-odn-coverage-e2e.sh
2. **P6 设计-施工(新增 T8-T9)**:
   - T8 设施状态机:odn_facility/odn_site/odn_device 增加 lifecycle_status(PLANNED/IN_BUILD/IN_SERVICE/RETIRED,默认 IN_SERVICE 兼容存量),变更留审计。
     验收:go test ./internal/domain/odn/...
   - T9 施工项目与竣工回填:construction_projects(单号/关联设施清单/状态 PENDING→BUILDING→ACCEPTED)+回填接口(批量把设施 PLANNED→IN_SERVICE,写 asbuilt_note/竣工人/竣工时间);GIS 图层按状态分色。
     验收:bash scripts/e2e/verify-odn-construction-e2e.sh
3. **P7 运维影响面(新增 T10)**:impact 接口(输入光缆段/设施 → 沿链路反查 → 受影响 address_coverage → 客户清单);报障工单可选挂 odn 实体。
   验收:bash scripts/e2e/verify-odn-impact-e2e.sh
4. **P2 端口占用态(已裁定,排 P7 后)**:物理口状态机+预占迁移+下单硬校验(SERVED+余量>0 才放单)。
5. **P3 绑定与售后(已裁定)**:激活绑定表+GIS 反查+售后工单挂设施(届时 Amended 2026-08-25 note)。

## 纪律

- 每任务独立 worktree 分支,验收命令绿了才 commit;合并走 worktree 协议四步收尾。
- 中央注册(menu.def/i18n/fields.md)改动压独立小提交。
- 迁移号动工前查两处(main 水位 000195 起 + feat/aaa-* 分支占号核对)。
