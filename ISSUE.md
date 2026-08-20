# ISSUE.md（上游/工具问题清单）

## web/admin

- **信息不准｜`web/admin/scripts/dev-token.mjs` 已失效**：脚本按旧前缀 `POST {baseUrl}/auth/login` 请求登录，后端实际前缀是 `/api/admin/v1`（`src/lib/serverConfig.ts` 的 `API_PREFIX`），运行直接 HTTP 404。应改为 `/api/admin/v1/auth/login`，或删除脚本改由 curl + localStorage 注入（skill docs 已记录替代做法）。
