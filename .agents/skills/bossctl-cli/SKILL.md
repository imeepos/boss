---
name: bossctl-cli
description: BOSS 业务平台 CLI 工具,用于操作全部 REST API 接口(121+ 端点),支持免登录 API key 认证。使用场景:自动化 CI/CD 流水线、日常运维脚本、API 调试与测试、批量数据操作。当用户要求 CLI 操作 API、免登录认证、或者提到 bossctl 时触发。
---

# bossctl CLI 工具使用指南

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

## 命令参考

| 命令 | 说明 |
|------|------|
| `call METHOD PATH [--data JSON] [--query k=v]` | 调用任意 API 端点 |
| `login USERNAME PASSWORD` | 登录获取 JWT |
| `me` | 查看当前登录身份 |
| `routes` | 列出所有 API 路由(125 个端点) |
| `apikey list` | 列出 API key |
| `apikey create <account\|worker\|customer>/<id> <name>` | 为指定主体创建 API key |
| `apikey revoke <id>` | 吊销 API key |
| `identity save <name> --api-key KEY` | 保存身份档案(复合场景切换用) |
| `identity list` | 列出身份档案 |
| `identity remove <name>` | 删除身份档案 |
| `ai config [--url URL] [--key KEY] [--model M]` | 查看/更新 OpenAI 集中配置(apiKey/apiUrl 平台统一管理) |
| `ai chat [--model M] <文本>` | AI 对话补全 |
| `ai embed [--model M] <文本...>` | 文本向量化(需网关支持 embedding 模型) |

### call 命令详解

```bash
# GET 请求(查询参数)
bossctl call GET /orders --query page=1 --query status=active

# POST 请求(JSON body)
bossctl call POST /orders --data '{"customerId":100,"productId":200}'

# PUT/DELETE
bossctl call PUT /legal-entities/1 --data '{"name":"新公司名"}'
bossctl call DELETE /addresses/42

# 路径自动补全(以下等价)
bossctl call GET /orders        # 自动补全为 /api/v1/orders
bossctl call GET /api/v1/orders # 完整路径
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

- **用途**:沿用/复盘师傅注册→后台审核→实名认证等流程时,直接读取该 JSON 复用账号与 API key,无需重新建号。
- **内容**:`admin`(总管理员)、`reviewer1`(审核人员,ops 角色持 menu:dispatch)、`workers[]`(每个注册的师傅,含 workerId/registrationId/API key)。
- **约定**:
  - 每次新建**任何**测试账号/师傅/审核人员,都必须把用户名、密码、API key、关联组织 ID 追加写入该 JSON(保持结构一致)。
  - 密码与 API key 为敏感信息;该文件权限建议 `0600`,不要提交到公共仓库。
  - 用 `--as <身份名>` 或 `--api-key <key>` 切换身份调用,详见上文"认证方式"。

**示例用法(读取账号发起调用)**:

```bash
# 用 admin 创建 API key
ADMIN_KEY=$(python3 -c "import json;print(json.load(open('.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")
bossctl --api-key "$ADMIN_KEY" call POST /legal-entities --data '{"code":"DEMO","name":"演示企业"}'
```