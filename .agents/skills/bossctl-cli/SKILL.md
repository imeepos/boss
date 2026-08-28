---
name: bossctl-cli
description: BOSS 业务平台 CLI 工具,用于操作全部 REST API 接口(121+ 端点),支持免登录 API key 认证。使用场景:自动化 CI/CD 流水线、日常运维脚本、API 调试与测试、批量数据操作。当用户要求 CLI 操作 API、免登录认证、或者提到 bossctl 时触发。
---

# bossctl CLI 工具使用指南

## 基础原则(踩坑 3 次沉淀,务必先读)

1. **接口存在 ≠ 业务正当**。admin 端有 `POST /orders` 路由,但下单是 customer 的本职;admin 真正的职责是建号/审批/调度/状态机推进。看到路由不要默认"我能调就归我"。
2. **admin 是"账号管理员",不是业务角色**。sysadmin 不下客户订单、不替调度派单、不替财务收费;它通过 `POST /accounts` 把业务岗位(客服/调度/财务/网维)受权建出来,各业务岗位用自己的 API key 干自己的活。
3. **一笔订单 = 4 个真人 + 系统协作**,客户/客服(财务)/调度员/师傅各管一摊:
   - 客户自己 `POST /api/user/v1/orders` 下单(`CustomerID` 由服务端注入,不可传别人的)
   - 客服坐席只接待 customer 转入的人工咨询,不要替客户下单(任何"代客下单"的真实业务都是少数线下场景,不是 demo 主路径)
   - 调度员推进 `check-resource` / `reserve` / `dispatch` / `assign` 段2/3/8
   - 财务推进 `charge` 段4
   - 师傅接单、扫码绑定、激活、签收(`/api/worker/v1/...`)
4. **三表登录边界**:`accounts` 只给后台员工用;`workers` 只给师傅端用;`customers` 只给客户 App 用。三类主体的 API key 不互通——customer key 不能调 admin 接口,worker key 不能调 admin 接口。
5. **RBAC permCode 决定可见性**:同一 admin 端点,`menu:order` / `menu:dispatch` / `menu:billing` / `menu:resource` 是不同岗位;sysadmin 全有,但真实业务用对应 permCode 的受权账号去干。

## 工作职责分工(一张表)

> 数据来自 `test-accounts.json`;新增任何业务账号必须按相同结构追加,见后文"测试账号存储"。

| 角色 | 主体类型 | roleCode | 主营接口 | 主要干的事 |
|---|---|---|---|---|
| **客户** | customer | (无) | `/api/user/v1/*` | 自助下单、催单、取消、改地址、评价 |
| **装维师傅** | worker | (无) | `/api/worker/v1/*` | 接单、扫码、激活、签收、反馈 |
| **客服坐席** | account | ops | `/orders` (GET)、`/complaints` | 受理客户咨询、投诉办结;**不下单** |
| **装维调度员** | account | technician | `/orders` workflow、`/dispatch/pool`、`/dispatch/my-tickets` | 资源核查、端口预占、派单给师傅 |
| **财务收款员** | account | ops | `/billing-runs`、`/payments`、`/invoices` | 段4合同收费、对账、出票 |
| **网络运维工程师** | account | resource_admin | `/ports` `/resources` `/provision/*` | 端口/资源置备、网络监控 |
| **审核员** | account | ops | `/customer-registrations/*`、`/worker-registrations/*` | 注册申请审批、实名核验 |
| **管理员(admin)** | account | sysadmin | `POST /accounts`、`POST /api-keys`、跨域审计 | 受权建号、签发 API key、审批 |

### 一笔新装订单的协作示例

```
1. customer   POST  /api/user/v1/orders         → 订单 ORD-XXX
2. dispatch   POST  /orders/XXX/check-resource  → 段2 资源核查
3. dispatch   POST  /orders/XXX/reserve         → 段3 端口预占
4. cashier    POST  /orders/XXX/charge          → 段4 合同收费
5. dispatch   POST  /dispatch/pool/.../assign   → 段8 派单给师傅
6. worker     POST  /api/worker/v1/tickets/.../accept
7. worker     POST  /api/worker/v1/tickets/.../scan-bind
8. worker     POST  /api/worker/v1/tickets/.../activate → DONE
```

每步用对应角色的 API key,不要把 5-8 步都拿 admin 去干(那样等于绕过了平台的 RBAC 设计意图)。

## 使用二进制

技能内预编译了各平台二进制文件,位于 `assets/` 目录下:

```bash
# 直接使用当前平台的二进制
./assets/bossctl-darwin-arm64 --help

# 或复制到 PATH
cp assets/bossctl-darwin-arm64 /usr/local/bin/bossctl
bossctl --help
```

## 从源码构建

技能包含构建脚本,可在任意平台重新编译:

```bash
# 当前平台
./scripts/build.sh

# 交叉编译到其他平台
GOOS=linux GOARCH=amd64 ./scripts/build.sh

# 构建产物在 assets/ 目录下
```

## 快速开始

```bash
# 列出所有 API 路由
bossctl routes

# 查看当前认证身份
bossctl me

# 调用 API
bossctl call GET /orders
```

## 认证方式

### 1. API key 认证(推荐,免登录,三类主体)

API key 与三类主体绑定(与三表登录边界对齐):
- **account**(管理后台账号):注入完整 RBAC 身份,可访问全部管理接口
- **worker**(师傅):扫码/工单接口取真实师傅身份;菜单门禁接口一律 403
- **customer**(客户):下单/查询类身份;菜单门禁接口一律 403

```bash
# 为主体签发密钥(需 sysadmin 身份)
bossctl apikey create account/103 ops-main
bossctl apikey create worker/5 field-test
bossctl apikey create customer/100 home-user

# 环境变量
export BOSS_API_KEY=boss_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
bossctl me

# 命令行参数
bossctl --api-key boss_xxx call GET /orders
```

### 2. 身份档案(复合业务场景切换)

复杂场景要不断切换身份(customer 下单 → admin 派单 → worker 上门 → admin 管理),
把每种身份的 key 存为档案,用 `--as` 一键切换:

```bash
bossctl identity save customer --api-key boss_c1...
bossctl identity save admin    --api-key boss_a1...
bossctl identity save worker   --api-key boss_w1...

bossctl --as customer call POST /orders --data '{"customerId":100,"offerId":1}'
bossctl --as admin    call POST /dispatch/pool/TK-20250818-001/assign --data '{"masterId":5}'
bossctl --as worker   call POST /tickets/TK-20250818-001/scan-bind --data '{"epc":"EPC-001"}'
bossctl --as admin    call GET /orders/ORD-20250818-001
```

### 3. JWT 认证(引导创建 API key 时使用)

```bash
# 登录后 JWT 自动缓存到 ~/.bossctl/token
bossctl login admin your-password
```

### 认证优先级

`--as 身份名` > `--api-key` > `--jwt` > `~/.bossctl/token` > 无认证(仅公开端点)

### 服务端地址

`--server URL` > `BOSS_SERVER` 环境变量 > 缺省 `http://192.168.0.102:28080`(102 部署环境)。
日常无需传 `--server`;本机起服务调试时才需要 `BOSS_SERVER=http://localhost:8080 bossctl ...`。

## 命令参考

| 命令 | 说明 |
|------|------|
| `call METHOD PATH [--data JSON\|@file] [--query k=v]` | 调用任意 API 端点;路径可带端前缀 `user:` / `worker:`(缺省 admin);`--data @file` 从文件读大载荷 |
| `login USERNAME PASSWORD` | 登录获取 JWT |
| `logout` | 退出登录并清除本地缓存 JWT |
| `me` | 查看当前身份(三类主体各自返回身份视图) |
| `routes [admin\|user\|worker]` | 列出三端 API 路由(626 条 = admin 464 + user 95 + worker 67,由 api/openapi 生成) |
| `upload [--portal admin\|user\|worker] FILE` | 附件上传(multipart,字段 file,单文件 32MB) |
| `apikey list` | 列出 API key |
| `apikey create <account\|worker\|customer>/<id> <name>` | 为指定主体创建 API key |
| `apikey revoke <id>` | 吊销 API key |
| `identity save <name> --api-key KEY` | 保存身份档案(复合场景切换用) |
| `identity list` | 列出身份档案 |
| `identity remove <name>` | 删除身份档案 |
| `release upload --app user\|worker --version 1.2.0 --code 12 [--min N] [--notes S] [--status DRAFT\|GRAY\|PUBLISHED] [--rollout N] [--whitelist 1,2] FILE.apk` | 上传 CI 产 APK 创建发版(POST /client-releases,menu:release) |
| `release patch <id> --status GRAY --rollout 20 [--whitelist 1,2] [--min N] [--force]` | 状态迁移/灰度比例/白名单编辑 |
| `release list [--app user\|worker]` | 发版列表 |
| `ai config [--url URL] [--key KEY] [--model M]` | 查看/更新 OpenAI 集中配置(apiKey/apiUrl 平台统一管理) |
| `ai chat [--model M] <文本>` | AI 对话补全 |
| `ai embed [--model M] <文本...>` | 文本向量化(需网关支持 embedding 模型) |
| `version` | 打印版本号(报障请附带) |

**退出码约定**: 0 成功;1 失败(HTTP 错误、业务 `code!=0`、认证 401)。CI/脚本以退出码判断成败;
`--as` 身份遇 401 时会提示档案 key 可能已吊销及重新保存命令。

**注意**: 全局 flag(`--server`/`--api-key`/`--as`/`--jwt`)必须放在子命令之前。

### call 命令详解

```bash
# GET 请求(查询参数,值自动 URL 编码,中文/空格/& 安全)
bossctl call GET /orders --query page=1 --query keyword=宽带安装

# POST 请求(JSON body)
bossctl call POST /orders --data '{"customerId":100,"productId":200}'

# 大载荷从文件读
bossctl call POST /products --data @product.json

# PUT/DELETE
bossctl call PUT /legal-entities/1 --data '{"name":"新公司名"}'
bossctl call DELETE /addresses/42

# 路径端前缀与自动补全
bossctl call GET /orders        # admin 端,自动补全为 /api/admin/v1/orders
bossctl call GET user:/orders   # user 端,即 /api/user/v1/orders
bossctl call GET worker:/home   # worker 端,即 /api/worker/v1/home
bossctl call GET /api/user/v1/orders # 完整路径原样发送
```

## API 路由发现

```bash
# 列出全部端点
bossctl routes

# 按方法过滤
bossctl routes | grep "POST"

# 按模块过滤
bossctl routes | grep "orders"
```

## 服务端前置条件

API key 认证依赖服务端已部署对应能力:

1. 服务端数据库需有 `api_keys` 表(迁移 000042 + 000044,密钥经 subject_type/subject_ref 绑定三类主体)
2. 服务端需启用 `X-API-Key` 认证中间件(优先于 JWT 校验)
3. 首次 API key 需通过 JWT 登录后创建(需 `menu:apikey` 权限,默认仅 sysadmin 角色持有)
4. 创建 worker/customer 主体密钥前,对应主体须已存在(悬空主体会被 401 拒绝)

## 安全约定

- 密钥格式: `boss_<32hex>`,仅在创建时返回一次,丢失需重新创建
- 服务端只存 SHA-256 哈希,明文永不落盘
- worker/customer 主体的密钥不带菜单权限(RBAC 恒拒),只能访问其身份对应的接口
- 停用主体即停用其所有 API key
- 推荐通过环境变量 `BOSS_API_KEY` 或身份档案传递密钥,避免 shell 历史记录

## 测试账号存储(唯一事实源)

**存储位置:本技能目录下的 `test-accounts.json`**
(即 `.agents/skills/bossctl-cli/test-accounts.json`,与 SKILL.md 同级)

- **用途**:沿用/复盘客户注册→后台审核→师傅注册→下单→调度→财务→师傅上门 12 环节全流程时,直接读取该 JSON 复用账号与 API key,无需重新建号。
- **内容**(按"基础原则"中的角色齐全):
  - `admin`(总管理员,sysadmin)— **只做**受权建号/签发 API key/审计,不下单
  - `reviewer1`(测试审核员,ops)— 注册审批、实名核验
  - `kefu_xu` / `kefu_zhao`(客服坐席,ops)— 受理客户咨询、投诉办结
  - `dispatch_li`(装维调度员,technician)— 资源核查、端口预占、派单
  - `cashier_wang`(财务收款员,ops)— 段4 合同收费、对账、出票
  - `noc_chen`(网络运维工程师,resource_admin)— 端口/资源置备
  - `workers[]`(每个注册的师傅,含 workerId/registrationId/API key)
  - `customers[]`(每个注册的客户,含 customerId/phone/realName 状态/API key)
- **约定**:
  - 每次新建**任何**测试账号(无论 account/worker/customer),都必须把用户名、密码、API key、roleCode、关联组织(legalEntityId/deptId/postId)ID 追加写入该 JSON(保持结构一致)。
  - 业务账号结构:`{username, password, realName, accountId, roleCode, org:{legalEntityId, deptId, deptName, postId, postCode}, apiKeys:[{name,key}], note}`;新增时必须包含 roleCode + org 完整四元组(否则后续派单/审计会查不到归属)。
  - 师傅结构:`{workerId, registrationId, name, phone, idCardNo, groupId, groupCode, regionId, status, realNameStatus, apiKey, note}`。
  - 客户结构:`{customerId, registrationId, name, phone, idCardNo, legalEntityId, addressId, regionId, status, realNameStatus, serviceStatus, apiKey, note}`。
  - 密码与 API key 为敏感信息;该文件权限建议 `0600`,不要提交到公共仓库。
  - 用 `--as <身份名>` 或 `--api-key <key>` 切换身份调用,详见上文"认证方式"。

**示例用法(读取账号发起调用)**:

```bash
# 用 admin 受权建一个调度员账号
ADMIN_KEY=$(python3 -c "import json;print(json.load(open('.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")
bossctl --api-key "$ADMIN_KEY" call POST /accounts --data '{"username":"dispatch_xx","password":"Disp@123","realName":"X调度","phone":"1390000xxx","roleCode":"technician","legalEntityId":1,"deptId":1,"postId":1,"status":1}'

# 读取调度员 API key 并以调度员身份推进订单
DISP_KEY=$(python3 -c "import json;print(json.load(open('.agents/skills/bossctl-cli/test-accounts.json'))['dispatch_li']['apiKeys'][0]['key'])")
bossctl --api-key "$DISP_KEY" call POST /orders/ORD-20260822-000378/reserve --data '{}'

# 读取 customer API key 让客户自助下单
CUST_KEY=$(python3 -c "import json;print(json.load(open('.agents/skills/bossctl-cli/test-accounts.json'))['customers'][1]['apiKey'])")
curl -X POST "http://192.168.0.102:28080/api/user/v1/orders" -H "X-API-Key: $CUST_KEY" -H "Content-Type: application/json" -d '{"productId":"111","addressId":"290","channelId":"111"}'
```

### ⚠️ 反模式(踩坑警示)

- **不要用 admin API key 调 `/orders` POST** —— 这是 customer 的本职,admin 调等于绕过 RBAC。真正"代客下单"仅限极少数线下场景,不是 demo 主路径。
- **不要用一个 API key 跑完整流程** —— 每一步用对应角色(段2/3/8 用调度,段4 用财务,段9-12 用师傅),这才是平台的真实分工;用一个 key 全干等于把 RBAC 设计意图架空。
- **不要在新单接口 body 里编 customerId** —— user 端 `/orders` 的 CustomerID 是服务端从 token 取的,body 里传了也不会用;admin 端 `POST /orders` body 才传 customerId,但那是给极少数代客场景用的,不要当主路径。