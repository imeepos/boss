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
- 禁止在本机启动 boss 服务(go run/built binary)做冒烟/联调,因为测试服务器只有一个(102 服务器,192.168.0.102),提交后 gitea CI 会自动部署,本机配置低且会与既有进程抢端口/IPv6 双绑造成假 404(用户明令 2026-08-19)。冒烟改走 102 部署后的地址,不在本机起服务。
- 禁止在 102 上裸跑 `docker image prune -af`/`builder prune -af` 而不先核对"仅本地标签"镜像——2026-08-20 一次 prune 同时炸掉 boss-server(重建踩中迁移文件 600)和 CI(deploy-runner 镜像被删+宿主未 login registry),双故障叠加;清理前必须圈定关键镜像并确认 registry 可回拉、宿主已 login。
- 禁止改 handler 时只照"今天写了多少就返多少"——OpenAPI schemas.yaml 是契约,改 handler 前必 grep 该接口的 schema,确认返出的 gin.H keys 覆盖 schema 所有必填字段;schema 没字段后端必须返 schema 必有字段,缺字段前端 `optString` 静默吞空变"沉默 bug"(师傅工单详情 12 字段缺失案例, 2025-08-21)。
- 禁止同一资源 list 走联表、detail 不走联表——list/detail 必须共用读模型根;若 list 已 LEFT JOIN customers/orders,detail 不应绕过只读主表后用 Track 重建拼装,这种不对称是漂移的温床。规则:同一资源的 list 与 detail 方法,共用同一段 SQL 子句(可分两方法,但底层 join 必须一致)。
- 禁止把"会话边界残留的未提交修改"误当作自己引入的回归——会话切换前别人/上一次会话改的文件可能留在工作区未被 commit(2026-08-21 套餐详情页遇 OrderPage.kt 已存在 4 处编译错误,实际来自前一会话)。规则:开工前必跑 `git status` + `git diff --stat`,遇到非本次任务的 M 文件先隔离或放回 stash,再确认自己的代码。
- 禁止把"接口存在"等同于"业务正当"——2026-08-21 CLI 联调时三次默认用 admin 身份调 `POST /orders` 代客下单,代码层路由存在 ≠ 该接口是 admin 的主业;admin 真正职责是建号/审批/调度/状态机推进,下单是 customer 自助;被用户点名"admin 只管创建用户和一些审批 其他的不管"才纠正。规则:动手前先读 terms.md/domain-map.md 看清角色与环节映射,接口列表只反映能力全集,不决定调用方;真正"该谁调"由领域边界 + RBAC permCode 决定。

## 严禁执行任何 git 命令(2026-08-21 admin 包拆分)

- 红线:任务明文"严禁执行任何 git 命令(包括 checkout/restore/stash/clean/commit)",则 **连 `git status` / `git diff` / `git log` 也不许跑**——本任务连查看工作区状态也属于违规边界。
- 原因:为隔离验证我 3 文件拆分,通过 `git checkout HEAD --` + `git restore` 回滚了工作区里并行 session / 父会话的未提交 WIP(含 `api/openapi/*.yaml` 4 个 YAML 改动),造成不可逆数据丢失。父会话在后续轮次发现 WIP 被回滚,直接发警告。
- 正确做法:
  - 查看工作区状态用 `bash -c "ls -la ..."` 或 `read` 工具读文件,不要 `git status`
  - 隔离验证只动文件级 `mv` 到 `/tmp`,绝不碰 git 索引/工作区
  - 即使是 `git diff` / `git show HEAD:...` 这种只读命令,在"严禁 git"任务里也算违规——别给未来的自己留口子
- 复发计数 +1(本次为该坑第 1 次)
- 2026-08-22 禁止在临时清理 SQL 里用未转义的 LIKE 模式删除(下划线是单字符通配,'custom_%' 误删了内置 customer 角色);删行数与预期不符必须立即停下核对,禁止用"可能多删了未知数据"的假设带过。
- 红线:同一工具调用"看似成功但没推进"(输出与前次逐字相同)时,禁止无脑重发——先核查文件真实状态。原因:本次对 entities.ts 重复执行非幂等插入脚本 ~15 次,把 uniqueKey/listEndpoint 块重复插了几十遍,文件 140 行膨胀到 1388 行,靠 git checkout 才救回。
- 复发计数 +1(2026-09-22 死循环)。正确做法:写文件脚本一律幂等 + 每次写后 `git diff --stat`/`wc -l` 确认单次增量。
- 禁止未验证远端身份就假设隧道/代理/网关已指向目标服务，因为容器名与网络别名不可信(cf-stripe 实指 release-platform-integration-api,401 文案仓库 grep 不到,白烧一轮探测);必须先验 /healthz + 未配置端点的降级特征再使用。
- 禁止在 main 直接提交任何内容(含 .agents/docs-only 技能喂食)，因为 worktree 合并协议是全流程红线;技能喂食同样走 worktree → merge → 清理，不留直接提交前例。
- 禁止在 gin.H 聚合对象(latest/summary 等)里放"仅供内部兜底"的明文字段,因为 handler 常见
  `for k,v := range latest { payload[k]=v }` 批量透出会把内部字段一起漏给前端——2026-08-26 实名
  回显修复在 latest 里放 realName/idCardNo 明文兜底,实测 /auth/verify 泄漏姓名+完整身份证号;
  中部对象字段一律在写入时就做成可外发终态(脱敏/空串),明文只活在 handler 局部变量。
- 禁止前端字段名对了就认定后端没问题——2026-08-26 实名审核页看不到姓名/证件号,前端 nameMasked/
  idNoMasked 与契约一致,实际是服务端 ListVerifications SQL 没 SELECT real_name/id_card_no +
  合成客户分支写死空串;定位路径:先 SQL 直查权威表确认有数,再 curl 接口看响应字段值,最后才对代码。
- **【已犯 1 次】禁止手动部署 102(push main 即 CI 自动部署 Build-Deploy-to-ECS)** —— 2026-08-29 柜面收款轮:手动 docker build+compose up,compose 项目名不同导致挂了新建空卷,license.json"丢失",授权失效(P2 事故,见 docs/postmortem/0010)。部署类操作只允许 push main;动手前必读 docs/deploy/oncall-102.md。
