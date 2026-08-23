# 配置模板自定义能力交付与验收

## 已完成

- `provision_templates.content` JSONB 自定义配置内容
- `version` 版本自增
- `status` ENABLED/DISABLED
- `updated_at` 更新时间
- 后台编辑、启停、删除未被任务引用模板
- 页面 `/provision/template`
- 接口 PUT/DELETE/status 与 OpenAPI、CLI 路由目录同步

## 验证

本地后端 `go test ./...` 通过；前端 typecheck/test/build 通过（45 文件、247 测试）。

已推送 main 提交 `85b8bf4`，102 的旧 `GET /provision-templates` 当前仍返回不含 `content/version/status/updatedAt` 的旧响应。说明新迁移和新 PUT/DELETE/status 路由尚未在 102 镜像生效，不能宣称线上模板自定义已验收；需等待 deploy-102 runner 重建后逐路由回放。
