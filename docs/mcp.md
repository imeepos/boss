# BOSS MCP Server(bossmcp)

> 面向 agent 的 MCP(Model Context Protocol)接入面:把用户端(user)/师傅端(worker)/
> 管理端(admin)REST 接口暴露给任意 MCP 客户端(Claude Code / Cursor / Claude Desktop 等)。
> 路由目录由 `api/openapi` 生成(`make bossctl-routes`),契约即工具能力,无手工清单;
> admin 端目录另按当前登录账号权限过滤(见安全边界),暴露面 = 账号拥有面。

## 构建

```bash
make bossmcp   # 产物 ./bossmcp(stdio server,单文件)
```

## 客户端配置

stdio server:宿主负责拉起进程,环境变量注入认证。

```json
{
  "mcpServers": {
    "boss": {
      "command": "/path/to/bossmcp",
      "env": {
        "BOSS_USER_API_KEY": "boss_<客户主体key>",
        "BOSS_WORKER_API_KEY": "boss_<师傅主体key>",
        "BOSS_ADMIN_API_KEY": "boss_<管理账号key>",
        "BOSS_SERVER": "http://192.168.0.102:28080"
      }
    }
  }
}
```

| 环境变量 | 说明 | 缺省 |
|---|---|---|
| `BOSS_USER_API_KEY` | user 端(客户主体)API key | 无 |
| `BOSS_WORKER_API_KEY` | worker 端(师傅主体)API key | 无 |
| `BOSS_ADMIN_API_KEY` | admin 端(账号主体)API key;决定 admin 目录/调用面的权限 | 无 |
| `BOSS_API_KEY` | user/worker 两端兜底 key(仅在端专属 key 缺省时生效;admin 端不参与兜底) | 无 |
| `BOSS_SERVER` | 后端地址 | `http://192.168.0.102:28080` |

key 签发(需 sysadmin,与 bossctl 同一体系):

```bash
bossctl apikey create customer/213 home-user   # 客户主体 → BOSS_USER_API_KEY
bossctl apikey create worker/1    field-test   # 师傅主体 → BOSS_WORKER_API_KEY
bossctl apikey create account/257 cli-staff    # 管理账号主体 → BOSS_ADMIN_API_KEY(调度等业务岗)
```

未配置 key 时 `boss_call`/`boss_whoami` 拒绝调用并提示对应变量名;启动日志
(stderr)也会以 `user_key=MISSING(...)` 留痕。

## 工具

| 工具 | 参数 | 说明 |
|---|---|---|
| `boss_routes` | `portal`(user/worker/admin)、`filter?` | 列端路由目录(方法/路径/描述),子串过滤;agent 用它发现接口。admin 端只列当前账号有权调用的接口 |
| `boss_call` | `portal`、`method`(GET/POST/PUT/DELETE/PATCH)、`path`、`body?`、`query?` | 调用任意端内接口;path 相对端前缀(如 `/orders`),`{param}` 占位换成实值 |
| `boss_whoami` | `portal` | 查当前身份(user/worker GET `/profile`;admin GET `/auth/me` 含权限码),开工前确认 key 有效 |

典型会话:先 `boss_routes` 发现 → `boss_whoami` 确认身份 → `boss_call` 执行。
`GET /orders` 示例返回:

```
code=0 msg="ok"
data:
{
  "hasMore": true,
  "items": [ ... ]
}
```

## 安全边界

- **三表主体不通用**:customer/worker/account 的 API key 互不通用(服务端 RBAC 恒拒);
  本 server 按端绑定 key,`boss_call` 传入跨端完整路径(如 user 端调 `/api/worker/v1/...`)
  直接拒绝,结构性防止把 key 发错面。
- **admin 端目录 = 账号权限**:`boss_routes(admin)` 先查 `GET /auth/me` 取当前账号
  权限码全集,再对路由目录做子集过滤——路由→permCode 映射由 `scripts/genrouteperms`
  从服务端 `requirePerm` 装配源码静态投影(`internal/apiroutes/admin_perms_gen.go`),
  与服务端 Authz 同源。权限查询失败一律拒发目录(fail-closed);账号数据范围由服务端
  按账号归属过滤,超出范围的行不会出现在返回里。
- **admin 权限外调用被拒**:权限外接口即使被误调,服务端 403 `no permission: <permCode>`
  以 `isError:true` 文本回传,agent 可理解原因并自纠。
- **受限模板 key**:带权限模板(如 partner-orders-read)的 admin key,目录/调用面
  只含模板授权码集(`/auth/me` 回模板真相,见 `internal/httpapi/admin/root_handlers.go`)。
- **业务失败 = isError 结果**:`code!=0` / HTTP 4xx/5xx 以 `isError:true` 文本回传
  (含 HTTP 状态、业务 code/msg),协议层不报错,agent 可读错自纠。
- **key 不落盘**:密钥仅经环境变量注入进程,无配置文件、无缓存。

## 协议

stdio 传输,每行一条 JSON-RPC 2.0 消息;支持 `initialize`(回显客户端 protocolVersion)、
`tools/list`、`tools/call`、`ping`;通知一律静默;日志只写 stderr(带 `[bossmcp]` 前缀)。

## 测试与门禁

```bash
go test ./cmd/bossmcp/ ./internal/apiclient/ ./internal/apiroutes/   # 协议循环+工具行为+目录权限一致性(httptest)
make bossctl-routes-check                      # 路由目录漂移门禁(含 internal/apiroutes)
make route-perms-check                         # admin 路由→permCode 映射漂移门禁
make mcp-smoke                                 # 真实环境冒烟(10 断言,直连 102,不经 LLM)
bash scripts/mcp-admin-acceptance.sh           # admin 验收矩阵(14 断言:三岗位正反例成对+跨组织数据隔离)
```

102 实测记录(2026-09-06):双端 `boss_whoami` 返回真实客户/师傅档案,
`boss_call` 双端列表接口返回真实业务数据;跨端路径/无效 key/未知端三负路径全部按预期拒绝。
真实环境冒烟已固化:`make mcp-smoke`(直连 102,不经 LLM)。
linux-amd64 资产已在 102 主机(linux)真机实测:initialize 握手 + 双端 whoami 真档案 3/3 通过。

admin 端实测记录(2026-08-29):调度/财务/网维三岗位受权账号经 MCP 拿到真实业务数据,
权限外调用 403 且目录面零泄漏;主品牌/家庭宽带两客服账号跨组织互查零串数据(39 单 vs 0 单)。
验收矩阵与证据:`scripts/mcp-admin-acceptance.sh` + `docs/acceptance/2026-08-29-admin-mcp-102.md`。
linux-amd64 资产 102 主机真机复测:initialize + admin whoami + admin /orders 3/3 通过。
