# Red Lines

<!-- 格式：禁止 X，因为 Y 发生过。真的付出过代价才记。 -->

- 禁止用 bash cat/head 代替 Read 工具读"待编辑"的文件，因为 edit/write 会因未观察而拒绝；同一会话踩过两次。
- 禁止复用上轮 CDP 截图的 Chrome profile 拍对照图，因为 localStorage 状态泄漏让"亮色"截图拍成了暗色。
- 禁止用假数据/mock 替代真实后端做开发验证，因为造假掩盖后端真实问题（用户明确驳回 mock 登录方案）；dev 免登录用真实 /auth/login 换来的 JWT 经 `?token=` 注入（scripts/dev-token.mjs）。
- 禁止在需要定制观感的顶栏/工具栏里用原生 `<select>` 做语言等枚举切换器，因为 option 弹层由系统渲染无法用 CSS 定制（暗色主题下仍是系统白色），且方框样式与 ghost 图标按钮视觉割裂，被用户点名"不美观、与 antd pro 不符"。
- 禁止改完布局 CSS（width/padding/flex/overflow）只跑 tsc/测试就交付，因为类型检查对视觉回归零覆盖——width:100%+padding 横向溢出（无 box-sizing 重置）就是这么漏出去的；必须目视或 CDP 截图确认。
