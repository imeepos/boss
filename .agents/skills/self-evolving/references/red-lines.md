# Red Lines

<!-- 格式：禁止 X，因为 Y 发生过。真的付出过代价才记。 -->

- 禁止用 bash cat/head 代替 Read 工具读"待编辑"的文件，因为 edit/write 会因未观察而拒绝；同一会话踩过两次。
- 禁止复用上轮 CDP 截图的 Chrome profile 拍对照图，因为 localStorage 状态泄漏让"亮色"截图拍成了暗色。
- 禁止用假数据/mock 替代真实后端做开发验证，因为造假掩盖后端真实问题（用户明确驳回 mock 登录方案）；dev 免登录用真实 /auth/login 换来的 JWT 经 `?token=` 注入（scripts/dev-token.mjs）。
