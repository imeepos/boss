# 上线就绪审计报告（后端管理系统 · 核心业务场景稳定性与齐全性）

> 日期：2026-08-30 ｜ 审计 HEAD：`3701e0b9`（main） ｜ 方式：devloop A/B 循环 + 机械验收（每项门禁以退出码为准，拒绝自报通过）
> 结论先行：**四条门禁全绿，1 项真实缺陷已修复并入主线，建议放行上线**（残留风险见 §5）。

## 1. 审计范围与方法

- 对象：admin 后端（/api/admin/v1，388 条路由）+ admin 前端（web/admin）+ 权限模型（102 真实环境）。
- 基准：docs/contract/terms.md（订单 12 环节 + 状态枚举权威字典）、domain-map.md、fields.md。只读不猜，冲突以契约为准。
- 方法：五任务账本（.devloop/loop-state.json），T1/T2/T4 门禁按 HEAD 落盘 rc+log（`.devloop/gates/<task>-<sha>.*`），验收命令对当前 HEAD 断言 rc=0——HEAD 前进即失效，杜绝陈旧绿。

## 2. 门禁结果（全部 @ 3701e0b9）

| 任务 | 门禁 | 结果 | 关键证据 |
|---|---|---|---|
| T1 后端稳定性 | `make check` | **绿** rc=0 | 70 个 Go 测试包 -race 全过；契约对账 A（三端 545 路由全有契约）/A2（方法级无漂移）/B（json tag lowerCamel）/C（行数红线）/D（迁移编号无撞号）全 OK；bossctl 路由目录、admin 权限映射、UI 一致性基线全 OK |
| T2 管理端前端 | `CI=true make web-admin-check` | **绿** rc=0（修复后） | typecheck+单测+生产构建过；DS 采用率 12 模块全部达标（boss 25/29=86%，below=0） |
| T3 场景齐全性 | `scripts/audit-admin-core-scenarios.sh` | **绿** exit 0 | 期望 59 条全命中：订单 12 环节（9 个 admin 管理面 + 3 个系统自动仅可观测）+ 47 条核心域，逐条附契约依据；防作弊线（路由<100 判异常）在位 |
| T4 权限真实冒烟 | `scripts/mcp-admin-acceptance.sh`（102，零造数） | **绿** rc=0 | PASS=17 FAIL=0：受权账号（调度/财务/网维）权限内可调、权限外 403、跨组织数据零泄漏（40400）、越界与不存在不可区分 oracle |

## 3. 发现与处置

### 3.1 阻断级（已修复闭环）

- **F-1 · admin 前端 DS 采用率回归**：`boss` 模块 23/29=79% 低于 80% 阈值，T2 首跑红。docs/admin/page-patterns.md 声称 2026-08-23 已收敛为 0，即其后新增 6 页未接入设计系统，属回归而非存量。未接入清单：order/AddressChainBadge、AddressChainDrawer、AddressChainParts、site/MarkdownEditor、site/editor、worker/TeamDialogs。
  - 处置：定点迁移 AddressChainDrawer（自绘按钮→ui Button）+ site/editor（自绘 input/卡片/按钮→ui Input/Card/Button），`e7f59890` 并入主线，迁移后 25/29=86% 留余量；typecheck/test/build 全绿后合并。
  - 残留：其余 4 页仍未接入（worker/TeamDialogs 237 行为最大），采用率 86% 有 6pp 余量，不阻断上线；建议下个迭代按「页面改动时顺手迁移」惯例收敛。

### 3.2 非阻断（记录在案）

- **F-2**：admin.yaml 为聚合契约（路由行 `  /path: {$ref: ...}`，/api/admin/v1 前缀由 servers 段声明）。审计脚本已按真实格式适配；后续写契约工具者勿假设内联前缀。
- **F-3**：订单环节 5（标签预绑定）/6（创建账号）/12（更新 GIS）为系统自动环节，admin 侧仅台账可观测，terms.md §1 执行方=系统，属契约预期，非齐全性缺口。
- **F-4**：admin 前端构建有 >500kB chunk 警告（最大 index chunk 1.29MB）。不阻断功能，建议后续 manualChunks 代码分割。
- **F-5**：devloop_accept 内置等待窗口不足以容纳 make check 全量时长（实测 SIGTERM）。已改用「后台全量跑 + HEAD 键控 rc 断言」模式，判定依然机械；该模式已固化进账本验收命令。

## 4. 核心业务场景齐全性明细（T3，59 条全过）

- 订单 12 环节 admin 管理面：1 下单（代客下单）/2 资源核查（触发+预览）/3 端口预占（触发+查询+释放）/4 合同收费（主执行面，未收费不派单 REQ-CL-001）/7 预配置（进度+重试）/8 派单（工单池+指派）/9 扫码绑定（补录+日志）/10 激活（补激活入口）/11 激活回调（失败重试）。
- 系统自动、admin 仅可观测：5 标签台账 / 6 认证账号视图 / 12 GIS 图层。
- 47 条核心域：权限（账号/角色/权限码）、客户（档案/实名/停复机）、产品、订单、缴费、渠道对账、账单、欠费、发票、报障、拆机、资产（台账/端口/设备/换新/盘点）、告警、话单/认证日志/认证账号、四码合一、预配置模板/日志、经营分析、券（模板/实例/兑换码）、积分、师傅（管理/入驻审核）、消息、区域、法人、apikey、开放平台、渠道审核、ODN 局点、官网内容、客户端发版、地理数据、备份、审计日志。

## 5. 放行建议

**建议放行**。依据：四条门禁在放行 HEAD 3701e0b9 全绿，其中 T4 为 102 生产部署环境真实 RBAC 成对验证，T3 证明契约层面无核心场景缺口。

上线前建议关注（均不阻断）：
1. F-4 大 chunk 影响首屏加载，建议排期代码分割；
2. 其余 4 个未接入 DS 的 boss 页面按惯例顺手收敛；
3. 上线后按 docs/ops/patrol-cron.md 每日巡检与 SLO 基线（slo-baseline-102.md）观察。

## 6. 证据归档

- 门禁原始日志：`.devloop/gates/T1-T4-3701e0b9.{log,rc}`（本地，gitignore 外排除）
- 场景审计脚本：`scripts/audit-admin-core-scenarios.sh`（60f59439 并入）
- DS 修复提交：`e7f59890`（rebase 后主线 sha 3701e0b9）
- 账本：`.devloop/loop-state.json`（五任务全 done，验收命令可随时重放）

## 7. 收尾轮（2026-08-30 同日续，消化非阻断项 + 部署复核）

最终 HEAD `83b9ad4a`，七项证据全绿：

- **T6 · 102 部署复核（rc=0）**：server healthz 与 admin-web 5180 双 200；部署镜像 tag=`e3e4f2e0`（出处链完整：CI 按推送 sha 构建，含首轮 DS 修复 `3701e0b9`）；RBAC 复验 PASS=17 FAIL=0。方法论勘误：资产哈希清单比对因 `VITE_BUILD_COMMIT` 注入构建 sha 两端必异，属无效证据，已改用镜像 tag 出处链验证（见 `.devloop/gates/T6-e3e4f2e0.log` v2 段）。
- **T7 · DS 残留消化（rc=0）**：3 页真实迁移——ReviewBadge→ui Badge warning 变体；MarkdownEditor 工具栏 8 钮→ui Button outline+sm、编辑框→ui Textarea；TeamDialogs 手写 inputCls/primaryBtn/plainBtn 常量整体退役→ui Input/Button（解散确认走 destructive）。采用率 **28/29=97%**（below=0）。`3bcc1894`。
- **T8 · 大 chunk 分割（rc=0）**：manualChunks 扩展——ol/pmtiles→vendor-map、recharts+d3 系→vendor-charts、swagger-ui-react/xlsx 独立、其余三方按「最后一个 node_modules 段」逐包成 chunk（pnpm `.pnpm` 虚拟存储首段陷阱已注释）。业务主包 **1290→422KB**，最大单 chunk **422KB<1MB**（432241B 断言过），340 chunks 缓存粒度到包级。`83b9ad4a`。
- **最终 HEAD 全量复绿**：T1 `make check` rc=0（-race 全量）；T2 web 门禁 rc=0；T3 场景齐全性 59/59 PASS；T4 RBAC PASS=17 FAIL=0。
- **残余风险**：①逐包拆分后的浏览器运行时加载行为未实测（本环境 CDP 受限），本次推送将再触发 102 部署，建议人工过一遍核心页面；②AddressChainParts 保持自绘微型 chips/面包屑（DS 无同密度组件，强套 Button 撑破布局，诚实跳过）；③TeamDialogs 的 PerfPanel 仍用原生 table（语义等价未强迁）。

**结论维持：放行。** F-1/F-4 两个非阻断项已消化闭环，唯一开放项为残余风险①的人工过页建议。
