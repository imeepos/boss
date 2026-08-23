# Q1 CS/AR 知识库部署验证（2026-08-24）

## 本地交付

知识库 CRUD 已合并主分支：

- `GET/POST /api/admin/v1/knowledge-articles`
- `PUT/DELETE /api/admin/v1/knowledge-articles/{articleId}`
- 页面 `/boss/knowledge`
- `cs_knowledge_articles` 使用既有 000118 表

本地后端 `go test ./...`、前端 `typecheck/test/build` 已通过。

## 102 结果

已将 `main` 推送到 gitea，远端 main 与本地提交一致（`bfd1dc6`）。102 的 `boss-server` 与 `boss-admin-web` 容器健康运行，但真实请求：

```text
GET http://192.168.0.102:28080/api/admin/v1/knowledge-articles
→ HTTP 404，非 JSON `404 page not found`
```

结论：当前 102 容器仍未包含知识库路由，线上页面不能宣称已验收。需要 102 的 Gitea runner 完成 `.gitea/workflows/deploy-102.yml` 对 main push 的构建/重建后，再复测 CRUD 和页面。未使用 mock 数据替代线上验证。
