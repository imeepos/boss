# 管理后台 · 组织与权限管理模拟清单（部门 / 岗位 / 子公司 / 区域 / 数据权限）

> 目标：从需求出发，以电信工作人员**不同部门 / 岗位 / 子公司 / 区域**的运维管理场景为起点，
> 逐轮模拟"日常常见事件"（每轮一个不同事件），每个事件走**完整生命周期**，
> 在系统中逐一找到对应功能点；凡生命周期中出现而菜单未覆盖的组织 / 数据权限触达点，即为需补全的菜单/页面。
> 本文档（含 `menu.js` + `account.html` + `company/department/post/region/datascope.html`）即为补全《管理后台"组织与权限"》菜单的推导依据与最终落地。

> 与 `multi-org-audit.md`（五维 9 轮 A01~A09）互补：后者是"审查缺什么"，本文是"部门与权限管理场景的逐轮日常事件生命周期 + 落地产物对账"。
> 原则沿用：**角色 = 功能权限（HasPermission），组织 = 数据范围（HasDataScope）**，两者正交。

## 〇、组织与权限基线（来源 `migrations/000002_stage1_org.up.sql` 种子）

| 维度 | 实体 | 数据范围载体 | 落地菜单 |
|---|---|---|---|
| 经营区域 | `regions`（集团/大区/省/城市 4 级 ltree） | `accounts.region_scope` | `region` 经营区域 |
| 子公司/法人 | `legal_entities`（LEG-A/B/C 品牌） | `accounts.legal_entity_id` | `company` 子公司/法人 |
| 部门 | `departments`（挂靠子公司） | `accounts.dept_id` | `department` 部门管理 |
| 岗位 | `posts`（组织编制，与角色分离） | `accounts.post_id` | `post` 岗位管理 |
| 岗位→角色 | `post_roles`（功能权限绑定） | —（供岗位间复用） | `post` 岗位管理（绑定列） |
| 数据范围 | `accounts.{legal_entity_id,dept_id,post_id,region_scope}` | 快照 + `HasDataScope` 判定 | `datascope` 数据权限 |

---

## 一、模拟总览（24 轮 · 每轮一个不同部门/权限日常事件）

> 覆盖：部门数据范围、岗位-角色分离、子公司隔离、区域数据范围、数据权限授权与越权拦截、跨域协同。
> 编号 O##；"菜单现状"以 `menu.js` 为权威，"有"= 存在菜单项/字段，"缺"= 需补全。

| 轮 | 场景（角色/岗位×组织维度） | 生命周期 | 落地菜单 |
|---|---|---|---|
| O01 | 系统管理员新建账号并授予组织归属 | 新建→选子公司/部门/岗位→配角色→配数据范围→即时生效 | account（已补全组织字段） |
| O02 | 客服坐席接单（本部门工单队列） | 登录→接单池(本部门)→取他部门工单→拦截→主管看 SLA | department + dispatch |
| O03 | LEG-A 客服查 LEG-B 客户（品牌隔离） | 查询→命中本公司档案→输他司客户号→无权限/无结果→审计 | company + datascope |
| O04 | 财务收款员 Cashdesk 最小化视图 | 登记缴费→只见姓名/账单号/余额/金额→无客户其他数据→打印 OR | department(财务部) + payment |
| O05 | 装维调度员调度工作台（岗位≠师傅） | 看跨区工单池→按技能/区域分派→绩效统计 | post(dispatcher) + dispatch |
| O06 | 同名岗位跨子公司复用同一角色码 | LEG-A/LEG-B 各有客服坐席→复用 ops→数据各自隔离 | post |
| O07 | 区域经理只看本大区经营数据 | 登录→选本区→看指标/资源/订单→看邻区→被数据权限拦截→审计 | region + datascope |
| O08 | 跨大区资源调配审批 | Luzon 缺端口→申请调 Visayas→总部审批→归属变更→账实一致 | region + resource |
| O09 | 系统管理员全组织数据范围授权 | 改数据范围→快照即时生效→越权被拒+提示→审计留痕 | datascope + audit |
| O10 | 岗位-角色绑定调整（调度员扩容到看经营指标） | 改 post_roles 绑定→即时生效→下次登录权限刷新 | post + account |
| O11 | 部门新增与队列归属（新开客服二部） | 新建部门→挂子公司→建岗位→建队列→账号挂靠 | department + post |
| O12 | 子公司停用与数据冻结 | 停用 LEG-C→其账号无法登录→历史数据只读 | company |
| O13 | 经营区域树扩展（新增城市） | 大区下新增城市节点→region_scope 可绑定→区域经理覆盖新城市 | region |
| O14 | 越权访问审计追溯（跨部门/跨区域） | 越权尝试→审计落库→多维检索→定位责任人→追责 | audit + datascope |
| O15 | 数据权限最小化（收款员隐藏客户档案） | 配置最小化视图→收款页只暴露必要字段→隐藏已脱敏 | datascope + payment |
| O16 | 岗位编制与账号一对多（一部门多坐席） | 部门建岗位→岗位挂多账号→各自 region_scope 不同 | post + account |
| O17 | 总部分析师看品牌×区域交叉汇总 | 看 LEG-B 两区合并→下钻两区→汇总=明细 | analytics + company + region |
| O18 | 部门绩效 SLA（客服部主管看本部门坐席） | 主管选本部门→看坐席接单/解决/SLA→无法看别部门 | department + dispatch |
| O19 | 岗位改绑角色即时生效（不换账号） | 岗位绑定角色调整→关联账号权限即时变更→审计 | post + account |
| O20 | 区域资源归属变更与台账一致 | 端口跨区调配→region 归属变更→台账=两区之和 | region + resource |
| O21 | 子公司内部门数据隔离（调度部 vs 客服部） | 调度部看派单、客服部看工单→互不越界→队列隔离 | department |
| O22 | 数据范围授权到"城市"级粒度 | 区域经理绑定到城市→只见该市→上级大区可见其下所有 | region + datascope |
| O23 | 账号组织归属变更（转岗） | 坐席转调度→换岗位→换角色→数据范围同步→审计 | account + post |
| O24 | 组织树全量对账（账号无孤儿归属） | 巡检→找出无部门/无岗位/无区域账号→补全/禁用→上报 | account + datascope |

---

## 二、逐轮生命周期 ↔ 功能点 ↔ 现状（有/缺）

> 格式：`生命周期环节 → 应触达功能点 → 现状(有/缺)`；"✅有"=菜单/字段已承载，"✘缺"=需补全（已在落地产物中补全）。

### O01 系统管理员新建账号并授予组织归属 · 岗位(系统管理员)×组织
新建 → 选所属子公司 → 选部门 → 选岗位 → 配角色 → 配数据范围(本区/本部/全集团) → 即时生效
- 账号组织归属字段（子公司/部门/岗位/数据范围）：`account.html` ✅（已补 4 列 + 数据范围列）
- 数据范围授权入口：`datascope.html` ✅
- 即时生效：后端 RBAC + 数据范围走快照，账号页承载 ✅

### O02 客服坐席接单（本部门工单队列）· 部门×岗位(客服坐席)
登录 → 接单池(本部门队列) → 取他部门工单 → 拦截 → 主管看本部门 SLA
- 部门数据范围/队列归属：`department.html` ✅（部门队列列）
- 派单队列：`dispatch.html` ✅（派单管理与队列联动）
- 越权拦截提示：`datascope.html` 承载越权策略 ✅

### O03 LEG-A 客服查 LEG-B 客户（品牌隔离）· 子公司×岗位(客服坐席)
查询 → 命中 LEG-A 档案 → 输 LEG-B 客户号 → 无权限/无结果 → 审计记越权尝试
- 品牌数据隔离：`company.html` ✅（品牌隔离说明）
- 账号绑定子公司：`account.html` ✅（子公司列）
- 越权审计：`audit.html` ✅

### O04 财务收款员 Cashdesk 最小化视图 · 部门×岗位(财务收款)
登记缴费 → 仅见姓名/账单号/余额/金额 → 无客户其他数据 → 打印 OR
- 最小化数据暴露：`department.html`（财务部最小化说明）✅ / `datascope.html`（最小化视图）✅
- Cashdesk 缴费：`payment.html` ✅

### O05 装维调度员调度工作台（岗位≠师傅）· 岗位×角色(technician 同源岗位)
看跨区工单池 → 按技能/区域分派 → 绩效统计
- 调度岗独立于师傅岗：`post.html`（dispatcher 岗位 + technician 角色）✅
- 调度工作台：`dispatch.html` ✅（岗位视图）
- 岗位-角色分离：`post.html` ✅

### O06 同名岗位跨子公司复用同一角色码 · 岗位×子公司
LEG-A 与 LEG-B 各有客服坐席 → 复用 ops → 数据各自隔离
- 岗位→角色绑定表格：`post.html` ✅（"同名岗位跨子公司复用"卡片）
- 数据域隔离：`company.html` / `datascope.html` ✅

### O07 区域经理只看本大区经营数据 · 地区×岗位(区域经理)
登录 → 选本区 → 看指标/资源/订单 → 看邻区 → 被数据权限拦截 → 审计
- 区域树：`region.html` ✅（集团/大区/省/城市）
- 区域数据范围拦截：`region.html`（区域数据范围卡片）+ `datascope.html` ✅
- GIS 按区域图层化：`gis.html` ✅

### O08 跨大区资源调配审批 · 地区×岗位(资源运维/区域经理)
Luzon 缺端口 → 申请调 Visayas → 总部审批 → 归属变更 → 两区账实一致
- 跨区调配：`region.html`（跨区域归卡片）+ `resource.html` ✅
- 总部汇总=区域之和：`analytics.html` ✅

### O09 系统管理员全组织数据范围授权 · 岗位(系统管理员)×组织
改数据范围 → 快照即时生效 → 越权被拒+提示 → 审计留痕
- 数据范围授权：`datascope.html` ✅（授权策略 + 即时生效说明）
- 越权审计：`audit.html` ✅

### O10 岗位-角色绑定调整（调度员扩容到看经营指标）· 岗位×角色
改 post_roles 绑定 → 即时生效 → 下次登录权限刷新
- post_roles 绑定编辑：`post.html`（绑定角色列）✅
- 权限即时生效：`account.html` ✅

### O11 部门新增与队列归属（新开客服二部）· 部门
新建部门 → 挂子公司 → 建岗位 → 建队列 → 账号挂靠
- 新建部门：`department.html`（新建部门按钮）✅
- 岗位挂部门：`post.html` ✅
- 账号挂部门：`account.html` ✅

### O12 子公司停用与数据冻结 · 子公司
停用 LEG-C → 其账号无法登录 → 历史数据只读
- 子公司停用：`company.html`（停用操作）✅
- 账号登录关联子公司状态：`account.html`（账号状态）✅

### O13 经营区域树扩展（新增城市）· 地区
大区下新增城市节点 → region_scope 可绑定 → 区域经理覆盖新城市
- 经营区域树新增节点：`region.html`（新增区域按钮 + 下钻）✅

### O14 越权访问审计追溯（跨部门/跨区域）· 组织
越权尝试 → 审计落库 → 多维检索 → 定位责任人 → 追责
- 越权拒绝留痕：`datascope.html` ✅
- 审计多维检索：`audit.html` ✅

### O15 数据权限最小化（收款员隐藏客户档案）· 组织
配置最小化视图 → 收款页只暴露必要字段 → 其余脱敏
- 最小化视图配置：`datascope.html` ✅ / `department.html` ✅
- 收款页脱敏：`payment.html` ✅

### O16 岗位编制与账号一对多（一部门多坐席）· 岗位×账号
部门建岗位 → 岗位挂多账号 → 各自 region_scope 不同
- 岗位挂多账号：`account.html`（岗位列 + 数据范围列）✅ / `post.html` ✅

### O17 总部分析师看品牌×区域交叉汇总 · 子公司×岗位(analyst)
看 LEG-B 两区合并 → 下钻两区 → 汇总=明细
- 品牌维度：`company.html`（品牌×区域交叉经营卡片）✅
- 交叉汇总：`analytics.html` ✅

### O18 部门绩效 SLA（客服部主管看本部门坐席）· 部门×岗位
选本部门 → 看坐席接单/解决/SLA → 无法看别部门
- 部门绩效：`department.html`（部门绩效/队列）✅
- 部门数据范围：`department.html` ✅

### O19 岗位改绑角色即时生效（不换账号）· 岗位×角色
岗位绑定角色调整 → 关联账号权限即时变更 → 审计
- 岗位改绑角色：`post.html` ✅
- 即时生效 + 审计：`account.html` / `audit.html` ✅

### O20 区域资源归属变更与台账一致 · 地区×岗位(资源运维)
端口跨区调配 → region 归属变更 → 台账=两区之和
- 区域归属变更：`region.html`（区域数据范围/归属卡片）✅
- 端口台账：`resource.html` ✅

### O21 子公司内部门数据隔离（调度部 vs 客服部）· 部门
调度部看派单、客服部看工单 → 互不越界 → 队列隔离
- 部门隔离 + 队列归属：`department.html` ✅

### O22 数据范围授权到"城市"级粒度 · 地区×岗位
区域经理绑定到城市 → 只见该市 → 上级大区可见其下所有
- 城市级 region_scope：`region.html`（城市节点）✅ / `datascope.html` ✅

### O23 账号组织归属变更（转岗）· 岗位×账号
坐席转调度 → 换岗位 → 换角色 → 数据范围同步 → 审计
- 账号组织字段可编辑：`account.html` ✅
- 转岗审计：`audit.html` ✅

### O24 组织树全量对账（账号无孤儿归属）· 组织
巡检 → 找出无部门/无岗位/无区域账号 → 补全/禁用 → 上报
- 账号组织字段完整性：`account.html`（公司/部门/岗位/数据范围列）✅
- 数据权限全量视图：`datascope.html` ✅

---

## 三、补全结论（组织与权限管理缺失菜单）

综合 24 轮，最初"角色之外四维全空"的缺口已全部补全为落地菜单/页面：

| # | 缺失能力 | 关联轮次 | 落地菜单/页面 | key |
|---|---|---|---|---|
| 1 | 子公司/法人（品牌隔离） | O03/O12/O17 | `company.html` 子公司/法人 | company |
| 2 | 部门实体（数据范围/队列/绩效） | O02/O04/O11/O18/O21 | `department.html` 部门管理 | department |
| 3 | 岗位实体 + 岗位↔角色分离 | O05/O06/O10/O16/O19 | `post.html` 岗位管理 | post |
| 4 | 经营区域树（大区/省/城市） | O07/O08/O13/O20/O22 | `region.html` 经营区域 | region |
| 5 | 数据范围授权（功能权限之外） | O07/O09/O14/O15/O22 | `datascope.html` 数据权限 | datascope |
| 6 | 账号组织归属字段 | O01/O23/O24 | `account.html`（子公司/部门/岗位/数据范围列） | account |
| 7 | 跨区域调配审批入口 | O08/O20 | `transfer.html` 跨区域调配（归网络资源组） | transfer |
| 8 | 品牌×区域交叉汇总 | O17 | `analytics.html`（品牌/区域维度 + 交叉汇总表） | analytics |
| 9 | 调度工作台（调度岗与师傅岗分离） | O05/O18 | `dispatch.html`（调度工作台 + 岗位视图） | dispatch |
| 10 | 菜单权限（角色→菜单可见性） | O01/O10/O19 | `menuperm.html` 菜单权限 | menuperm |

> 后端落地点：`internal/domain/user`（`org.go` 组织实体 + `service.go` 接口增 `ListRegions/ListLegalEntities/ListDepartments/ListPosts/GetDataScope/HasDataScope`），
> `internal/pkg/middleware/auth.go` 增 `DataScopeChecker`/`DataAuthz` 数据范围授权中间件；迁移 `000002_stage1_org.up.sql` 补全岗位/岗位-角色种子；迁移 `000003_stage1_menuperm.up.sql` 补全菜单权限种子（`menu:*` 权限码 + 角色→菜单绑定）。

> 三层权限模型（正交，取交集）：**菜单权限**（role→menu，本页/menuperm）∪ **功能权限**（role→perm code，`role_permissions` + Authz）∪ **数据权限**（account→org，datascope + HasDataScope）。

---

## 四、落地校验（验收证据）

### 4.1 菜单项 ↔ 页面 ↔ 后端三方对账

| 菜单 key | 页面 | 后端承载 | 状态 |
|---|---|---|---|
| company | company.html | `legal_entities` + `ListLegalEntities` | ✅ |
| department | department.html | `departments` + `ListDepartments` | ✅ |
| post | post.html | `posts` + `post_roles` + `ListPosts` | ✅ |
| region | region.html | `regions` + `ListRegions` | ✅ |
| datascope | datascope.html | `accounts` 组织字段 + `GetDataScope`/`HasDataScope` | ✅ |
| account | account.html（补组织列） | `accounts` 组织字段 | ✅ |
| transfer | transfer.html | 跨区调配单（归属变更 + 台账联动） | ✅ |
| analytics | analytics.html（补品牌×区域） | 品牌/区域维度交叉汇总 | ✅ |
| dispatch | dispatch.html（补调度工作台） | 岗位视图（调度岗/师傅岗分离） | ✅ |
| menuperm | menuperm.html | `permissions`(menu:*) + `role_permissions` 角色→菜单 | ✅ |

### 4.2 鉴权三层判定落地

- 菜单权限：role→menu（`role_permissions` menu:*，控制菜单可见性）→ `menuperm.html` ✅
- 功能权限：`HasPermission`（RBAC 快照）→ `middleware.Authz` ✅
- 数据范围：`HasDataScope`（组织快照）→ `middleware.DataAuthz` ✅
- JWT 不塞组织字段，数据范围走服务端快照，避免 token 膨胀 ✅

> 结论：24 轮（≥20 达标）部门/权限管理场景模拟完成，每轮完整生命周期均有对应功能点；缺失 10 类组织/权限能力（含菜单权限）全部落地到菜单 + 页面 + 后端 + 迁移，并核验通过。与 `simulation-register.md`（26 轮角色功能菜单）互补，覆盖"功能菜单"与"部门/权限管理"两个目标维度。
