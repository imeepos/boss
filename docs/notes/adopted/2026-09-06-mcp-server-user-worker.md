# MCP server 接入裁定(bossmcp stdio,2026-09-06)

## 决策

MCP 支持以**本地 stdio server**(`cmd/bossmcp`)承载:agent 宿主拉起进程,
经 `X-API-Key` 调 102 既有 REST 接口。不新增服务端端点、不动网关路由、不加 Go 依赖
(MCP JSON-RPC 子集自实现,~120 行)。

工具面收敛为 3 个:`boss_routes`(目录发现)/ `boss_call`(任意端内接口)/
`boss_whoami`(身份确认),路由目录由 `api/openapi` 生成
(`internal/apiroutes`,与 bossctl 目录同源同门禁)。

## why

- stdio 是 MCP 客户端最通用传输,零部署变更;102 既有 API key 认证直接复用。
- 契约单一事实源:接口增删改只需 `make bossctl-routes` 再生成,工具能力自动跟随。

## 放弃了什么

- **服务端内嵌 MCP(SSE/HTTP transport)**:要求 102 部署变更与长连接网关配置,
  收益仅是远程 agent 免装二进制;需要时再加 transport(生成器/客户端已按端参数化)。
- **每接口一个 MCP tool(用户端 95 + 师傅端 67 = 162 个)**:客户端上下文爆炸,
  工具选择噪声大;3 工具 + 目录发现覆盖全部端点。
- **复用 cmd/bossctl 客户端代码**:bossctl 是 package main 不可 import,抽共享包要大改
  CLI;改为 `internal/apiclient` 精简实现(信封+认证+query 编码,~140 行,独立单测)。

## 边界

- admin 端不开放(三表主体边界):customer/worker key 不进管理面;`boss_call`
  跨端完整路径结构性拒绝。
- multipart 上传(附件/APK)不经 MCP 工具,agent 场景走 `bossctl upload`。

## 关联

- 使用文档: `docs/mcp.md`
- 实现: `cmd/bossmcp`、`internal/apiclient`、`internal/apiroutes`
- 决策先例: 2026-08-22 worktree 合并协议(本功能同样走 worktree→ff-only 合入)
