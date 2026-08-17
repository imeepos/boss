# mock 假数据服务

按 `api/openapi/` 契约返回假数据,零依赖手工路由,不解析 yaml。

## 启动

```bash
node api/mock/combined.js   # 用户+师傅合一,默认 0.0.0.0:8090,MOCK_PORT 覆盖
node api/mock/admin/server.js      # 管理后台,默认 0.0.0.0:8092,MOCK_ADMIN_PORT 覆盖
```

守护循环启动用 `scripts/mock-{user,worker,admin}.sh`(worker 脚本以 MOCK_PORT=8091 起 combined)。

## 结构

- `combined.js` + `lib/` + `routes/` —— 用户端+师傅端合一入口:`/api/v1`(用户)、`/api/worker/v1`(师傅)、静态托管 docs/user 与 docs/worker
- `data.js` —— 用户端假数据;`worker/data.js` —— 师傅端假数据,JSON 字段统一 lowerCamelCase
- `admin/` —— 管理后台(前缀 `/api/admin/v1`),假数据按域拆分 `admin/data/*.js`
