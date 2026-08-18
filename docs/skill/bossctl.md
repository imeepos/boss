# bossctl: BOSS API CLI 工具

## 概述

bossctl 是 BOSS 综合业务支撑平台的命令行工具,用于操作全部 REST API 接口。
支持免登录 API key 认证,适合 CI/CD 流水线、自动化脚本、日常运维等场景。

## 安装

```bash
# 从源码构建
cd boss
go build -o /usr/local/bin/bossctl ./cmd/bossctl

# 或使用 Makefile 构建
make bossctl

# 验证安装
bossctl --help
```

## 快速开始

### 1. 免登录 API key 认证(推荐)

API key 与 BOSS 账号绑定,权限随账号角色走 RBAC。

```bash
# 设置环境变量
export BOSS_API_KEY=boss_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 直接调用 API
bossctl me
bossctl call GET /orders
```

### 2. JWT 登录认证

首次使用或需要管理 API key 时,可用账号密码登录:

```bash
# 登录
bossctl login admin your-password

# JWT 自动保存到 ~/.bossctl/token
# 后续命令自动使用缓存的 JWT

# 查看当前身份
bossctl me
```

### 3. 管理 API key(system-admin 权限)

```bash
# 列出所有 API key
bossctl apikey list

# 为指定账号创建 API key
bossctl apikey create 1 "ci-pipeline"

# 创建成功返回完整密钥,请立即保存(仅此一次)
# 输出: boss_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 吊销 API key
bossctl apikey revoke 5
```

## 命令参考

### 全局选项

| 选项 | 环境变量 | 说明 |
|------|----------|------|
| `--server URL` | `BOSS_SERVER` | API 服务器地址,默认 http://localhost:8080 |
| `--api-key KEY` | `BOSS_API_KEY` | API key(免登录,优先级最高) |
| `--jwt TOKEN` | `BOSS_JWT` | JWT 令牌 |

认证优先级: `--api-key` > `--jwt` > `~/.bossctl/token` > 无认证(仅公开端点)

### 命令列表

| 命令 | 说明 |
|------|------|
| `call METHOD PATH [flags]` | 调用任意 API 端点 |
| `login USERNAME PASSWORD` | 登录获取 JWT |
| `me` | 查看当前登录身份 |
| `routes` | 列出所有可用的 API 路由 |
| `apikey list` | 列出 API key |
| `apikey create <accountId> <name>` | 创建 API key |
| `apikey revoke <id>` | 吊销 API key |

### call 命令详解

通用调用命令,可操作所有 API 接口。

```bash
# 基本用法
bossctl call GET /orders
bossctl call GET /orders/ORD-20250801-001

# 查询参数
bossctl call GET /orders --query page=1 --query status=active

# POST 请求(JSON body)
bossctl call POST /orders --data '{"customerId":100,"productId":200}'

# PUT 请求
bossctl call PUT /legal-entities/1 --data '{"name":"新公司名"}'

# DELETE 请求
bossctl call DELETE /addresses/42

# 路径自动补全 -- 以下路径等价
bossctl call GET /orders          # 自动补全为 /api/v1/orders
bossctl call GET /api/v1/orders   # 完整路径,直接使用
```

### 响应格式

所有 API 响应统一 envelope:

```json
{
  "code": 0,
  "msg": "ok",
  "data": { ... }
}
```

- `code=0` 表示成功
- `code!=0` 表示失败,`msg` 包含错误信息

## 认证机制

### API key 认证(免登录)

API key 通过 `X-API-Key` 请求头传递,适用于:

- CI/CD 流水线自动化
- 定时脚本/批量任务
- 第三方系统集成

**安全约定:**

- 密钥格式: `boss_<32hex>` (如 boss_abc123...)
- 服务端只存 SHA-256 哈希,永不落明文
- 密钥仅在创建时返回一次,丢失需重新创建
- 停用账号即停用其所有 API key
- 支持单一吊销,不影响其他 key

### JWT 认证

通过 `BOSS_JWT` 环境变量或 `~/.bossctl/token` 缓存传递,适用于:

- 交互式使用
- 需要创建 API key 的管理场景

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BOSS_SERVER` | http://localhost:8080 | 服务器地址 |
| `BOSS_API_KEY` | (空) | API key(免登录) |
| `BOSS_JWT` | (空) | JWT 令牌 |

## 实战示例

### 日常运维

```bash
# 查看运营总览
bossctl call GET /dashboard

# 列出今日订单
bossctl call GET /orders --query createdAfter=2025-08-01

# 查看告警
bossctl call GET /alarms --query status=active

# 查询欠费客户
bossctl call GET /arrears

# 查看资产台账
bossctl call GET /assets
```

### 订单管理

```bash
# 创建新订单
bossctl call POST /orders --data '{
  "customerId": 100,
  "productId": 200,
  "serviceType": "fixed_broadband"
}'

# 查看订单详情
bossctl call GET /orders/ORD-20250801-001

# 查看派单池
bossctl call GET /pool

# 领取任务
bossctl call POST /pool/TK-20250801-001/assign
```

### 地理信息管理

```bash
# 列出国家
bossctl call GET /geo/countries

# 新建国家
bossctl call POST /geo/countries --data '{
  "code": "JP",
  "name": "Japan"
}'

# 导入地址数据
bossctl call POST /geo/import --data '{
  "rows": [{"path": "jp.tokyo.shinjuku", "name": "新宿"}]
}'
```

### 四码合一查询

```bash
# 按客户查询
bossctl call GET /quad-links/by-customer --query customerId=100

# 按端口查询
bossctl call GET /quad-links/by-port --query portId=500

# 按资产查询
bossctl call GET /quad-links/by-asset --query assetId=300
```

### 配置下发

```bash
# 查看下发模板
bossctl call GET /provision-templates

# 查看下发任务
bossctl call GET /provision-tasks

# 重试失败任务
bossctl call POST /provision-tasks/TASK-001/retry
```

## 开发指南

### 构建

```bash
# 构建 CLI
go build -ldflags="-s -w" -o bossctl ./cmd/bossctl

# 构建 server(含 API key 支持)
BOSS_API_KEY_SALT=my-salt go build -o server ./cmd/server
```

### 路由发现

`bossctl routes` 列出所有可用 API 路由(121 个端点),方便快速查找:

```bash
bossctl routes | grep "orders"
bossctl routes | grep "POST"
```

### 测试

```bash
# 单元测试
go test ./cmd/bossctl/...

# 集成测试(需运行中 server)
bossctl call GET /healthz
bossctl login admin admin123
bossctl me
```

## 安全建议

1. API key 遵循最小权限原则:为指定账号创建,而非系统管理员
2. 定期轮换密钥:创建新 key → 切换使用 → 吊销旧 key
3. 不使用 `--api-key` 参数直接传递密钥(会记录到 shell 历史)
4. 推荐通过环境变量 `BOSS_API_KEY` 或 CI 机密管理

## 术语表

| 术语 | 说明 |
|------|------|
| 免登录 | 无需先调用 /auth/login,直接通过 API key 认证 |
| API key | 与账号绑定的长期密钥,格式 boss_<32hex> |
| JWT | 短期令牌,通过登录获取,默认 24h 过期 |
| RBAC | 基于角色的权限控制,API key 权限随账号角色 |