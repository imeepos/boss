# Red Lines

<!-- 格式：禁止 X，因为 Y 发生过。真的付出过代价才记。 -->

- 禁止用 bash cat/head 代替 Read 工具读"待编辑"的文件，因为 edit/write 会因未观察而拒绝；同一会话踩过两次。
- 禁止复用上轮 CDP 截图的 Chrome profile 拍对照图，因为 localStorage 状态泄漏让"亮色"截图拍成了暗色。
- 禁止用假数据/mock 替代真实后端做开发验证，因为造假掩盖后端真实问题（用户明确驳回 mock 登录方案）；dev 免登录用真实 /auth/login 换来的 JWT 经 `?token=` 注入（scripts/dev-token.mjs）。
- 禁止在需要定制观感的顶栏/工具栏里用原生 `<select>` 做语言等枚举切换器，因为 option 弹层由系统渲染无法用 CSS 定制（暗色主题下仍是系统白色），且方框样式与 ghost 图标按钮视觉割裂，被用户点名"不美观、与 antd pro 不符"。
- 禁止改完布局 CSS（width/padding/flex/overflow）只跑 tsc/测试就交付，因为类型检查对视觉回归零覆盖——width:100%+padding 横向溢出（无 box-sizing 重置）就是这么漏出去的；必须目视或 CDP 截图确认。
- 禁止 CI 部署时随机生成 JWT/密钥类 env 兜底,因为每次部署轮换会使全部登录 token 失效、用户被集体登出;必须 gitea repo secret 固定注入,未配置直接失败并提示配置位置。
- 禁止把"healthz 返回 ok"当作"本次 CI 部署成功"的证据,因为部署在 compose up 前失败时旧容器仍在服务;必须核对 actions 日志或容器镜像 tag。
- 禁止新写/改任何组件或页面时出现硬编码文案或硬编码颜色,因为多主题与多语言是硬性规则(用户明令):文案必须走 i18n(本项目闭环= types.ts + zh-CN/en-US/ms-MY 三份 locale 同步),颜色必须走主题 token/全局 CSS 变量;写代码前先扫一遍现有 JSX 是否残留裸字符串标签和裸色值。
- 禁止完成任务后不提交代码就收尾,因为用户明令"完成一项任务一定要及时提交相关代码"——门禁(typecheck/build/test)通过后立即 git commit,一笔任务对应至少一笔提交,提交信息写清改动点与验证方式。
- 禁止在 UI 里用原生 <select> 新增下拉——option 弹层系统渲染无法随主题定制,已两次被用户点名;一律用 components/Dropdown.tsx(触发器+浮层 listbox+打勾+外部收起)。表单内遗留的原生 select 待逐步替换。
- 禁止用文字字形(▾ ✓ × →)当图标——视觉重量不足且各平台渲染不一;一律描边 SVG。
- 禁止在总结里声称"已适配/已验证"而没有对应验证动作(grep 令牌定义、双主题截图、build)——用户会信以为真,静默失败就是这样漏过去的。
- 禁止把“新增 light/dark CSS 令牌 + typecheck/build 通过”当作表单多主题适配已完成，因为浏览器实际计算样式仍可能错误；必须在真实页面验证输入背景、文字、placeholder、边框、focus、只读态和按钮。
- 禁止把 build/test 通过当作 UI 交互已验证——下拉/搜索问题必须在真实业务 DOM 中断言点击后的控件文本、筛选结果和 URL 同步，否则用户会再次发现“能展开但不能选中”。
- 禁止自造设计体系结构(分页/表格/抽屉等)而不先取 antd 一手规范——antd GitHub components/<name>/index.zh-CN.md 是一手来源。
- 禁止收尾总结后留未提交改动——"完成"的定义含 git commit;门禁四件套 = typecheck + test + build + commit(git status 必须干净),已 3 次靠用户提醒才提交。
- 禁止在机制层面证明前把复现的 bug 结论为"环境/工具怪象"(模拟器 input tap 怪、机型差异)——本会话因此放走真 bug 数轮,用户真机复现才回头;凡"不可能"行为一律先插桩拿调用栈 ground truth。
- 禁止在并行 agent 共享的工作区里让已验证的修复停留在未提交状态——工作区会被 git checkout/clean 随时回退;验证通过的下一个小动作就是 commit。
- 禁止自研 Compose 组件调用点用裸尾随lambda传点击动作(组件末位是 @Composable 插槽时必绑错)——动作一律 onClick = 命名参数显式传递。
