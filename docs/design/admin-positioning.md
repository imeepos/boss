# admin 定位裁定：操作视角与需求视角（design/admin-positioning）

> 版本 V1.0（2026-08-18）｜依据：`docs/contract/{terms,domain-map,fields}.md`、需求全案 V1.2 第 2.3/4.1 节
> 定位：回答"admin 为谁服务、解决什么"。与他处冲突时以 contract 文档为准。

## 1. 两个视角的裁定

### 操作视角放在 admin

admin 是**运营侧唯一的受权操作台**（封闭账号模型，见 auth.yaml 裁定）。
"操作视角"指：凡是对平台公共数据、组织、账号、权限的**写操作**，只能发生在 admin，
由持权角色（sysadmin 等 7 角色码，fields.md 1.2）执行，全程审计留痕。

### 需求视角放在企业内部人员

admin 的"客户"是**企业内部人员**——装维师傅、客服坐席、财务收款员、调度、区域经理、
网络运维工程师（岗位清单见 migrations/000002 posts 种子）。他们不是 admin 的用户，
是 admin 的**需求方**：日常业务在各自作业端（师傅端/客服工单/计费台账）完成，
只有撞上"跨不过去的墙"时才需要 admin。

### 一句话

> admin 的存在意义 = 集中解决内部人员"只有 admin 能解决"的问题；
> 其余一切业务操作都应该在业务端自助完成，admin 不做业务端的第二入口。

## 2. "只有 admin 能解决"问题清单（能力矩阵）

按内部人员的求助场景归纳。列：场景｜求助人（岗位/角色）｜admin 能力｜落地页面｜后端端点｜现状。

### A. 账号与进入权（SYS 域）

| 场景 | 求助人 | admin 能力 | 页面 | 端点 | 现状 |
|:--|:--|:--|:--|:--|:--|
| 新员工入职要账号 | 全岗位 | 受权建号（分角色/组织/区域范围） | base/account | POST /accounts | 已上线（102 验证） |
| 忘记密码进不了系统 | 全岗位 | 管理员改密（个人中心自助改密需旧密码，忘密只有 admin 能重置） | base/account 编辑 | PUT /accounts/{id} | 已上线 |
| 离职/停用 | 全岗位 | 禁用账号（API key 联动失效） | base/account | PUT /accounts/{id} status | 已上线 |
| 调岗/换公司/换区域 | 全岗位 | 改组织归属与数据范围 | base/account 编辑 | PUT /accounts/{id} | 已上线 |

### B. 组织与编制（SYS/BRAND 域）

| 场景 | 求助人 | admin 能力 | 页面 | 端点 | 现状 |
|:--|:--|:--|:--|:--|:--|
| 新子公司/品牌开张 | 管理层 | 建法人 | org/company | POST /legal-entities | 已上线 |
| 新部门编制 | 管理层 | 建部门 | org/department | POST /departments | 已上线 |
| 新岗位/岗位换角色 | 管理层 | 建岗位/绑功能角色 | org/post | POST/PUT /posts | 已上线 |
| 菜单可见性争议 | 各角色 | 查三层权限模型矩阵 | org/menuperm | GET /menu-perms | 已上线（只读，契约如此） |

### C. 公共数据与规则（SYS 域横切）

| 场景 | 求助人 | admin 能力 | 页面 | 端点 | 现状 |
|:--|:--|:--|:--|:--|:--|
| 业务规则要调（欠费阈值/预占有效期） | 财务/调度 | 业务参数热更 | base/params | GET/PUT /params | 已上线（本轮） |
| 数据纠纷要查"谁改的" | 审计/管理层 | 审计日志查询 | base/audit | GET /audit-logs | 已上线（本轮，items+操作人联查） |
| 批量地址/地理数据初始化 | 网络运维 | 数据导入中心 | base/importer | POST /addresses/import、/geo/import | 已上线（地址/geo；客户/端口/资产导入 planned） |

### D. 自动化凭证（SYS 域）

| 场景 | 求助人 | admin 能力 | 页面 | 端点 | 现状 |
|:--|:--|:--|:--|:--|:--|
| 内部系统要 API 免登对接 | 网络运维/开发 | API key 签发/吊销（bossctl 即消费者） | org/apikey | GET/POST /api-keys、DELETE /:id | 已上线（签发/吊销/明文一次展示） |

## 3. 明确不属于 admin 的

- 客户侧自助（注册/充值/报障）：客户门户 PORT（user 端，REQ-PORT-007），admin 不代客操作。
- 日常业务作业（派单/扫码/出账/缴费）：boss/billing/ams 分组页面属运营作业，按角色权限开放，
  但"求助型"场景只有上表 A–D 四类。
- 师傅端内容：boss/worker-ops，非 sysadmin 独占。

## 4. 缺口 backlog（按优先级）

1. ~~P0 审计日志后端~~ 已上线（V1.0 轮）。
2. ~~P0 业务参数后端~~ 已上线（V1.0 轮）。
3. ~~P1 API key 管理页~~ 已上线（org/apikey）。
4. P2 导入任务清单 `GET /import-tasks`（客户/端口/资产批量导入）。

## 5. 使用规则

1. 新增 admin 能力时先入本矩阵：答不上"哪个内部岗位会为此求助 admin"的能力不做。
2. 页面交互按 antd Pro 规范（工具栏+表格+Drawer 表单），列名以 fields.md 为准。
3. 本文档随能力落地更新"现状"列；缺口清空后本文档转为维护态。
