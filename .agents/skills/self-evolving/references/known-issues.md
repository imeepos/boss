# Known Issues

<!-- 格式：症状 → 原因 → 修法。排查超过 5 分钟的 bug 才值得记。 -->

## 新 worktree pnpm install 报 EACCES: permission denied, mkdir '/Volumes/sker'

症状 → worktree 内 `pnpm install --frozen-lockfile` 全部依赖解析完成后死于 `EACCES mkdir '/Volumes/sker'`(主仓库目录能装)。
原因 → 宿主全局 `pnpm config get store-dir` = `/Volumes/sker/dev-cache/pnpm-store`,该卷当前未挂载;主仓库 node_modules 是旧卷在线时装的,新目录无缓存可用才触发建 store。
修法 → 显式覆盖存储位置:`pnpm install --frozen-lockfile --store-dir ~/.pnpm-store-boss`(本地可写目录,~30s 装完);新环境跑 pnpm 前先 `pnpm config get store-dir` 探活。

## edit 报 "edit requires reading the file first"，但该文件明明看过

症状 → 用 bash `cat` 看过文件内容后调用 edit/write 覆盖，被拒："edit requires reading ... first — read the file, then retry"。
原因 → edit/write 只认 Read 工具的观察记录，bash 输出不算（同一会话内两次踩中）。
修法 → 要编辑/覆盖的文件一律先用 Read 工具读一遍。

## edit 报 "old_string was not found"，内容肉眼完全一致

症状 → old_string 与文件末尾段落逐字符相同却匹配失败。
原因 → old_string 末尾带了 `\n`，而目标文件没有结尾换行。
修法 → 匹配文件末尾段落时去掉 old_string 的尾换行。

## 同一 CDP profile 连拍两套主题截图，"亮色"图拍成了暗色

症状 → 第二次运行截图脚本时，本应亮色的截图呈暗色。
原因 → `--user-data-dir` 复用，localStorage 里上一轮的 `boss.theme=dark` 仍在，首屏内联脚本按它渲染。
修法 → 每个状态显式 `localStorage.setItem` + reload 后再拍；或每次运行 `mktemp -d` 新 profile。

## 脚本收尾 rmSync 临时 profile 报 ENOTEMPTY

症状 → `rmSync(profile, {recursive:true, force:true})` 抛 `ENOTEMPTY, Directory not empty`。
原因 → `proc.kill('SIGTERM')` 后 Chrome 仍在写 profile 目录，立刻 rm 产生竞态。
修法 → kill 后先 `await Promise.race([once(proc,'exit'), sleep(2000)])` 再 rm（scripts/cdp-capture.mjs 已修）。

## 新增 i18n key 后 tsc 报 TS2353 "does not exist in type"

症状 → web/admin 里给 locale 增加 `common.profile` 后 `pnpm exec tsc --noEmit` 报 TS2353。
原因 → `src/i18n/types.ts` 的 Translations 是类型闭环，locale 对象字面量触发 excess property check。
修法 → 加 key 必须同时改 types.ts 对应块 + zh-CN/en-US/ms-MY 三份 locale，改完立即跑 tsc。

## 同一文件第二轮 edit 报 "old_string was not found"

症状 → LangSwitch 改造后再改 RightTools，old_string 明明是之前看到的内容却匹配失败。
原因 → 同一会话内上一轮编辑已改变该片段，凭旧记忆拼 old_string 与最新文件不一致。
修法 → 同一文件第二次 edit 前先 Read 最新目标片段再拼 old_string。

## 内容区出现意外横向滚动条（width:100% + padding 溢出）

症状 → `.shell-main` 改为 `width:100%; padding:0 24px` 后，`.shell-main-wrap` 出现横向滚动条；tsc/测试全绿，纯视觉回归。
原因 → 项目全局无 `box-sizing:border-box` 重置，content-box 下实际宽 = 100%+48px；且滚动容器一轴为 auto 时另一轴 visible 被规范计算为 auto，横向溢出直接渲染成滚动条。
修法 → styles.css 加 `*,*::before,*::after{box-sizing:border-box}`（已加），块级容器去掉冗余 `width:100%`（auto 本就撑满且含 padding）。布局 CSS 改动后必须目视/截图验证。

## sysadmin 访问 /base/geo 全量 403 no permission:menu:geo

症状 → admin（sysadmin）登录成功，/auth/me 角色正确，但 /api/v1/geo/* 全部 403。
原因 → 102 库迁移只到 000037，缺 000038（geo 表+种子）与 000039（menu:geo 权限+sysadmin 授权）；HasPermission 只查 role_permissions 显式行，sysadmin 没有隐式全权。
修法 → 对比 `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1` 与 migrations 目录，漏跑的 .up.sql 用 pgx 整文件 Exec 后手动 INSERT schema_migrations 记录版本。

## pgx 读 schema_migrations.version 报 cannot scan text into *int

症状 → QueryRow(...).Scan(&intVar) 报 "cannot scan text (OID 25) in text format into *int"。
原因 → 本项目 schema_migrations.version 是 TEXT（如 '000037_alarm_retest'），不是 golang-migrate 的整型。
修法 → Scan 进 string。

## 深色表面上 button 文字不可见（span 换 button 后继承断裂）

症状 → 顶栏姓名（深底）浅色主题下不可见，深色主题"看似正常"。
原因 → `<span>` 会继承父级 color，`<button>` 不会——UA 样式默认 `color: buttontext`（黑），深色顶栏上黑字黑底。
修法 → 深色/彩色表面上的 button 一律显式写 color；元素标签类型互换（span↔button/a）时把颜色继承列入自查。

## cdp-capture --eval 报 SyntaxError: await is only valid in async functions

症状 → --eval 里写 `await new Promise(...)` 报 "SyntaxError: await is only valid in async functions and the top level bodies of modules"。
原因 → eval 代码以普通脚本形式执行，不是模块，顶层 await 不可用。
修法 → 整段包 `(async()=>{ ... })()`；返回 Promise 会被脚本 await。

## --logs 里找不到程序化验证结果

症状 → 把验证对象挂 `window.__verify`，--logs JSON 里搜不到。
原因 → --logs 只采集 console 事件/网络请求，window 状态不进日志。
修法 → 验证结束显式 `console.log("VERIFY:"+JSON.stringify(v))`，再从日志 grep VERIFY。

## 多文件并行 edit 全部被拒 "edit requires reading the file first"

症状 → 用 bash grep/sed 定位到 4 个 i18n 文件的插入点后并行 edit，4 个全被拒。
原因 → bash 输出的行不算 Read 工具的观察记录；并行批量编辑时更容易只 grep 不 Read。
修法 → 每个 target 文件先 Read 目标片段（offset/limit 局部读即可），再并行 edit。

## 语义废话 JSX 通过 tsc 但渲染无意义

症状 → 空态写出 `{g.loadFail ? '' : ''}{rows.length===0 ? '— 0 —' : ''}` 这类条件恒假/重复判断的片段，tsc 不报错。
原因 → tsc 只拦类型不拦语义；生成式写 JSX 时局部片段会"看似合理"。
修法 → 写完每个 JSX 片段通读一遍条件与数据流再提交；抽成小组件（如 EmptyState）比内联三元更不易写废。

## app.env 超管口令对既有 admin 不生效

症状 → app.env 的 BOSS_ADMIN_PASSWORD 配好、服务重启,admin 用该口令登录仍 40100。
原因 → EnsureSuperAdmin 是 ON CONFLICT DO NOTHING:admin 账号在更早时间已存在(开发期手建,real_name=开发管理员),bootstrap 按设计跳过,密码保持旧值。
修法 → pgx 直连 `UPDATE accounts SET password_hash=$1 WHERE username='admin'`(bcrypt 新哈希);或删号重启由 bootstrap 重建。口令对齐后 app.env 里保留同一值,保证口径一致。

## CSS 引用不存在的令牌 → 暗色主题静默白块(2026-08-18 geo 下拉)

- 症状:组件在亮色主题正常,暗色主题下触发器/浮层呈纯白,控制台无任何报错。
- 原因:CSS 写了 `var(--shell-bg, #fff)`,但 tokens.css 里根本没有 `--shell-bg`(真实令牌是 `--shell-card-bg`/`--shell-card-border`);CSS 变量缺失不报错,静默走 fallback。
- 修法:`grep -rn -- "--xxx" theme/ styles.css` 确认每个引用的令牌已定义;自研组件按 geo.css 模式自带 `:root[data-theme='light'/'dark']` 两套组件级令牌(如 --dd-bg/--pager-bg),不依赖记忆中的外壳令牌名。
- 加重情节:总结里声称"全部走令牌、主题自适应"但未验证——凡是没 grep/没双主题截图支撑的适配声明都是假的。

## 文字字形当图标 → 视觉过小(2026-08-18 geo 下拉箭头)

- 症状:下拉箭头/打勾看起来特别小、若有若无。
- 原因:用字符 `▾`/`✓` + font-size 10px 冒充图标,字体字形笔画细,缩小后视觉重量远低于真图标。
- 修法:描边 SVG(24 viewBox/stroke 1.8-2/round/width=height=14/currentColor),与项目菜单图标同规格。
## 共享开发库(192.168.0.102:25432/boss)曾被集成测试污染(2026-08-19 已清)
- 症状:addresses 表 276 条全是 e2e*/w8*/an*/g* 测试残留;customers 196/196、ports 275/275 全挂在垃圾地址上;另积累 97 个 e2e 账号/渠道/套餐。
- 原因:e2e_pg_integration_test / analytics / gis 三个集成测试经 BOSS_PG_TEST_DSN 指向共享库且无自清理,每跑一次留一批时间戳后缀数据。
- 修法:已全量清理(按 FK 依赖序);三个测试均已补自清理(e2e 走 t.Cleanup 模式匹配 ^(e2e|w8)[0-9]+$,analytics/gis 走 defer);已实证跑 e2e 后残留为 0。
- 余险:今后若再把 BOSS_PG_TEST_DSN 指向共享库,自清理是唯一防线;建议测试用独立库。

## pgx 可空列缺 COALESCE → Scan(&int16) 报错 → 500 内部错误(2026-08-18 geo/subdivisions)

- 症状:GET /api/v1/geo/subdivisions 返回 `{"code":50000,"msg":"内部错误"}`，库里已有种子数据（osm_admin_level 非空），但其他场景下该字段为 NULL 时触发。
- 原因:ListSubdivisions 的 SELECT 写 `d.osm_admin_level` 而未用 COALESCE，该列在 DB 定义为 SMALLINT（可空，无 NOT NULL 约束）。pgx 的 `Scan(&int16)` 遇到 NULL 值直接报错，该错误不属于 `ErrNoRows`/`ErrDuplicate` 等已知类型，落入 `respondErr` 的 `default` 分支返回 500。
- 修法:把 `d.osm_admin_level` 改为 `COALESCE(d.osm_admin_level,0)`。同文件的 `GetSubdivision` 已经正确使用了 COALESCE，属于同一函数的遗漏。
- 检视:排查 Go API 500 时，先看 SQL SELECT 的 nullable 列（SMALLINT/INTEGER/BIGINT 且无 NOT NULL）有没有 COALESCE 包裹——这是 pgx 最常被遗漏的扫描保护。

## 主题令牌已补但表单视觉仍未确认

症状 → typecheck/test/build 均通过，用户中心表单仍可能在某主题下显示错误的输入背景、文字或 focus 状态。
原因 → CSS 门禁不覆盖视觉；只把固定亮色变量替换为主题变量，未在真实页面切换 light/dark 并读取 computed style 或截图对照，无法证明浏览器实际应用了正确令牌。
修法 → 修改前盘点表单状态，grep 每个 `var(--x)` 的定义；修改后使用全新浏览器 profile，在真实用户中心页面分别验证 light/dark 的背景、文字、placeholder、边框、focus、只读态和按钮，再总结实际验证范围。

## URL 查询参数与本地交互 state 互相覆盖 → 下拉/搜索值回弹

- 症状:下拉可以展开，但点击选项后显示仍是旧值；搜索输入可能刚输入就回退；URL 变化后组件状态不稳定。
- 原因:控件渲染直接使用 `useSearchParams` 派生值，交互更新 URL 触发父树重渲染，旧 query 值又覆盖刚设置的本地值；只改成本地 state 又会丢失刷新恢复能力。
- 修法:采用“双轨状态”：首次挂载用 URL 初始化本地 state；交互只更新本地 state，同时通过独立 setter 写回 URL；不要让 URL 后续变化反向覆盖本地 state。下拉选项提交也应避免依赖全局 `mousedown` 收起顺序。
- 检视:验证必须同时断言选中后的触发器文本、本地筛选结果和 `location.search`，仅检查 URL 或仅跑 build 不能证明控件可用。

## 扫码绑定永不 MATCH：标签未绑定资产 bound_asset_id(2026-08-19 开网 CLI 模拟)

- 症状:按 flow 顺序创建资产且标签设了 tagId 关联、status=UNBOUND,扫码返回 `40920 扫码与预绑定不一致`,怎么扫都不 MATCH。
- 原因:`VerifyScan`(pg_scan.go)的 MATCH 判定是 `tags WHERE epc_code` 必须 `bound_asset_id != 0` 且 `asset_id == link.AssetID`;`CreateAsset` **不会写回** `tags.bound_asset_id`(设计如此),所以"先建资产、建个 UNBOUND 标签"不等于标签绑上了资产。二者是独立表,关联靠标签侧显式填 `bound_asset_id`。
- 修法:先建资产拿 assetId,再创建标签时显式填 `boundAssetId=<assetId>` + `status:"BOUND"`;或补 PATCH 把已有标签 bound_asset_id 指向资产。置备接口 `/provision/assets` 的 TagID 已放宽为可 0(让"资产先建、标签后绑"成立)。
- 检视:核实 epc 对应的标签 `SELECT bound_asset_id,status FROM tags WHERE epc_code='...'`,必须是实际资产 id 而非 0。

## subagent 中断后仍异步写盘/擅自 commit+push
- 症状:list_agents 显示 ready,interrupt 已确认,但工作区仍不断出现新 diff;本地出现本人未做的 commit 且已 push。
- 原因:子代理回合的文件写/ git 操作在 interrupt 生效前已排队执行。
- 修法:interrupt 后 sleep 再 git status 复核;不明提交先 git show 查内容再决定 revert 或保留;清理 untracked 一律 git clean -nd 预览后执行,禁 rm -rf 整目录。

## Compose 多参数组件尾随lambda绑到渲染插槽 → 组合期执行导航(2026-08-20 worker 我的页)
- 症状:点底部Tab"我的"应进个人中心,却直接落到"满意度反馈",返回依次经过 服务公告→接单设置;模拟器与真机均复现;logcat 同毫秒内连发 8 个 push,调用栈全在 Recomposer.performRecompose→Cell。
- 原因:Cell(title, desc, onClick, right: @Composable) 的末位参数是渲染插槽 right;调用点写 `Cell("历史工单") { nav.push(...) }`,尾随lambda绑定到 right 而非 onClick,组合期 `right?.invoke()` 把 push 当内容直接执行,页面上屏即连环压栈。`() -> Unit` 可赋给 `@Composable () -> Unit`,编译器不报错。
- 修法:调用点显式命名 `onClick = { ... }`;若要组件层面根治,把 onClick 放最后一位(但需先审计既有 right 插槽用法,本项目 Hall/Pickup/Maintenance 等 8 处正确用了尾随lambda当 right,不能盲改)。排查利器:NavHost.push/switchTab 里临时 Log.d(tag, msg, Throwable()) 打调用栈。
- 检视:自研 Compose 组件凡"最后一个参数是 @Composable lambda"的,调用点一律禁止裸尾随lambda传动作。

## App 登录成功但全部业务请求 401(2026-08-20 worker/user 双端)
- 症状:登录跳转正常,首页/个人中心/反馈全报"HTTP 401";curl 用同样端点+token 却全 200。
- 原因:登录响应对接 mock 时是平铺 {token},真实后端是 {code,data:{token}};App 从顶层 optString("token") 取到空串,setToken("") 等于没存,Authorization 头永远不带。SharedPreferences 是空 map(run-as 读出)实锤。
- 修法:从 data.token 取值且为空时报错不跳转;排查手法 `adb shell run-as <pkg> cat shared_prefs/*.xml` 直接看 token 落没落盘,比抓包快。
- 症状:后端新起的服务 `curl 127.0.0.1:<port>/api/...` 全部 404,但 `/healthz` 返回 ok;启动日志里路由明明都注册了。原因:`lsof` 不在默认 PATH,真正占用该端口的另有其进程(本例一个孤儿 `./server-new` 绑在 127.0.0.1:18080 IPv4),而新服务器绑 `*:18080` IPv6;curl 走 IPv4 命中旧进程。修法:`/usr/sbin/lsof -nP -iTCP:<port> -sTCP:LISTEN` 看全部监听者,`kill` 陈旧 IPv4 监听者后重测。2026-08-19。

## gitea actions 容器 exec 报 chdir to cwd ... set in config.json failed

症状 → job 第一步(Clone)即失败: `OCI runtime exec failed: chdir to cwd ("/workspace/.../<子目录>") set in config.json failed: no such file or directory`。
原因 → 把 `defaults.run.working-directory` 设在 job 级且指向克隆后才存在的 matrix 子目录;docker 容器 runner 在 exec 前就要 chdir,克隆前目录不存在。
修法 → 去掉 job 级 defaults,Clone 在仓库根执行,仅对克隆后的 run 步骤单独加 `working-directory:`。
## build-install-user-android.sh 在 macOS 报 mapfile: command not found
- 症状:脚本构建成功,安装阶段(select_device)死在 `mapfile: command not found`。
- 原因:macOS 自带 bash 3.2 无 mapfile(bash 4+ 特性)。
- 修法:直接手动 `adb -s <serial> install -r <apk>`;根治需把 mapfile 换成 while read 循环(未修,待办)。
## git add 目录报 "ignored by .gitignore" 但文件已被跟踪
症状: `git add internal/pkg/server/server.go` 报 paths ignored。
原因: .gitignore 第 59 行裸词 `server` 匹配了 internal/pkg/server 目录名(本意是忽略构建产物 server 目录)。
修法: 已跟踪文件不受 ignore 影响,忽略警告正常 add/commit 即可;新增文件若被误伤用 `git add -f` 或先改 .gitignore 为 `/server` 锚定根目录。

## 「我的」页滚动区圆角:clip 了但视觉上看不到
- 症状:Box clip 顶部圆角像素断言"缺口存在"但用户说没圆角,只有静止时首卡自己的圆角。
- 原因:滚动区透明,底下是同色渐变,蓝裁蓝不可见;且圆角挂在内容 Column 上时只有滚到顶才碰到。
- 修法:滚动区 Box 加不透明 Palette.bg 底色,圆角轮廓即刻与渐变头分界;滚动中恒在。

## dev-token.mjs 免登录脚本失效(404)
- 症状:`node web/admin/scripts/dev-token.mjs` 报"登录请求失败(HTTP 404)"。
- 原因:脚本按旧前缀请求 /auth/login,后端实际前缀是 /api/admin/v1。
- 修法:绕过脚本,curl POST http://192.168.0.102:28080/api/admin/v1/auth/login (admin/admin123) 取 data.token,浏览器 localStorage 注入 boss.token/boss.servers/boss.server.active 后直访目标页。

## docker prune 后 boss-server 重启循环 + CI 拉不到 runner 镜像
- 症状:boss-server 反复重启(RestartCount 13),日志 "migrate: read 000066_product_category: permission denied";同时 CI 报拉 192.168.0.102:5000/boss/deploy-runner:latest "no basic auth credentials"。
- 原因:~/docker-clean.sh 的 `docker builder prune -af; docker image prune -af`。(a) 重建镜像时构建上下文里迁移文件恰为 600,COPY 原样保留,容器 USER app 读不了;(b) prune 删掉仅本地标签的 deploy-runner 镜像,而宿主 docker 从未 login registry,一直靠本地缓存掩盖。
- 修法:(a) server.Dockerfile COPY migrations 后加 `RUN chmod -R a+rX /app/migrations`(commit 60b91ca);(b) 宿主 `docker login 192.168.0.102:5000`(凭据 102:~/boss/deploy-image/dotdocker/config.json 的 auth base64)+ `docker pull` 恢复镜像,空提交重触发流水线。验证:`docker run --rm --entrypoint sh <img> -c 'ls -l /app/migrations'` 全 a+r,healthz ok。

## 症状: /payments/stripe/{intent,checkout} 在 102 恒返回 42200 参数非法
- 原因: handler 在 BindBody 之前先查 PayGateway.Get("stripe");102 容器未配 BOSS_STRIPE_API_KEY,网关未注册即按裁定"密钥未配即降级"返回 CodeInvalidParam(42200),与请求体无关。
- 修法: 配置 BOSS_STRIPE_API_KEY(可选 BOSS_STRIPE_WEBHOOK_SECRET/BOSS_STRIPE_API_BASE)后重启 boss-server;排查时先看 internal/httpapi/user/stripe.go 的校验顺序。

## 症状: ETL RecordRun 报 "ERROR: operator is not unique: unknown - unknown (SQLSTATE 42725)"
- 原因: pgx 参数化 SQL `EXTRACT(EPOCH FROM ($3-$2))*1000` 中,当 FinishedAt 为 NULL(RUNNING 记录)时,`$3` 与 `$2` 的类型无法从 NULL 上下文推断,PG 在 unknown-unknown 上找不到唯一运算符。
- 修法: 显式 cast `$3::timestamptz-$2::timestamptz`(commit a77e28b)。排查线索: docker logs boss-server | grep "etl executor" 会出现 record running: ERROR,而 etl_job_run 表为空——先看应用日志再查表。

## CI deploy run 日志只剩首尾行、连败但代码没变(2026-08-25)
- 症状:deploy-102 全部 run 5 秒内死于 Clone 后,run 日志(zst)丢失全部中间步骤输出;build/push/compose 手动执行全过;重启 gitea-runner 无效。
- 原因:act_runner 0.2.11 run 日志流丢失(工具 bug);真实失败=workflow 脚本 `$(docker images | grep | head -1)` 在 pipefail 下的 SIGPIPE 竞态(exitcode 141),本地镜像 tag 累积增多后必现。
- 修法:① 定位:runner config.yaml level 改 debug → docker logs gitea-runner 看步骤名+exitcode(排查完调回 info);② 根治:管道尾加 `|| true`(c2a2df61);③ 预防:pipefail 脚本禁裸 `| head -N`。

## 症状: Stripe confirm 返回 400 "This PaymentIntent is configured to accept payment methods enabled in your Dashboard..."
- 原因: 测试账号启用了重定向型支付方式(Dashboard 配置),API 直接 confirm 未带 return_url 被拒。
- 修法: confirm 请求带 `return_url=...`;或建 intent 时设置 `automatic_payment_methods[enabled]=true` + `allow_redirects=never`(仅当允许降级非重定向方式时才需要后者)。
- 排查线索: 400 body 的 error.message 含 "provide a `return_url`"。

## 症状: cloudflared 快速隧道打到别的服务(返回的 401/JSON 文案非本仓库所有)
- 原因: 隧道容器名(`cf-stripe`)不保证指向 boss-server;其 `--url http://api:8080` 在所属网络里解析到
  release-platform-integration-api(另一项目),返回别家 401 文案。
- 修法: 建新隧道容器挂目标项目网络(如 boss-app)用服务名 `--url http://server:8080`;用前先验后端身份
  (/healthz + 未配置端点降级特征,见 techniques)。
- 排查线索: 响应的 error code/文案在仓库 grep 不到 = 不是自己的服务。

## 症状: 门户注册成功但返回 customerId=-1,账单插入报 bills_customer_id_fkey
- 原因: portal 注册走合成客户空间(portal_seq 负数 id,000057),不在 customers 表;而 bills/payments 硬 FK 指向 customers。
- 修法: E2E 先建真实 customers 行(注意 customer_code/legal_entity_id/address_id 等必填列,查 live 表防迁移漂移),
  再 UPDATE portal_accounts 把 phone 改指该 id,密码模式重登拿新 token。
- 排查线索: `SELECT customer_id FROM portal_accounts WHERE phone=...` 看符号是否负数。

## 症状: /auth/sms-code 或注册 42200 参数非法
- 原因: 短信通道对 phone 做 E.164 归一化,当前仅支持 +86/+60 区号(Region() 决定),639xxx 等其他开头被拒。
- 修法: 测试用 138 开头的 11 位中国手机号;新市场需 internal/pkg/sms 的 Region() 追加区号并配通道。

## 症状: Stripe Checkout 支付成功但 payment_intent.succeeded 事件 metadata 为空 → settle 静默跳过
- 原因: Checkout Session 上的 metadata(如 pay_no/bill_no/customer_id)不会自动出现在底层 PaymentIntent
  上,payment_intent.succeeded 事件只带 PI 对象,寻址字段全空,webhook 返回 200 但不落账(静默失效)。
- 修法: `CreateCheckoutSession` 表单同时写 `payment_intent_data[metadata][key]` 才透传;Session 级
  metadata 保留双保险。验签层查 Stripe: `GET /v1/payment_intents?created[gte]=...` 看 PI metadata。
- 排查线索: webhook 200 但无新 payments 行 → 查 Stripe PI 的 metadata 是否为空。

## 症状: docs/user 静态演示页列表全空(locale.js 语法错 + 信封不匹配)
- 原因: ① docs/user/locale.js 被模板片段污染(~30 行 ×3 语区,如 `'user.profile.text11': '在用'' : '...`)js 解析挂;
  ② 13 个 demo 页都写 `d.items`(平铺响应),真实 user API 返回 `{code,data:{items},msg}` 信封。
- 修法: demo 页消费处改 `(d.data||d).items`;locale.js 需按语区重建被污染键。此系既有 demo 缺口
  (domain-map PORT 待完善),与支付链路本身无关;真实门户客户端(Android)按信封消费正常。

## Material3 2026.06+ BOM：ExposedDropdownMenu 已不能作为顶层 Composable 调用

**症状**：写 `androidx.compose.material3.ExposedDropdownMenu(expanded=..., onDismissRequest=...) { items }` 编译报 `Unresolved reference 'ExposedDropdownMenu'`；`ExposedDropdownMenuBox` 内 content lambda 也报 `@Composable invocations can only happen from the context of a @Composable function`。

**原因**：Material3 新版 API 把菜单直接吸收进了 `ExposedDropdownMenuBox` 的 `content: @Composable ExposedDropdownMenuBoxScope.() -> Unit`，不再提供独立的 `ExposedDropdownMenu` Composable；旧 import 完全限定名 `androidx.compose.material3.ExposedDropdownMenu(...)` 已经移除。

**修法**：在 `ExposedDropdownMenuBox { ... }` 的 content lambda 里**直接**放 `DropdownMenuItem(...)`，不要再包一层。`menuAnchor(...)` 仍照旧。详细见 `knowledge/android.md`。

## Google Play Services Tasks 在 Kotlin 协程里 .await() 编译报 Unresolved reference

**症状**：写 `client.getCurrentLocation(...).await()` 编译报 `Unresolved reference 'await'`。

**原因**：`play-services-location`、`play-services-tasks` 等不内置 `kotlinx-coroutines-play-services` 的 `.await()` 扩展，需要单独依赖。

**修法**：`gradle/libs.versions.toml` 加
```toml
coroutines = "1.10.2"
kotlinx-coroutines-play-services = { group = "org.jetbrains.kotlinx", name = "kotlinx-coroutines-play-services", version.ref = "coroutines" }
```
`app/build.gradle.kts` 加 `implementation(libs.kotlinx.coroutines.play.services)`，再 `import kotlinx.coroutines.tasks.await`。

## Material3 ExposedDropdownMenu 的 API 路径：必须 ExposedDropdownMenuBox scope 上下文

**症状**：写 `androidx.compose.material3.ExposedDropdownMenu(expanded=..., onDismissRequest=...) { items }` 编译报 `Unresolved reference 'ExposedDropdownMenu'`；在 ExposedDropdownMenuBox content lambda 里直接放 `DropdownMenuItem(...)` 不报编译错但**渲染时菜单项被识别为 anchor 子项**，跟输入框 placeholder 重叠显示。

**原因**：Material3 1.3.x（含 2026.06 BOM）的 `ExposedDropdownMenu` 是 `androidx.compose.material3.ExposedDropdownMenuBoxScope` 的**扩展函数**，不是顶层 Composable；Box content lambda 里的第一个非 TextField 子项会被识别为 anchor 内容的一部分。

**修法**：
```kotlin
ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
    OutlinedTextField(value = value, ..., modifier = Modifier.menuAnchor(...))
    // 必须用 ExposedDropdownMenu 包裹菜单项
    ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
        options.forEach { item ->
            DropdownMenuItem(text = { Text(item) }, onClick = { onChange(item); expanded = false })
        }
    }
}
```

**症状**：`catch (e: Exception)` 之后 launchLocation 静默失败，hint 和 door 都 set("") 用户没反馈。

**修法**：`catch (e: SpecificFailure)` + `catch (e: Exception)` 双层；Failure 子类型把对应 message 写入 err 文案位。

- 症状: bossctl --server ... call POST user:/orders 返回 404 非 JSON,但服务端确实注册了 /api/user/v1/orders。原因:cmd/bossctl/routes.go 的 portalPrefixes 里 user 前缀 = "/api/v1"(历史遗留),与服务端实际 "/api/user/v1" 不一致;upload_test.go 也固化旧值。修法:调 user 端一律写完整路径,或修 routes.go 前缀。

## UI 双主题断言假阴性:同步读 computed style 吃进 transition 中间值(2026-09)

**症状**:CDP/Puppeteer 同步 eval 中 `documentElement.setAttribute('data-theme','dark')` 后立即 `getComputedStyle(el).backgroundColor`,返回仍是亮色值,误判"暗色令牌没生效",实际元素带 `duration-200` 等过渡类,200ms 内读到的是插值起点。

**修法**:改属性与读结果拆成两次 eval,中间走工具的 settle(≥2500ms);或 eval 内 `new Promise(r=>setTimeout(()=>r(getComputedStyle(el).backgroundColor),600))` 返回 Promise 由 awaitPromise 接住。

## 后台菜单某项不显示图标(2026-08-27)
- 症状:侧边栏菜单项文字前空白,Network 里 /icons/items/<key>.svg 404。
- 原因:Sidebar 的 MaskIcon 按 menu.def.ts 的 key 映射 public/icons/items/<key>.svg;新增菜单项(如 crashlogs)时只登记了 key 没放 SVG 文件。
- 修法:补一个 24x24 stroke 风格 SVG(stroke=#8b98a5, stroke-width=1.8, 参考同目录 audit.svg/storageconfig.svg 画法),CI 推 main 自动构建 admin-web 镜像上线。先例:stripeconfig.svg(5cedb1d8)、crashlogs.svg(8f6b6c34)。realname-review 也缺,归属并行分支 fix/realname-review-icon-theme。

## pgx []byte 入参 vs JSONB 列:bytea→jsonb 隐式转型不存在,必 22P02(2026-08-27)

**症状**:`INSERT INTO t (..., jsonb_col) SELECT ..., $N FROM ...` 中 `$N` 绑定 Go `[]byte`(如 `json.Marshal(payload)` 结果)。pgx 默认把 `[]byte` 按 bytea 编码(`\x...` hex),PG 的 bytea→jsonb 隐式转型不存在,执行报 SQLSTATE 22P02 `invalid input syntax for type json`。该 bug 在匹配订阅数=0 时**完全不暴露**(SELECT 0 行,PG 不对 `$N` 求值/转型),仅在环境首次出现匹配订阅时触发——典型"subscriptions=0 长期未暴露"型隐蔽断链。首例:`internal/domain/openplat/webhook_pg.go InsertDeliveries`。

**修法**:两种,与 `internal/domain/billing/pg_ar_closure.go` 等既有 JSONB 入参写法对齐:
- 入口侧:`string(payload)` 传 Go 字符串(pgx 发 text),SQL 用 `$N::jsonb` 显式转型(text→jsonb 走 PG 内置 cast);
- 或 SQL 侧加 `$N::jsonb` + 确保 pgx 发的是 text(用 string 而非 []byte)。
任何 INSERT/SELECT 带 JSONB 参数的语句,code review 必查入参是 string 还是 []byte。

## INSERT...SELECT 无 RETURNING 用 QueryRow.Scan 必返 ErrNoRows(2026-08-27)

**症状**:`INSERT INTO t SELECT ... ON CONFLICT DO NOTHING` 没有 RETURNING 子句,代码却 `db.QueryRow(...).Scan(&n)` 读取插入行数。SELECT 0 行时(无匹配),整条 INSERT 返回 0 行结果集,Scan 必报 `pgx.ErrNoRows`,调用方把"本该成功的 0 插入"判为失败并向上冒泡 50000。即使修好 JSONB 转型,该 bug 仍让首条 InsertDeliveries 走 ErrNoRows 路径,Emit 永远返错。环境从未配置订阅时 0 插入 = 永远 ErrNoRows,与 JSONB 22P02 叠加形成双重假失败。

**修法**:无 RETURNING 的 INSERT/UPDATE 一律用 `db.Exec(...)` + `tag.RowsAffected()` 读取行数;QueryRow.Scan 只用于有 RETURNING 的语句或 SELECT。Code review 对 `QueryRow(...).Scan(&n)` 必查 SQL 是否含 RETURNING。

## 102 健康检查通过但业务 API 被 LICENSE_REQUIRED 阻断(2026-09-26)

**症状**: `GET /healthz` 返回 `{"status":"ok"}`,登录接口可用,但带 token 请求业务 API 返回 `{"code":"LICENSE_REQUIRED","msg":"system license required"}`。

**原因**: healthz 只检查 HTTP 进程存活;授权中间件在业务路由前阻断,与代码是否已部署是两个独立事实。

**修法**:部署验收分三层记录:健康检查、登录/鉴权、目标业务路由。目标路由遇 `LICENSE_REQUIRED` 时标记为环境授权阻断,保留响应证据,不要修改业务代码绕过授权或声称已完成真实回归。

## cdp-capture DOM 断言 innerText.includes 撞上外壳同子串文案(2026-09-26)

**症状**:断言"收起"按钮是否存在用了 `b.innerText.includes('收起')`,匹配到侧栏菜单项"收起菜单",点击是 no-op,断言读回行数不变 → collapse 功能"假阴性"。首轮报 ok:false,实为选择器错。

**修法**:断言目标按钮一律 `innerText.trim() === '精确文案'`,或先锚定业务容器(section/table 祖先)再在其内查;外壳(侧栏/顶栏/菜单)文案与页面按钮共享常用词(关闭/收起/展开/刷新)是常态。

## 侧栏兄弟菜单双击亮:NavLink 前缀匹配(NavLink isActive prop 已移除)(2026-10-01)

**症状**:访问 `/boss/site/cats`(官网分类),侧栏 `/boss/site`(官网内容)与 `/boss/site/cats` 同时高亮(CDP 断言 nav aria-current 返回两条)。

**原因**:react-router-dom 6.30.4 的 NavLink 默认 `end=false` 前缀匹配(`locationPathname.startsWith(toPathname)`),`/boss/site` 是 `/boss/site/cats` 前缀即算 active。此版本 NavLink **已无 `isActive` prop**(6.26 有,6.30 从 props 移除,只剩 className/children 函数收 `{isActive,isPending,isTransitioning}`)。

**修法**:Sidebar 从 NavLink 改 Link + 显式 `aria-current={active?'page':undefined}`;激活判定抽到 `router/menu.def.ts` 的 `isNavActive(to,pathname)`:精确路径激活;深层路由仅当其不是其它菜单项完整路径时算同页(如 `/boss/site/new` 仍高亮官网内容,`/boss/site/cats` 不高亮)。同一缺陷顺带修掉 `/bss/marketing` vs `/bss/marketing-recon`。改动前先 `grep -n isActive node_modules/.../react-router-dom/dist/index.d.ts` 确认版本 API。
