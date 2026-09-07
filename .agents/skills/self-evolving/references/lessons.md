# Lessons

<!-- 一条经验一行。格式：当 X 发生时，修复是 Y。skill 没提前警告我。 -->

- 当两个控件共用同一个 aria-label(如趋势周期与列表筛选都叫「周期筛选」),querySelector 与可达性同时受损:屏幕阅读器分不清控件,自动化定位拿到错误元素——断言失败先怀疑「标签不唯一」,修正文案键本身(trendPeriodLabel 拆分),而不是绕道换选择器。2026-09-07 报告中心轮。

- 当需要给"需登录的 Web 页面"截图且没有 Playwright 时，修复是系统 Chrome `--headless=new --remote-debugging-port` + Node>=22 全局 WebSocket 裸 CDP（脚本见 scripts/cdp-capture.mjs）。skill 没提前警告我。
- 当 CDP 截图要覆盖 localStorage 驱动的状态（如主题）时，修复是每个状态显式 `localStorage.setItem` 后 `Page.navigate` 重载再拍，或每次运行换全新 `--user-data-dir`；不复用上轮 profile。skill 没提前警告我。
- 当自动化填充 React 受控 input 时，修复是用 `Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set` + `dispatchEvent(new Event('input',{bubbles:true}))`，直接 `el.value=` 不触发 React 状态。skill 没提前警告我。
- 当要给描边色写死的第三方 SVG 图标换色（随 currentColor/主题）时，修复是 CSS `mask: url(...) center/contain no-repeat; background: currentColor`（alpha 遮罩，与原 SVG 色无关），而不是 `<img>`（无法改色）或逐个内联（量大）。skill 没提前警告我。
- 当悬浮按钮要钉在滚动容器可视区角落时，修复是 `position: fixed`（相对视口，配合底栏高度算 bottom），`position: sticky; bottom` + float 的组合对放置位置敏感、容易失效。skill 没提前警告我。
- 当 edit 的 old_string 覆盖到文件末尾时，修复是去掉 old_string/new_string 的结尾换行（文件可能无尾换行导致匹配失败）。skill 没提前警告我。
- 当页面表现异常（白屏/交互失效/数据不对）时，修复是先采浏览器 console 报错 + 网络请求清单再分析，而不是只看截图猜；`cdp-capture.mjs --logs out.json` 一次拿到两者（用户经验传授）。skill 没提前警告我。
- 当枚举切换器（语言/单位/主题）混在一排 ghost 图标按钮里时，修复是"图标按钮 + 自定义下拉浮层（role=listbox + 当前项打勾 + 点击外部收起）"，样式复用现有 token，明暗主题自动适配。skill 没提前警告我。
- 当顶栏要放退出登录时，修复是收进头像下拉（信息头 + 个人设置 + 危险色退出项，antd Pro 惯例），不在顶栏平铺独立退出按钮。skill 没提前警告我。
- 当页面出现第二个及以上下拉浮层时，修复是把浮层骨架（定位/背景/边框/阴影/选项行）写成多选择器共用样式，新菜单只写差异部分。skill 没提前警告我。
- 当侧栏菜单激活项可能滚出视野时，修复是 useEffect 监听 pathname，仅目标在可视区外时 `scrollTo({behavior:'smooth'})` 对齐到视野中央；在视野内则不滚。skill 没提前警告我。
- 当同一文件在本会话已被编辑过、又要做第二轮 edit 时，修复是先 Read 最新内容再拼 old_string，凭上一轮记忆拼会 not found。skill 没提前警告我。
- 当块级容器想占满父级宽度时，修复是不写 `width:100%`（display:block 的 auto 已撑满且 padding 内含），`width:100%`+padding 在无 border-box 重置的项目里必横向溢出。skill 没提前警告我。
- 当滚动容器（overflow-y:auto）意外出现横向滚动条时，修复是量 `scrollWidth > clientWidth` 定位溢出元素，再查 box-sizing/固定宽/白溢出（img、whitespace、flex 最小宽）。skill 没提前警告我。
- 当在深色/彩色表面放 button 时，修复是显式写 color 不指望继承（span 继承、button/input/select 不继承，UA 默认 buttontext 黑），否则深底黑字不可见。skill 没提前警告我。
- 当用户报"某主题下颜色不对"时，修复是同时检查全部主题同位置——本例深浅主题顶栏同为深色，深色主题其实同病只是未被注意。skill 没提前警告我。
- 当模型不支持读图（read_image 报错）时，修复是跳过自动核验继续推进修复，改用 --eval getComputedStyle 做程序化颜色验证，最后总结列出"请人类目测"清单（用户明确指示）。skill 没提前警告我。
- 当在新 bash 会话里跑 go/docker 等命令报 command not found 时，修复是先 `export PATH=/opt/homebrew/bin:$PATH`（brew 工具全在此），不要结论"工具没装"。skill 之前未警告。
- 当给域 Service 接口（如 user.Service）追加方法时，修复是同一提交内同步补齐测试桩（fakeUser 等），编译错误清单就是桩清单。skill 没提前警告我。
- 当初始化超管/首个账号时，修复是走「启动引导 + ON CONFLICT DO NOTHING」：密码只从环境变量注入、绝不写进迁移或种子文件、已存在不覆盖（防重启重置密码）。skill 没提前警告我。
- 当部署 compose 需要密钥时，修复是 env_file 管 secrets（app.env 不入库，加 .gitignore），environment 段只留非密默认值；记住 compose 优先级 environment > env_file，要覆盖默认值得两处一起动。skill 没提前警告我。
- 当 sysadmin 角色被菜单门禁拒（403 no permission:menu:x）时，修复是先查远端库 `schema_migrations` 最新版本对比 `ls migrations/*.up.sql`——本项目权限全是 role_permissions 显式行，sysadmin 无隐式全权，迁移漏跑（漏插权限/授权）是首要嫌疑。skill 没提前警告我。
- 当手测 API 收到 42200 参数非法时，修复是先读后端请求 struct 再拼 JSON（如 geo attrs 的 timeZones 是 string[] 而非对象数组），不凭直觉猜字段类型。skill 没提前警告我。
- 当本机没有 psql 却要查/改远端 PG 时，修复是 /tmp 临时 go 程序 + pgx 直连 DSN（configs/config.example.yaml 有现成连接串），迁移文件是纯 SQL 可整文件 Exec。skill 没提前警告我。
- 当新增数据库迁移文件时,修复是先 `ls migrations/*.up.sql | tail -5` 确认真实最大编号——`ls | head` 截断列表曾让我险些撞号 000031(已被 order_no_seq 占用);代码注释里的迁移号(internal/domain/order/pg.go)也要 grep 交叉验证。
- 当 CI 全新 clone 后 compose up 报 env file not found 时,修复是 workflow 里从 example 生成 env 文件、密钥从 gitea repo secret 注入、缺失即 fail fast——被 .gitignore 的文件在无人值守环境必然缺失。
- 当验证 CI 部署结果时,修复是看 actions 日志或比对镜像 tag(GITHUB_SHA),curl healthz 只证明"有容器活着"——部署在 compose up 前失败时旧容器照常应答 ok。
- 当本机无 docker/psql 而要验证 SQL 迁移时,修复是 sqlglot(pip install --user sqlglot)按 postgres 方言 parse 全文件拦语法错;约束语义仍需真实 PG。
- 当 Go 工具链不在 PATH 时,修复是 export PATH=/opt/homebrew/bin:$PATH(AGENTS.md 已声明 brew 在此);go build 失败先查这个再怀疑代码。
- 当 cdp-capture 的 --eval 需要多步操作或含 await 时,修复是整体包进 `(async()=>{ ... })()`(顶层 await 直接 SyntaxError),登录→设主题→跳页→验证合成一条链,中间用 setTimeout Promise 等渲染。skill 没提前警告我。
- 当用 --logs 做程序化验证时,修复是结果必须 `console.log("VERIFY:"+JSON.stringify(v))` 再从日志里 grep VERIFY——挂在 window 上的状态不会出现在日志里(日志只收 console 事件)。skill 没提前警告我。
- 当设计抽屉/弹层组件 props 时,修复是 onClose(关闭 UI)与 onChanged(数据变化要刷新)分开两个回调,绑错会让抽屉永远关不掉或刷新不了。skill 没提前警告我。
- 当给后台页面写"每主题强调色"的主操作按钮时,修复是复用 --shell-fab-bg/--shell-fab-bg-hover/--shell-fab-icon 令牌(亮=品牌蓝、暗=品牌金),它是项目现成的主题自适应强调色,不必新造。skill 没提前警告我。
- 当用户报"某菜单/按钮缺图标"时,修复是先同时 `ls` 资源目录 + `grep` 渲染点,判定"资产缺"(补 SVG)还是"渲染缺"(补引用)——同一症状两种病因,曾连续两轮分别是这两种。skill 没提前警告我。
- 当新增菜单遮罩图标 SVG 时,修复是拷贝同目录现有图标的规格(24 viewBox/stroke 1.8/round cap),stroke 色值随意——mask 方案下实色由 background:currentColor 决定,与文件内颜色无关。skill 没提前警告我。
- 当顶栏空间紧张要做收展式搜索框时,修复是收起态复用 shell-tool-btn(与相邻工具按钮同排同规格),展开态切回 shell-search 椭圆;Esc 全清收起/空值失焦收起/提交成功收起三路径一次写全。skill 没提前警告我。
- 当多会话编排(DSH session_link_talk)目标连续返回空回复时,修复是不再反复 talk——空回复多为目标会话正处于纯工具轮;改查外部权威数据(102 接口/DB)判进度,既省墙钟又避免给目标会话注入噪音消息。
- 当编排多角色并行模拟时,修复是 Promise.all 并行发 talk(各 480s 超时,程序总墙钟≤600s),轮间等待用 bash sleep;曾一轮三会话并行 7m26s 完成,较串行省一半以上。
- 当批量归档会话后,修复是用 session_link_list 复核剩余列表——archiveSession 返回的 archivedSessionIds 是工作区累计归档清单而非本次提交集,曾险些误判;本次另发现一个历史会话被连带归档(单向不可逆)。
- 当用户嫌 gitea secret 配置麻烦时,修复是评估"内网私有仓库直接把 env 文件入库"(固定密钥+注释公网风险)——homelab 场景标准 secret 流程是过度设计,简单性优先;skill 之前推的 secret 方案被现场驳回。
- 当 app.env 配了超管口令但登录仍 40100 时,修复是查该账号 created_at/real_name:历史遗留账号(如开发期手建的 admin)会被 EnsureSuperAdmin 的 ON CONFLICT DO NOTHING 正确跳过,口令不会同步——直接 UPDATE password_hash(bcrypt 新哈希)对齐即可。
- 当 pgx/simple protocol 报 42P18 "could not determine data type of parameter $1" 时,修复是检查占位符编号:必须从 $1 连续编号($2 起头会报 $1 类型不明),必要时补 ::text 显式类型。
- 当接手一个"页面已上线才补多语言"的任务时,修复是按"标题→表单字段标签→占位符→按钮/提示"清单全量扫硬编码,而不是只改用户点名的那两处标题(本次 geo 两个编辑抽屉:标题早已走 i18n,真缺的是抽屉内十几个字段标签)。
- 当收尾总结前发现工作区仍有未提交改动时,修复是先 git status 分辨哪些是本任务产物(含上轮遗漏),按功能分笔提交再回复用户;多任务改动交织在同文件时,合并为一笔但提交信息逐项列明。
- 当同会话遗留了别的任务的未提交代码时(本次 geo 分页组件),修复是一并验证(typecheck)后随本任务补提交,不让工作区长期脏着。
- 当 edit 报 "requires reading ... first" 时,修复是用 read 工具读目标文件(部分行也行)再重试;bash 的 cat/sed/grep 输出不算"已观察",edit 门禁只认 read 工具。skill 曾在 notes 提过但没喂进 lessons,现补上。
- 当 react-router-dom v6(BrowserRouter)页面要"刷新/分享链接后搜索条件与分页不丢"时,修复是 useQueryState 封装 useSearchParams + setParams(...,{replace:true})(见 web/admin/src/lib/useQueryState.ts);与 lib/urlPrefs.ts(一次性覆盖写 localStorage 后抹参数)语义相反,别混用。
- 当后台列表页要补分页时,修复是复用 web/admin/src/components/Pagination.tsx(总数/区间 + 每页 10/20/50/100 + 上下页,safePage 自动钳位);本项目此前无任何分页组件,geo 是首例。
- 当给列表页加 URL 驱动的筛选状态时,修复是"筛选条件变化时同步 setPage(1)",否则翻到第 2 页再改关键词会出现空页。
- 当 i18n 新增 key 需要同步 3 份 locale + types.ts 四处时,修复是 locale 用 python 脚本批量替换(count==1 断言防错位)、types.ts 用 edit,改完跑 pnpm typecheck 一次验闭环。
- 当要自研组件对齐 antd/Pro 规范时,修复是先 curl ant-design GitHub 仓库的 components/<name>/index.zh-CN.md 拿一手 API/设计说明(比搜索博客准),按其默认值与语义实现(分页例:首末页恒显+当前±2+省略号、showTotal 区间文案、sizeChanger 10/20/50/100、quickJumper、单页不隐藏、aria-current)。skill 没提前警告我。
- 当任何新组件(分页 size changer、筛选器)需要下拉时,修复是复用 components/Dropdown.tsx(触发器+浮层 listbox+当前项打勾+点击外部收起);原生 <select> 的 option 弹层系统渲染、无法随主题定制,工具栏场景已被用户点名"奇怪"两次。skill 曾警告过顶栏场景,现推广到全部场景。
- 当组件 CSS 引用 var(--xxx) 却始终呈现 fallback 颜色时,修复是先 grep 该令牌在 theme/tokens.css / styles.css / geo.css 里是否真的定义——本项目曾引用不存在的 --shell-bg/--shell-border(实际叫 --shell-card-bg/--shell-card-border),暗色主题下静默变白块;自研组件按 geo.css 模式自带 :root[data-theme='light'/'dark'] 两套组件级令牌最稳。
- 当小图标(下拉箭头/打勾)显得异常小时,修复是别用文字字形(▾/✓)当图标——字体渲染笔画细、size 缩小后视觉更小;一律用描边 SVG(24 viewBox/stroke 1.8-2/round/currentColor),14px 显示即可与 antd 图标视觉重量一致。
- 当本机没有数据库却要验证迁移时,修复是先 grep configs/ 找现成远端 DSN 并问用户库在哪(本项目真库在 192.168.0.102:25432),不要 brew 装本地 PG(10 分钟超时白等);skill 没提前警告我(lessons 28 其实已有正解,开工前没回看)。
- 当 raw.githubusercontent 下载大文件(数 MB)反复断流时,修复是 `for i in $(seq 1 15); do curl -s --max-time 40 -C - -o f URL; python3 -c "import json;json.load(open('f'))" && break; done` 断点续传拼完——exit 0 不代表下完,完成判据是内容可 parse。
- 当"数据必须真实"的任务要选开源数据集时,修复是先验新旧口径标志(如菲律宾行政区划:ARMM 是 2019 前旧称,现叫 BARMM;省数 81→82 含马京达瑙分省),再对照官方统计口径(PSA PSGC)核对总数,第一个搜到的镜像可能是多年前旧版。
- 当要验证迁移可回滚且目标库是共享库时,修复是单事务 down→up 回环:脚本剥掉文件内 BEGIN/COMMIT,外层显式起事务依次 Exec 两个文件,down 后数行数、up 后回原量再 commit——零风险验证双向可执行+幂等。
- 当多轮下载/安装等待中被打断或超时(工具 spawn ENOENT、SIGTERM)时,修复是直接原样重试一次再排查——本轮 spawn bash ENOENT 与 600s 超时重跑均自愈,先怀疑环境抖动再怀疑命令。
- 当 pgx Exec/Query 报 `unused argument: 1` 时,原因是严格参数校验——语句占位符少于传入参数个数直接报错;修复是每条语句只喂自己的参数,或对内部生成的纯数字后缀直接 fmt.Sprintf 拼进 SQL(无注入面)。
- 当测试用 t.Cleanup 做数据自清理而清理里用连接池时,同测试的 `defer pool.Close()` 会先关池导致全批清理静默失败;修复是 pool.Close 也改 t.Cleanup(LIFO,先注册后执行,数据清理先跑)。
- 当按精确后缀清理测试数据总有残留时,原因是同一测试的子场景常自造独立后缀(e2e 的 W8 用 orderNo6() 另起一个);修复是按测试专用命名模式匹配(^(e2e|w8)[0-9]+$),顺带覆盖历史残留。
- 当 edit 工具修改后出现重复函数头/两行并一行时,原因是 new_string 与 old_string 范围不对称(顺手带了函数头/只删换行的 no-op);修复是 new_string 严格镜像 old_string 的边界,改完立刻 build。
- 当怀疑库里是测试垃圾数据时,先确认迁移无种子(grep INSERT),再按 path 前缀分组+created_at 对到具体测试文件,最后查 pg_constraint 依赖图定删除顺序;共享库被集成测试污染的入口几乎都是 BOSS_PG_TEST_DSN 指向了共享库。
- 当 Go API 返回 500 内部错误、且已确认非权限/参数问题（调用链走到 PG 实现层）时，第一步查 SQL SELECT 的 nullable 列有没有 COALESCE 包裹——pgx 的 `Scan(&int16)` 遇到 NULL 列会直接报错，不属于 `ErrNoRows` 等已知错误类型，落入 `respondErr` 的 `default` 分支返回 500。`GetSubdivision` 已正确使用 `COALESCE(osm_admin_level,0)`，但 `ListSubdivisions` 遗漏了。skill 没提前警告我。
- 当列表筛选既要刷新恢复又要保证控件即时响应时，修复是 URL 只做首次初始化，交互更新本地 state 并通过独立 setter 同步 URL；不要把 `useSearchParams` 的实时值直接作为控件渲染源。skill 没提前警告我。
- 当用户报告控件“点不中/选不中”时，修复是用真实页面 DOM 断言点击后的控件文本、筛选结果和 `location.search` 三者同时变化；build/test 只能证明代码可编译，不能证明交互链路。skill 没提前警告我。
- 当要制作个人中心或设置类页面时，修复是先调研 Ant Design Pro 一手范式并输出信息架构/状态清单，再采用“紧凑账号头部 + 左侧分区导航 + 右侧单任务内容面板”，不要先堆叠多张功能卡片。skill 没提前警告我。
- 当用户要求多主题表单适配时，修复是先盘点表单的背景/文字/placeholder/边框/focus/按钮/只读态，再为 light/dark 各自定义语义令牌；typecheck/build 不能证明视觉正确，必须在真实页面做双主题 computed-style 或截图验证。skill 已有相关警告，但本次前一轮未执行。
- 当在 worktree 的子目录执行 git add 时，修复是先用 `git rev-parse --show-toplevel` 确认仓库根目录，路径按根目录解析；门禁结束必须在根目录检查 `git status --short`。skill 没提前警告我。
- 当项目有定制设计系统（非 shadcn default theme）且需要 shadcn-style UI 组件时，修复是手动创建组件（参考 shadcn 编码模式：forwardRef + cn + cva），不走 `npx shadcn@latest add` CLI。原因：生成的组件使用 `hsl(var(--primary))` 等默认 CSS 变量，与项目现有的 `--color-brand-*`/`--shell-*` 设计令牌不兼容，手动改写的工作量不亚于直接写。skill 没提前警告我。
- 当工具超时时，不要急着归因到网络。先做排除：① 去掉管道重试看真实输出；② 检查是否在等交互输入（加 `-y` 或 `--yes`）；③ 检查目标 URL 是否可直达（`curl -v` 看连接耗时）；④ 检查本地 registry 配置（`npm config get registry` / `pnpm config get registry`）。skill 没提前警告我。
- 当整理跨年度研发计划时，修复是先以契约和现有路线图建立基线，再用年度主题、季度出口指标、横向工程主线、明确不做项和季度治理机制约束范围；不要把未建能力域直接承诺为无验收条件的功能清单。skill 没提前警告我。
- 当把年度路线继续拆成季度时，修复是每季固定写“目标、重点计划、交付物、验收指标”，并同时标注规划序号与自然季度，避免周期起点和季度编号混淆。skill 没提前警告我。
- 当季度计划涉及待建能力域时，修复是先核对 domain-map 的已建/待建状态，再在每季写明确不做项；这样能避免 WHO、RA、FMS、SET 等未具备边界的域被远期愿景顺手承诺。skill 没提前警告我。
- 当规划 AI、预测维护和规模化交付时，修复是按“数据底座先于模型、建议先于自动执行、单区域验证先于多区域复制”安排依赖，并为每季写人工兜底、审计、回滚和隔离验收；不要把智能化目标写成脱离事实源的自动化口号。skill 没提前警告我。
- 当审计路线图与实现状态时，修复是同时核对代码、OpenAPI、迁移、测试/验收报告和真实环境记录，并把“有代码”“部分闭环”“已验收”分开；页面或局部接口存在不能直接证明季度目标完成。skill 没提前警告我。
- 当审计 PORT 与开放平台时，修复是把“后端/API 或仓库功能已落地”与“门户前端/真实环境生产交付已验收”分开；PORT 不能用 user 端点替代前端门户，openplat 不能用本地 e2e 替代 102 写路径复验。skill 没提前警告我。
- 当审计稳定性与税务时，修复是把“基础迁移/小库恢复”与完整生产灾备（保留、异地、告警、凭据、大库 RTO/RPO、多节点自动恢复）分开，并把内部税务轨迹与外部税局适配器、签名、异步回执和生产联调分开。skill 没提前警告我。
- 当路线图季度标签与日期周期错位时，修复是以实际日期季度作为审计主键，并将“提前实现代码”与“对应季度正式生产验收”分开；同时检查主链路残余状态和性能报告，不只看成功率摘要。skill 没提前警告我。
- 当审计 CS/AR 服务域时，修复是把迁移、模型、只读指标和查询回放与完整业务闭环分开，逐项核对写入、状态审计、SLA/升级、回访评价、账龄快照、催收生成、承诺还款/核销、任务回放和统一工作台。skill 没提前警告我。
- 当把差距审计转成开发计划时，修复是按依赖将生产基线、灾备税务、异常运营、业务工作流、数据治理、AI、预测维护和规模复制分阶段，并为每阶段定义可验证出口；未达出口不得进入后续智能化阶段。skill 没提前警告我。
- Playwright 断言页面标题时,侧边栏菜单/面包屑/页内 h2 三处同文案会触发 strict mode violation:一律用 getByRole('heading') 而非 getByText(2026-08-18, e2e 冒烟首跑 2 失败均此因)
- 跑前端 e2e 前先确认 vite proxy 的 BOSS_API_TARGET 指向真实后端(vite.config 默认 102:28080),playwright webServer.env 里覆盖才生效(2026-08-18)
- 当用户反复强调"保存/记录账号密码 API key"时,修复是当场用 write 工具落盘 JSON 并 `ls` 确认存在,不要只口头答应"会保存"——本会话因只答应不执行被用户连催四次,浪费多轮(2026-08-19)
- 当创建带组织绑定的资源(部门/岗位/账号)报 42200 时,修复是先读对应契约字段类型再拼 JSON:整数 id 传字符串必 42200(如 POST /departments 的 legalEntityId 必须整数);"先读后端请求 struct 再拼 JSON"适用于一切 42200(2026-08-19)
- 当要给业务流选审核/操作账号时,修复是先 GET /menu-perms 查矩阵里目标权限码归属哪些角色(如 menu:dispatch -> ops/sysadmin/technician),再据此决定建号 roleCode——不要默认只有 sysadmin 能操作(2026-08-19)
- 当 check-contract-sync A 报"路由已实现但契约未登记"、而子文件已加路径时,修复是往顶层 api/openapi/admin.yaml 补同路径的 `$ref` 行(~1 编码 /),正则只匹配同行带 $ref 的路径、不递归子文件(2026-08-19)
- 当 go test 某包报编译错误而自己没改过那些文件时,修复是先 git status + stat 时间戳判断是否为并行 Agent 正在同工作区开发(未跟踪新文件+时间戳接近当前),识别为非自己回归,只验证自己领域包、不改他人正在写的文件(2026-08-19)
- 当前端从 Vite 代理切换为后端绝对地址直连时,修复是服务端新增按环境变量白名单校验 Origin 的 CORS 中间件，并用 httptest 覆盖允许来源 OPTIONS 预检与未知来源拒绝；不要用 `*` 配合 credentials。(2026-08-19)
- 当 POST /provision/ports 等「置备/联调」接口报 FK violation(50000 内部错误)时,修复是相关联的 FK 字段(resourceId/orderId 等)必须用**前面创建接口真正返回的 id**,不能凭 fixture/直觉写小整数(如用 resourceId:1 但真实资源是 228)——先看创建响应的返回 id 再拼后续请求(2026-08-19, 开网 CLI 模拟)
- 当扫码绑定(VerifyScan)报 40920「扫码与预绑定不一致」时,修复是检查标签是否真的绑定了资产的 `bound_asset_id`:VerifyScan 查 `tags WHERE epc_code` 要求 `bound_asset_id != 0`,且 MATCH 需 `asset_id == link.AssetID`;正确顺序是**先建资产,再以 `boundAssetId`+`status:BOUND` 创建标签**(CreateAsset 不会回写 tags.bound_asset_id),创建时 status=UNBOUND 的标签永远扫码不 MATCH(2026-08-19, 开网 CLI 模拟)
- 当要为 customer 主档保存"账号密码"时,修复是明确 BOSS 客户主档(customers)**没有登录口令**,鉴权全走 API key:`POST /api-keys` 支持 `subjectType: customer`(subjectRef=customers.id,名称必填),签出的 plainKey 可直接 `bossctl --api-key <key> call GET /auth/me` 以客户身份自证;客户主体密钥不带菜单权限(RBAC 恒拒),test-accounts.json 的 username/password 记 null、放 apiKey(2026-08-19, 开网 CLI 模拟)
- 当 UI 出现"理论上不可能"的行为(一次点击多次跳转/事件穿透)时,修复是立即在关键函数加 Log.d(msg, Throwable) 打调用栈拿 ground truth——本会话空谈"Compose 事件不可能跨重组派发"数轮,真机复现+栈日志五分钟定位是渲染期执行。
- 当自研 Compose 组件最后一个参数是 @Composable 渲染插槽(right/content)时,修复是动作回调用显式命名参数 onClick = 传,禁止裸尾随lambda——尾随lambda永远绑最后一位,且 () -> Unit 可静默赋给 @Composable 版本,编译器不拦。
- 当"登录成功但所有请求 401"时,修复是先 `adb shell run-as <pkg> cat shared_prefs/*.xml` 看 token 是否落盘,再 curl 同端点对照——多半是响应信封解析错位(mock 平铺 vs 真实 {code,data} 嵌套)。
- 当对接真实后端替换 mock 时,修复是先用 curl 打一遍关键端点核对响应结构(信封/字段层级),再写解析代码;mock 的平铺结构会掩盖信封差异。
- 当与并行 agent 共享工作区/真机时,修复是修复一验证通过立即 git commit(未提交的工作区会被僵尸进程 git checkout 掉);装完 APK 用 `dumpsys package <pkg> | grep lastUpdateTime` 确认没被覆盖再下结论。
- 当真机与电脑时间对不上时,修复是先 `adb shell date` 对时区差(本例手机慢 9 小时),再比对 lastUpdateTime——直接比数值会得出"我的安装没生效"的错误结论。
- 当后端验证码只落库不发短信(未接短信网关)时,修复是发码接口 curl 触发后用临时 go+pgx 查 portal_sms_codes 表拿真码(5 分钟有效,一次性);测试师傅账号存 .agents/skills/bossctl-cli/test-accounts.json。
- 当"已注册的路由返回 404 / 服务看起来没上新代码"时,大概率不是代码问题,先查端口是否被另一个进程用 IPv4/IPv6 双绑;curl 走 IPv4 会打偏。2026-08-19。
- 当为 Go 接口造测试 fake 时,让 go vet 一次性列出全部缺失方法再批量补,不要撞一个补一个。2026-08-19。
- 当单测 `:=` 赋值报 mismatch 时,目标函数返回多值就改成 `v, _ :=`。2026-08-19。
- 当 Android App(targetSdk 35+) 页内返回键点不到/系统返回直接退出时,修复是根布局加 windowInsetsPadding(WindowInsets.safeDrawing) 让顶栏避开状态栏(edge-to-edge 默认绘制到屏幕顶端,顶部~90px 触控被状态栏吃掉),并给自维护导航栈配 BackHandler(enabled=stack.size>1){pop()};症状:uiautomator 显示按钮 bounds 正常但 input tap 无响应(2026-08-19 worker App)
- 当需要冒烟/联调后端时,修复是等 102 服务器在提交后自动部署,直接用部署地址验证——本机只有一台测试服务器(102)且本机配置低,不要在本地 go run 起服务(2026-08-19 用户明令;本地起服务还撞端口双绑假 404)。
- 当规格/设计稿要求"mock 数据"时,修复是仍按用户裁定拒绝 mock(2026-08-20 再次点名),直连 102 真实服务(192.168.0.102:28080)curl 取真数;user 端取 token 全流程:POST /auth/sms-code → go+pgx 查 102 库 portal_sms_codes(列: phone/scene/code/expires_at/used,注意无 created_at/expired_at) → POST /auth/login {mode:"sms"} 拿 data.token。
- 当本机 gradlew 报 "Unable to locate a Java Runtime" 时,修复是 export JAVA_HOME=/opt/homebrew/opt/openjdk@17( brew openjdk@17 已装);且管道接 tail 会吞退出码,看 EXIT=${PIPESTATUS[0]}。
- 下游编码 AI 无视觉能力时，"设计稿→提示词"模板必须强制转写与默认组件外观的差异（容器色/指示器/渐变/异形头部），只写组件名（如 NavigationBar）会让无视觉模型退回 material3 默认样式，视觉完全走样。
- 写"设计稿→提示词"类模板时，占位符内禁止出现裸的具体数值/色值示例：填模板者可能不对照设计稿直接照抄，把臆造值当成真实规格。示例只给"要回答哪些维度"，数值必须标注"从设计稿量取后填入"。
- (2026-08-20) 模型不支持图像输入时,Compose 视觉验收可 screencap 拉回本地后用 PIL 逐像素断言(渐变入状态栏/底栏高度 px=dp×density/选中 tab 蓝色像素数),比肉眼读图更可量化;配合 uiautomator dump 断言文案与 bounds。
- cordis 预设 YAML 的 `!!js` 标签只支持 scalar：标记数组必须逐项 `- !!js >- expr`，整表打 `!!js` 会 YAML 解析失败（schema kind: "scalar"）。
- 动态插件沙箱里没有 setTimeout/setInterval：用 ctx.timeout 需 inject ["timer"]；宿主 console.log 外部读不到，探针结果用"失败即抛错 + cordis_inspect_self"回传。
- gpt-image-2 图生图（/v1/images/edits）的 multipart 文件字段名是 `image`，不是 `image[]`；传错返回 400 "Missing required file field image"。（2025-08-20 实测）
- bossctl 二进制全局 flag 是 `-server`(不是 --url/_base query);报 flag not defined 先看 -h。
- 三等分 weight 列里的大字号数值(金额/日期)必须按最长内容校验宽度:maxLines=1 + softWrap=false + 字号留余量;32sp 在 1080p 下必溢出变形(首页账单卡片踩过)。
- 移动端生图 prompt 不写"可滚动/最小行高48dp/按钮44dp/边距16dp"等硬约束，gpt-image-2 会把所有区块塞进一屏并全面缩水（profile-v1 实测：行高24dp、卡片间距4dp、通栏大红退出按钮）。硬约束清单已固化为 ui-proto 预设 M5 元模板。
- spec 的设计 token 表要和生图 prompt 同源：先写 spec 后生图时必须把 token 逐项翻译进 prompt，否则 spec 约束实现、图却另一套（profile-v1：spec 写了 44dp，图里 31dp）。
- AI 设计稿"丑"的六大结构性根因：塞一屏/间距崩坏/组件样式混用/平台归属混乱/破坏性操作过重/小字低对比——评审时按这六类逐项打勾，不用自由心证。
- 生图前先从项目基准页截图识图提取 UI-SPEC（token/组件/图标/平台特征），再按 spec 生图——盲写 prompt 生成的设计稿必然与项目风格漂移（profile v2/v3 教训：prompt 自造的 #1E3A8A 渐变和基准页 #0872F4→#1698FA 完全不同）。
- boss 用户端基准页是 iOS 视觉语言（9:41/Home Indicator/四栏 Tab），新设计稿默认沿用 iOS 风格，禁止混 Android 元素；规范已固化在 designs/UI-SPEC.md。
- "设计优秀"可逆向为可执行规则：从优秀基准稿识图提取的 15 条决策（色彩80/15/5比例、三级表面、单一焦点、卡片三段式、状态双线索、动作分级、悬浮快捷卡）已固化 designs/UI-PARADIGM.md + 预设 M6 元模板——生成前直接套用整套 prompt，不再凭感觉。
- gpt-image-2 位图生成没有精确尺寸概念：行高/字号/间距写进 prompt 也只能按"比例感"渲染，5 轮实测(v2-v7)行高稳定 24-34dp 达不到 48dp。位图只做风格方向稿，dp 精确值由 spec.md 承担，像素级还原靠实现后 --diff 验收。
- 参考图风格锁定：单屏裁剪图远优于多屏拼图（拼图稀释信号）；sips -c H W --cropOffset y x 可零依赖裁剪。
- 生图 prompt 粗粒度四层（用途/元素功能/核心风格/token色字体，~150词）效果优于逐dp微观详述——用户点名+五轮实测确认：越细越死板，dp标准归验收和spec，设计自由度留给模型；参考图（单屏裁剪）优先于文字描述。
- 生图prompt首句必须先定场景："高保真UI设计稿+{手机|平板|PC}端+页面名+全屏预览(整图即屏幕,无边框无标注无水印)"，再跟四层要求——开篇不定场景模型会自加画板装饰。
- Compose 的 Modifier.offset 只移视觉不缩布局槽:压卡/上移要用在"整个容器"上,套在单个卡片上会给后续元素留出原高度的死间隙(profile 快捷卡 32dp 空隙踩过)。
- Compose Text 直接挂 heightIn(min)+background 当按钮,文字不居中:最小高度交给外层 Box(contentAlignment=Center),Text 只做内容。
- uiautomator dump 抓不到 Compose 渐变头等未暴露语义的文本(搜不到≠没渲染);能抓到的节点用 bounds 数值断言(单行/位置/间距)比截图靠谱,模型不支持看图时是首选验证法。
- 短信验证码 5 分钟一次性:发给用户前提醒时效,报"登录失败"先查 portal_sms_codes 的 expires_at/used 再怀疑链路。
- 当 cdp-capture 断言登录后页面但总是跳回 /login 时: 每次运行是新 profile,localStorage 种子不跨运行;单次运行内 seed→location.href→eval 三段式。
- 共享工作区可能被并行进程自动 commit;验证收尾用 git log -- <file> 而非仅 git status。
- 2026-08-20 部署唯一正道:push gitea main 触发 .gitea/workflows/deploy-102.yml(CI 构建镜像→推 192.168.0.102:5000→compose up 102:28080);无 102 ssh 权限,禁止本机自启后端对接。
- 2026-08-20 CORS 白名单禁止枚举 vite 端口:vite dev 端口随占用漂移(5173→5175…),localhost/127.0.0.1 源应不限端口放行(internal/pkg/server/server.go originAllowed)。
- 2026-08-20 共享工作区有并行会话:暂存区文件可能被别的会话一并 commit;代码就绪后立即自行提交,不留 staged 过夜。
- Compose clip 圆角要可见,前提是被裁区域有与背景可区分的不透明底色;透明区域 clip 在同色背景上等于没裁(滚动区圆角蓝对蓝踩过)。
- "滚动区域整体圆角且滚动中恒在"的标准实现:包裹 Box(padding 交点 + clip(顶部圆角) + 不透明 bg),内部 Column 只管 verticalScroll;不要依赖某张卡自己的圆角。
- 空 Box 靠 padding 撑高度:内容移走后高度塌缩;量高度(onSizeChanged)的层必须就是视觉呈现层。
- statusBarsPadding/windowInsetsPadding 会消费 insets,同 Box 内后绘 sibling 再取 statusBars 高度得 0;需提前在消费前量取(asPaddingValues 先算好)。
- 首帧计算出的 padding 可能为负(测量前默认 0),Compose 负 padding 直接 IllegalArgumentException 闪退;一律 coerceAtLeast(0.dp) 或改用 Spacer。
- shell 链上 && adb install 在 build FAILED 时仍会执行到旧 APK 并报 Success;自动化里必须先判定 BUILD SUCCESSFUL 再 install。
- UI 需求含空间词(内圆角/交点/遮挡)且第一次实现被打回时,第二次就问,选项按"卡片级/区域级/头部级"分层给,不要同层连猜。
- 多会话共用工作区:自己的改动 build+验证通过后立刻 commit,否则会被并行会话的 git add -A 裹进无关提交。
- "多页面视觉/行为保持一致"的需求,解法是抽共用组件+单点参数对象(PinnedHeaderSpec),不是各页调参后对比;组件一致则视觉必然一致,对比修补是无底洞。
- 多会话共享仓库:pull 后先 assemble 一次确认基线可编译,再开始自己的改动;别人提交坏代码会阻塞你,最小修复(unlock)优于绕行。
- 工具调用被打断(abort)后,该轮的 build/install/commit 可能悬空——继续工作前先 git status 核对。
- 用户报告"UI 元素时有时无/悬停才出现"类异常,先让他硬刷新并确认访问地址(dev localhost / 102 部署 / 构建产物),再查代码;DOM 计算样式正常而用户看不到 = 客户端陈旧(HMR/缓存),不是代码 bug。
- admin 免登录冒烟不写表单 eval:登录页是 placeholder 受控 input 无 id;直接 localStorage 注入 boss.token + boss.servers + boss.server.active(见 docs/boss-admin-web.md),再导航目标页。
- 后端 API 前缀是 /api/admin/v1(不是 /api/v1);登录 POST /api/admin/v1/auth/login,信封 data.token。
- 新增 admin 菜单项必须同时补 public/icons/items/<key>.svg(描边 #8b98a5, viewBox 24, stroke 1.8),否则侧栏该行无图标——menu.def.ts 的 key 就是文件名。
- 2026-08-20 102: docker-clean.sh 的 `docker image prune -af` 会删掉"仅本地标签、从未 push"的镜像(如 boss/deploy-runner)和 registry 登录前依赖本地缓存的一切;清理脚本后必须验证 deploy-runner 等关键镜像可从 registry pull,且宿主要先 docker login 192.168.0.102:5000(凭据在 ~/boss/deploy-image/dotdocker/config.json)。
当 X:mobile/user/android 下直接 ./gradlew 报 "Unable to locate a Java Runtime" 时,修复是 Y:export JAVA_HOME 为 /opt/homebrew/Cellar/openjdk@17 下 Contents/Home(同 scripts/build-install-user-android.sh)
- pgx QueryExecModeSimpleProtocol 下把 []byte(json.Marshal 结果)直接当 jsonb 参数传,会被格式化成 "[123 34 ...]" 文本,PG 报 22P02 invalid input syntax for type json;jsonb 参数一律传 string(raw)(2026-08-20 portal SavePrefs/PutMessage 实锤)
- Gitea CI(102)按 commit 构建镜像并自动重启 boss-server:本地 docker build 因镜像源 TLS 超时失败时,git push gitea 即等效部署(2026-08-20)
- 多 subagent 并行改同一 Android 模块:按页面文件白名单分组+共享文件(Widgets/Theme)由主 agent 独占,5 agent 并行零冲突(2026-08-20)
- 共享工作区提交前必看 `git diff --cached`:并行进程可能已把它的文件暂存进 index,直接 `git add 我的文件 && git commit` 会把 index 里别人的暂存一并卷入;混文件(如 openapi user.yaml)用 `git hash-object -w` + `git update-index --cacheinfo` 只暂存自己的 hunk(2026-08-23 stripe 接入踩过,reset --soft 重做)。
- 契约命名门禁(check-contract-sync B)会扫描 json tag:解码外部渠道 snake_case 响应别用 struct tag,用 map[string]any 取字段。
- 当门禁脚本输出"OK"但带计数时,先检查计数是否为 0 或异常小——扫描目录迁移后 0 条路由也能全绿,假阴性比 FAIL 更危险(2026-08-23 contract-sync A 扫旧目录 internal/app,22 条路由漂移无人发现)。
- Compose 组件内部链了 `.padding(top=X)` 时,外部再传 `Modifier.padding(top=0)` 是 no-op(内部 padding 在后,覆盖归零);要归零必须给组件加显式参数(如 topPadding=0)。(2026-08-21 worker 首页 OverviewCard 顶距,用户真机二次点名)
- 2026-08-21 共享真机禁盲目 reboot:先 `dumpsys window policy | grep secure=true` 判断有无 PIN 锁,锁死即永久失去 adb 可控性;SystemUI 卡死优先转 emulator 而不是重启真机。
- 2026-08-21 Kotlin 不存在 `Modifier.包名.函数` 点号调用扩展;Modifier 链必须先 import 再裸函数名连缀,编译错误一眼即辨。
- 2026-08-21 when 校验的 else 分支必须显式返回"通过"哨兵(如 ""),`else -> "agree"` 会把门控条件变成永久失败,UI 自动化点击无响应时优先查这类恒假分支。
- 2026-08-21 实名流程: 后端 scene=verify 验证码端点 + verifications 附件三列(000070)已在 102 上线;合成客户(注册未建主档)无手机号,发码端点正确 40400,端上须有失败提示通道。
- 2026-08-21 多状态设计稿先做频率分层:高频状态等分占屏,低频状态(如注册)降为小字链接——用户点名"常用放最显眼位置",prompt 须写明入口级元素"视觉层级最低"。
- 2026-08-20 用户否定'订单按客户归属判公司'的推导:多地址客户场景下应按安装地址→区域→运营主体判定;领域判定问题先枚举实体的物理约束(一客户多地址)再选主判据,不要默认外键继承.
- 并行会话共享工作区时,提交前必须 `git add <明确文件清单>` 而非 `git add -A/-.`;2026-08-20 ad27799 把并行会话未提交的 6 个文件扫进 docs 提交,只能靠未推送的 rebase 拆分补救。
- cdp-capture 冒烟带鉴权页面:URL 先指到 /login,settle 后 eval 写 localStorage(boss.token/boss.servers/boss.server.active)再 location.replace('/目标页');直接reload会停在 /login(初始无token已被守卫重定向)。
- 脚本批量在 import 区插行时,不要按"以 import 开头的行"定位插入点——多行 `import {` 会被拦腰插入;应找完整 import 语句(或 `} from` 行)之后插入。(2026-04-11 admin confirm 替换)
- cdp-capture.mjs 每次运行是新浏览器上下文:localStorage 注入必须与页面导航在同一次调用里完成,且最后一个 --eval 用 async 等待+返回文本做断言;eval 执行在 --settle 之前。(2026-04-11)
- 页面直访 404/空白时先 grep router/menu.def.ts 的真实 path(菜单分组带前缀,如 /alarm/alarm),不要猜 /alarm。(2026-04-11)
- 当订单处于 stage 2 时，修复是预占入口先打开资源核查，只有核查 available 后才 POST `/orders/:orderNo/reserve`；直接预占会返回 42200 参数非法。skill 没提前警告我。
- 当 React 页面动作按钮可能被表单包裹时，修复是所有非提交动作显式声明 `type="button"`，避免隐式提交导致导航。skill 没提前警告我。
- 移动端"点击按钮无反应"先怀疑服务端状态没翻转,别只查前端:本例 accept 返回 code:0 但只回填 worker_id 不改 status,列表刷新后观感"没反应"。排查顺序 = 前端事件链(uiautomator dump 断言) → curl 同端点 → 查服务端写库逻辑。(2026-08-21 师傅端领取工单)
- 测试替身里手工改状态(fw.tickets[0].Status="DOING")会掩蔽服务端不落库的 bug;fake 必须真实执行状态流转,断言才能守住回归。(2026-08-21)
- 当测试报 "invalid worker token" 时,先检查 signWorkerToken 是否发生在 t.Setenv(BOSS_JWT_SECRET) 之后——顺序颠倒会签出错误密钥的 token。(2026-08-24)
- 当部署文件存在于本机与服务器两侧且非 git 同步时,改文件前必须先 `ssh server cat <path>` 确认服务器版本,edit 本机副本后用 `scp` 推上去;改完用 `ssh ... git status` 或 `docker compose config` 在服务器侧验证,而非凭本机 git diff。skill 没提前警告我。(2026-08-21 MinIO compose)
- 当需要给容器注入密钥时,优先 long-syntax `volumes: [bind]` 挂文件;`top-level secrets:` + `secrets: [..]` 块在某些基础镜像(UBI Micro、Distroless)上即使 compose config 渲染正常,容器内 `/run/secrets/` 也可能不存在。skill 没提前警告我。(2026-08-21 MinIO)
74. Node22/Chrome 现版 JSON.parse 报文是「Unexpected token 'x', "片段" is not valid JSON」,无 position/line;行号定位只能靠 Firefox 的 "line N" 正则,其余回落无行号文案(2026-08-2x importer 预览)。
75. 提交时暂存区存在并行会话遗留文件,用 `git commit -m ... -- <本任务路径>` pathspec 提交,不动他人在场改动(2026-08-2x importer 提交避开 mobile/*.kt)。
76. pgx v5 回扫 timestamptz 得到的是「Go 进程本地时区」的 time.Time，不是 UTC——代码注释声称"DB 时间戳按 UTC 扫描"属错误假设；时区审计先实测（pgx 连库 SELECT now() 回扫看偏移），再信注释(2026-08-21 时区审计)。
77. 时间正确性常靠「DB会话=进程=UTC」三重巧合维持：审计时区须同时查 SHOW TimeZone、DSN 是否带 TimeZone、容器 TZ env 三处；任何一处单方面改变都会碎(2026-08-21)。
- #78 门禁(typecheck/test/build)通过后立即 git commit,再跑耗时的 E2E/CDP/双主题验证;共享工作区有并行会话时,验证耗时窗口就是被扫提交/被回退的窗口(2026-08-21 importer Excel 导入被并行会话混提交)。
- 当 INSERT 拼 `const factSnap + 显式列` SQL 时,占位符总数必须现场重数列数(factSnap 是 7 列不是直觉的 8),pgx 报 insufficient arguments 第一反应应是数列,不是加参数。pgxmock 单测对多余占位符不报错,会静默通过,必须跑真库集成测试才能抓住。
- lesson: locale/types.ts 插入 i18n 块时,同形尾部 key(pageUnit/rangeText)在多个 section 重复,必须用"目标段的段名行+下一段段名"做唯一锚点,不能凭尾部 key 模式定位(2026-08-25 notif 块插进 company 段,三份 locale 全错返工)。

- 重发同一 edit 前先确认上次结果:一次消息里重复的 edit 会双倍插入,重试成功的 edit = 制造重复块(2026-08-25 wiring.go Push 块 x2)。
- 共享工作区有并行 agent 时:自己 write 的文件可能被对方改写/提交,提交前用 git log/diff 认领自己的产物,不重提对方的中间态(2026-08-25 push-config 与 backup 并行)。
- admin 页面 UI 实测:localStorage 注 boss.servers 必须 [{id,name,baseUrl}] 且 active=id;无 vite 代理,API 直连 baseUrl;DOM 断言优先于截图(2026-08-25)。
- lessons 74: 共享工作区并行会话可能把你的未提交改动卷进它的混合提交——完工即自commit,不给别人代提交的机会;发现被卷提交不可 revert 拆分,只能记录(7d0d984/fe30e60)。
- lessons 75: cdp-capture 首个 --eval 偶发在 about:blank 上执行(localStorage SecurityError)——注入与 location.href 合并成一个 eval,第二个 eval 只做轮询断言,一次成功。
- lessons 76: vite 突然 500 "Failed to resolve import"先查并行会话是否 mid-edit 共享入口文件(App.tsx),等 1-2 分钟再curl该模块确认,别急着改自己代码。
| 74 | 2026-08-21 admin | lazy 路由页首载 chunk 时整页闪烁 = 唯一 Suspense 边界挂在布局树外层(整壳被 fallback 卸载);修法:在布局 Outlet 外加局部 Suspense,壳层保持挂载 |
- lessons 77: Tailwind v3 无动态 spacing 刻度,v4 写法(min-w-45/w-27/w-130)被 JIT 静默丢弃不报错——宽度塌缩/换行错乱先查 getComputedStyle 的 width/minWidth;v3 一律任意值 min-w-[180px]。全仓已知 ~20 处同类残留(ServerManagerDialog/params/servers 等),未修。
- lesson: 102 容器以 app 用户(uid 1000,alpine adduser -D)运行,空命名卷首挂继承镜像目录属主——镜像里 mkdir+chown 才保险;已存在的旧卷必须 docker run --rm -v <vol>:/d alpine chown -R 1000:1000 /d 人工修一次。
- lesson: apiFetch 发 FormData 时绝不能带 Content-Type: application/json,要让浏览器补 multipart boundary(client.ts 已修,新调用方直接传 FormData 即可)。
- lesson: Go 服务新增依赖可写目录的功能时,deployments compose(BOSS_BACKUP_DIR + 命名卷)与 Dockerfile(预建目录)必须与功能代码同一批提交,否则 102 部署即 nil 服务。
- lesson(2026-09-06 P5-W2):run_code 生成含 ${/'/反斜杠 的 bash 脚本,占位符法一次成型——内容行全用单引号 JS 串,@@SQ@@/@@BS@@/@@DS@@ 代替三类禁写字符,写盘后 python3 replace(chr(39)/chr(92)/chr(36)+chr(123)) 一步还原;逐行手工转义必炸 parse error。
- lesson(2026-09-06 P5-W2):bash 管道 `cmd | tail` 会吞退出码,`pnpm test | tail` 假绿 exit 0——判断门禁结果必须 `echo ${PIPESTATUS[0]}` 或不带管道单独跑;同轮在主树复跑同用例可判定存量/新增。
- lesson: CI(deploy-102.yml)只构建/部署 Go server 镜像;web/admin 前端验证一律本地 pnpm dev + localStorage 注入 102 token 直连。
- lessons 77: CDP 断言按钮文案前先 grep locale 实际 key 值再写正则;样例数据先读目标 schema(如 preview.ts ADDR_FIELDS),shape 不符会误判组件故障。
- lessons 78: push 后 102 新接口仍 404 = CI 部署延迟(~1-2 分钟),20s 轮询直到生效,勿回滚排查自己的路由。
- lessons 79: 本仓库 push 远端名是 gitea(ssh://git@192.168.0.102:222/sker/boss.git)不是 origin,git push origin 会直接报无权限;push 后 CI 自动部署 102。
- lessons 80: 并行会话可能把你编辑中/已测完的文件抢先 commit(2026-08-21 worker keyword 被 41a0125 收走且混入他人 popover.ts);提交前后各 git status 一次,发现他方文件混入自己提交时在总结中明确说明而非默默接受。
- lessons 81: admin 选择器三件套已沉淀在 web/admin/src/components/pickers/(EntityPicker 基座:服务端 keyword 检索 + DetailDrawer 详情 + 前往管理页跳转),新表单选用户/师傅/客户直接复用,别再各页自造。
- lessons 82: CDP 断言自研 Dropdown:打开浮层必须对触发器 click(),选中选项必须 dispatchEvent(new MouseEvent('mousedown',{bubbles:true}))——onChange 绑在 onMouseDown 上,click() 选不中;页面常有多个 listbox 触发器,先按 aria-label/innerText 枚举定位再操作。
- lessons 83: CDP --logs 的 network 条目是最硬的断言证据:选中选择器后刷新,直接在日志里 grep 请求参数(如 bills?customerId=213),比读 DOM 文本可靠(React 重渲染时机会让 textContent 读取扑空)。
75. subagent failed 通知不可信:先 list_agents 看实况,僵尸 agent 会继续写文件;提交前对每个非预期未跟踪文件查来源(mtime/内容)再处置。
76. 拆分大文件先扫同目录既有拆分模式(components/AttachmentManager、geo/styles.ts、app/wiring_*.go),新文件命名与职责边界对齐惯例,review 成本最低。
77. commit message 里的量化结论(行数/用例数)必须来自实测命令输出,不许凭记忆估。
78. 并行 agent 在场时父会话禁用 git stash 诊断——stash -u 会卷走 agent 半成品,pop 又与其并发写冲突;诊断"失败是否预存在"用 git show HEAD:<file> 对比或临时 worktree。
79. 补 OpenAPI 契约先 grep 域 yaml:多数"漂移"只是聚合 face.yaml 缺 $ref 行(定义早已存在),2 行修复而不是重写定义。
80. 未提交的关键改动在有并行会话/agent 的环境里立即 commit——工作区随时可能被别人的 git 写操作回滚。
- worktree 基线编译失败先 `git stash -u` 验基线再自查;并行方会往 gitea/main 推破损中间态(漏 add 新文件最常见)(2026-08-25 worker-android)。
- JPush 5.x 集成:只加 cn.jiguang.sdk:jpush 依赖 + manifestPlaceholders["JPUSH_APPKEY"],不写 meta-data(AAR 已带占位符)(2026-08-25)。
81. 免登录 CDP 验证 admin 页时,?token= 会在 auth 探测失败(如 boss.servers 未配)时被 logout 清掉;正解=先开 /login,eval 注入 boss.token+boss.servers+boss.server.active 后再 location.href 目标页(2026-08-21 admin 状态组件验证)。
82. cdp-capture 的验证结果用 eval 返回对象(stdout 打印)而不是 console.log——涉及 location 跳转的用例里,console 采集常落在导航前,VERIFY 抓不到(2026-08-21)。
83. 代码改完跑门禁前先 commit 一版草稿:长验证流程(DOM 双主题断言)中途,并行会话可能把工作区改动扫进它的巨石提交,提交纪律已被破坏且无法干净拆分(2026-08-21 2c34af5 混装)。
84. 改集成测试前先跑基线:红了一片时先分类(环境漂移/存量测试债/自己的改动),别默认是自己改坏的——本仓 e2e 曾整红 1 天无人发现(2026-08-21 e2e 归属推导测试债)。
85. 部分唯一索引(WHERE status IN ...)的表做批量 UPDATE 到索引内状态时,同键多行会 23505:批量标记语句必须带"每键至多一行"守卫(已有活跃行不标/取最小 id)(2026-08-21 Reconcile CONFLICT)。
86. 画稿"中规中矩"根因是 5 个维度都打 5 分;高级感是在对的维度上克制(颜色/圆角/装饰)、对的维度上极致(节奏/字号/留白)。Linear 用 510/590 字重、Stripe 用 300 细体大字、Vercel 用 box-shadow 代替 border——每个"反常识"决定背后都是反 SaaS 默认值的克制。设计稿前必读 `references/knowledge/design-aesthetics.md`。
85. 机械验收运行器有秒级硬超时时,e2e 冷/热分离是标准解:对源码树做内容指纹(git ls-files -s + status --porcelain 哈希),指纹未变且产物含特征串则跳过重建直接断言;ssh 类高频断言按「一次连接跑多条 SQL 只回显 SELECT 行」批处理(2026-09-06 P5-W1 验收 35s→稳态 4s)。
87. 生图 prompt 的 Style 段不要写"现代/简洁/专业"等空词;翻译成可执行的设计语言——editorial / restrained / technical luxury / like Stripe or Linear + 具体的字号跳跃/字距收紧/圆角上限/焦点圈双层。空洞词被模型按"通用 SaaS"理解,正是"中规中矩"的源头(2026-08-25)。
88. 节拍检测法:设计稿缩到 25% 后眯眼看——能立刻找到 3 个明确"组"说明节奏对,平均分布就是 24/24/24/24 的平庸节奏。节拍三件套=字号敢跳(14→24 不是 14→17)+留白敢空(主标题上下 32px+)+分组敢疏(区块 32-48px、组内 8-16px)。
- 当 102 验收/E2E 出现「订单 DONE 但 provision 任务 FAILED/卡住」时，修复是先查 boss-provisioner 的 BOSS_PROVISION_DRIVER 与 TL1 解析链四要素(nms_oltid/PON 三维/TL1 内容模板/task_no 预置)，部署驱动切换会让旧验收夹具静默失配。skill 没提前警告我。
89. 多 worktree 并行时 vite/playwright 默认 5173 易被同机别的工作区占用(测试串台,看着像组件挂掉实则打到了别人的 dev server)。正解:playwright.config 把 baseURL/port/webServer.command 都从环境变量读(PW_PORT),执行时 `PW_PORT=5291 pnpm exec playwright test` 隔离。验证后端端口也别忘同样处理,前端先 curl `/src/App.tsx` 看返回的源码路径就能秒判(2026-08-26 remote-desktop trackC)。
86. 给领域 service 接口加方法前,先 grep 全部实现方(含测试 fake):embedded interface 的 fake 不受影响,显式逐方法实现的 fake 会漏,build 红一轮才发现一处(2026-08-21 RollbackStage)。
81. 路由注册必须放在 register* 前缀函数内:check-contract-sync 的提取器只扫 register 开头函数,把 g.POST 写进 setup*/其他命名函数会静默逃出契约对账(362<->363 少计无告警)。重构 root.go 时警惕。
82. 多 cd 链式命令一处失败后续命令会在错误目录执行(含 git stash!)——跨 checkout 操作一律用绝对路径或 workdir 参数,绝不用相对路径串联 cd。

## 2026-08-22 partner 入驻域(后端+前端全链路)
74. PG 行含 NULL 列时,SELECT 列表必须对每个可空列 COALESCE(x,0) 再扫 int64;否则 List/详情首查即 500("can't scan into dest: cannot scan NULL into *int64"),本地无库跑不出来,102 实测才暴露。
75. 新增域错误必须同步登记 internal/pkg/httpx/error.go 的 RespondErr 映射表,否则业务冲突(重复审核/重复提交)落 default 500"内部错误",用户无法判断原因。
76. 多阶段 Dockerfile 的依赖安装层,COPY 清单必须包含全部 install 前置输入(pnpm-workspace.yaml 的构建白名单也是);pnpm 新版把 ignored build scripts 升级为硬失败,pnpm@latest 漂移会让 CI 无代码变更也断,固定版本号。
77. 102 admin-web 部署在 5180,nginx 同源代理 /api/→boss-server;浏览器冒烟不要设 boss.servers(直连 28080 会撞 CORS 白名单预检 404),让 apiFetch 走相对前缀即可。
78. 102 部署 CI(deploy-102)的 compose up --force-recreate 会把容器留在 Created 不启动,部署后需人工 docker start;遇到 404 别急着重试 push,先 docker ps -a 看容器状态。
79. React 受控 input 程序化填值直接 el.value= 无效,必须 Object.getOwnPropertyDescriptor(原型,'value').set.call(el,v)+dispatchEvent(input);(补强 techniques #4,本次再次验证)
- 当内网服务(102:xxxx)要经公网 138(43.240.223.138, ssh ops@22, ufw 只放行表内端口)暴露时,修复是 102 上 systemd 常驻 `ssh -N -R 127.0.0.1:15180:127.0.0.1:5180 ops@43.240.223.138`(服务名 boss-5180-tunnel) + 138 nginx sites-enabled/boss-5180 listen 5180→15180(带 websocket 头) + `ufw allow 5180/tcp`;GatewayPorts=no 恰好把远端绑死 loopback,由 nginx 出公网。skill 没提前警告我。
- 当 sshd 拒绝带选项的 authorized_keys 行时,修复是 `sudo sshd -d -p 2223` 起 debug 实例 + ufw 临时放行该端口,日志 "bad key options" 一行即定位;注意 sed 的占位前缀会被 sshd 当选项解析。skill 没提前警告我。
- 当连不上"记忆中的"公网端口时,先看 `sudo ufw status`:138 只放行 22/3773/8787/4873/43770/80/8888/8899/8080/8090-8092/8788/8789/5173/3478/49160-49360,其余 TCP 全 DROP,症状=ping 通但端口超时。skill 没提前警告我。
- 当 cdp-capture.mjs 跨多次运行共享 localStorage 状态(语言/主题)时,修复是不行——每次运行全新 profile,setItem 后另一次运行读到 null;必须在同一次调用的多个 --eval 里 set→location.reload()→轮询断言。skill 没提前警告我。
- 当冒烟脚本点"下一步"没跳步时,先核对测试数据本身是否满足校验(如信用码必须整 18 位),再怀疑页面逻辑——本次连续两次自造数据长度不够,校验其实一直在正确拦截。skill 没提前警告我。
- 当 gpt-image-generate.mjs 文生图报 400 "Unknown parameter: 'response_format'/'style'" 时,修复是从 buildPayload 删掉对应字段——当前代理端点不认这两个参数,gpt-image-2 默认即返回 b64_json,输出分支本已兼容 b64_json/url 两种。
- 当在 web/admin 子目录用相对路径跑 `.agents/skills/.../xxx.mjs` 时,修复是 cd 回仓库根再跑——Node 报模块找不到只打出版本号尾巴,先想路径再想环境。
- 当列表型配置页前端用统一 `Row.id` 取主键列却显示 undefined 且 toggle/disable 死链 `xxx/undefined/...` 时，修复是 Tab 定义自带 `idKey: string` 按后端 SQL `AS "..."` 别名一一标注（addonId/couponId/denomId/customerId/faqId/guideId/id），渲染与动作 URL 都按 `row[def.idKey]` 取；同步加 tabs.test.ts 锁住映射。后端主键名与前端假设不一致是无告警漂移，grep `pg_lists.go` 的 SELECT 列表是唯一对账源。skill 没提前警告我。
- 2026-08-22 批量脚本插入 locale 行要自带尾逗号,插完立即 typecheck(types/三份 locale 四处同步时用脚本尤其注意)。
- 2026-08-22 regen 生成物(如 bossctl 路由目录)会夹带他人域的历史漂移,提交前必看 diff,只手工保留本次变更相关条目。
# 84 (2026-08-22): 发现半成品未提交代码时,先 git worktree list 判断是否他人(并行会话)的活跃 worktree——是则绝不进入编辑,另起自己的 worktree;用户点名"一人一个 worktree"。
# 85 (2026-08-22): 临时验证分支永远用 `git add <明确路径>` + commit,绝不 -am;删除前 `git status` 确认主工作未随行。
# 86 (2026-08-22): 多 worktree 并行时命令必须显式传 workdir,git mv/add 这类"就地生效"操作在默认目录执行会直接污染主分支。
- 2026-08-22 仓库远端名是 `gitea`(ssh://git@192.168.0.102:222),无 `origin`;worktree 收尾第①步 push 前先 `git remote -v` 确认远端名,push origin 必 128。
- 2026-08-22 macOS 系统 bash 是 3.2(无 mapfile/readarray/关联数组declare -A部分可用),写 shell 脚本须兼容 3.2:用 while read + 数组 append 代替 mapfile,写完 bash -n + 实跑验证。
- 2026-08-22 worktree 收尾 ff-merge 失败的正确动作序列(协议已固化 docs/notes/adopted/2026-08-22-worktree-merge-protocol.md):回 worktree `git rebase main` → force-with-lease 更新备份 → 重试 ff-merge;全程绝不 worktree remove。
78. 本地起 stub 代理后端做浏览器验证时,必须先应答 OPTIONS 预检(204 + Allow-* 头)再转发,否则跨域 fetch 静默全灭,断言全空误判页面没渲染。(2026-08-22 工作台空态验证)
### pgxmock ExpectQuery 参数是正则,SQL 中的括号需成对转义(如 count\(\*\)),否则报 error parsing regexp
- 2026-08-26 worktree 收尾时 `git merge --ff-only X | tail -1` 管道会吞退出码,失败后 && 链继续跑掉 worktree remove;合并命令必须单独执行或显式检查退出码,失败唯一动作是回 worktree rebase 重试。
D 门禁撞号先 merge main 反向同步再复跑:worktree 基点过旧会看到已被让号修复的旧撞号
pgx 参数类型必须与 SQL 推断类型严格匹配:int 喂 text 位($1||str)报 unable to encode,Go 侧显式转换或 SQL 写死常量
迁移 SQL 文件注释里禁止写分号:朴素 split(";") 工具(回环脚本/migrate 某些模式)会被注释内分号毒害
- 2026-08-23 验证线上行为前先核对 registry 镜像 Created 时间与本地 commit 时间;容器 Up 时间/健康检查通过都不代表二进制已更新(CI 异步部署,验证失败先怀疑没部署)。
- 2026-08-23 前端代码已 commit/已 push/容器已 rebuild,用户仍报"看不到变化"时,修复是三步自检:`curl -sI 域名/` 看 `Cache-Control` 是否命中 `max-age=...immutable`、`curl 域名/assets/index-*.js | grep <新代码符号>` 验证新 chunk 是否真到位、`curl 域名/index.html | grep index-...js` 看 index.html 引用的 hash 是否为新 chunk;命中 immutable 时用户需硬刷新才能看到新版本(nginx sites-enabled/boss-5180 当前未加 no-cache for index.html,这是已知改进点)。skill 没提前警告我"修完前端先 curl 远端 bundle 自检"。
- 对接外部 API:先用真实凭据发一次最小调用(哪怕报参数错),端点存在性+签名正确性立刻可知;SignatureDoesNotMatch 之外的一切参数级错误都说明签名已通过。
- 旧供应商文档里的域名可能整个下线(NXDOMAIN),"no such host"先在本机 dig 一次再怪容器 DNS。
- 2026-08-24 ETL 真实执行器接入模板(台账驱动):`ListJobs 过滤 enabled → 执行器注册表 map[jobKey]func(ctx)(rows int64, err error) → RecordRun(RUNNING, startedAt) → 执行 → RecordRun(SUCCESS/FAILED, finishedAt, rows, err)`;cadence 判定 `LastRunAt==nil 立即执行,否则 now-LastRunAt >= cadence`;无真实执行器的台账任务跳过不伪造记录(不发明业务行为)。
- 2026-08-24 SQL 参数运算必须显式类型标注:pgx 参数化查询里 `$3-$2` 当 $3 为 NULL(如 RUNNING 记录 FinishedAt)时 PG 报 `SQLSTATE 42725 operator is not unique: unknown - unknown`;修法是 `$3::timestamptz-$2::timestamptz`。经验:凡参数参与算术/比较,先想 NULL 路径的类型推断。
- 2026-08-24 docs/* 下的 HTML 是文档/原型不是项目页面,不纳入三语/功能验收;真实 worker 页面是 web/admin `boss/worker*` 三页(useT)+ mobile android `app_name`。判定验收范围前先问"这是真实项目代码还是 docs 原型"。
- 2026-08-24 分支清理判定法:残留分支先 `git merge-base --is-ancestor <分支> main`;非祖先但内容已合入(如 backup-main tip 与 main 同标题提交 diff 为空、intel NPE 修复以新 hash 合入)可用 `git diff main <分支> --stat` + 逐提交 `git log main --grep=<主题>` 双确认后 -D,不必保留。
- 拆分同包 Kotlin 文件前先 `grep -rn "fun <拟用名>"` 全包扫一遍：private 改 internal 跨文件后与邻居页面同名函数直接 conflicting overloads，编译才炸（2026-08-24 OrderTimeline 撞 FaultDetailPage.TimelineCard）。
- 调既有 API 前先 `grep -n "^object" <Api文件>` 确认函数归属哪个 object：一个文件多个 object（ProductApi/OrderApi 同文件）时凭文件名 import 必报 unresolved（2026-08-24 changeAddress）。
- Compose test 的 DeviceConfigurationOverride 宽度覆盖：ForcedSize(DpSize) 兼容 1.7~1.11，Width(Dp) 是 1.9+ 才有；且都是 Companion 扩展，须显式 import 函数名（如 import androidx.compose.ui.test.ForcedSize），只 import 类名报 Unresolved（2026-08-24 360dp 基线）。
- 2026-08-24 CI 按变更分类跳过部署时,凡 HEAD 是合并提交必须并看两个父的 diff:feature 侧 merge main 后直接推 main,HEAD^ 是 feature tip,只看第一父会把带入 main 的运行时变更误判 docs-only 静默跳过部署(gitea task 2606 实例);修法 `files=$(git diff --name-only HEAD^ HEAD); git rev-parse -q HEAD^2 && files+="$(git diff --name-only HEAD^2 HEAD)"`。
- 2026-08-24 git amend 前先确认 HEAD 指向哪个提交:并行多提交在途时 amend 默认打进 HEAD,不是"打进我想改的那个";误并后 reset --soft + 按文件重拆可恢复提交原子性。
- 2026-08-24 日期边界类测试的正确隔离是给时钟加 SetFixed 测试缝钉死绝对时刻(生产默认真实墙钟),而不是依赖真实 now + 相对偏移:夹具与 handler 各取一次 now 就存在日界毫秒竞态,宿主时区也会渗入。
- 2026-08-24 双父并集分类的代价是保守:main 刚前进过运行时提交后,即便只合 docs 也会触发一次重复部署(P2 侧 diff 含运行时文件)。这是可接受的取舍——漏部署真实变更比多一次幂等重启危害大;docs-only 直推(非合并形状)仍稳定跳过。
- 新增后台菜单页时,权限码迁移必须同步登记 permissions + role_permissions(sysadmin CROSS JOIN 或单授),否则 102 回放直接 403 no permission(2026-08-28 menu:site 踩坑,先例 000039 geo_menu)。
- INSERT 与 UPDATE 对同一状态字段的副作用必须对称:UPDATE 置 PUBLISHED 落 published_at 而 INSERT 不落,导致创建即发布的文章倒序错乱、日期为空(2026-08-28 cms_posts 102 回放发现)。
- 102 部署是 push gitea main 触发 CI;回放验证要等容器真正换新(端点可达≠新代码),按"制造可观测差异再轮询"确认部署完成。
- 2026-08-28 新路由的契约登记源是 api/openapi/*.yaml(A 检查),cmd/bossctl/routes_admin.go 只是 CLI 目录;两处都要加,漏 yaml 必红灯。
- 2026-08-28 免鉴权公开端点三原则:只读最小投影手动挑字段、非公开态统一 404 不泄露存在性、挂靠既有 public 子路由先例不新开路由组。
- 2026-08-28 前端区块接后端数据且业务上可能为空时,空态/失败态整块 return null 隐藏,不留空白占位区(官网首页 NewsSection 模式)。
- AGP 未声明 testInstrumentationRunner 时用老 InstrumentationTestRunner，JUnit4/Compose 用例 connectedDebugAndroidTest 静默跑 0 个且 BUILD SUCCESSFUL——必须看结果 XML tests 数（2026-08-24 user/worker 端均中招）。
- 模拟器 uiautomator 点 Compose 控件：每次点击前重新 dump 取 bounds（键盘开合/滚动/懒加载都会漂移）；判定页面跳转不要 grep 旧文本可能残留，应用标题类唯一节点。点击级 E2E 不稳时降级为 API 级闭环（登录→端点→DB 断言）（2026-08-24 变更地址 E2E）。
- Go 后端 map[string]any 出 JSON 的 ID 字段：pgx bigint 扫成 int64，toStr 若只认 string 则 ID 恒空；handler 层 toStr/toBool 工具必须覆盖 int64/int（2026-08-24 /addresses addressId 全空）。
- 2026-08-28 部署探针必须先做区分度审查:旧代码同样回 404 的路径不能当"新代码已上线"的判据(cover 端点踩坑);可靠判据只有两类:前端 bundle 里 grep 新 i18n 字符串/符号,或后端制造新代码专属状态(如带封面的 200 图片流)再轮询。
- 2026-08-28 公开图片类资源走"父资源门控"端点(/site/posts/:slug/cover 先验 PUBLISHED 再吐附件),不暴露 /attachments/:id 通用读,防枚举他人附件;非 image/* 一律 404。
- 2026-08-28 匿名页图片不能引用需鉴权的附件端点;/attachments/:id/content 带 token 才 200,公开场景必须由后端提供免鉴权受控流。
- 2026-08-28 SPA 懒加载页面符号不在 entry bundle,验证部署看 entry 里的 i18n 文案字符串(必在主包);markdown 渲染断言用 DOM eval(li 数/strong 文本/img naturalWidth),不靠截图目测。
- 用脚本向 TSX 批量插代码时,锚点必须选模块级唯一行(如 `export default function`),插进 JSX 内部只有 typecheck 能兜底;插完立刻跑 typecheck。
- ff-merge 在多人并行仓库可能连续失败 2-3 次(main 实时前进),协议动作是"merge main→门禁→push→ff-only"循环,不是放弃或删树。
- Gitea Actions job 容器内写文件的持久化只有宿主 docker 资源(命名卷/镜像/registry):job 容器文件系统随任务销毁,"绿了但产物没落盘"比红更隐蔽(run 1526 打印产物路径后 /srv/boss/apk 实不存在);跨容器传文件用 docker create+cp,bind mount 的 $PWD 对宿主 daemon 不可见(2026-08-28 android-apk 两连修)
- 102 出网受限:registry-1.docker.io 直连超时,daocloud 镜像源仅白名单(library/* 可,cimg/mobiledevops 不可);但 dl.google.com/services.gradle.org/maven central 可达,自建镜像走 daocloud 基座+Google 源是正解(android-builder:1 已推 192.168.0.102:5000)
- 共享 GOMODCACHE(~ /go/pkg/mod)文件系统异常时,go 命令全部无输出挂起(open 卡死);隔离 GOPATH+GOCACHE 到 /tmp 即可绕过完成验证,修法先于根因。
- 死会话的卡死 go 进程特征:PPID=1 + stdout unix socket "->(none)"(对端已消失);确认孤儿后 kill 不影响在途会话。
- cdp-capture 免登录注入:守卫页先载会重定向到 /login,reload 无效;注入 localStorage 后必须 location.href='/目标路径'。boss.servers 元素需含 id/name/baseUrl 三字段。
- 接手无主 worktree 双证法:ps 无归属进程 + 文件 mtime 超 6 小时,即可安全当归属者完成收尾。
- (修正上条)~/go/pkg/mod 挂起根因已确诊:~/go 是符号链接→外置卷 /Volumes/sker(USB APFS),该卷 I/O 停摆时所有依赖它的 go/pnpm 命令集体卡死在 open();停摆常自愈(复查时 ls 7ms、go list 0.2s)。若复发,根治方案是把 ~/go 指回内置盘;应急仍是隔离 GOPATH/GOCACHE=/tmp。外置盘上还住着 .vite-plus node 运行时,同停摆会连带 pnpm。
- (落地)2026-08-24 已把 ~/go 从外置卷 /Volumes/sker 迁回内置盘 ~/.local/go(原符号链接原子替换,缓存 1.7G rsync 保留,旧副本留在 sker 作备份可删)。此后外置盘停摆不再影响 go 门禁。
- worktree 未提交编辑 + worktree 被外部清理 = 改动直接丢失(2026-08-28 round5:boss-wa5 在签名实测中途被外部 prune,gitignore 改动未 commit 即失);防御:worktree 内编辑后立即 commit(每个文件/小簇),即使测试未跑完也先 stash 不留无 commit 文件
- ~/.gradle 是指向 /Volumes/sker/.gradle 的符号链接,该目录会被清空导致 wrapper 冷启动卡住;解决:GRADLE_USER_HOME=本地路径(/Users/imeepos/.gradle-local),首次下载 gradle 发行版+依赖即可(2026-08-28 round5)
- go/gradle 命令"无输出挂死"且 CPU 0%:先查共享卷 I/O(fs_usage/sample)与孤儿进程(PPID=1),再怀疑代码或依赖(2026-08-24 user-android round4)。
- 一次 E2E 观察到的"点错页"先核对 onClick 接线与调用链,再定性 UI bug;uiautomator 坐标漂移可制造假象,未复现不擅自改代码(2026-08-24 首页重叠排查)。
- worktree 环境每项改动即刻 commit:并行会话可能随时 ff-merge+清理 worktree,未提交改动随树消失不可逆(2026-08-24 round4 实证)。
- 2026-08-28 menu 权限 baseline 消化应先补 permissions+role_permissions(sysadmin) 再谈后端 requirePerm 收敛:一次迁移改变多域鉴权会放大风险;专属码先服务自定义角色菜单可见性,端点级收敛分批做。
- 2026-08-28 前端本地 pnpm store 被并行 worktree 抢锁时,不要无限重试 install;保留已完成 Go/契约门禁,先 commit 原子变更让 CI 构建成为第二道 web 门禁,但最终 102 回放必须补齐 typecheck/build 证据。
- 2026-08-28 SEO SPA 元数据应在异步文章数据到达后设置并在卸载还原(title/meta);og:image 必须使用匿名可访问的受控封面流,不能引用带 JWT 的 /attachments/:id/content。
- 2026-08-28 nginx 缓存修复的正确配对是 index.html/no-cache + /assets hash 文件 immutable;只改一侧会分别导致旧入口或资产缓存问题。
- check-contract-sync E 项要求 menu.def key 与 menu: 权限码同名(camelCase 一一对应,无下划线);初版 menu:crash_logs 与菜单 key crashlogs 不匹配即红,且涉及迁移/handler/契约/i18n 错误文本多文件同步更名,宜首次命名即对齐
- admin handler 单元测试不拉全 Register,自建最小路由仅装目标 handler + admin Authn 中间件更省力(避免依赖全部 user.Service fake)
- ssh HOST arg1 arg2 ... 把 argv 按空格拼接发给远端 shell,含管道/重定向的命令必须整体作为一个字符串参数('ssh HOST "cmd | head"' 而非 'ssh HOST cmd head'),否则远端 shell 按字面管道解析;此坑在取件脚本 docker 链式调用反复出现
- 2026-08-28: go build 连 -x 都零输出且挂死 → 先查 GOPATH/GOMODCACHE 是否在外置卷(~/go 软链 /Volumes/sker 停摆),本地 GOPATH 绕行再定位。
- 2026-08-28: pgx 把 nil 切片编码为 SQL NULL,显式 INSERT 列不吃表 DEFAULT → 撞 NOT NULL;落库前 nil 兜底空切片。
- 2026-08-28: multipart/DB 落库链路的 bug 域单测挡不住(缺省值/驱动编码/语义),必须 102 实测冒烟;gin 同段 :id 与 static 冲突注册期 panic,公开面用独立段(/site/downloads)。
- 2026-08-28: UI 验证用 cdp-capture --eval 打 innerText 断言比截图可靠(无图像输入能力时);admin 页登录注入 localStorage boss.token 后 location.href 跳转。
- (2026-08-24 全面迁移)sker 盘(disk7,1TB USB,历史多次 I/O 卡死)上的全部构建工具链已迁离:.vite-plus 8.4G(node/pnpm/dsh/claude)→ ext512/dev-toolchain;.gradle 25G、.android 4.3G、.venvs、.hermes/.kimi-code/.cloakbrowser → ext512/dev-cache;pnpm store-dir(~/.config/pnpm/rc)与 npm cache(~/.npmrc)改指 ext512/dev-cache。全部用符号链接原位替换,零 shell 配置改动(Android SDK 本就在 ~/Library 内置盘)。内置盘仅 9Gi 放不下,ext512(disk4,独立外置盘,450G 空闲,boss 仓库所在)是唯一可行落点。sker 上仍留纯数据(workspace/archives/gitea/verdaccio 等 26 个链接)与旧副本备份,确认稳定后可清理。
- 迁移时遇并行会话 Gradle/Kotlin daemon classpath 已解析到旧盘绝对路径:不杀 daemon(误伤在途构建),换链接后旧 daemon 读旧路径继续跑,自然消亡后新构建走新盘;rsync 报 exit 23(源文件中途消失)补一轮增量即可。
- (2026-08-24 重复缓存清理)多代缓存共存根因:同一工具的缓存位置被历史配置改过三代(默认内置盘→sker USB盘→ext512),每代迁移都没删旧副本。清理 sker 旧副本约 60G(sker 用量 347G→287G)+ 内置 ~/Library/pnpm/store 2G + /tmp 隔离缓存。两个坑:①go 模块缓存文件只读,rm 报 Permission denied,先 chmod -R u+w 再删(等价 go clean -modcache);②Kotlin daemon classpath 钉在旧盘 jar 上,须等 daemon 闲置自退(sker/.gradle 留待后续删)再清。t3-release 13G 是构建产物非缓存,留待业务确认。
- rsync -a 迁移含符号链接的工具链时会原样保留绝对路径链接:指向旧位置的链接迁移后全变死链(vite-plus/bin/dsh 指向已删的 sker 路径,致 dsh 在用户 shell MISSING)。迁移后必须 find -type l 逐个 readlink 检查旧盘前缀,改相对路径或新绝对路径;仅凭"目录复制 rc=0"不够。
- 死 PATH 条目无害但可顺手报:/etc/paths.d/dotnet-cli-tools 的 ~/.dotnet/tools 是 dotnet 安装器不展开 ~ 的已知 bug,/pkg/env/global 来自 pmk;清理性质,非故障。
- Android 资源字符串含撇号必须转义为 \\'(反斜杠+撇号);双引号包裹法 \"...\" 在 aapt2 string-array 场景下触发 NPE(StartElement.getAttributeByName 返回 null),不可用(2026-08-28 round6)
- aapt2 报错常带混淆行号("Failed to flatten XML at line N")实际指向被合并的 merged.dir/values-XX/values-XX.xml 资源条目,需逐键验证;尤其 values-en 中历史 string-array item 直含撇号未转义会被本次构建暴露(本次踩坑)
- Kotlin Composable 中回调 lambda(permissionLauncher / pickers)若引用 ctx,ctx 声明必须在其前;Compose 顺序敏感且编译器不报"use before declaration"而是"Unresolved reference"(2026-08-28 round6)
- 资源字符串拼接模板 "${x} 张" 不能直接复用为 <string>:Android 资源需 <string>%1\$d 张</string> 占位符,Kotlin 侧 stringResource(R.string.key, x) 传入;OptionRow/Dropdown 的 listOf(...) 在 Composable scope 内 stringResource 即可(2026-08-28 round6)
- Compose stringResource() 只能在 @Composable scope 内直接调用,不可放在 onClick lambda / scope.launch / try-catch 块内;编译器报 \"Try catch is not supported around composable function invocations\" 或 \"@Composable invocations can only happen from the context of a @Composable function\"。解决:hoist 到所属 Composable 函数顶部声明 val(2026-08-28 round7 batch6)
- Android 模块级常量(listOf(...))无法用 stringResource,因为 stringResource 只能在 @Composable scope 调用;两种解决:(a) 把常量从模块顶层移到 Composable 函数体内用 stringResource 取,(b) 重构为 Composable 函数返回列表(2026-08-28 round7 batch7)
- 加 stringResource 调用务必同步加 import androidx.compose.ui.res.stringResource,编译器报 Unresolved reference 而非提示缺失导入;ProfileCards/ProfileScreen/SafetyScreen 三处踩坑(2026-08-28 round7 batch7)
- 2026-09-01: Tauri 内嵌资产验证方法——assets 被 brotli 压缩,HTML 全文不会出现在 strings 输出;应该 grep 入口 hash 文件名(如 `strings binary | grep -c "index-xxx"`)验证当前 dist 已被嵌入,keys 会以明文出现。
- 2026-09-01: CARGO_TARGET_DIR 跨 worktree 共享会导致 tauri-build 的 build script 输出缓存命中,新 dist 文件列表不更新;二进制嵌入陈旧资产。必须用 worktree 自己的 target 目录。
- 2026-09-01: pnpm 工作空间根 pnpm-workspace.yaml 无 packages 字段时,`pnpm install` 在子项目报"No projects found";需 `--ignore-workspace` 标记。`pnpm build` 脚本不受影响。
- 2026-09-01: worktree 间 symlink node_modules 触发 pnpm 模块状态检测不一致,报`ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`;应在 worktree 内`pnpm install --ignore-workspace` 真实安装。
- 2026-09-01: 后台 job 的 bash 调用不继承父 shell 的 PATH;export PATH 必须写在命令字符串内。
- 2026-09-01: gitignored 的 target 目录在多 worktree 共享盘上可能被并行会话意外覆盖/回退;不要信任另一个 worktree 的 target 目录内容。
- 2026-09-01: 磁盘将满时 `go test ./...` 的 build failed 是假信号——先 `df -h`,余量不足先 `go clean -cache`(本项目 go-build 缓存可达 13G)再重跑,勿当代码错排查。
- 2026-09-01: 300 行红线检查(check-contract-sync C 项)按 `strings.Count("\n")+1` 计数(比 wc -l 多 1);改大文件前先查余量,主分支可能已被并行分支推高到红,顺手拆文件守红线(findByRequestID→pg_query.go、回退/取消→pg_reopen.go 先例)。
- 2026-09-01: 并行 worktree 会话会让 local main 领先 gitea/main(未推 commit);ff-merge 前 `git merge-base --is-ancestor HEAD <分支>` 确认,rebase 应以 local main HEAD 为基而非 gitea/main。
- 2026-09-01: make check 与 git rebase 不可并发——rebase 改写工作区文件,运行中的测试读到半成品,结果作废必须重跑。
- 2026-09-01: 本机无 psql/DB 客户端时,用 `cat x.sql | ssh imeepos@192.168.0.102 'docker exec -i boss-infra-postgres-1 psql -U boss -d boss -X'` 管道执行(host 无 psql,DB 在容器 boss-infra-postgres-1,映射 25432);多层 shell 引号必炸,SQL 写文件管道最稳;执行前 `\set ON_ERROR_STOP on` + BEGIN/COMMIT 包单事务。
- 2026-09-01: 删除父表前先做 FK 依赖扫描(`grep REFERENCES orders migrations/*.up.sql`)+ pg_dump 备份受影响表(`pg_dump -t t1 -t t2 ...`),子表先行删除防 FK 阻断;破坏性清理走 快照→预检→备份→事务清理→复扫=0→API 冒烟 六步。
- 给被 pgxmock 锁 SQL 的函数加前置查询时,先 `grep -rn "ExpectQuery.*<原SQL片段>"` 列全受影响 mock 再动手,避免逐个跑测试发现。
- "表在迁移里有但库里不存在"先查是否有后续迁移 DROP/吸收(如 000059 三表归一),再下"缺失"结论。
- cdp-capture 每次运行是全新浏览器 profile:localStorage 不跨次保留,token/servers 注入必须在同一次运行内完成(首个 eval setItem + location.href 重新导航,后续 eval 用 Promise+setTimeout 等待)。
- cdp-capture 多个 --eval 在同一页面上下文顺序执行:顶层 const/let 会跨 eval 撞名,一律用 IIFE 包裹。
- vite dev 默认只绑 localhost(::1),127.0.0.1 连不上:CURL 探活和 CDP 访问都要用 http://localhost:PORT。
- cdp-capture 操作自定义 Dropdown:触发器按钮文本是"当前选中项"(如'账号'),不是目标项;查找触发器用 aria-haspopup=listbox + 当前值文本,点开后 option 才按目标文本找。
- 死循环预防:同一 bash 命令连续执行 ≥2 次且输出逐字相同,说明是"重放"而非"推进"——先 `git diff`/`wc -l`/`grep -c` 核查真实状态,再决定是否重跑;绝不盲目重发同一命令。
- 改文件的 python 脚本必须幂等:插入前先查"该标记是否已存在"(check-then-insert),否则 `assert count==1` 只防"已存在",防不住"每轮 search() 又命中、又插一份"(用 re.search + 插入后同一正则继续命中 → 每跑一次多一份)。
- 一个 bash 调用只放一段 heredoc:不要把 python 拆成 `<<'EOF'...EOF` 后接裸 bash 再 `<<'EOF'`(第二段 `s=open(f).read()` 会被 bash 当命令报 syntax error);要改多文件用单一 python heredoc 内循环。
- python 内嵌多行 Go/TS 代码用三引号字符串 `'''...'''`,别用单引号+真实换行(直接 SyntaxError EOL)。
- 替换脚本锚点要足够特异:短文本(如 `product: '产品资费',`)会在多个页面/段落重复出现,`assert count==1` 炸;先 `grep -n` 定位,锚点带上下文(整行 entityNames: {...})。
- 给数组(如 IMPORT_ENTITIES)加元素后,toEqual 断言会因"插入位置与测试预期顺序不一致"失败:新增元素要么按现有顺序插入,要么同步改测试期望顺序(看 diff 的 -/+ 定谁错)。
- 加域方法会把文件顶过 300 行红线(contract-sync C 项):写前 `wc -l`,超了压缩空行或换文件。
- 大段手改后跑 `gofmt -l <file>`,非空就 `gofmt -w`,再 build(空白被脚本合并后 gofmt 会补齐)。
- 隧道/代理容器名不可信:先验后端身份(/healthz + 未配置端点的降级特征),再注册回调或断言连通。
- Stripe webhook endpoint 凭 sk 用 REST 建(POST /v1/webhook_endpoints),whsec 创建时一次返回、随 endpoint 长期有效。
- 账号启用重定向支付方式时,API confirm PaymentIntent 必须带 return_url,否则 400。
- portal 注册给合成负 id 不在 customers 表;支付/账单 E2E 先建真实 customers 行再 UPDATE portal_accounts 改指 + 密码重登。
- cloudflared 快速隧道(--url)URL 重启即变:webhook endpoint 需重建,重建步骤写进 adopted note。
- E2E 造数清理按序:payments(含 bill_id 兜底)->bills->customers->addresses->portal_accounts->portal_sms_codes;孤儿巡检门禁兜底。
- pgx v5 向 jsonb 列写参时 `[]byte` 默认按 bytea 编码,即使 SQL cast `$4::jsonb` 也报 SQLSTATE 22P02;必须传 `string`(text 编码,`text::jsonb` 合法)——与 import_tasks $7::jsonb+string 成功先例一致,pgxmock 模拟不出该行为,只在真库暴露。
- 2026-09-04: 102 cron 脚本提交前先在目标机真实路径跑正常/DOWN 两路,本地 shell 通过不代表远端 ROOT/JSON 拼接安全;外发 JSON 字段统一清洗 CR/LF。
- 2026-09-04: check-contract-sync 的 OpenAPI 路径扫描应按 portal 隔离并同时读取根 manifest 与对应子目录,不能把同行 `$ref` 当作唯一 path 形态。
- 2026-09-04: cdp-capture 需要跨命令保留登录态时使用 `--user-data-dir`;默认临时 profile 仍保留以防状态泄漏。
- `git worktree add ../name` 建的是仓库外兄弟目录;往仓库内 `./name` 写文件再 commit,git 向上解析到主仓库,commit 静默落 main。写前 `git worktree list` 核绝对路径,commit 输出方括号看分支名(2026-08-26)。
- 迁移 .up.sql 上真库演练必须先 sed 删掉文件内 COMMIT 再追加 ROLLBACK,否则"演练"变直接提交(2026-08-26,NCR 物化靠幂等 SQL 兜底无分叉)。
- ModalBottomSheet 内收起软键盘用 input keyevent 4(BACK);keyevent 111(ESC)会把整个 sheet 关掉,真机自动化断流(2026-08-26)。
- 真机保存报 50000 先 docker logs boss-server 查 SQLSTATE:FK 违约多半是陈旧 token 指向已删 customer,pm clear 重登即愈,不是新代码 bug(2026-08-26)。
- 推 main 后部署是否发生必须以 boss-server 运行镜像 sha / schema_migrations 最新版为准去核对,不能假设 CI 已完成;CI runner 被其他项目任务占满时 deploy 任务会只创建不执行(2026-08-27 激活回调闭环,3116 任务未执行)。
- pgx 列可空性必须真库验证一次:mock 桩永远返回你写的非 NULL 行,`*int` 扫 NULL 列在 mock 下全绿、首个 fresh 行(如 open_webhook_deliveries.http_status)即炸,整条投递循环每轮报错。凡"从零第一次读"的行优先用 pgtype.Int4/Text 或 COALESCE(2026-08-30 webhook 集成测试当场抓获,单测全绿未能预警)。
- SQL 语义改动(SKIP LOCKED 领取/租约窗口/事务边界)先上真库 EXPLAIN 验证语法与计划再写代码;pgxmock 只能对字符串做正则匹配,验不出 `FOR UPDATE OF 别名` 级别的合法性。
- DSH 工具的后台执行靠工具参数(如 run_in_background: true),把它当环境变量写进命令体不会生效,命令会被前台超时杀掉。
- 当 make lint 红在 `gofmt -l` 且文件不在本次改动集:是 golangci-lint 缺席时的存量兜底拦截,gofmt -w 单独 style 提交修掉,不要混进 feature 提交(2026-08-26)。
- 双主题断言在同一个同步 eval 内改 data-theme 并立即读 computed style,读到的是 CSS transition(如 duration-200)的中间值——必然产生"没生效"的假阴性;改属性与读结果必须隔开一次 await/settle(2026-09 用户详情重构,浪费最多时间的一坑)。
- 页面首访被守卫(Navigate to /login)弹回后,eval 里的 location.reload() 重载的是 login 页而非目标页;带着 token 回内页要显式 location.href='/目标路由',reload 不会还原被替换掉的 URL。
- 判断 Tailwind 任意值 utility 是否真的进了产物:`grep -o ".\{0,60\}令牌名.\{0,80\}" dist/assets/*.css` 直查构建产物最权威;vite dev 的 CSSOM 遍历有 @layer 嵌套盲区,扁平 for cssRules 会漏,需递归 walk。
- 幽灵令牌不止害一个文件:发现一个未定义 var() 时应立即跑全量审计(techniques.md「幽灵 CSS 令牌全量审计」),同类错误往往成批潜伏(一次揪出三个)。修复只用已定义等价物:主按钮 --shell-fab-bg 系列(亮藏青暗金自适应)、待审/警示色用 --color-brand-gold-600、文字链用 --color-text-link,不要现造新令牌除非在 tokens.css 双主题块里成对补齐。
- 外层 SELECT 无 FROM 时引用任何列名必报 42703(PG 无法解析裸列);2026-08-26 实名 PASS 门禁 guardRealNameIdentity 因此在线上全量 500 而 pgxmock 正则匹配全绿——正则只对字符串,不校验列归属。
- "字段在表里存在"推不出"SQL 能跑":子查询外的裸列属于外层作用域,靠真库一次真实调用兜底,别信 mock。
- 当 102 API 返回 `LICENSE_REQUIRED` 而健康检查仍为 ok 时,修复是把它记录为部署环境不可验收事实:healthz 只能证明进程存活,不能证明业务路由或授权链路可用;不得把该响应误判为代码回归失败。
- 老持久 profile 做 i18n/主题切换断言会拿到"localStorage 已变而 DOM 未变"的竞态假样本;修复是 fresh user-data-dir + 「目标节点渲染就绪」条件等待(.st-tag 出现再取文本),不要固定 sleep 后直接读。
- ?token= 写入的 boss.token 会被 AuthGuard 的失败预取静默清掉:先配 boss.servers 再带 token 导航,顺序反了就"未配置服务端"弹回登录页;localStorage 里 token 莫名消失 ≠ urlPrefs 失效,先查 adminLogout 调用链。
- 页面组件里裸 `<input className="w-full">` 在 color-scheme:dark 下渲染 UA 默认样式,与 tokens 体系观感割裂;表单输入一律用 ui/Input(shell-input-* 全令牌),下拉用 Dropdown,label 一律 i18n——audit 关键词:grep '<input className="w-full"'。
- httpx.RequireString/RequireNonNegativeFloat 返回具体指针 *ValidationError,在返回 error 的函数里直接 return 构成 typed-nil(接口非 nil、打印 <nil>);修复是 httpx.CollectErrors(...) 包一层再 return——2026-08-27 openplat 订阅校验三用例同坑。
- 返回"副本"的枚举函数(EventCatalog 等)要配"返回长度=源长度"的测试,防止后人改成直接返回内部 slice 被调用方污染。
- 格式化函数已兜底 undefined,调用方 `String(r.createdAt)` 强转会击穿兜底把 undefined 渲染成字面量(fmtTime 收 undefined 返回 '—',String 后成真字符串绕过判空);修复是列渲染抽纯函数传原值,测试锁定"缺字段渲染占位符"——2026-09 用户列表注册时间 undefined 事故。
- 部署后验证接口新字段,轮询条件必须是"目标特征字段出现"而不是"服务有响应":服务常驻时后者恒真,第一轮就 break 拿到旧版本假阴性(2026-09 用户列表 createdAt 验证空转一轮);正确姿势 grep 响应体里的新键名,40 次×20s 内等到即判成功。
- 当注释声称的行为与 SQL 实现不符时(如"发全部启用订阅"却 `WHERE event_type=$2` 精确匹配),修复是把它当真 bug 顺着写回归:目录外事件类型(openplat.test)经目录制+精确匹配双重锁死永远命中 0 条,自检功能整体空转;2026-08-27 开放平台测试事件即此,按应用匹配新增 InsertAppDeliveries 修复。
- 当"下拉/清单只有一项"被报为反常时,修复是先查数据源登记表(eventCatalog)与 emit 侧调用点数量再谈 bug:登记制目录如实反映"emit 侧只挂了一个事件"不是缺陷,渲染链路(后端→API→前端)用 102 curl 逐层复核排除;2026-08-27 订阅事件选择器调研结论。
- 当门禁(check-contract-sync 等)在 feature 分支红时,修复是先回主树复跑同门禁:同红=并行会话存量(不碰、不修、总结里注明),仅自己改动引入的红才属于本次修复范围;若存量红恰好落在本任务调研域内,做成独立小提交(零行为变更)单独 revert。
- 当侧栏兄弟菜单项互为路径前缀(官网内容 /boss/site 与官网分类 /boss/site/cats)时,NavLink 默认前缀匹配会让访问子项时父项也高亮(双击亮);修复是激活判定精确化:精确路径激活,深层路由仅当其自身不是其它菜单项完整路径时算"同页"(如 /boss/site/new 高亮官网内容,/boss/site/cats 不高亮)——2026-10-01 抽成 menu.def.ts 的 isNavActive 纯函数,配 vitest;react-router 6.30.4 已移除 NavLink isActive prop,改前先查 d.ts。
- 列表页 `columns` 数组若存字段标识符(code/name/sort)而非译文,三语 locale 下表头都裸英文(zh-CN 界面也显示 code);列头数组要直接放本地化标签(如 ['标识码','名称','排序','启用']),与 releasePage/customer 等既有页面口径一致——2026-10-01 siteCatsPage 修正,sitePage/knowledgePage 仍留同源缺口。

- 项目 vitest environment=node 且无 @testing-library/react 时,组件测试走 renderToStaticMarkup(SSR 骨架)+ 抽纯函数单测,不写 useState 异步交互测试。开工前 grep vite.config.ts test.environment + pnpm-lock.yaml testing-library/react + 项目内 fireEvent 引用计数三件套。
- 写反向回填 SQL 时,严禁 `WHERE x IS NULL` 哑条件(业务流先填 x 时 UPDATE 静默跳过 0 行,无法区分"已绑别人"与"无须回填");改用 `WHERE x IS NULL OR x = $expected`,UPDATE 0 行**必然**是冲突,可靠触发 ErrXxxConflict,不留下孤儿。配套必须有可观测 slog 日志(冲突对象 id + 原因字段),便于排查时 grep。skill 没提前警告我。
- 收到"数据没关联上/对不上"类反馈,第一动作是 SQL 查表给出数量级证据(双向一致 / A 端孤儿 / B 端孤儿),再判断是历史数据还是接口问题;直接看接口或写迁移容易越界。skill 没明确沉淀"双向孤儿诊断三件套"(一致性对 / A 单向 / B 单向)。
- 收到"真实验证失败"反馈时,先 SSH 102 看服务端日志找真实 SQLSTATE(23505/22001/42P07 等),针对性拆解 pgconn.PgError 按 ConstraintName 分类映射为业务错误;不要笼统返 50000,那样前端只能看到"内部错误"。
- DB 唯一约束触发后,pgxmock 的 `pgxmock.NewResult("UPDATE", 0)` 模拟的是"冲突"场景,但 PG 真实行为是 `UPDATE 值相等仍返 1 行`;写幂等测试时必须用 `WillReturnResult(pgxmock.NewResult("UPDATE", 1))` 模拟幂等场景。
- Docker 容器内替换镜像层文件必须用 bind mount 注入或 docker commit 保留可写层,docker cp 改的二进制/文件**容器重启即丢失**(因为镜像层只读、cp 只写可写层、commit 不显式保留就没了)。
- 真实验证脚本应走业务接口(免 license gate/admin token),不走 license status(被 license gate 拦截)。102 dev 环境镜像里 LicensePublicKeyHex 已注入,门禁启用;本地 build 没注入公钥,二进制行为不同。skill 没沉淀"license-gated vs dev build"的环境差异。
- 迁移让号:同时改 schema_migrations.version + 改 up.sql 用 `CREATE INDEX IF NOT EXISTS`(兼容已落库索引);否则 server 启动时撞 42P07 触发重启循环。skill 没沉淀"迁移让号 + 已落库兼容"完整三步。
- macOS bash 3.2 会把全角标点(如「」)字节并入 `$VAR` 变量名——`"$SCOPE」"` 报 unbound variable;中文旁引用变量一律写 `${VAR}`。
- BSD/macOS `head` 不支持负行数(`head -n -1` 直接报错);「取除末行外全部」用 `sed -e '$d'`。后端 envelope HTTP 恒 200、业务码在 body 的 `.code`,E2E/脚本断言必须看业务码,断言 HTTP 状态码会静默漏判。
- push 后 CD 部署竞态:上一 run 的镜像可能盖住你的 push(并发取消只对未开始的 run 生效)。部署验证第一步先 `docker images --format '{{.Tag}}'` 对齐镜像 tag 与预期 commit sha,再开测;没触发就推空提交重触发(classify 按「已部署 sha」比对,runtime diff 会补部署)。
- pgxmock v4 的 `NewPool()` 返回 `PgxPoolIface` 接口而不是 `*Pool`,helper 函数签名要写接口类型。
- 当 Portal 弹层(Portal 挂 body)与内联 fixed 遮罩同现时,修复是先确认两者处于同一层叠上下文,再比 z 数值;Portal 不改变层叠上下文归属,别被"弹层在 DOM 更深处"迷惑。
- 当用 checkout/stash 之外的方式临时篡改工作区验证测试时,修复是用 stash push/pop 或 sed 双向改回;`git checkout -- file` 会抹掉该文件全部未提交修改,包括真正的修复。
- 当 httpx.BindAndValidate 的 extraChecks 里 return RequireXxx(...) 直接返回时,修复是包一层 CollectErrors——Require* 返回 *ValidationError,nil 指针被装箱成非 nil error 接口,校验通过路径也会 panic(typed-nil)。
- 当 pgx/pgxmock Scan 目标是 **time.Time 时,修复是用 pgtype.Timestamptz 扫描再取 .Time(仓库 ListAssignments 既有模式),pgx 不支持 double pointer 目标。
- 当长任务跨多个提交窗口时,修复是每次 merge 回 main 前 git fetch + 重新跑 check-contract-sync D 项——并行会话会随时占走迁移号,开工时查过的号中途会失效。
- 当师傅端"任务列表"类接口按 worker_id 查询时,修复是显式带 status 过滤(DOING)——单测 mock 不会暴露漏过滤,只有真实环境回归会抓到。
- 当写 dated artifact(决策 note 文件名/注释/commit message 日期)时,修复是先 `date +%F` 取系统时钟为准——本仓库存在会话间日期漂移(main 上已有未来日期的 note),凭印象写日期必返工。
- 当 commit 后链式 `git status --short` 看到未预期文件时,修复是 `git show --stat HEAD` 确认提交内容——status 输出的是未提交改动,不是提交内容。
- 当在全新 worktree 里首次构建 Android 时,修复是先复制 gitignored 的机器本地文件(local.properties 的 sdk.dir 等)——worktree 是干净的检出,缺它 gradle 直接找不到 SDK。
- 当生成密钥/证书类资产时,修复是放在主树 gitignored 位置而非 worktree——git worktree remove 会物理删除目录,worktree 内的 keystore 随之丢失,须重生成+重新留档指纹。
- 当 Android instrumented 测试用 PackageManager 拿权限声明时,修复是 `requestedPermissions?.contains(...) == true` 安全调用——当前 API 返回可空 Array,直接 contains 编译不过。
- 当本地有 AVD 且要跑 connectedDebugAndroidTest 时,修复是 `-no-window -gpu swiftshader_indirect` 无头启动,轮询 `sys.boot_completed=1` 后再连测,收尾 `adb emu kill`。
- 当在 withContext/coroutine lambda 里写 while(true) 重试循环时,修复是把循环抽到显式返回类型的 private suspend helper——循环语句类型是 Unit,label return 不计入 lambda 返回类型推断,直接内联必报 "Argument type mismatch: actual type is Unit"。
- 当 commit 消息含全角括号「」、箭头 →、冒号等字符时,修复是一律用 `git commit -F 消息文件`(git commit -q + 多行 -m 会被 bash 拆裂,报 pathspec 错)。
- 当修复"映射/枚举转换"类 bug 时,修复是重写测试为逐项 spec 断言(如逐 stage 12 个输入断期望里程碑)——不要只换两三个采样值,旧的采样测试会把错误公式锁成"正确行为"。
- 当探测后端端点可用性时,修复是先读 openapi 契约确认 HTTP method——同一 path 的 GET/POST 路由可分别存在,P**OST-only 端点用 GET 探测恒 404 会误报"契约-部署漂移"(2026-08-27 /push/device 实例,ISSUE.md 误报到更正)。
- 当客户端调后端"注册类"端点时,修复是读服务端入参校验(形态/长度/字符集)——本仓 push validRegistrationID 仅收 [0-9a-zA-Z],UUID 带横线必 42200,须先规范化。
- 当脚本判断 API 调用成败时,修复是以响应信封 code==0(或 ok:true)为准,不能只看 http_code——本仓错误信封 42200/40100 也返回 HTTP 200(2026-08-28 slo-cruise emit 静默失败实例)。HTTP 200 + grep '"code":0' 双条件才对。
- 当要调用存在枚举/白名单的端点时,修复是先读契约与实现确认枚举值(如 ops/notify-emit 的 refType 白名单 stripe_tunnel/slo_cruise,见 ops_notify.go),别按语义猜一个新值——白名单外必 42200。
- 当复用现成"对账/巡检"口径做根因分析时,修复是先 grep 该指标在代码里的权威 SQL 定义(如 pg_recon_counts.go),再照抄直查——自拼表名/列名(port_reserves/reserve_expires_at)会连撞两回(2026-08-28 孤儿端口实例)。
- 当把 merge/cleanup 串在同一条 bash 链里时,修复是先跑 merge 看结果再单独清理——ff-merge 失败(并行会话推进 main)后链式 worktree remove 仍会执行,靠 branch -d 拒绝才保住 commit(2026-08-28 实例,recidivism #9 变体)。
- 当给既有 pgxmock 提交流程加新查询时,修复是同步补 mock 期望并提供 helper(如 expectDirectRiskClean)——漏补会让 4 个存量子测试一起红,报"could not match actual sql"。
- 当全量生成工具(gen-*.mjs)的输出 diff 混入他人未同步的路由时,修复是 git checkout 还原生成文件再手工增量——不代偿别人的欠债,也不被工具"顺带同步"绑架。
- 当 CLI/工具的退出码语义要变更时,修复是先 grep 仓库内全部消费方(scripts/、CI、`set -e` 脚本里 `result=$(cmd)` 的 command substitution 会被新非零退出码杀掉),再改默认值——退出码是接口,不是实现细节(2026-09-06 bossctl 退出码改 1 杀死 api_test.sh)。
- 当本地身份档案/凭证文件 401 时,修复是先与唯一事实源(test-accounts.json)逐 key 比对有效期,过期档案直接刷新,而不是反复重试调用(2026-09-06 admin 档案旧 key 已吊销)。
- 当给后端接口造测试载荷连报 42200 时,修复是先读 handler 的 httpx.BindAndValidate/Require* 校验清单(字段名、类型形态如字符串带宽"100M" vs 数字)再重试,不要猜(2026-09-06 products/price-history 连错三次)。
- 当 M3 1.4.0 ModalBottomSheet 内要自定义系统返回语义时,修复是把返回路由进 onDismissRequest(「到达时面板 Hidden 且栈非空=回上一级,否则真关闭」的确定性语义)——内容层 BackHandler 收不到弹窗期返回事件(注册在 Activity dispatcher),触摸时间戳也区分不了返回/上滑(两者都先 hide 再回调,2026-08-28 地址选择器 A3 三轮实证)。
- 当 uiautomator dump 断言 Compose 控件可点性时,修复是找文本的包裹节点而不是文本子节点——Compose 文本子节点会单独暴露成 clickable=false 幽灵行,QA 按它断言会误报"按钮不可点"(2026-08-28 面包屑 chip 误判实例)。
- 当真机 input text 打不进字母时,先怀疑 MIUI 搜狗输入法吞事件,修复是改走选区回填路径造数,不与 IME 纠缠(2026-08-28 MI 9 SE 实测)。
- 当 license/证书类"激活成功"后,修复是必须验证持久层文件真实落盘(docker exec ls 卷挂载路径)——激活写的是容器层时,任何重部署都会静默蒸发证书,activated:false 且无 reason 即文件缺失(2026-08-28 102 license 卷挂载缺失事故,两次复发)。
- 当交付含复杂 SQL(递归 CTE/多 CTE)的脚本时,修复是提交前必须真库实跑+用合成 VALUES 替换数据源 CTE 验证全分支——没被数据 exercise 过的分支(string_agg 内嵌祖先链子查询、apply plan 过滤)都是未验证代码;2026-08-28 地址治理脚本连踩三错(缺 WITH RECURSIVE、递归分支漏带 addr_id、CTE 漏选原始列)全被 ON_ERROR_STOP 即时暴露,合成路径 8 行再验证 matched/中断/归一/plan 过滤后才提交。
- 当会议型 subagent 成员首轮/中途 failed 且无 closing message 时,修复是重催消息明确三点:纯文字作答、禁调任何工具、限字数(如600字)——老周连续两次失败后按此一次成功;成员自行大量读文件/调接口是会话失败头号诱因,与"审查型成员给阅读预算"同根(2026-08-28 权限评审会 4/5 首轮失败实证)。
- 当要断言"远端是否已有某修复"时,修复是比对 gitea/main 的文件 blob 或 git show 远端内容,而不是 grep refs/remotes/gitea 找分支名——合并后功能分支按协议删除,查分支必误报"未推送"(2026-08-28 obs2 收口向主持人误报 push 缺失,被回执纠正)。
- 当代码改完还要做长周期真机验证时,修复是先 git commit 存档再上设备——验证中途收尾进程会按协议把"未提交的 worktree"提交+合并+清理,目录随时可能消失;本例侥幸被收尾 commit 原样带走,重来就是重写(2026-08-28 obs2 实例,是"验证通过立即 commit"红线的前移变体:改完即 commit,别等验证)。
- 当 SQL 列是 NOT NULL DEFAULT '' 时,插入侧不要包 NULLIF(x,''):NULLIF 把空串转 NULL 直接 23502;NULLIF 只用于真正可空列(如 bill_id NULLIF(0))。
- 当后端在 handler 里调 domain 且 domain 内部修改入参副本字段(如兜底生成 payNo)时,handler 拿不到新值:让 domain 返回回执结构体带该字段,或 handler 层先生成。
- 当 102 部署验收(真环境)时,先 bossctl 打一发只读端点确认服务健康再跑迁移类操作;真实验证会暴露 mock 层永远拦不住的 SQL NULL/约束类缺陷,验收预算不能省。
- 当 docker build 走远程 builder(102-remote)时,先查 .dockerignore 是否排除本仓大型编译产物(web/desktop/target 等),否则上下文传输+写入层会撑爆远端盘。
- (2026-08-29 郑稳) ModalBottomSheet 表单 IME 开启时 sheet 按聚焦字段上移平移(实测~111px),陈旧坐标的 tap 会打在搜狗候选条上——把拼音组字连同候选词一起提交进**错误字段**(字段出现"bar be"=键入bar+候选be,即此坑签名)。铁律:每次焦点变化/键盘开合后必须重新 uiautomator dump 取坐标再 tap;见到"值=键入串+空格+意外词"先查候选条误触,别判产品缺陷。
- (2026-08-29 郑稳) 断言"打字过程浮层保持展开":比对 dumpsys 弹出式窗口的 **Window hash**——同一 hash 贯穿按键全程=浮层从未关闭(比 frame 有无更硬);注意光标手柄窗(62×75px)也计为弹出式窗口,frame 尺寸按 192px/行折算行数区分,别被 refs 计数假阳性骗。
- (2026-10-17 陈端) 把 `Color(0xAABBCCDD)` 换成常量引用(如 `RN.error`)时不能照抄原右括号数——`Color(` 自带一个 `)`,换成裸常量后要少写一个;正确做法是按新实参重新数括号(listOf/linearGradient/background 各一个)。症状:Syntax error: Unexpected tokens + 连锁 "Expecting an element"。改完立即 build 可一轮抓出。
- (2026-10-17 陈端) check-ui-consistency.mjs 基线按文件粒度"只降不升",把字面量收编进某文件的色板区会让该文件计数上升超基线;修正是最小化手改基线里对应文件那一行,严禁 --write-baseline 全量重生成(会把并行 worktree/同事在途文件的新计数一起固化,压缩别人的余量)。

- 当表单需要下拉选择关联实体(客户/工人/用户/法人)时,修复是先查 `web/admin/src/components/pickers/`(CustomerPicker/UserPicker/WorkerPicker/EntityPicker 均带服务端检索+详情抽屉+跳转管理页),禁止手写 remote Dropdown——CustomerPicker 建好未接线,被手写轮子顶替一个提交周期,复盘才被发现(2026-08-29)。
- 当 domain/handler 需要读业务配置(biz_params)而服务接口不含该方法时,修复是定义单方法接口窄口在调用点断言(`type paramGetter interface{ GetParam(...) }`),测试 fake 零改动;扩公共服务接口是广度膨胀(2026-08-28 cash 限额实测)。
- 当部署前做真实环境验收时,修复是按编号断言表逐项执行并记录响应原文(V1..Vn + 重验 R1..),暴露的缺陷直接升级热修——本轮 8 项断言表暴露 3 个单测拦不住的缺陷(NULLIF 空串/可空 Scan/payNo 回传),验收预算不可省(2026-08-29)。
- 当自动化流水线(订单环节6建档等)会新建表行时,验收造数清理脚本(acceptance-cleanup.sh)的计数/备份/DELETE 三处清单必须同步补该表,否则孤儿巡检门禁拦部署(2026-08-29 lo_accounts 实证)。
- 给表单选项"补数据源"前先盘点各目录端点的权限门禁:跨域目录(组织/资源域)挂的是管理菜单码,受理角色读不到;正确做法是新增跟随受理域本域权限的只读目录端点,而不是放宽既有菜单码授权(2026-08-29 代客受理目录裁定)。
- 新后台能力若做成独立菜单页,会触发 menu-sync 快照对账(需 102 部署迁移后重新采集基线)与 menu.def 总数测试;挂进既有同权限页做抽屉可完全绕开登记链(2026-08-29 注册审核抽屉裁定)。
- 当在仓库新建与根 .gitignore 条目同名的目录下加文件时(internal/pkg/server/ 撞根二进制名 `server`),修复是新文件 git add -f、已跟踪文件 git add -u;git add 报 ignored 先查根 .gitignore 名字碰撞(2026-09-07)。
- 当脚本调 102 admin API 时,认证头是 X-API-Key: <key>,用 Bearer 返回 401 invalid token;写脚本前先 curl 探认证格式(2026-09-07 stripe-recon 实测)。
- 当沙箱环境跑 go 报 ~/Library/Caches/go-build operation not permitted,修复是 export GOCACHE=<repo>/.cache/<name>(该目录已在 .gitignore);~/go/pkg/mod 只读可用,无需升权(2026-09-07)。
- 当 pgx v5 判写操作是否生效,用 res.RowsAffected()(单返回值 int,非 database/sql 的 (int64,error));0 行即假成功应显性报错(2026-09-07 RollbackStage 守卫)。
- 当 vitest 断言两个 Promise 引用相等时（toBe），必然为假——async 函数 return 另一个 Promise 会新建包装；改断言 `expect(await p2).toBe(await p1)` 比较底层值对象身份（2026-08-30 session-handoff）。
- 当解析 `git status --porcelain` 时，禁止对整行 trim——行首两个状态码列（如 ` M`）被 trim 后 slice(3) 错位；解析前只去 \r 和空行，保留行首空格（2026-08-30 session-handoff）。
- 当 npm 需要装包而 vendor 用符号链接提供 @deepseek-ai/* 时，先从 devDependencies 去掉 scope 包再 npm i，装完再补符号链接；否则 npm 试图写链接目标目录 EPERM（2026-08-30 session-handoff）。
- 当验收器/devloop_accept 等待窗口短于长门禁（make check 全量 -race）时长,修复是把门禁放后台跑、rc/log 按 `git rev-parse --short HEAD` 落盘,验收命令只对当前 HEAD 断言 rc=0;HEAD 一动旧结果自动失效,不吃陈旧绿（2026-08-30 上线审计）。
- 当子代理要写主仓库工作区外的路径,先由主会话把 worktree 建在工作区内（`<repo>/.worktrees/<name>` + .git/info/exclude）再派发;子代理会话审批禁用,sandbox_permissions 对其不可用,成品可落 /tmp 由主会话 cp+commit（2026-08-30）。
- 当 pnpm 报 ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY,用 CI=true 重跑（非交互环境允许 purge modules）;且 wrapper 退出码 0 ≠ 门禁本体过,必须读真实 rc（2026-08-30 T2 假红）。
- 当合成客户（负数段 ID,无 customers 主档）走任何旅程,默认它要连过 N 道独立校验闸（uploader_id<=0 / 路径参数<=0 / 主档 LEFT JOIN 查无行）,修掉一道闸下一道才显形——必须全链路复现到终点再宣布修复完成（2026-08-30 实名三道闸:50000→40400→42200 串行显形）。
- 当 curl 复现接口先 grep handler 的 req struct 对齐 JSON 形状,不要凭端上代码或直觉猜字段名（门户登录 mode/smsCode ≠ 猜的 method/code,同会话浪费 3 轮还撞短信冷却）。
- 当轮询 healthz 等 deploy,commit 字段是 7 位短 sha,必须前缀比对而非全等（拿 8 位比对永不命中,白等一个轮询周期）。
- 当对 jsonb 列做 LIKE 模糊匹配,先 `payload::text` 转型,否则 operator does not exist 且整事务回滚（清理脚本半途而废,须重跑）。
- 当给 admin 端新增任何路由,提交清单固定四件套:handler+路由注册 / OpenAPI yaml(否则 check-contract-sync 红) / `make route-perms-check` 再生成 admin_perms_gen.go(不在 make check 链里,102 镜像构建才拦) / 回归测试;漏第三件本地全绿照样部署失败(2026-08-30 施工看板轮)。
- 当本地门禁全绿但 CI 失败,第一反应查「CI 有而本地没有的门禁」,不要怀疑代码——镜像构建里的 genrouteperms --check、web-ui-audit 等都不在 make check 默认链(2026-08-30)。
- 当 git 链式命令关键步骤(merge/push)用 `| tail -1` 看结果,"Updating..." 只是首行不是成功凭证;失败要保留完整 stderr,以 `git log`/`git status` 复核落点为准(2026-08-30 ff-merge 假成功险些连锁删分支)。
- 当排查 gitea actions 失败,run/job 状态在 gitea-postgres `action_run`/`action_run_job`(status: 1=success 2=failure 3=cancelled),完整日志在宿主 `/var/lib/gitea/actions_log/sker/<repo>/<hash前缀>/<taskId>.log.zst`,`docker cp` 出来 zstd -dc 解压即得,不要在 DB/minio 里绕路(2026-08-30)。
- 当往 OpenAPI YAML 的内联 flow map(`{ type: ..., description: ... }`)里写含逗号或 `>` 的描述,必须整体加引号——flow 内 `,` 会终结 plain scalar,`>=6 位` 变成"无法作为 token 开头的字符",整个 bundle 测试红(2026-09-01 worker.yaml POST /workers)。
- 当给 WorkerService 这类跨包宽接口加方法,预期 admin(dispatch_test fakeWorkerSvc、worker_test fakeWorkerOps)、user(portal_actions_test fakeWorkerSvc)三包手写桩各补 3 个 stub;嵌入接口的桩(如 worker 端 fakePortalWorkerSvc)自动吸收但运行期调用即 nil panic,凡新路径过桩必显式 override(2026-09-01)。
- 当部署 102 验证新路由,起一个后台轮询(未带 token 打新路由,404=旧二进制 / 401=新二进制)同时准备验证脚本,轮询命中即跑,省掉盯 CI 的空等(2026-09-01 POST /workers 约 90s 上线)。
- 当在 JSX/style 里写设计令牌,禁止动态拼接 var(--前缀+变量) 的 token 名——web-ui-audit 静态 grep 会把残片(如 --color-)判成幽灵令牌红 build;写两份完整静态 style 对象按条件选用(2026-09-01 userdata 页 notice 横幅)。
- 当 run_code 程序在一个 program 里连做多个写操作,中途报错退出不会回滚已成功的 edit;重跑前必须先 grep 校验上一轮是否已落盘(fields.md 8D-4 双插教训,2026-09-01)。
- 当重构老页面,列设计先读后端列表 SQL 的 SELECT ... AS 别名(pg_lists.go 类)再定前端列——旧 UI 的泛化列(名称/状态)会丢真实字段,SQL 别名就是字段契约,还能顺带发现 fields.md 漏登记(2026-09-01 /bss/userdata 七 Tab)。
- cdp-admin-capture 不带 --base 默认打 localhost:5173;验 102 部署必带 --base http://192.168.0.102:5180。断言先看 bodyHead——页面显示 This site can't be reached 是连错地址,别误判成旧版本未部署(2026-09-01 userdata 验部署绕了 4 分钟)。
- worktree 收尾协议隐含第五步:ff-merge 后必须 push gitea main——CI(deploy-102)只监听 main push,只推 feature 分支则部署永不发生;「合并完成」的判定标准是 102 上出现对应 SHA 的镜像,不是本地 main 指针动了(2026-09-01 干等 20 分钟教训)。
- 当审计/巡检类探针 SQL 只有 pgxmock 测试,mock 会把错误表名固化成"绿"——幽灵引用(loy_ledgers/gis_points 类)必须补 BOSS_PG_TEST_DSN 真库回归,一条 ReconCounts 全量执行就能把 CRITICAL 当场炸出(report/probe_sql_integration_test.go 可复制,2026-09-02 补偿巡检轮)。
- 当给存量表补枚举 CHECK,走"数据归一 UPDATE → ADD CONSTRAINT NOT VALID → VALIDATE CONSTRAINT"三步;NOT VALID 不阻塞 DML,VALIDATE 失败即存量脏值曝光(2026-09-02 coupons 000177,102 真库验证 convalidated)。
- 当展示编码要防超列宽,按 000085 惯例从 id 派生(A-/RK-/C- + lpad(id,N,'0')),别把 orderNo 等变长语义塞进 UNIQUE 码——orders 变长即整笔写入必炸 value too long(2026-09-02 procurement asset_code 39>32)。
当看到「指令格式不对却执行成功」类日志时,先查 SUCCESS 的判定条件与对端真实身份——自研仿真器/桩回的 OK 是闭环自证,不代表业务成功(2026-09-03 provision apply telnet 链)。
- 负例工厂函数默认返回全合规报文时,构造「去合规」用例必须把每个要偏离的字段显式写进 mod(尤其清空默认 Tag/ctag),漏一个默认值就会让负例首跑变正例失败(2026-09-03 tl1sim strict_test loose 模式)。
- worktree symlink 主树 node_modules 后,前端门禁严禁走 pnpm 脚本:pnpm 11 的 deps-status-check 会报 ERR_PNPM_UNSAFE_MODULES_DIR(modules 目录解析目标不是项目子目录)并试图重装;一律分步直调 node_modules/.bin/tsc|vitest|vite + node scripts/web-ui-audit.mjs(2026-09-03 picker-lib 轮实测)。
- 当用 read 分页读大文件后要整文件 write 时,修复是循环 read 直到累计行数==totalLines 再写;否则静默截断丢内容(fields.md 曾丢 1430 行,git checkout 恢复)。skill 没提前警告我。
- 当在 run_code 的 JS 双引号串里写 Go 反引号 raw string 时,修复是直接写反引号字符(无需转义);从模板字面量带来的 ` + BT + ` 拼接会被当字面量落盘。skill 没提前警告我。
- 当 OpenAPI yaml flow mapping {} 内的 plain scalar 含 ASCII 逗号或以 > 开头的片段时,修复是用单引号包裹 scalar,否则解析报 "found character that cannot start any token"。skill 没提前警告我。
- 当要改 SQL 拼装逻辑时,修复是抽成纯函数先写纯单测再接 DB——本次纯单测先于集成抓出 FROM 出现在 WHERE 之后的真 bug。skill 有可测性预告,无此具体坑。
- 当验收/自测脚本要解析工具二进制时,非登录 shell 的 PATH 常不含 ~/bin,command -v 找不到即静默回落到技能 assets 里过期的预编译产物(bossctl 464 vs 线上 509 误判实证 2026-09-04);修复是指定解析链 env > PATH > 显式安装目录 > 源码临时编译,并显式禁止回落过期资产。
- 当 run_code 里调无参工具(如 session_link_list)报 binding arguments must be lossless JSON 时,修复是显式传 {}。skill 没提前警告我。
- 当 node -e 的双层引号命令静默无输出且 exit 0 时,修复是写临时 .mjs 文件再 node 执行,不在 -e 里叠引号(2026-09-04 实证)。
| 当门禁命令需要 tail/head 截取输出时,退出码取的是管道尾(No projects found in "/Users/imeepos/ext512/ymm-001/boss"
[ERR_PNPM_NO_IMPORTER_MANIFEST_FOUND] No package.json (or package.yaml, or package.json5) was found in "/Users/imeepos/ext512/ymm-001/boss". 的 test 失败被吞, 照打假绿) | 修复:门禁一律裸跑看退出码,或 set -o pipefail;「OK 标记」只能由真退出码守卫的分支打印(2026-09-05 T20,build 前置 test 假绿,靠 make check 后 web test 显红才发现) |\n
- 当 worktree 目录尚在但 gitdir 注册失联(目录内 git 报 not a git repository),「能否合并」不能看目录猜——走主仓侧三方核对:git worktree list 注册表、本地+远端分支 refs、diff -rq 对主树找独有文件;独有文件只剩构建缓存且 mtime 全冻结在 checkout 时刻=无未提交工作,再抽一个差异文件用 git log --all --find-object=<blob> 确认内容在历史里,即可安全 rm -rf(2026-09-05 wt-chain-contract 实证:分支本地远端均不存在,快照为 8/29 已并入时代的旧 checkout,gitdir 早被删)。
- 当 pgxmock 单测要扫 timestamptz 到 *time.Time(双指针)报 destination kind 'ptr' not supported 时,修复是域代码用 pgtype.Timestamptz 中转再转指针(先例 pg_ledger.go effTo),测试 AddRow 直填 time.Time 即可(2026-09-05 quadfixa)。
- 当域接口加方法导致全仓 fake 桩编译失败时,修复是接口变更同一提交里 grep『实现该接口的 struct 名』逐个补零值方法,不等 make check 兜底(2026-09-05 quadfixa:LatestActivationCallback/ResolveFactSnapshot 连累 admin 两桩)。
- 当上游查询要判断『同资源活跃行是否存在』且唯一索引只约束活跃态时,修复是置状态前先 SELECT 活跃行判同客户(刷新复用)或跨客户(业务冲突码),而不是等 23505 裸唯一键错变 50000(2026-09-05 任务A:uq_quad_links_address 部分唯一索引)。
- 当依赖宿主把 run_code 程序体包进模板字符串时,修复是程序体内禁现美元符花括号序列本身也要动态构造(fromCharCode(36)+'{'),连改红线 11 条目都能被红线 11 咬(2026-09-05 实证)。
- 当复合 bash 里 cd 子目录后还要操作仓库根文件时,修复是 workdir 钉仓库根 + pnpm --dir 代替裸 cd,或 cd 后全程绝对路径——cd 后相对路径按新 cwd 解析,git add 会静默找错目录报 fatal(2026-09-05 pp1b 两连)。
- 当长 markdown 要落盘时,修复是 tools.write + JS 行数组 join 换行,不走 bash heredoc(三箭头手滑与引号定界符吃掉转义两连败,2026-09-05 pp1b)。
- 当怀疑 API 查询参数是否生效时,修复是用不存在的资源值打反例(999999):返回空=真过滤,返回全量=参数被忽略;只看默认排序首行同值会误判巧合为功能(2026-09-05 pp1b:payments customerId 假过滤)。
- 当并行会话共享同一台机构建时,修复是给 make/长构建钉私有缓存(GOCACHE=/tmp/<会话名>-gocache make check),绝不去 go clean -cache 清共享缓存——既修不了竞争(unlinkat directory not empty)还会打断别人正在跑的构建(2026-09-06 P3-E:共享缓存条目损坏致全包 build failed,私有缓存一次全绿)。
- 当本地无 docker 但要真库验证迁移 SQL 时,修复是用 homebrew postgres 全套二进制(initdb --no-locale -E UTF8 + pg_ctl 起 55432 高位端口 + 停后 rm -rf 数据目录)起一次性集群;pg_ctl 报 FATAL postmaster became multithreaded 就是没给 LC_ALL,启动前 export LC_ALL=en_US.UTF-8(2026-09-06 P3-E:000188/000189 全链真库自检零 docker 依赖)。
- 当新特性的 e2e 只能在部署后才能全绿时,修复是先在旧部署上真机冒烟并按『FAIL 断言与新特性一一对应、清理/残留类断言必须全绿』判读——红得其所即脚本机制已被证明,部署后重跑转绿(2026-09-06 P3-E:sn 列不存在/无唯一索引/无 EPC 校验各断言恰好逐条变红)。
- 当轮询「部署是否完成」时,修复是排除**全部已知旧 hash** 而非「与上次采样不同」——并行会话部署会让 5180 的 index.html 在多个旧 bundle 间翻转,hash 一变就当部署完成会在旧包上白验一轮(2026-09-06 采购白屏轮:10 秒假阳性);且 hash 命中后必须再做行为断言(点详情开抽屉)才算数。
- 当列表页正常而点开单条(详情/编辑)白屏时,修复是先 curl 单条接口看 data 第一层 key 是否又包了一层(单资源 {item} vs 列表 {items}),再对照前端取用形态——TypeError 定位到的 toFixed/属性读取处就是信封错位点(2026-09-06 采购单)。
- 当 bash case 模式要匹配「变量 + 固定字符」时,修复是变量一律裸写(*$c,* 形态)并以哨兵包裹 subject(前后加逗号)实现全包含匹配;变量夹在引号段之间(*",$c,"*)实测永不匹配(2026-09-06 P5-W3:六检查码只剩排序末位命中)。
- 当程序化替换 shell/脚本内容后行为诡异时,修复是用 od -c 或 cat -A 看真实字节——单个字符级损坏(如尾部 * 变 ))在 Read/grep 渲染里完全不可见(2026-09-06 P5-W3)。
- 当要在硬 FK 表上造「引用不存在」类稽核违例时,修复是 SET session_replication_role = replica 直插脏行再 DEFAULT 还原(postgres 官方镜像首用户即 superuser,102 实测可行);造数行用独立标记前缀,清理断言零残留(2026-09-06 P5-W3)。
- 当 e2e 断言依赖部署后端点时,修复是先对旧部署跑一轮:失败断言应恰好红在新能力上、清残留/语法类断言全绿,即脚本机制已证明;部署触发是 push gitea main(deploy-102 workflow),本地 ff-merge 不推 main 就不会有新部署,而脚本本身的 fix 无需等重部署(2026-09-06 P5-W3)。

- 当验收运行器有硬超时(如 8s)而脚本因 N 次独立 ssh+docker exec 往返超时,修复是合并为 1-2 次批量连接:同段多条 SQL 合一个 heredoc 一次连接执行;模板取值内联进 SQL(INSERT..SELECT 带常量标记,免 shell 变量拼接);条件分支改 CASE WHEN EXISTS 内联;标量回执用 SELECT 'LABEL=' || value 单行输出、脚本端 sed -n 按行解析;改完必须连续两轮计时实测并回报数字(2026-09-06 P5-W3:15 次往返 8.4s 被杀 -> 3 次 2.28s/2.49s 双绿)。

- 当 run_code 里要用 session_link_talk 等待执行会话回复,等待窗口必须小于宿主 run_code 程序墙钟上限(600s):talkTimeoutMs 给满 600000 会连程序一起被掐,claimToken 拿不到、事后 collect 报凭证无法识别;给 420-540s 留返回余量,长等待改为磁盘观察(git worktree list / ls-remote)轮替(2026-09-06 AAA 负责人轮:600000 顶满被杀,10 分钟白等)。

- 当同一元素的激活/空闲状态用「常挂基础类+激活才拼的类」实现且两组都含 bg-*/text-* 同名 utility,修复是抽互斥类组纯函数(monthlyTabClass(active):base+active/idle 二选一)+ vitest 断言两组零重叠:Tailwind 冲突 utility 的生效方由 CSS 产物顺序裁决、与 className 书写顺序无关,曾致激活页签亮色白字白底、暗色深底深字(2026-09-06 月度填报,commit 2edcd51d)。
- 当 pgxmock(v4) ExpectQuery/ExpectExec 的 SQL 带占位符而不设 WithArgs,修复是必须补 WithArgs(参数用 AnyArg),报错形如 expected 0, but got N arguments;裸 WillReturnRows/WillReturnError 只匹配零参查询(2026-09-06 AAA-A5 nas_pg_test)。
- 当测试断言异步 goroutine 写入的标准 log/ErrorLog 缓冲,修复是用互斥缓冲(结构体内嵌 sync.Mutex 的 Write/String),strings.Builder 直连必被 -race 拦截;PacketServer.ErrorLog 用 log.New(互斥buf, 完成双保险(2026-09-06 AAA-A5 secret_source_test)。
- 当 go build ./cmd/xxx 会在 cwd 留下同名二进制污染 git status,修复是验证编译用 go build ./... 或 go vet、或 -o /dev/null;残留二进制(abp 包名场景)会被误当成新目录(2026-09-06 AAA-A5 ./aaa 事故)。
- 当 RFC 层面某报文「服务端不可验证」(Access-Request 的 Request Authenticator 为随机数),错密钥的可观测面只在可验证报文(Accounting/CoA 的 MD5 验证子)与 PAP 解密垃圾间接暴露——测试造错密钥场景要选可验证报文类型,别对不可验证协议行为硬断言(2026-09-06 AAA-A5,RFC 2865/2866)。
- 当自定义下拉/浮层嵌在 Radix Dialog/Drawer 里,修复是键盘分支(Esc 尤甚)必须 stopPropagation,否则事件冒泡到 document 被全局 dismiss 层接住、整层容器连同表单一起被关(2026-09-07 PP2-W0 走查实证:Esc 想关下拉却关了整只下单抽屉);面板关闭态按 Esc 仍要放行,保留容器级 Esc 语义。
- 当 vitest 全量跑出现「Test timed out in 5000ms + jsdom 环境启动 80-160s」,修复是先单文件隔离复跑 + main 树基线对照再定性——资源型 flake(并发 transform 挤占)会伪装成业务回归,本例 519/519 两轮全过(2026-09-07 PP2-W0)。

