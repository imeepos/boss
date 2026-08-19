# Known Issues

<!-- 格式：症状 → 原因 → 修法。排查超过 5 分钟的 bug 才值得记。 -->

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
