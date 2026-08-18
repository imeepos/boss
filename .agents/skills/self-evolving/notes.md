# Notes

## 2026-08-18 web/admin app shell 任务反思

**哪个坑浪费了最多时间？**
无 Playwright/Puppeteer 环境下给需要登录的后台页面截图。最终方案：系统 Chrome +
`--headless=new --remote-debugging-port` + Node>=22 全局 WebSocket 裸写 CDP。
次要坑：复用 `--user-data-dir` 导致上一轮写入的 `boss.theme=dark` 泄漏，"亮色"截图拍成了暗色，白跑一轮。

**这个 skill 有没有提前警告我？**
没有（skill 为空）。本次沉淀 CDP 脚本与 localStorage 泄漏两条。

**重来一次我会怎么做？**
- 每次截图运行用全新临时 profile（`mktemp -d`），或每个状态显式写入后 reload 再拍。
- 编辑文件一律先 Read 工具，不用 bash cat 代替（edit 会因未观察而拒绝）。
- React 受控输入框的自动填充，第一轮就该用 native setter + input 事件，不试 `el.value=`。

## 2026-08-18 web/admin 语言切换重构反思

**哪个坑浪费了最多时间？**
不算坑，但原生 `<select>` 的问题值得总结：视觉密度与相邻 ghost 圆形图标按钮不一致（方形带边框 vs 圆形无边框），
且 `option` 下拉部分是系统渲染、CSS 无法定制，暗色主题下仍是系统白色弹层，观感割裂。用户主动指出"不美观、与 antd pro 不符"。

**这个 skill 有没有提前警告我？**
没有。本次沉淀为红线 + 经验各一条。

**重来一次我会怎么做？**
- 顶栏/工具栏内的语言、单位、主题等枚举切换器，第一版就不要用原生 `<select>`，
  直接写"图标按钮 + 自定义下拉浮层"（antd Pro 惯例：地球图标 + listbox 浮层 + 当前项打勾）。
- 浮层复用现有 token（背景/边框/阴影/ hover），保证明暗主题自适应，不新造颜色。
- 记得补 aria-haspopup/listbox/aria-selected 与点击外部收起，成本低但一次写对。

## 2026-08-18 web/admin 顶栏/侧栏系列优化反思

**哪个坑浪费了最多时间？**
UserMenu 改造时 edit 报 old_string not found：函数体在上一轮（LangSwitch）编辑后已变化，我凭旧印象拼 old_string 失败。
次要坑：新增 `common.profile` i18n key 只改了 3 份 locale，漏改 `i18n/types.ts` 的 Translations 类型，tsc 报 TS2353——好在门禁命令立即抓住，一次修复。

**这个 skill 有没有提前警告我？**
edit 失配没有（有"结尾换行"变体但没有"会话内多轮编辑后凭记忆拼 old_string"这条）；i18n 类型闭环没有。

**重来一次我会怎么做？**
- 同一文件第二轮编辑前，先用 Read 重新看目标片段再拼 old_string，不凭上轮记忆。
- 本项目 i18n 是类型闭环：加 key 必须同时改 `types.ts` + zh-CN/en-US/ms-MY 三份 locale，改完立即 `pnpm exec tsc --noEmit`。
- 顶栏设计对齐 antd Pro：工具区 = ghost 图标排（主题/通知/语言下拉），收尾 = 头像+姓名下拉（信息头 + 个人设置 + 危险色退出）；退出不常驻顶栏。
- 多个下拉浮层共用一套 CSS 骨架选择器（.shell-lang-menu, .shell-user-menu 并列写），新增菜单零成本。
- 侧栏激活项对齐：仅当目标在可视区外才 smooth 滚动到视野中央，不打断用户手动浏览。

## 2026-08-18 内容区横向滚动条排查反思

**哪个坑浪费了最多时间？**
自己上次改动引入的回归：`.shell-main` 从 `max-width+margin:auto` 改为 `width:100% + padding 24px`，
而项目从未设置过 `box-sizing: border-box`（全局 grep 为零）——content-box 下实际宽 = 100%+48px，横向溢出。
另外 `.shell-main-wrap` 只有 `overflow-y: auto`，但按规范滚动容器另一轴的 visible 会被计算为 auto，横向溢出直接变横向滚动条。

**这个 skill 有没有提前警告我？**
没有。这是最典型的"静默失败"：改动当场无任何报错，tsc/tests 全绿，只有目视才能发现。

**重来一次我会怎么做？**
- 动布局 CSS（width/padding/flex）后必须目视或 CDP 截图验证，类型检查对 CSS 回归零覆盖。
- 给项目第一次写全局样式时就上 `*,*::before,*::after{box-sizing:border-box}` 重置；块级容器想撑满父级时
  不写 `width:100%`（auto 已撑满且自动扣 padding），width:100%+padding 在 content-box 下必溢出。

## 2026-08-18 顶栏姓名浅色主题不可见 + 读图受限反思

**哪个坑浪费了最多时间？**
`<span>` 换 `<button>`（UserMenu 改造）后，button 不继承父级 color，UA 默认 `color: buttontext`（黑），
叠在两个主题都是深色的顶栏上 → 姓名不可见。且这是"分主题报障"：用户只报浅色主题，深色其实同病只是难察觉。
次要坑：验证阶段想用 read_image 核验截图，模型不支持读图，探测 computed style 的 eval 又被中断。

**这个 skill 有没有提前警告我？**
没有。文字颜色继承问题与"模型读图受限"都是首次遇到。

**重来一次我会怎么做？**
- 在深色/彩色表面放任何 button 时，立刻显式写 color，不指望继承（span 会继承，button/input/select 不会）。
- 改完 UI 元素标签类型（span→button、a→button）时，把"颜色/字体继承断点"列入自查项。
- 颜色问题优先程序化验证：--eval 取 getComputedStyle 对比前景/背景色，不依赖读图。
- 模型不支持读图时：跳过自动核验，继续完成修复，最后总结时明确列出"请人类目测"清单。





## 2026-08-18 超管初始化 + docker compose env 管理

**哪个坑浪费了最多时间？**
不算大坑但有两个：一是 `go` 命令不在默认 PATH（在 /opt/homebrew/bin），首次构建失败后才发现，Makefile 已有 `GO ?= go` 约定但新 shell 每次都要重设 PATH；二是给 `user.Service` 接口加方法后忘了 fakeUser 测试桩要同步补，好在编译期立即暴露。另外 `docker compose config` 验证 env 合并时也因 PATH 问题第一轮无输出，容易被误判为命令失败。

**这个 skill 有没有提前警告我？**
没有。PATH 问题和接口-桩同步都是首次记录。

**重来一次我会怎么做？**
- 每个新 bash 调用一律先 `export PATH=/opt/homebrew/bin:$PATH`，brew 装的工具（go/docker/graphviz）全在那。
- 给 Go 接口加方法时，同一 commit 里就补齐所有测试桩，编译报错清单就是桩清单。
- 验证 compose env 合并用 `docker compose config | grep BOSS_`，输出为空先怀疑 PATH/命令没跑，再看业务。

## 2026-08-18 /base/geo 增删改查检查（403 排查 + 远端库补迁移）

**哪个坑浪费了最多时间？**
geo 全部接口 403 `no permission: menu:geo`，但 /auth/me 确认 admin 就是 sysadmin。绕了一圈看 HasPermission 实现（纯 role_permissions 表 JOIN，sysadmin 无隐式全权）才想到查库：
102 库 `schema_migrations` 只到 000037，缺 000038（geo 表）+ 000039（menu:geo 授权）。次要坑两个：
`schema_migrations.version` 是 TEXT（'000037_alarm_retest'）不是 int，Scan 报错一轮；本机无 psql，用 /tmp 临时 go 程序 + pgx 直连 25432 完成查询和补迁移。
另有一次 42200 是自己拼请求体格式错（timeZones 是 string[] 不是对象数组），看 Go struct 前先猜了格式。

**这个 skill 有没有提前警告我？**
没有。冒烟账号已有（techniques），但"403 → 先比 schema_migrations 与 migrations/ 目录"这条没有。

**重来一次我会怎么做？**
- sysadmin 被门禁拒（403）时，第一步就查 `SELECT max(version) FROM schema_migrations` 对比 `ls migrations/*.up.sql`，权限来自 role_permissions 显式行、迁移漏跑是首要嫌疑。
- 手工补迁移：迁移文件是纯 SQL（含 BEGIN/COMMIT），pgx `Exec` 整文件执行即可，随后手动 INSERT schema_migrations 记录版本。
- 测请求体先读后端 struct（geo.CountryAttrs 等），不凭直觉拼 JSON 字段类型。
- geo 域的"删"= 软删除 is_active=false，测试数据留停用态即可，不物理删（契约 1.5.1）。

---

## 2026-08-18 · geo 全栈落地 + 102 CI 部署(接上轮,续踩新坑)

**哪个坑浪费了最多时间?**
- 迁移编号撞号:`ls migrations | head -60` 截断了列表,以为最新是 000030,新建了 000031_geo_intl;实际已有 000031_order_no_seq,真实最新是 000037(还有 000032~000037)。靠 grep 代码里 "migrations/000031" 的注释才发现,改名 000038 才避免撞号。
- CI 部署 app.env 缺失:compose env_file 引用 app.env 但它被 .gitignore 刻意不入库,CI 全新 clone 必然缺文件,Deploy 步骤直接炸。第一次修还自作聪明用随机 JWT 兜底,被用户指出"部署肯定要固定 app.env"——随机 JWT 每次部署轮换,全部登录 token 失效。二次修复:强制走 gitea 仓库 secret,未配置即 fail fast。
- healthz ok 的歧义:workflow 在 compose up 之前失败时,旧容器还在跑,curl healthz 依然 ok——我误判为"部署成功"。healthz 只能证明"有容器活着",不能证明"本次部署生效"。

**这个 skill 有没有提前警告我?**
- env_file 不入库会炸 CI 这条,上轮 lessons 里只写了"部署时 cp example 填写",没覆盖"无人值守 CI 拿不到这个文件"的场景。本轮已补。
- 迁移编号检查、healthz 歧义,均无预警。

**重来一次我会怎么做?**
- 新增迁移前先 `ls migrations/*.up.sql | tail -5` 看真实最大编号,不信被截断的列表;代码注释里的迁移号(order/pg.go 提到 000031)也要 grep 交叉验证。
- CI 里 compose 需要的密钥文件:一律 gitea repo secret 注入 + 缺失即失败,绝不随机兜底;固定密钥(BOSS_JWT_SECRET)配置一次永不轮换。
- 验证 CI 部署是否真生效:看 actions 运行结果日志,或比对镜像 tag(GITHUB_SHA),healthz ok 只是必要条件。
- 无 docker/psql 的本机验证 SQL:sqlglot(pip install --user)按 postgres 方言 parse 全文件,能拦语法错,拦不了约束语义,真实验证仍需 PG。

---

## 2026-08-18 · geo 页 Ant Design Pro 化重构(双主题适配)

**哪个坑浪费了最多时间?**
- CDP --eval 第一版直接写顶层 `await`,SyntaxError 一轮才想起要包 async IIFE。
- 给 detail 抽屉绑 onClose 时错绑成 onChanged(刷新回调),抽屉永远关不掉——写码自检发现,没浪费运行轮次但属设计失误。
- CountryPanel 空态写出无意义 JSX(`{g.loadFail ? '' : ''}`),tsc 拦不住(合法语法),自检时才改掉。
- 程序化验证第一版把结果挂 window.__verify,--logs 里根本没有(日志只收 console 事件,不收 window 状态),必须 console.log 出来。

**这个 skill 有没有提前警告我?**
- edit 前必须 Read:警告过,但本次 4 个 i18n 文件并行 edit 全被拒——bash grep 定位不算观察,多文件并行编辑前每个都要 Read。
- 模型不能读图:警告过(直接复用"程序化验证"方案,这次零浪费)。
- 顶层 await / window 状态不进 logs / 回调语义混用:均无预警。

**重来一次我会怎么做?**
- cdp-capture 的 --eval 一律先包 `(async()=>{...})()`,多步操作(登录→设主题→跳页→验证)合成一条链。
- 程序化验证结果必须 `console.log("VERIFY:"+JSON.stringify(v))`,--logs 里 grep VERIFY 即得。
- 抽屉/弹层组件的 props 设计:onClose(关)与 onChanged(数据变了要刷新)必须分开,写之前先想清楚父组件两个回调分别做什么。
- 写完 JSX 片段立刻通读一遍再跑 tsc——tsc 只拦类型,拦不住语义废话。
- 主题适配正解已验证:新页面样式全部走 --shell-*/--color-* 令牌 + 页级自定义令牌在 [data-theme] 块切换,零写死色值;主操作按钮复用 --shell-fab-bg(亮=品牌蓝/暗=品牌金)是现成的"每主题强调色"。

---

## 2026-08-18 · 历史记录盘点:经验 → 事实手册固化

**做了什么?**
通读六轮反思(notes + references),发现一类内容错位:geo API 语义、102 库架构事实、冒烟数据状态这类"查证过才知道的项目事实"散在 lessons/techniques 里,而按 skill 分工它们应进 docs/boss-admin-web.md 事实手册。已固化两个新章节:geo 域速查(API 前缀/软删除/attrs 请求体/locale 口径/停用态测试数据/前端样板位置)、102 库与权限架构(schema_migrations TEXT、role_permissions 显式行模型、DB 直连方式)。

**历史暴露的复发性问题(跨会话统计):**
- "bash 查看 ≠ Read 观察"已踩 4 次(2 次记录在案后仍复发),本条是红线但强度不够——批量并行编辑前应逐文件 Read,本次又中招一次。
- PATH(/opt/homebrew/bin)问题出现 3 次;教训已有,靠肌肉记忆执行。
- i18n 4 处同步(types+3 locale)已从"踩坑"变成"顺手做对",经验闭环生效的证据。

**下次盘点的触发条件:** notes 累计 5 轮以上、或发现 references 里同一条经验被重复记录时,做一次"去错位"整理(经验归 references,项目事实归 docs)。

## 2026-08-18 · 菜单图标补齐 + 顶栏搜索收展

**哪个坑浪费了最多时间?**
用户报"国家行政规划缺图标"后我先 grep 菜单定义,再 ls icons/items 才确认缺 geo.svg——顺序对、没浪费。但第二轮"顶部菜单也缺图标"暴露了关键盲区:我默认走"缺 SVG 资产"思路,ls 一看 13 个组图标全在,真正缺的是 TopNav 渲染代码根本没引用图标(侧栏有、顶栏无)。同一个"缺图标"症状,两轮病因不同:一轮是资产缺,一轮是渲染缺。

**这个 skill 有没有提前警告我?**
没有。"图标缺失先分清资产缺 vs 渲染缺"这条不存在;docs 事实手册也没记菜单图标体系(组图标 /icons/<id>.svg、项图标 /icons/items/<key>.svg、MaskIcon currentColor 自适应)。

**重来一次我会怎么做?**
- 收到"X 缺图标"类报障,第一步同时做两件事:`ls` 资源目录 + `grep` 渲染点,先判定"资产缺"还是"渲染缺",再动手。本次第二轮若直接先 ls 就能立刻定位。
- 新增遮罩图标只需拷现有 SVG 规格(24 viewBox/stroke 1.8/round),颜色字段是死值无妨——mask 方案下 background:currentColor 决定实色。
- 收展式搜索框:收起态直接复用 shell-tool-btn(与主题/通知/语言按钮同排同规格),展开态才是 shell-search 椭圆;Esc 全清收起、空值失焦收起、跳转成功收起,三种收起路径一次写全。

---

## 2026-08-18 · gitea secret 方案被驳 + admin 密码不生效排查

**哪个坑浪费了最多时间?**
- 我给的 gitea secret UI 配置方案被用户直接驳回("太麻烦了,为什么不用环境变量")——对内网 homelab,secret UI + 两条 secret + 手动重跑属于过度设计;直接把 app.env 入库(固定 JWT+超管口令)是用户想要的最简路径,而且完全成立。
- admin 登录 40100 排查:app.env 口令配了、bootstrap 跑了,但 admin 是 08-17 就存在的开发账号(real_name=开发管理员),EnsureSuperAdmin 的 ON CONFLICT DO NOTHING 正确跳过了它——"已存在不覆盖"防重置密码的安全设计,反过来让口令对不上。最终直接 UPDATE password_hash 对齐。
- pgx 手写探针连续两个参数坑:SQL 只写了 $2 没有 $1 → 42P18 "could not determine data type of parameter $1";参数占位必须从 $1 连续编号。

**这个 skill 有没有提前警告我?**
- "已存在不覆盖会导致 env 口令对既有账号无效"没有预警(上轮 lessons 只写了 bootstrap 幂等不覆盖是好事,没写它的反面)。
- "判断部署是否真生效"上轮已沉淀(比对镜像/看日志),这次实际用了"admin 是否被新建"当探针,有效。
- 42P18 参数编号坑无预警。

**重来一次我会怎么做?**
- 内网私有仓库 + 用户要简单:第一步就提"app.env 直接入库"选项并说清公网风险,让用户选,而不是默认上 secret UI 标准流程。
- env 口令登录失败时,先查账号 created_at/real_name 判断是"新引导账号"还是"历史遗留账号",后者直接 UPDATE 哈希对齐,不折腾。
- 手写 pgx SQL:占位符从 $1 连续编号;报 42P18 先查编号,再查 ::text 类型标注。

## 2026-08-18 · geo 编辑抽屉多语言适配反思

**哪个坑浪费了最多时间?**
不算大坑,但定位有偏差:用户说"编辑 · 国家 / 编辑 · 行政区划 需要适配多语言",这两个抽屉标题其实早已走 i18n(g.edit/g.tabCountry),真正硬编码的是抽屉内部的字段标签(short name、continent、status、level (1-4)、geonameid 等)。先 grep 语言包确认键已存在、再读组件找裸字符串,才定位到真问题。次要:改 locale 后 tsc 报错才想起 types.ts 类型闭环(notes 里已有此教训,这次又靠门禁兜住)。

**这个 skill 有没有提前警告我?**
i18n 三份 locale + types.ts 闭环有(上轮沉淀);"组件页面必须同时考虑多主题与多语言"没有明文红线——本次用户把它定为硬性规则,已沉淀。

**重来一次我会怎么做?**
- 写任何新组件第一版就让所有文案走 t.*、所有颜色走 token,后补成本远高于首写。
- 接多语言任务先全量扫该页面 JSX 的裸字符串,不按用户点名范围窄化。

## 2026-08-18 · 完成任务未提交被用户指出

**哪个坑浪费了最多时间?**
完成了 geo 编辑抽屉多语言任务(门禁全过)却没 git commit,用户发现工作区脏着点名批评;且工作区还残留更早一轮的分页改动(Pagination/useQueryState)同样未提交。

**这个 skill 有没有提前警告我?**
没有——"任务完成即提交"此前不在任何红线/经验里,我把"验证通过"当成了终点。

**重来一次我会怎么做?**
- 定义"任务完成"= 门禁通过 + 代码已提交,缺一不可;总结回复前跑一遍 git status。
- 分页与多语言改动在同文件交织,按功能拆分成本高,合并一笔但提交信息逐项列明,另单独提交 skill 沉淀。

## 2026-08-18 /base/geo URL 状态 + 分页任务反思

**哪个坑浪费了最多时间？**
很小:编辑 i18n/types.ts 前用了 bash `sed -n` 查看内容,edit 直接被拒("requires reading first")——观察策略只认 read 工具,不认 bash 输出。重 read 一遍即过,但这是第二次踩同类坑(上次是 cat)。

**这个 skill 有没有提前警告我？**
lessons 里已有"编辑文件一律先 Read 工具,不用 bash cat 代替"这条,是 notes 里写的,但 references/lessons.md 没有对应条目——沉淀时喂错了文件,警告没生效。

**重来一次我会怎么做？**
- edit 之前查文件内容只用 read 工具,bash 查看仅用于 grep/定位行号。
- i18n 三份 locale + types.ts 四处同步改动,直接用脚本一次改三份 locale(types.ts 用 edit),比三条 edit 快且不会漏。
- react-router-dom v6 项目里"刷新后搜索条件不变"一律 useSearchParams + {replace:true},不要再造 history.replaceState 轮子;注意与 lib/urlPrefs.ts(一次性覆盖后抹除)语义相反,别混用。

## 2026-08-18 /base/geo 分页+下拉系列纠错反思(用户连续纠正 4 次)

**哪个坑浪费了最多时间?**
不是时间,是返工轮数。同一个分页/下拉被用户连续纠正 4 轮:
1. 初版分页只有"上一页 x/y 下一页"——没查 antd 规范就自造结构;
2. 每页条数用了原生 <select>——顶栏语言切换时已被用户点过一次的问题,换个场景我又犯;
3. 下拉"多主题适配"实际是坏的:CSS 引用了不存在的 --shell-bg/--shell-border,静默走 fallback 白底,暗色主题全错——我声称"全部走令牌"但没 grep 验证令牌存在;
4. 箭头图标是 10px 文字字形 ▾,视觉过小——图省事用字符当图标。

**这个 skill 有没有提前警告我?**
第 2 条有(lessons 有原生 select 教训),但我没在写新组件前回看;第 1、3、4 条都没有。最严重的模式是第 3 条:**静默失败 + 我在总结里说了没验证过的假话**("明暗主题自适应"是抄来的意图,不是验证过的事实)。

**重来一次我会怎么做?**
- 写任何对齐某设计体系的组件,第一步先取一手规范(antd GitHub 的组件 md),不自造结构。
- 写新组件前先 grep lessons.md 里相关关键词(select/图标/主题),旧教训按场景检索,不靠记忆。
- CSS 里每引用一个 var(--x),提交前 grep 确认 x 在 tokens.css/styles.css/组件令牌块中真的定义过;自研组件直接按 geo.css 模式自带 [data-theme] 双令牌块。
- 图标一律描边 SVG(24/stroke 1.8-2/round/currentColor),禁止文字字形当图标。
- 总结里写"X 已适配/已验证"之前,必须有对应的验证动作支撑(grep 令牌、双主题截图或 build);没有就写"未验证,请目测"。

## 2026-08-18 会话收尾又忘提交(用户点名)

**哪个坑浪费了最多时间?** 无技术坑,纯流程坑:geo 全会话改动(URL 状态/分页/下拉/主题修正)做完、门禁全绿、反思也做了,就是没 git commit,等用户点名。这是本会话第 3 次犯"完成后不提交"(前两次教训已在 lessons 里)。

**这个 skill 有没有提前警告我?** lessons 里有两条,但 SKILL.md 高频红线区没有——按场景检索没做,收尾清单不存在。

**重来一次我会怎么做?** 把"git status 干净"纳入收尾门禁:门禁从"typecheck+test+build"扩为"typecheck+test+build+commit";反思流程第 0 步先 git status,有产物先提交再反思。

## 2026-08-19 菲律宾行政区划 PSGC 内置任务反思

**哪个坑浪费了最多时间?**
两个:① raw.githubusercontent 下 7MB JSON 反复断流,裸 curl 一次次假完成(文件在增长但 json parse 永远失败),最后靠 `curl -C -` 断点续传循环十几轮才拼完整;② 本机没有 PG,我直接 `brew install postgresql@16` 装了 10 分钟超时——而项目的真库一直跑在 102(192.168.0.102:25432,configs/config.example.yaml 里明写着 DSN),用户一句话点破。另外第一个搜到的数据集(ciatph/psgc2)是 ARMM 时代的旧口径,差点直接用。

**这个 skill 有没有提前警告我?**
有,但我没在开工前回看:lesson 28"本机没有 psql 却要查/改远端 PG → /tmp 临时 go + pgx 直连"就是这次的正解,我等于重新发明了一遍。教训:**开基础设施/数据库类任务前,先 grep lessons.md 关键词(psql/DSN/PG),再决定装什么**。

**重来一次我会怎么做?**
- 动手装本地基础设施前,先 grep configs/ 找现成 DSN + 问用户"库在哪",homelab 项目几乎都有远端真库。
- 大文件下载一律 `for + curl -C - + 每轮校验(curl 完成判据是内容可 parse,不是 exit 0)`。
- "信息一定要真实"类数据任务:先看数据集的新旧口径标志(ARMM vs BARMM、省数 81 vs 82),再对照官方口径数字,不用第一个搜到的镜像。
- 迁移可回滚性用"单事务 down→up 回环"验证:stripTx 去掉文件内 BEGIN/COMMIT,外层起事务跑完两个文件再 commit,对共享库零风险。

## 2026-08-18 · geo/subdivisions 500 修复：pgx 可空列缺 COALESCE 导致扫描报错

**哪个坑浪费了最多时间？**
没有大坑。唯一的弯路是首先在 server-ts（TypeScript 实体层）里搜 "subdivisions"，结果当然是空的——后端是 Go，路由在 `internal/app/http_geo.go`，SQL 实现层在 `internal/domain/geo/pg_subdiv.go`。从 curl 报错 → 路由 → 调用链 → 查 SQL 实现，这条路本身是对的，但第一轮 grep 偏向了 TS 层（因为刚做过前端任务），浪费了约 1 分钟。

**这个 skill 有没有提前警告我？**
没有。这属于"pgx 可空列扫描"的专门坑：`SELECT d.osm_admin_level` 返回 NULL（SMALLINT 可空），`Scan(&int16)` 直接报错，因为 pgx 的 zero-value 约定只适用于 `*int16` 指针，不适用于 `int16` 值类型。`GetSubdivision` 已正确用 `COALESCE(osm_admin_level,0)`，`ListSubdivisions` 遗漏了。

**重来一次我会怎么做？**
- 排查 Go API 500 时，先看 SQL SELECT 的 nullable 列有没有 COALESCE 包裹——这是 pgx 最常见的扫描错之一。
- 如果 `GEO` 是 Go 后端，第一轮 grep 就限定 `internal/` 目录，不先搜 server-ts。

## 2026-08-19 经验知识分类整理（前端/后端/实施）

**做了什么？**
把 accumulated 在 references/ 里的所有经验（lessons 66 条 + known-issues 19 条 + red-lines 18 条 + techniques 23 条）按"前端/后端/实施/通用"四类重做索引，每一条经验标注来源和要点，按场景分组。

**产出：** `references/knowledge/README.md`（总说明 + 速查统计）+ `前端.md`（49 条）+ `后端.md`（24 条）+ `实施.md`（14 条）+ 每类末尾附"开工前 grep 关键词"。原文不动，只增索引。

**关键决策：**
- 分类原则：按"经验适用场景"而非"这个文件是什么"归类。一条经验可能跨类，优先归入最常使用的场景。
- 通用类（编辑工具使用、流程规范、模型限制等）不单独建文件，分散在三类中按需列出，在总 README 统计表中体现。
- 每条索引只保留"一句话要点"，原文细节在 references/ 原始文件里。

**贯彻了 skill 的"references/ 只增不改"原则**——knowledge/ 是新目录，不碰任何现有文件。同时更新了 SKILL.md 的目录结构和"开工前必查"部分，要求后续每次喂经验后同步更新 knowledge/ 索引。

**哪个坑浪费了最多时间?**
e2e 自清理写对了三轮才闭环,三个坑各废一轮全量验证:① pgx 严格参数校验——六参数喂给只含 $1 的语句直接报 `unused argument`,整批 DELETE 全灭;② `t.Cleanup` 注册的清理跑在同测试 `defer pool.Close()` 之后,池已关,全批静默失败(只有 -v 看日志才发现);③ W8 子测试用 `orderNo6()` 自造独立后缀,按 seed suffix 精确匹配永远漏删 w8 树。另外 edit 工具两次构造失误:old_string 只含 SQL 而 new_string 顺手带了函数头 → 头部重复;一次"只想删个换行"的 no-op edit 把两行并成一行 → TS 语法错。

**这个 skill 有没有提前警告我?**
部分。"edit 前先 read"警告过且我照做了,但没警告 old/new 范围必须对称——这是 read 之外的独立坑。pgx 参数校验、t.Cleanup 与 defer 的顺序、子测试独立后缀,均无预警,全靠真跑 e2e + psql 计数残留才暴露。上一会话的 lesson"真库在 192.168.0.102"这次直接受益,垃圾数据溯源一步到位。

**重来一次我会怎么做?**
- 写批量清理 SQL 前先跑 pg_constraint 依赖图查询,照拓扑序排语句(这次做了,方法值得固化)。
- 清理目标按"测试专用命名模式"(如 `^(e2e|w8)[0-9]+$`)匹配,不信精确后缀——同测试内子场景常自造后缀。
- t.Cleanup 里用连接池时,资源释放必须同走 t.Cleanup(LIFO),不能 defer 与 t.Cleanup 混用。
- 验证闭环 = BOSS_PG_TEST_DSN 指真库跑测试 + psql 按前缀计数残留为 0,缺一不可(单看测试 PASS 不够)。
- edit 的 new_string 严格镜像 old_string 的范围边界,不顺手增删行。
