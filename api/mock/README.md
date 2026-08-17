# mock 假数据服务

按 `api/openapi/` 契约返回假数据,零依赖手工路由,不解析 yaml。

## 启动

```bash
node api/mock/combined.js   # 三合一,默认 0.0.0.0:8090,MOCK_PORT 覆盖
```

守护循环启动用 `scripts/mock-{user,worker,admin}.sh`(worker=8091、admin=8092,均以 MOCK_PORT 起 combined)。

## 结构

- `db.js` —— 三端共享的关系型事实库(客户/订单/工单/端口/资产/LOID/账单外键关联),单一事实源;主键统一 uuid(`lib/store.js`),支持真实增删改查:`/api/admin/v1/crud/{table}[/{uuid}]`(GET 列表/详情、POST 新建、PUT 更新、DELETE 删除,内存态重启即还原)
- `selfcheck.js` —— 关系不变量自检(`node api/mock/selfcheck.js`),改 db 后必须全绿
- `combined.js` —— 唯一入口,三前缀聚合:
  - `/api/v1` 用户端(路由 `routes/user.js`,视图 `data.js` 由 db 派生)
  - `/api/worker/v1` 师傅端(路由 `routes/worker.js`,视图 `worker/data.js` 由 db 派生)
  - `/api/admin/v1` 管理后台(路由 `routes/admin.js`,视图 `admin/data/*.js` 由 db 派生,新增域只需加 data 文件)
  - 静态托管:`/` → docs/user,`/worker/` → docs/worker,`/admin/` → docs/admin
- `lib/http.js` —— CORS / 请求体 / JSON 响应 / 静态文件(防穿越)
- JSON 字段统一 lowerCamelCase;调试用 `GET /api/admin/v1/_routes` 列出全部 admin 路由
