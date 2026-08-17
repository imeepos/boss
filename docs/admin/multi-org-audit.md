# 管理后台 · 多维度审查模拟（地区 / 子公司 / 部门 / 岗位 / 角色）

> 目标：在 `simulation-register.md` 的 7 角色功能模拟之上，叠加**组织维度**——
> 不同**地区**、不同**子公司**、不同**部门**、不同**岗位**、不同**角色**五个维度，
> 审查模拟本系统（`docs/admin` 管理后台 + 阶段1 后端）在真实集团组织的使用情况。
> 每个维度组合走一条完整日常事件生命周期，凡生命周期中出现而系统未承载的**组织/数据权限触达点**，即为缺失项。

## 〇、五维基线

| 维度 | 含义 | 与「角色」的关系 | 当前系统是否承载 |
|---|---|---|---|
| 地区（区域） | 地理/行政经营区域，数据范围 | 角色相同、区域不同 → 数据可见范围不同 | 仅 `addresses` 5 级地理树（市/区/街道/小区/楼栋），**无经营区域树，无数据范围** |
| 子公司（品牌/法人） | 独立经营主体，品牌隔离 | 跨子公司数据必须隔离 | **无任何实体** |
| 部门 | 组织内职能单元 | 岗位与部门强相关 | **无任何实体** |
| 岗位（职位） | 一人一岗（装维师傅岗/调度岗/坐席岗…） | 岗位≠角色；角色是功能权限集合，岗位是组织编制 | **无任何实体**（`accounts` 只有 `role_id`） |
| 角色 | 功能权限集合（7 类） | 已承载 | `roles` 表 + RBAC 快照，已落地 |

**结论先行**：当前系统仅实现「角色（功能权限）」一层，**地区数据范围、子公司、部门、岗位四维为空**，
`accounts` 只挂 `role_id`，JWT Claims 只有 `RoleCode`。多组织开展使用时会出现「同角色跨子公司/跨区域越权可见」的系统性数据泄露风险。

---

## 一、组织基线（用于模拟的集团样例）

> 以菲律宾多品牌宽带运营商（见 `需求全案` REQ-BRAND/REG-004）为蓝本，落到当前单体内可演示的最小集团结构。

```
集团（总部 HQ）
├── Luzon 大区（Region-L）
│   ├── 子公司A「主品牌·企业」（LEG-A）
│   └── 子公司B「家庭宽带」（LEG-B）
├── Visayas 大区（Region-V）
│   └── 子公司C「批发品牌」（LEG-C）
└── Mindanao 大区（Region-M）
    └── 子公司B「家庭宽带」（LEG-B，跨大区经营）
```

- **地区**：大区 → 省 → 城市 → Barangay（经营区域树，与 `addresses` 地理树解耦）。
- **子公司**：LEG-A / LEG-B / LEG-C（品牌隔离，客户/订单/账务/资产互不可见）。
- **部门**：总部=装维调度部/客服部/财务部/网络运维部；子公司=属地客服部/属地装维组。
- **岗位**：装维师傅/装维调度员/客服坐席/财务收款/信用专员/区域经理/大客户经理。
- **角色**：沿用阶段1 的 7 类功能角色（customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin）。

---

## 二、逐维度审查模拟（每轮一个不同「维度组合」）

> 编号 A##；「现状」以 `internal/` + `migrations/` + `menu.js` 为权威；
> 「有」= 系统可承载，「缺」= 需补全（模型/字段/菜单）。

### 地区维度

#### A01 区域经理只看本大区经营数据 · 地区×岗位(区域经理)
- 生命周期：登录 → 选本区 → 看经营指标/资源/订单 → 尝试看邻区 → **被数据权限拦截** → 审计留痕
- 功能点：区域数据范围、跨区访问拒绝、GIS 地图按区域图层化（GIS-003）
- 现状：`analytics/gis/order` 页面**无区域过滤**；后端 `HasPermission` 只判功能码，**无数据范围判定** 【缺】「区域数据权限」

#### A02 跨大区资源调配审批 · 地区×角色(资源运维)
- 生命周期：Luzon 缺端口 → 申请调 Visayas 空闲端口 → 总部审批 → 资产/端口台账归属变更 → 两区账实一致
- 功能点：跨区调配单、区域归属变更、总部汇总=各区域之和（REQ-REG-005）
- 现状：`resource` 有台账，**无「跨区域调配」入口**，`addresses` 树不支持「经营区域」语义 【缺】

### 子公司维度（品牌/法人隔离）

#### A03 子公司A客服查不到子公司B客户 · 子公司×角色(ops)
- 生命周期：LEG-A 客服查询客户 → 命中 LEG-A 档案 → 输入 LEG-B 客户号 → **返回无权限/无结果** → 审计记录越权尝试
- 功能点：品牌数据隔离（REQ-BRAND-001）、账号绑定所属子公司
- 现状：`customer.html` 无子公司维度；`accounts` 无 `legal_entity_id` 【缺】「子公司隔离」

#### A04 子公司B跨大区经营，总部看品牌汇总 · 子公司×角色(analyst)
- 生命周期：总部分析师看 LEG-B 在 Luzon+Mindanao 的合并指标 → 分项下钻到两个大区 → 汇总=明细
- 功能点：品牌×区域交叉汇总（REQ-BRAND/REG-004 跨层汇总）
- 现状：`analytics` 无「品牌/子公司」维度，无法交叉汇总 【缺】

### 部门维度

#### A05 客服部坐席只看到本部门工单 · 部门×岗位(客服坐席)
- 生命周期：坐席登录 → 接单池只含本部门队列 → 尝试取他部门工单 → 拦截 → 主管看本部门 SLA
- 功能点：部门数据范围、队列归属、部门绩效
- 现状：`dispatch/complaint` 无部门队列；`accounts` 无 `dept_id` 【缺】「部门数据范围」

#### A06 财务部收款员接 Cashdesk 不可见其他客户 · 部门×岗位(财务收款)
- 生命周期：收款员登记缴费 → 仅见「姓名/账单号/余额/金额」→ 无客户其他数据 → 打印 OR
- 功能点：最小化数据暴露（PAY-003）、Cashdesk 专属视图
- 现状：`payment` 页无「收款员最小化视图」，无部门隔离 【缺】

### 岗位维度

#### A07 装维调度员分派多区域工单 · 岗位×角色(technician 同源岗位)
- 生命周期：调度员看跨区工单池 → 按技能/区域分派 → 绩效统计
- 功能点：调度岗（区别于装维师傅岗）、调度工作台
- 现状：7 角色中**无「装维调度员」岗**，`dispatch` 无调度工作台，岗位与角色未分离 【缺】「岗位-角色分离」

#### A08 同名岗位跨子公司复用同一角色码 · 岗位×子公司
- 生命周期：LEG-A 与 LEG-B 各有「客服坐席」岗 → 复用 `ops` 角色 → 但数据各自隔离 → 权限=功能(角色)∪数据(子公司)
- 功能点：岗位→角色绑定、岗位归属部门/子公司
- 现状：无岗位实体，无法表达「同角色、多岗位、不同数据域」 【缺】

### 角色维度（承上核对）

#### A09 系统管理员做全组织数据范围的授权 · 角色(sysadmin)×组织
- 生命周期：管理员新建账号 → 选所属子公司/部门/岗位 → 配角色 → 配数据范围(本区/本部/全集团) → 生效即时
- 功能点：账号组织归属、数据范围授权、即时生效
- 现状：`account` 页只有「账号/姓名/角色/状态」，**无组织归属字段**；`accounts` 无 org 字段 【缺】

---

## 三、缺失结论（组织与数据权限模型）

综合 9 轮（A01~A09，覆盖五维），当前系统在「角色」之外的四个组织维度全部缺承载，核心缺失为：

> 落地状态：以下 9 项均已在 `org-perm-simulation.md`（O01~O24）驱动下落地——#1~#6 落地为「组织与权限」菜单 + 后端 + 迁移；#7~#9 落地为 `transfer.html` / `analytics.html`（品牌×区域）/ `dispatch.html`（调度工作台）。

| # | 缺失能力 | 关联轮次 | 归属 | 落地形态 | 落地状态 |
|---|---|---|---|---|---|
| 1 | 经营区域树（大区/省/城，独立于地理地址树） | A01/A02/A04 | 基础平台 | `regions` 表 + 与 `addresses` 解耦 | 【有】 `region.html` |
| 2 | 子公司/法人实体（品牌隔离） | A03/A04/A08 | 基础平台 | `legal_entities` 表 | 【有】 `company.html` |
| 3 | 部门实体 | A05/A06 | 基础平台 | `departments` 表 | 【有】 `department.html` |
| 4 | 岗位实体 + 岗位↔角色分离 | A07/A08 | 基础平台 | `posts` 表 + `post_roles` 关联 | 【有】 `post.html` |
| 5 | 账号组织归属（子公司/部门/岗位/区域） | A03/A05/A09 | 基础平台 | `accounts` 增 `legal_entity_id/dept_id/post_id/region_scope` | 【有】 `account.html` 补列 |
| 6 | 数据权限（数据范围，功能权限之外的约束） | A01~A09 | 基础平台 | `data_scopes` 表 + JWT 增数据范围声明 | 【有】 `datascope.html` + `HasDataScope` |
| 7 | 跨区域调配审批入口 | A02 | 网络资源 | `transfer` 调配菜单（或并入 reserve） | 【有】 `transfer.html` |
| 8 | 品牌×区域交叉汇总 | A04 | 经营分析 | `analytics` 增「子公司/品牌」维度 | 【有】 `analytics.html` 补维度 |
| 9 | 调度工作台（调度岗与师傅岗分离） | A07 | 订单与工单 | `dispatch` 增岗位视图 | 【有】 `dispatch.html` 补工作台 |

---

## 四、落地建议（阶段1 最小闭环）

> 遵循「角色=功能权限，组织=数据范围」的 RBAC + 数据权限二分原则，
> 与 `需求全案` REQ-SYS-001/005、REQ-REG-004、REQ-BRAND-001 对齐。

### 4.1 数据模型（迁移 000002）

```sql
-- 经营区域树（与 addresses 地理树解耦；挂 ltree 表示行政层级）
CREATE TABLE regions (
    id         BIGSERIAL PRIMARY KEY,
    path       LTREE NOT NULL,              -- 如 root.luzon.manila
    level      SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 4), -- 集团/大区/省/城市
    name       VARCHAR(128) NOT NULL,
    CONSTRAINT uq_regions_path UNIQUE (path)
);
CREATE TABLE legal_entities (                -- 子公司/品牌
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(64) NOT NULL UNIQUE,  -- LEG-A/LEG-B/LEG-C
    name       VARCHAR(128) NOT NULL
);
CREATE TABLE departments (
    id         BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    name       VARCHAR(128) NOT NULL
);
CREATE TABLE posts (                          -- 岗位，与角色分离
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(64) NOT NULL UNIQUE,  -- 如 dispatcher/cashier/agent
    name       VARCHAR(128) NOT NULL,
    dept_id    BIGINT NOT NULL REFERENCES departments(id)
);
CREATE TABLE post_roles (                     -- 岗位→角色(功能权限)绑定
    post_id    BIGINT NOT NULL REFERENCES posts(id),
    role_id    BIGINT NOT NULL REFERENCES roles(id),
    PRIMARY KEY (post_id, role_id)
);
-- 账号增组织归属 + 数据范围
ALTER TABLE accounts
    ADD COLUMN legal_entity_id BIGINT REFERENCES legal_entities(id),
    ADD COLUMN dept_id          BIGINT REFERENCES departments(id),
    ADD COLUMN post_id          BIGINT REFERENCES posts(id),
    ADD COLUMN region_scope     LTREE;       -- 空=全集团；否则限本城/本大区子树
```

### 4.2 鉴权两层判定

- 功能权限：沿用 RBAC 快照（`HasPermission(accountID, permCode)`）。
- 数据范围：新增 `HasDataScope(accountID, resourceOwnerOrg)`，由 `region_scope`+`legal_entity_id`+`dept_id` 判定，越权拒并审计。
- JWT：`Claims` 增 `legal_entity_id/dept_id/post_id/region_scope`（或只放 `accountID`，数据范围照旧走服务端快照，避免 token 膨胀——**推荐后者**）。

### 4.3 管理后台菜单影响

- `account.html` 增「所属子公司/部门/岗位/数据范围」列与编辑项。
- `analytics.html` 增「子公司/品牌」过滤维度。
- `dispatch.html` 增「调度工作台」岗位视图。
- 新增「跨区域调配」入口（`transfer`，归网络资源组）。

> 本清单与 `simulation-register.md` 角色功能模拟互补：前者补「功能菜单」，本文补「组织与数据权限」。
> 后端落地点为阶段1 `internal/domain/user`（账号/组织/数据范围）+ 新迁移 `000002_stage1_org.up.sql`。
