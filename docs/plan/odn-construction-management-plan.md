# ODN 无源网络施工管理开发计划

> 版本：2026-09-07。目标：让 ODN 资源从规划、施工到可运营落地可证据化，不把施工管理退化为状态按钮或文本仓库。

## 1. 定制原则

本系统服务对象是 ODN 无源网络：网格、设施、光缆段、纤芯、箱体/分光器、端口、覆盖地址。施工管理的完成标准不是“项目已点击验收”，而是每一条资源链都有设计基线、现场事实、测试结果、竣工几何和运维接管记录。

四条硬原则：
1. **资源链为主键**：项目范围必须落到 facility/segment/fiber/port/coverage，不接受只有描述的施工范围。
2. **现场事实可追溯**：进度、照片、坐标、测试文件绑定到具体资源或工序。
3. **设计与竣工双版本**：施工不能覆盖设计；as-built 是新版本，差异必须可审计。
4. **门禁由服务端执行**：许可、材料、设计、质量、测试不满足时禁止开工、验收或转运营。

## 2. 当前缺口与取舍

| 缺口 | 对 ODN 的实际后果 | 处理批次 |
|---|---|---|
| 许可只是模板文本 | 无法证明路权/PECE 可施工 | P0 |
| 项目状态无过程证据 | BUILDING 期间无法知道哪段已建 | P0 |
| 清单无资源/工序完成量 | 无法计量和定位返工 | P0 |
| 验收只批量改生命周期 | 设施、覆盖、测试可能互相矛盾 | P0 |
| 无设计/as-built 差异 | 运维拿不到可信网络台账 | P1 |
| 无材料出库和测试归档 | 成本、质量、资产无法追溯 | P1 |
| 无资产化和运维移交 | 已建网络仍是孤儿资源 | P2 |
| 无完整应付付款 | 结算不等于付款闭环 | P2 |

不在 P0 堆入通用 CRM、泛审批、长篇日报模板、无业务约束的附件中心；每个对象必须回答“关联哪条 ODN 资源、产生什么门禁、谁负责、什么证据”。

## 3. ODN 最小闭环

```text
规划资源/设计版本
  → 施工项目与资源范围冻结
  → 许可检查 + 技术交底 + 材料/设备提交批准
  → 开工门禁
  → 按资源段现场进度/照片/坐标/测试采集
  → 分项检查与整改复验
  → as-built 差异审查
  → 设施/链路/端口转运营 + 覆盖 SERVED 联动
  → 资产化与运维移交
  → 工程量核定、结算、付款
```

### 资源链验收最低证据
- 设施：编码、类型、坐标、生命周期、照片；
- 光缆段：起止设施、敷设方式、长度、路由几何、纤芯；
- 纤芯：芯号、连接关系、接续记录、OTDR/光功率结果；
- 端口：设备、端口号、占用关系、测试结果；
- 覆盖：地址、服务设施/设备、SERVED/PENDING 变化原因。

## 4. 领域对象与关系

保留现有 `construction_projects`、`construction_items`、`construction_settlements`，但重新定位为主单、范围/清单、结算；新增对象按批次建设：

| 对象 | 关键关联 | 作用 |
|---|---|---|
| construction_design_baselines | project + 资源版本 | 锁定施工依据 |
| construction_scope_resources | project + resource type/id | 资源化施工范围 |
| construction_wbs_tasks | project + parent task | 工序和里程碑 |
| construction_permits | project + ROW/PECE | 开工许可事实 |
| construction_progress | task + resource | 现场完成量/坐标/照片 |
| construction_inspections | resource/task | 检查结果和整改 |
| construction_tests | segment/fiber/port | OTDR、光功率、连通测试 |
| construction_asbuilt_revisions | project + resource | 竣工几何/属性版本 |
| construction_handover | project + resource | 运维接管与资产化凭证 |

跨域规则：施工域不直接 import procurement/billing implementation；通过 httpapi 组合供应商、材料和财务信息，沿用现有契约的软引用与快照原则。

## 5. 状态与服务端门禁

### 项目状态
`DRAFT → PLANNED → READY_TO_START → BUILDING → COMPLETION_SUBMITTED → ACCEPTING → ACCEPTED → HANDED_OVER → CLOSED`。另有 `SUSPENDED`、`RECTIFICATION`，不允许用备注表达。

### 门禁
- READY_TO_START：承包商类型为 CONSTRUCTION 且资质有效；设计基线已发布；范围非空；ROW/PECE 达标或明确 NA；材料/设备提交达到要求。
- BUILDING：服务端原子校验全部开工条件，记录 started_by/started_at。
- COMPLETION_SUBMITTED：范围内资源均有现场事实；必需测试和照片齐全；未关闭严重整改为零。
- ACCEPTING：验收按资源/工序逐项记录；失败进入 RECTIFICATION，不得直接 ACCEPTED。
- ACCEPTED：生成 as-built 版本；仅通过的资源转 IN_SERVICE；覆盖联动必须成功，否则整批失败并产生告警。
- HANDED_OVER：运维资产、网络关系、文档、测试包和责任人齐全。

## 6. 分期开发与逐项验收

### P0-A：资源化施工范围与开工门禁
范围：把项目明细从“设施+金额”升级为 ODN 资源范围、设计基线、任务分解；接入许可检查；服务端阻断不合格开工。
验收：创建项目→挂 facility/segment/port→发布设计基线→缺许可时开工返回明确 409；满足条件后只允许一次有效开工；审计可查。

### P0-B：现场事实与分项进度
范围：按资源/任务记录完成量、坐标、照片、人员、时间、异常；支持弱网暂存和幂等回传；管理端展示资源地图与未完成清单。
验收：同一资源重复回传不重复计量；照片/坐标能反查资源；项目总进度由明细聚合而非手工输入；失败留 `[odn-construction] ... FAILED` 日志。

### P0-C：质量、测试、整改与验收
范围：ODN 检查模板、整改、复验；按光缆段/纤芯/端口录入 OTDR、光功率、连通性；验收生成资源级结果。
验收：测试不合格不能验收；整改关闭后可复验；验收只转通过资源；覆盖门控开启时 ACCEPTED 同事务更新 SERVED，失败可观察且不产生半成品状态。

### P1：设计-as-built、文控和移交
范围：设计版本、竣工差异、竣工图/测试包、运维接管；建立资源链单一可信台账。
验收：设计与竣工几何/属性可对比；每个 IN_SERVICE 资源可追溯到项目、验收、测试；移交缺资料时拒绝关闭。

### P2：材料、资产化和工程支付
范围：项目领料/退料、施工资产凭证、网络资产盘点、工程量核定、应付台账和付款凭证。
验收：材料成本落到项目/资源；资产可反查施工来源；结算、应付、付款三态分离；禁止用 SETTLED 伪装已付款。

## 7. 执行纪律

每个 P0 子批次独立 worktree、独立提交；编辑前读取最新文件；先检查迁移号和其他 worktree。每个提交先运行针对性测试，再运行相关门禁并记录结果。完成后在 feature worktree merge main、重跑门禁、推送 gitea；回主树核对 `pwd` 与分支后仅 ff-only 合并，复核提交存在后再 remove worktree、删除本地和远端分支。

## 8. 本计划的第一执行任务

P0-A 不直接重写全部施工系统，先交付“资源化范围 + 开工门禁”的最小闭环。其 acceptance command：

```bash
bash scripts/e2e/verify-odn-construction-e2e.sh
```

若现有脚本无法覆盖新门禁，先扩展脚本和测试夹具，再写实现；禁止以 UI 截图或按钮点击代替业务验收。
