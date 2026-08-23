# Q1 CS/AR 知识库部署验证（2026-08-24）

## 本地交付

知识库 CRUD 已合并主分支：

- `GET/POST /api/admin/v1/knowledge-articles`
- `PUT/DELETE /api/admin/v1/knowledge-articles/{articleId}`
- 页面 `/boss/knowledge`
- `cs_knowledge_articles` 使用既有 000118 表

本地后端 `go test ./...`、前端 `typecheck/test/build` 已通过。

## 102 结果

已将 `main` 推送到 gitea，远端 main 与本地提交一致（`fb402b6`）。102 的 `boss-server` 与 `boss-admin-web` 容器现已重建并健康运行。

真实 CRUD 回放（使用 admin JWT，测试文章 `LIVE-Q1-VERIFY`，最后已删除清理）：

1. `POST /knowledge-articles` → `id=1`
2. `PUT /knowledge-articles/1` → `ok=true`，状态更新为 `PUBLISHED`，版本变为 2
3. `GET /knowledge-articles` → 返回更新后的文章
4. `DELETE /knowledge-articles/1` → `ok=true`
5. 再次 `GET /knowledge-articles` → `items=[]`

健康检查：

- `http://192.168.0.102:28080/healthz` → HTTP 200
- `http://192.168.0.102:5180/` → HTTP 200

结论：知识库后端 CRUD 已在真实 102 环境验收，测试数据已清理；页面构建和路由已通过本地前端门禁。
