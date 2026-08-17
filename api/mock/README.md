# mock 假数据服务

按 `api/openapi/` 契约返回假数据,零依赖手工路由,不解析 yaml。

## 启动

```bash
node api/mock/combined.js   # 三合一,默认 0.0.0.0:8090,MOCK_PORT 覆盖
```

守护循环启动用 `scripts/mock-{user,worker,admin}.sh`(worker=8091、admin=8092,均以 MOCK_PORT 起 combined)。

## 结构

- `combined.js` —— 唯一入口,三前缀聚合:
  - `/api/v1` 用户端(路由 `routes/user.js`,数据 `data.js`)
  - `/api/worker/v1` 师傅端(路由 `routes/worker.js`,数据 `worker/data.js`)
  - `/api/admin/v1` 管理后台(路由 `routes/admin.js`,数据 `admin/data/*.js` 按域拆分,新增域只需加 data 文件)
  - 静态托管:`/` → docs/user,`/worker/` → docs/worker,`/admin/` → docs/admin
- `lib/http.js` —— CORS / 请求体 / JSON 响应 / 静态文件(防穿越)
- JSON 字段统一 lowerCamelCase;调试用 `GET /api/admin/v1/_routes` 列出全部 admin 路由
