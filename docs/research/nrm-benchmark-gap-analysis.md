# 网络资源管理（NRM）市面对标与 OSS 域差距分析

> 版本 V1.0（2026-09-06）｜撰写：项目负责人会话
> 目的：以市面优秀 NRM/网络资源管理系统为基线，盘点本平台 OSS 域现状，给出差距清单与 P5 波立项依据。
> 事实来源：bossctl routes admin 实测路由、internal/domain/{resource,device,gis} 现码、docs/contract/* 契约。

## 1. 对标对象与能力基线

市面主流网络资源管理/网络库存（Network Inventory / NRM）产品的共性能力，取自 TM Forum Frameworx/SID 资源视图与主流厂商（华为 iMaster NCE、Nokia NetAct、Cisco Crosswork、VIAVI NITRO、Whale Cloud ZSmart 资源、VC4 S2C 等）公开能力面，归纳为十项：

| # | 能力 | 说明 |
|---|------|------|
| C1 | 统一资源模型 | 物理资源（机房/机架/设备/板卡/端口/光缆/接头盒/分光器）+ 逻辑资源（VLAN/IP/GEM Port/ONU ID）+ 关系（包含/连接/承载），参照 TMF SID |
| C2 | 资源全生命周期 | 规划→在建→在用→维护→退网的状态机管控，编码终身锁定不复用 |
| C3 | 服务驱动分配 | 开通流程驱动核查→预占→确认→释放，事务行锁防竞态 |
| C4 | 拓扑与路径可视化 | 物理拓扑 + 逻辑拓扑（PON 树：OLT→PON口→分光器→ONU），任意端点路径追踪（path trace） |
| C5 | 自动发现与账实对账 | SNMP/NETCONF/TL1 采集实装清单，与账面比对产出差异（实装无账面/账面无实装/状态不一致） |
| C6 | 容量管理与预警 | 端口/分光器/网格使用率统计、阈值告警、扩容建议 |
| C7 | 库存数据质量稽核 | 孤儿引用、状态机违例、命名/编码规范违例的周期稽核与整改闭环 |
| C8 | GIS 呈现 | 资源点位上图、地理与逻辑拓扑联动 |
| C9 | 开放北向 API | TMF639 Resource Inventory 类开放契约，供下游系统消费 |
| C10 | 仿真/数字孪生 | 割接模拟、配置仿真（前沿项，非必需） |

## 2. 本平台 OSS 域现状盘点（2026-09-06 实测）

已具备（映射到基线能力）：

| 现状 | 对应能力 | 证据 |
|------|----------|------|
| OLT/分光器/端口台账（IDLE/RESERVED/USED/DISABLED） | C1/C2 | GET /resources、GET /ports、端口变更历史 |
| ODN 全套台账：网格/设施/光缆段/纤芯/局点/核心链路设备，编码终身锁定 | C1/C2 | /odn/grids、/odn/facilities、/odn/segments(/fibers)、/odn/sites、/odn/devices |
| 订单 12 环节：环节2 资源核查、环节3 端口预占（事务行锁） | C3 | /orders/:orderNo/check-resource、/reserve；contract/terms.md |
| 设备指标 SNMP 采集 + 阈值告警 + 维护记录 | C5 部分（仅指标非清单） | cmd/collector、/device/metrics、/device/maintenances |
| GIS 点位图层 + 实体详情 | C8 | /gis/odn-points、/gis/resources/:id/detail |
| 四码一致性对账 | C7 部分 | /quad-links/reconcile（订单一致性视角） |
| 每日八域对账快照（含资源域行级计数） | C7 部分 | /reports/recon/latest |
| 软引用孤儿巡检 | C7 部分 | /db-patrol/orphans |
| OpenAPI 契约与 671 条三端路由 | C9 部分 | api-docs-openapidoc（2026-09-06 adopted note） |
| OLT CLI 仿真 + RADIUS 光猫上线 | C10 部分 | cmd/oltsim |

## 3. 差距矩阵

| 能力 | 现状评级 | 缺口 | 等级 |
|------|----------|------|------|
| C6 容量管理与预警 | 弱 | 仅 /odn/grids 有 >=80% 预警；端口/分光器维度无使用率统计 API、无容量视图、无阈值告警入告警体系 | P1，无外部依赖 |
| C4 拓扑与路径追踪 | 弱 | 四码反查是一致性视角，缺 ONU→分光器→OLT 逐跳物理链路 API 与前端链路视图 | P1，无外部依赖 |
| C7 库存质量稽核 | 中 | 孤儿巡检已有；缺资源域专项稽核（端口-分光器归属断裂、状态机违例、编码规范违例）与报表出口 | P1，无外部依赖 |
| C5 自动发现账实对账 | 弱 | collector 只采指标不采端口实装清单；端口级账实比对需设备侧协议扩展（oltsim/SNMP 清单采集） | P2，依赖设备侧改造 |
| C6 容量预测 | 无 | 无趋势预测与扩容建议（依赖数据治理底座） | P3 |
| C10 割接仿真/数字孪生 | 萌芽 | oltsim 已可测连通性；割接模拟未立项 | P3 |
| C9 北向开放 | 中 | OpenAPI 已就绪；TMF639 形状契约未对齐 | P2 |

## 4. P5 波立项（本波执行，三任务并行）

依据第 3 节，选三项 P1 无外部依赖差距立项，任务书与验收命令见 .devloop/loop-state.json：

- W1 资源容量视图与预警（对标 C6）：端口/分光器使用率聚合 API + >=80% 阈值告警入现有告警体系 + admin 容量视图页。
- W2 PON 端到端链路反查（对标 C4）：端点输入→逐跳物理链路 API + admin 链路视图入口。
- W3 资源台账稽核（对标 C7）：归属断裂/状态机违例/编码违例三类稽核 + 报表端点 + 挂每日巡检。

验收均为幂等 e2e 脚本（scripts/e2e/verify-oss-*.sh），Lead 机械执行后归档会话。

## 5. 后续路线（不承诺排期）

1. W4 设备侧端口实装清单采集（oltsim/SNMP 扩展）→ 端口级账实对账（补 C5）。
2. W5 容量趋势与扩容建议（依赖数据治理底座指标目录）。
3. W6 TMF639 形状北向契约（补 C9 对外开放）。
4. W7 割接仿真（基于 oltsim 扩展，C10）。

## 6. 参考来源

- TM Forum: Frameworx/SID 资源视图；inform.tmforum.org "If your inventory is wrong, your network decisions are wrong"
- VIAVI NITRO 网络库存（Telefonica HISPAM 选型公告）、Whale Cloud ZSmart 资源管理（Gartner Peer Insights 条目）、VC4 S2C GPON Inventory 能力面
- Gartner Peer Insights: Nokia NetAct Alternatives（OSS 市场格局）
