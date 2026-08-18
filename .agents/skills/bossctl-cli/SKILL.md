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

### 1. API key 认证(推荐,免登录)

API key 与 BOSS 账号绑定,权限随账号角色走 RBAC。

```bash
# 环境变量
export BOSS_API_KEY=boss_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
bossctl me

# 命令行参数
bossctl --api-key boss_xxx call GET /orders
```

### 2. JWT 认证(引导创建 API key 时使用)

```bash
# 登录后 JWT 自动缓存到 ~/.bossctl/token
bossctl login admin your-password
```

### 认证优先级

`--api-key` > `--jwt` > `~/.bossctl/token` > 无认证(仅公开端点)

## 命令参考

| 命令 | 说明 |
|------|------|
| `call METHOD PATH [--data JSON] [--query k=v]` | 调用任意 API 端点 |
| `login USERNAME PASSWORD` | 登录获取 JWT |
| `me` | 查看当前登录身份 |
| `routes` | 列出所有 API 路由(121 个端点) |
| `apikey list` | 列出 API key |
| `apikey create <accountId> <name>` | 创建 API key |
| `apikey revoke <id>` | 吊销 API key |

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

1. 服务端数据库需有 `api_keys` 表(密钥哈希与账号绑定)
2. 服务端需启用 `X-API-Key` 认证中间件(优先于 JWT 校验)
3. 首次 API key 需通过 JWT 登录后创建(需 `menu:apikey` 权限,默认仅 sysadmin 角色持有)

## 安全约定

- 密钥格式: `boss_<32hex>`,仅在创建时返回一次,丢失需重新创建
- 服务端只存 SHA-256 哈希,明文永不落盘
- 停用账号即停用其所有 API key
- 推荐通过环境变量 `BOSS_API_KEY` 传递密钥,避免 shell 历史记录