# mock 假数据服务

按 `api/openapi/` 契约返回假数据,零依赖手工路由,不解析 yaml。

## 启动

```bash
node api/mock/combined.js   # 三合一,默认 0.0.0.0:8090,MOCK_PORT 覆盖

node api/mock/server.js            # 仅用户端,默认 0.0.0.0:8090,MOCK_PORT 覆盖
node api/mock/worker/server.js     # 仅师傅端,默认 0.0.0.0:8091,MOCK_WORKER_PORT 覆盖
node api/mock/admin/server.js      # 仅管理后台,默认 0.0.0.0:8092,MOCK_ADMIN_PORT 覆盖
```

守护循环启动用 `scripts/mock-{user,worker,admin}.sh`。

## 结构

- `server.js` + `data.js` —— 用户端(前缀 `/api/v1`)
- `worker/` —— 师傅端(前缀 `/api/worker/v1`),假数据集中 `worker/data.js`,JSON 字段统一 lowerCamelCase
- `admin/` —— 管理后台(前缀 `/api/admin/v1`),假数据按域拆分 `admin/data/*.js`
- `combined.js` + `lib/` + `routes/` —— 三合一入口与共享工具
