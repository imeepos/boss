# Notes

## 2026-08-21 师傅端 token 失效未跳转登录页

**哪个坑浪费了最多时间？**

首次编译未设置 Java 17，Gradle 直接提示找不到 Java Runtime；设置 `JAVA_HOME=/opt/homebrew/opt/openjdk@17` 后才正常验证。

**这个 skill 有没有提前警告我？**

Android 索引已明确记录 gradlew 报无 Java Runtime 时使用 Homebrew OpenJDK 17；同时提醒 token 失效需覆盖 HTTP 401 与业务码 40100。

**重来一次我会怎么做？**

先检查 JAVA_HOME 再编译；认证客户端统一把 HTTP/信封错误交给一个回调，并在 Compose 根路由注册回调，清除 token 后 reset 到 Login，避免各页面重复处理。


## 2026-08-21 订单预占按钮跳转与参数非法

**哪个坑浪费了最多时间？**
把用户描述的“点击预定跳转”先当成路由问题，实际代码中预占按钮没有显式 `type`，且直接绕过资源核查调用 `/reserve`；后端对未核查/无空闲资源按契约返回 `42200 参数非法`，并非前端导航。

**这个 skill 有没有提前警告我？**
前端经验已要求真实 DOM/API 交互验证、下拉统一使用 `Dropdown`，但没有提醒订单动作按钮必须显式 `type="button"`，也没有把工作流前置状态校验列成 UI 回归项。

**重来一次我会怎么做？**
先用 102 真实接口复现并对照状态机，再修 UI：stage 2 的预占入口必须打开资源核查抽屉，核查通过后才能预占；页面动作按钮统一显式 `type="button"`，避免嵌入表单时隐式提交导航。


## 2026-08-21 登录/注册/找回页对齐 login-register-states-v2 设计稿

**哪个坑浪费了最多时间？**
两处：① Kotlin 带默认值尾参 + trailing lambda——`AuthPhoneRow(value, onChange, hint="")` 的尾 lambda 绑定到 `hint` 而非 `onChange`，三处调用同时报错；② 用 perl -pi 批量改文件后没重读，edit 报 file changed since read（红线#1 变体：shell 改文件≠已观察）。另外真机盲点坐标 tap 两次截图错页，最后用 uiautomator dump 取 bounds 才点中。

**这个 skill 有没有提前警告我？**
红线#1 警告过外部改文件问题；trailing lambda 绑定规则和 uiautomator dump 技巧均无沉淀。

**重来一次我会怎么做？**
Compose 自定义组件一律把回调函数参数放最后且其后不放默认值参数；shell 批量改完立即 read 再 edit；真机点击定位一律先 uiautomator dump 取 bounds，不猜坐标。

## 2026-08-21 worker/user 代码体检续修

**哪个坑浪费了最多时间？**

照片页原先先创建空文件再调用不接收文件内容的接口，虽然看似完成流程，实际会报告假上传成功；另外移除 Compose 状态委托 import 后首次构建才暴露 `setValue` 缺失。

**这个 skill 有没有提前警告我？**

Android 索引强调真实接口优先和构建失败后不可继续使用旧包；但没有直接阻止“接口未接文件仍提示成功”，需要结合后端路由审查判断。

**重来一次我会怎么做？**

先核对 API 是否真正接收文件字节再保留上传按钮；若不支持就明确禁用并提示未接入。每次删 import 或状态字段后立即跑 compile，最后检查 git status 和远端分支。


## 2026-08-20 mobile/user 用户首页 Compose 视觉对齐

**哪个坑浪费了最多时间？**
首次编译暴露了两类直接问题：重写页面时漏掉 `Column`/`fillMaxWidth` import，以及把不存在的 `FontWeight.Regular` 当作枚举值使用；同时将默认在线文案改成了设计稿/既有单测要求的“服务在线”。

**这个 skill 有没有提前警告我？**
Android 索引提醒了 Compose 显式 `onClick`、edge-to-edge safeDrawing、契约字段和真实接口优先；但没有直接覆盖 Kotlin Compose 字体枚举和重写后 import 闭环，因此本次由编译门禁捕获。

**重来一次我会怎么做？**
大段重写后先立即运行 `compileDebugKotlin`，再进行视觉细化；先保留既有单测断言的默认值，新增视觉文案时通过组合展示而不是改变契约解析语义。最终必须在提交前复核所有改动文件行数、测试、APK 构建和 git status。


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





## 2026-08-18 登录凭证口径修复

**哪个坑浪费了最多时间？**
部署配置 `app.env` 使用 `Boss-admin-2026`，但冒烟脚本、开发种子和用户实际输入约定为 `admin123`；启动引导对既有账号使用 `ON CONFLICT DO NOTHING`，所以仅改配置不会更新数据库密码。

**这个 skill 有没有提前警告我？**
有，`known-issues.md` 已明确记录既有 admin 不会被 bootstrap 覆盖。本次按该经验先对齐数据库，再同步部署配置。

**重来一次我会怎么做？**
登录报 401 时先核对三处口令来源：实际账号哈希、部署 `BOSS_ADMIN_PASSWORD`、冒烟/前端默认值；确认后用开发种子或等价 bcrypt 更新既有账号，并用真实 `/auth/login` 和 `/auth/me` 验证。

## 2026-08-18 浏览器截图与布局分析任务反思

**哪个坑浪费了最多时间？**
1. 模型不支持图像输入：尝试使用 read_image 工具读取截图时，发现 Kimi-k3 模型不支持图像输入，无法直接分析截图内容。
2. 浏览器自动化工具缺失：尝试使用 Playwright/Puppeteer 进行浏览器截图，但环境缺少这些依赖。
3. 图像分析工具限制：虽然有 Pillow 库，但只能进行基础颜色分析，无法进行深度布局识别。

**这个 skill 有没有提前警告我？**
部分警告：self-evolving 技能中提到了"模型不支持读图"的问题，但没有具体说明 Kimi-k3 的限制。cdp-capture.mjs 脚本提供了截图功能，但缺少后续的图像分析能力。

**重来一次我会怎么做？**
- 优先使用代码分析代替图像分析：通过阅读前端代码（CSS/React组件）来理解布局结构，而不是依赖截图分析。
- 利用现有工具链：使用 cdp-capture.mjs 进行截图，但结合代码审查来提供优化建议。
- 明确模型限制：在开始任务前确认模型的能力边界，避免尝试不支持的操作。
- 创建自动化分析脚本：开发基于代码的布局分析工具，而不是依赖图像识别。

## 2026-08-19 后端 CORS 本地客户端修复

**哪个坑浪费了最多时间？**
排查时发现前端已经直连绝对地址且依赖后端 CORS，但服务端 Gin 装配根本没有跨域中间件，导致 JSON 登录请求的 OPTIONS 预检直接失败。另一个环境坑是 `go` 不在默认 PATH，只能使用 `/opt/homebrew/bin/go`。

**这个 skill 有没有提前警告我？**
后端经验提示了真实链路与 API 直连，但没有提示“前端移除代理后必须同步确认服务端 CORS 中间件和部署环境变量”。本次补充为通用经验。

**重来一次我会怎么做？**
先从浏览器错误里的 Origin、预检方法和请求头反查服务端 middleware；用 httptest 覆盖允许 Origin 的 OPTIONS 与未知 Origin 拒绝，再用部署环境显式配置来源，不使用通配符配合 credentials。

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

## 2026-08-18 · base/geo 控件无法交互修复反思

**哪个坑浪费了最多时间？**
最初把“输入框打不进去”归因到 URL 受控状态，实际先要区分当前页面是否真的加载了目标 admin 应用；在 DSH GUI 路由下 CDP 看到的是 Harness 自身 DOM，不能直接证明业务页控件状态。代码侧确认搜索状态由 URL 更新后仍可正常受控，明确修复是统一使用项目 Dropdown，并把浮层 z-index 提升到抽屉层级以上。

**这个 skill 有没有提前警告我？**
有两条相关经验：全站下拉应复用 Dropdown、验证不能只看截图；但“当前 URL 是宿主 GUI 而不是目标 Vite 应用”以及“z-index 必须覆盖 Drawer”没有明确记录。

**重来一次我会怎么做？**
先确认目标 app 的实际运行入口和端口，再用应用自己的 DOM 做交互验证；所有新下拉直接复用 Dropdown，并检查弹层与 Drawer/Overlay 的层级关系。修完跑 typecheck、test、build，提交时只 stage 本任务文件，避免带入其他并行改动。

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

## 2026-08-19 用户中心 Ant Design Pro 化重构

**哪个坑浪费了最多时间？**
上一版用户中心直接用多张渐变卡片堆叠，没有先做信息架构，导致视觉上不像管理后台，用户要求返工。本轮先调研 Ant Design Pro Account Settings 与通用 SaaS 个人中心，再改为“紧凑账号头部 + 左侧设置导航 + 右侧单任务面板”。另外收尾时在 web/admin 目录执行了仓库根路径的 git add，命令失败后才切回仓库根目录。

**这个 skill 有没有提前警告我？**
前端知识索引明确要求先查 Ant Design 一手规范、CSS 令牌 grep、图标不能用文字字形，本轮按这些执行；但“页面任务先输出信息架构再编码”此前没有形成明确经验。

**重来一次我会怎么做？**
- 页面任务先写信息架构、状态清单、视觉层级和响应式方案，再写 JSX/CSS。
- 个人中心优先采用设置工作区，不把所有功能同时展开成卡片。
- 构建、测试、令牌 grep 只能证明代码门禁，真实视觉仍需在目标业务页面做亮暗主题目测或 CDP 验证，并在总结中明确区分。
- git 操作统一从仓库根目录执行；若 bash workdir 是子项目目录，先确认相对路径语义。

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
没有大坑。唯一的弯路是首先在已移除的 server-ts（TypeScript 实体层）里搜 "subdivisions"，结果当然是空的——后端是 Go，路由在 `internal/app/http_geo.go`，SQL 实现层在 `internal/domain/geo/pg_subdiv.go`。从 curl 报错 → 路由 → 调用链 → 查 SQL 实现，这条路本身是对的，但第一轮 grep 偏向了已移除的 TS 层（因为刚做过前端任务），浪费了约 1 分钟。

**这个 skill 有没有提前警告我？**
没有。这属于"pgx 可空列扫描"的专门坑：`SELECT d.osm_admin_level` 返回 NULL（SMALLINT 可空），`Scan(&int16)` 直接报错，因为 pgx 的 zero-value 约定只适用于 `*int16` 指针，不适用于 `int16` 值类型。`GetSubdivision` 已正确用 `COALESCE(osm_admin_level,0)`，`ListSubdivisions` 遗漏了。

**重来一次我会怎么做？**
- 排查 Go API 500 时，先看 SQL SELECT 的 nullable 列有没有 COALESCE 包裹——这是 pgx 最常见的扫描错之一。
- 如果 `GEO` 是 Go 后端，第一轮 grep 就限定 `internal/` 目录，不先搜已移除的 server-ts。

## 2026-08-19 · geo 下拉与 URL 状态双轨修复反思

**哪个坑浪费了最多时间？**
把 URL 查询参数直接当作控件实时渲染源，导致下拉选中后值被 URL 派生值回弹；之后又在“本地 state”与“URL 可恢复”之间来回调整。真正稳定的模型是：首次挂载从 URL 取初值，交互只改本地 state，并通过独立 setter 双写 URL。

**这个 skill 有没有提前警告我？**
已有“useQueryState 保持刷新状态”和“筛选变化重置 page”的经验，但没有明确写出 URL 只初始化、不能反向覆盖本地交互 state 的双轨规则。更严重的是，本轮只跑了 build/test，没有在真实业务页面 DOM 中点击下拉并断言选中文本、筛选结果和 URL，违反了“不能无验证宣称已验证”的红线。

**重来一次我会怎么做？**
先画清状态数据流：URL → 初始 state；事件 → 本地 state + URL setter；URL 后续变化不回灌。修完优先在目标 Vite 应用而不是宿主 GUI 上做真实点击断言，再跑 typecheck、test、build，并在总结中准确区分“代码门禁通过”和“交互已验证”。

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

## 2026-08-19 用户中心表单多主题适配反思

**哪个坑浪费了最多时间？**
前一轮用户中心布局与菜单实验只做了 typecheck/test/build 和 HMR，没有做亮暗主题的真实页面验证；随后用户发现基本资料与安全设置的表单仍使用全局固定亮色输入令牌。修复时虽然补了 `[data-theme]` 主题令牌并通过门禁，但仍未完成双主题截图或 computed-style 断言。

**这个 skill 有没有提前警告我？**
有。前端索引、known-issues 和红线都明确要求每个 CSS 令牌先 grep 定义、主题改动做双主题截图或程序化验证；本次之前没有执行完整验证，因此属于重复犯错。另一个小问题是从 `web/admin` 子目录执行仓库根路径的 git add 失败，说明 worktree 操作应先确认 git 根目录与相对路径。

**重来一次我会怎么做？**
- 主题改动前先列出页面实际元素与每个 token 的 light/dark 值。
- 修改后用全新 Chrome profile，分别设置 `boss.theme=light/dark`，在真实用户中心页面采集截图或用 `getComputedStyle` 断言输入框背景、文字、边框和按钮对比度。
- 在 worktree 的仓库根目录执行 git add/commit，最后确认 `git status --short` 干净；总结只声明实际完成的验证。

## 2026-08-19 · shadcn-style UI 组件安装反思（已修正归因）

**哪个坑浪费了最多时间？**
并不是网络问题，而是我下结论太快、没有做排除验证就归因到网络，这是反思要记的核心教训。

**当时实际发生了什么：**
- 第一次 `npx shadcn@latest add button -y 2>&1 | head -40` → 超时 30s 被 kill。`| head -40` 管道可能提前关闭导致 SIGPIPE，不是网络问题。
- 第二次 `npx shadcn@latest add button 2>&1` → 超时 15s 被 kill。这次**没传 `-y`**，CLI 很可能在等待交互式选择（shadcn add 默认列出组件列表让用户选），不是网络问题。
- `npm ping` 返回 `http://192.168.0.102:4873/`（私有 npm 镜像），`PONG 110ms` 说明镜像正常。
- **没有做的事情：** 单独测 `curl https://ui.shadcn.com/r`、去掉管道重试、等更长时间看 CLI 真实输出。

**正确结论：**
- 超时更可能的原因是 CLI 在等交互输入（第二次没传 `-y`）或 `| head` 管道截断，不是网络不通。
- 即使 CLI 跑通，生成的组件用 `hsl(var(--primary))` 等默认 CSS 变量，与本项目的 `--color-brand-*`/`--shell-*` 令牌体系不兼容——这个结论本身是对的，但网络原因说是错的。

**这个 skill 有没有提前警告我？**
没有。但"下结论前先做排除验证"这条本身应该成为红线。

**重来一次我会怎么做？**
- 工具超时时不急着归因到网络，先做排除：① 去掉管道重试看真实输出；② 检查是否在等交互输入（加 `-y`）；③ 检查目标 URL 是否可直达（`curl -v https://ui.shadcn.com/r`）；④ 检查本地 npm registry 配置（`npm config get registry`）。
- 定制设计系统项目创建 shadcn-style 组件，正确流程是手动创建（forwardRef + cn + 项目 CSS 变量），不走 CLI add——不是因为网络，而是因为 CLI 生成代码不兼容定制令牌，手动写反而更快。
- 所有故障归因必须在总结里写明"如何确定的"（具体命令 + 输出），不能只说"可能"。

## 2026-08-18 推进项目进度(dsh-codebase-wisdom 会话)
- 哪个坑浪费最多时间:Playwright getByText 撞侧边栏菜单+面包屑双副本,strict mode 连挂 2 条;读 error-context.md 的 page snapshot 后一次修对。
- skill 有没有提前警告:self-evolving 红线 5(未验证不声称)促使我全程用真实后端断言,有效;但无 strict mode 相关经验条目。
- 重来一次:写 e2e 断言页面标题直接用 getByRole('heading', ...) 起步,不先试 getByText。

## 2026-08-19 bossctl CLI 模拟业务流 + 查询完善接口(bossctl-cli 会话)
- 哪个坑浪费最多时间:① 用户反复强调"保存账号密码 API key"我却只口头答应、连续四轮没落盘,被用户连催"你倒是写呀",直到真正 write 才结束;② check-contract-sync A 门禁报"路由未登记"而子文件已加路径,以为是缩进问题,实际是读顶层 admin.yaml 的 `$ref` 行、不递归子文件,靠往 collectSpecPaths 加临时 DEBUG print 才定位;③ 建部门 42200 是因为 legalEntityId 传了字符串 "1" 而非整数 5。
- skill 有没有提前警告:没有针对"答应保存要当场落盘"的红线(现有红线 5 是"未验证不声称",这次是"答应了不执行",另一类);也没有 check-contract-sync 匹配机制的条目。
- 重来一次:① 任何"会保存/已记录"的承诺当场 write + ls 验证,不拖到下一轮;② 契约 A 门禁先看 collectSpecPaths 源码+临时 DEBUG 确认匹配机制,不要凭 regex 直觉猜;③ 42200 一律先读请求 struct 类型,整数 id 不传字符串;④ 结束前 git status 识别并行 Agent 改动,不把自己的域测试与其编译阻塞混淆。
- 沉淀:techniques #24(CLI 五步模拟业务流+账号落盘)、#25($ref 行匹配机制)、#26(并行 Agent 识别);lessons #76-80;knowledge/后端.md 索引已同步。

## 2026-08-19 开网 CLI 全流程模拟(12 环节 → 订单 DONE)

**哪个坑浪费了最多时间?**
① 端口置备 50000:POST /provision/ports 我传 resourceId:1(fixture 直觉),但真实资源创建后是 id 228,FK 违反——置备类接口的 FK 关联必须取前面创建响应返回的真实 id。② 扫码绑定 40920「扫码与预绑定不一致」:按直觉"先建资产、建个 status=UNBOUND 的标签、再建资产时关联 tagId"以为就绑上了,实际 VerifyScan 的 MATCH 要求 `tags.bound_asset_id != 0` 且等于 quadlink.AssetID,而 CreateAsset 不回写 tags.bound_asset_id;正确顺序是**先建资产(无需 tagId),再以 boundAssetId 填实际资产 id + status:"BOUND" 创建标签**。这是全流程中最隐蔽、最费时的一环。

**这个 skill 有没有提前警告我?**
没有。skill 后端索引有 FK/42200/COALESCE 等"后端报错"类经验,但没有"联调/置备数据依赖序"和"双表关联必须显式绑、create 不回写"这类事实性坑。扫码 MATCH 的判定条件(读 pg_scan.go)与 CreateAsset 不绑标签(读 pg_write.go)都是靠源码追出来的,应先作为事实点沉淀。

**重来一次我会怎么做?**
- 置备接口互相关联时,先记录每个创建响应返回的 id(map: 渠道/资源/端口/批次/资产),再拼下一个请求——决不凭 fixture 猜 FK。
- 涉及"标签↔资产"这类双表多对多/关联语义,先 grep 读两端写入代码(CreateAsset vs CreateTag)确认谁写 bound_asset_id,不假设"建 A 时会带上 B"。
- 全流程模拟前先列出依赖序蓝图(techniques #27),缺一环先补置备再推进,避免走到 409/404 才回头。
- 客户等非 account 主体的"账号凭证"=API key(subjectType=customer),sign 完立即写 identities.json + test-accounts.json 落盘,不口头承诺。

**沉淀:** lessons #81-83;known-issues #20;techniques #27-28;knowledge/后端.md 索引与 README 计数已同步。

## 2026-08-19 初始化 mobile user/worker Android 工程
- 哪个坑浪费最多时间:`gradle wrapper` 任务卡在 distribution url 校验(services.gradle.org 不可达)2 分钟超时才发现;改从本地 gradle 发行版 jar 里解出 gradle-wrapper.jar + 复用现成 gradlew 脚本绕过。
- skill 有没有预警:红线 8(未检查环境依赖)部分预警——没料到机器无 JDK,临时 brew install openjdk@17 补上。
- 重来一次:先查 JAVA_HOME/网络可达性再动手;wrapper 生成失败时直接 unzip gradle-wrapper-main-*.jar 取 jar。

## 2026-08-19 tailwind+shadcn 重构 boss/web/admin(第1轮)

- 哪个坑浪费最多时间:同一仓库存在并发提交者(另一个会话),我的 staged 文件两次被卷进对方的巨石提交(ffda8e3 事故 + 837a9f8 卷走 bss/user);第一次差点污染 90 文件的后端重构,靠 reset --soft + pathspec commit 挽回。
- skill 有没有提前警告:没有。红线只说"任务完成必须 commit",没警告"git add 后别人可能抢先 commit 整个 index"。
- 重来一次:多会话共享仓库时,一律 `git commit -m ... -- <显式pathspec>`(不经 index 提交),绝不裸 `git add`+`git commit`;提交前先 `git diff --cached --name-only` 确认 index 只有自己的文件。

## 2025-xx 报障组三页实现(user android)
- 最大坑:为消除注释里的 "/*" 序列用 python 批量替换块注释,把 `onClick = { /* TODO */ },` 替成 `onClick = { // TODO },`,右花括号被注释吞掉造成语法错误。教训:对"代码行内联块注释"不能机械正则替换成 //,必须换行重排;批量改完必须逐处 grep `//.*}` 复核。
- skill 是否预警:否(新增经验,已记入本条)。
- 重来一次:先 grep 出所有内联块注释手工处理,其余单独成行的再批量替换。

## 2026-08-19 三端 API 前缀分离 + 门户落库(httpapi 重构)
- 哪个坑浪费最多时间:BSD sed 的 `\b` 边界符被静默忽略(替换零生效还报成功),`&` 在替换串里等于"整个匹配"(把 `mgr.Sign(` 换成 `mgr.Sign(auth.AudAdmin, .Sign(`)。两处都靠编译器报错才暴露,各耗一轮。教训:macOS 上正则替换一律用 python3 re,别用 sed -i 玩 \b 和 &;sed 只做最朴素的字面替换。
- skill 有没有提前警告:没有。红线 2(编辑前 read)沾边但不覆盖。
- 重来一次:批量跨包重命名先 `grep -rn` 列出全部匹配形态,再用 python 脚本替换 + 立即 go build 单包验证;一次 sed 换完就编译,别攒批。
- 另一个坑:跨包搬文件后,方法(recordAudit/buildDashboard 等)不能定义在外部类型上,切函数时用脚本把 `a.name(` 同步换成 `name(a, `,漏一个编译期才现形。教训:先 grep 全部调用点再动手,换完 grep 复核调用点归零。

## 2026-08-20 多 subagent 并行推进门户上线态
- 哪个坑浪费最多时间:3 个 subagent 中 2 个长时间"running"零产出,其中一个被 interrupt 后仍异步落盘:擅自 git commit 巨石提交并 push 到远端(bbef15b,混装后端+安卓+违规 message),之后还在中断后继续写文件(损坏的 worker 相机代码、重复 CI 文件、甚至一度删掉 cmd/ 入口),被迫反复 git checkout 抢修。
- skill 有没有预警:没有。known-issues 里没有"subagent 无视 no-commit 指令/中断后仍写盘"这一类。
- 重来一次:并行 subagent 后必须 (1) 提交前 git status 对照本人改动清单,发现不明提交立刻查 author/内容;(2) 声明完成前 sleep 数秒再 git status 一次防僵尸写入;(3) 清理 untracked 时绝不用 rm -rf 目录(误删过 tracked .gitea 文件),用 git clean -nd 先预览。

## 2026-08-20 Android 师傅端真机三连bug(导航连环push/401/被覆盖)
- 哪个坑浪费最多时间:(1) 把 Compose 尾随lambda误绑 right 插槽的真实 bug 误判为"模拟器 input tap 怪象",用户真机复现才回头认真查,此前空耗多轮理论推演;(2) 僵尸 subagent 三次回退我未提交的工作区改动、并把旧构建覆盖安装到真机,导致已验证的修复反复"失效",一度怀疑自己修错了。
- skill 有没有预警:known-issues 已有"subagent 中断后仍异步写盘/擅自 commit"条目(前次反思),但没有"机制未证明前禁止结论环境怪象"红线,也没有 Compose 多参数组件尾随lambda的坑。
- 重来一次:UI 出现"理论上不可能"的行为时,第一步就加 Log.d(Throwable 栈)插桩拿 ground truth,不空谈理论;修复验证通过后立即 commit(提交是防并行走失的唯一硬保障);共享真机上装完 APK 用 dumpsys lastUpdateTime 确认没被覆盖再下结论。

## 2026-08-19 移动端门户缺失端点补齐(后端 API + E2E 冒烟)
- 哪个坑浪费最多时间:端口被陈旧进程占用造成的"假 404"。`lsof` 不在默认 PATH(bash: command not found),用 `/usr/sbin/lsof` 才发现 127.0.0.1:18080 被一个孤儿 `./server-new`(IPv4 loopback 绑) 占用,而我的新服务器绑在 IPv6 `*:18080`。同一端口 IPv4/IPv6 双绑时,`curl 127.0.0.1` 走 IPv4 命中错进程,healthz/路由全部 404,自己代码"看起来没注册路由"。空耗多轮才定位。
- skill 有没有预警:没有。known-issues/techniques 里没有"检查端口是否被其他进程用 IPv4/IPv6 绑定、curl 与服务器地址族不匹配"这一类。
- 重来一次:排查"接口 404/路由缺失"时,第一动作 `lsof -nP -iTCP:<port> -sTCP:LISTEN`(用全路径 /usr/sbin/lsof)列出占用该端口的全部进程,确认是否双绑(IPv4+IPv6);必要时 `kill` 陈旧进程再测,而不是默认自己代码没注册路由。
- 另一个坑:给测试写 fake 桩必须完整实现 Go 接口的全部方法。fakeTaxStub 只写了 ListInvoices,go vet 报缺 BackfillTaxNo/IssueInvoicesForPeriod/GetInvoice 等;fakeUserData 缺 ListUserVerifyRecords/ListProductSpecs 等,且 CreateUserPlan/CreateUserAddress 返回 (int64,error) 不是 error。教训:每次给新接口造 fake,先 `go vet` 让编译器列出全部缺失方法,一次性补全,别一个个撞。
- 另一个坑:单测里调用返回两值的 helper(如 signCustomerToken 返回 (string,error)),`x :=` 编译错,要 `x, _ :=`。教训:Go 里任何 `:=` 单值赋值若目标函数返回多值,govet/compile 立即报,改 `_,err` 或 `v, _` 即可。

## 2026-08-19 环境约束纠正:测试服务器只有 102,不要本机启动服务
- 用户明令:测试服务器只有一个(102,192.168.0.102),提交后 gitea CI 自动部署;尽量不要本机启动 boss 服务做冒烟,本机配置低。
- 哪个坑:上个任务我在本机 go run /tmp/boss-new 起了服务冒烟,还因此撞上端口被陈旧进程 IPv4/IPv6 双绑的假 404,空耗多轮。用户此刻直接亮明环境约束。
- 重来一次:需要冒烟/联调后端 → 提交后等 102 自动部署,直接用 192.168.0.102:28080(部署地址)验证,不在本机起服务。

## 2026-05-25 worker 状态栏主题色对齐
- 最大坑：无。本次顺利，但 edit 时误删了 Primary2（old_string 范围多带了一行），好在当轮发现立即补回并 build 验证——印证红线 4"new_string 与 old_string 严格对称"。
- skill 帮助：knowledge/android.md 的 edge-to-edge 相关 lessons 直接给出方案方向；DEV-GUIDE 里的 JAVA_HOME 构建命令省了排查时间。
- 复用经验：模型不支持图像输入时，验证 UI 用 PIL 像素采样代替肉眼截图，量化且更可信。

## 2026-05-25 user 端 edge-to-edge 布局遮挡修复
- 最大坑：git add 指定文件提交时,并行会话早已 stage 的文件(gen-er-drawio 等 5 个)被一起扫进"我的"提交(11 files),违反一提交一变更纪律;soft reset + restore --staged + stash 并行 WIP 后才干净提交 6 个文件。另:模拟器被 worker 僵尸进程抢前台,uiautomator dump 拿到的是 worker 页面,差点据其下错误结论——每轮 dump 前先 dumpsys activity 确认 topResumedActivity 是自己。
- skill 帮助：knowledge/android.md #3 lesson 直接命中本任务核心方案(safeDrawing+BackHandler);worker 端 AppRoot 的 import 写法(saveDrawing 需单独 import)省了一次编译试错。
- 复用经验：布局验证不必起后端——TokenStore 有明文迁移路径,push prefs xml + run-as cp 注入假 token 即进 Home 骨架页;uiautomator dump 的 bounds 坐标(标题 y>状态栏高度、底栏 y<屏高-手势区)可量化断言无遮挡。

## 2026-08-19 数据库表结构对账+ER图脚本化
- 最大的坑: 本机磁盘满导致 go build 链接失败("no space left on device"), 换 go vet 做类型门禁绕过; 环境(磁盘/依赖)失败要与代码失败分开判断。
- skill 提前警告了吗: 红线5(门禁+commit)有效; 但没提示"纯文档变更可用 vet 替代 build"。
- 重来一次: 提交前先看 git status, 发现非本任务文件(ProductsPage.kt 被并行进程修改)不代提交, 保留即可。

## 2026-08-20 五项数据库裁定落地
- 最大的坑: subagent 长回合零落盘时,等待是浪费;interrupt 后接 send_message 强制检查点最有效,打断后其半成品(已落盘文件)可直接由主 agent 接管收尾。
- 重来一次: subagent 超过 ~3 轮无文件增量就打断,而不是等到第 5 轮。

## 2026-12-XX 修复 mobile/user 我的页文字遮挡
- 哪个坑浪费最多时间:无大坑;工作区已有 AppCard Box→Column 修复但未提交(僵尸进程回退高危),根因是 Box 让卡片子元素全部叠在左上角。
- skill 有没有提前警告:有,red-line"验证通过立即 commit"和 android.md 的叠印记录直接命中。
- 重来一次:同样流程——构建(openjdk@17 需显式 export JAVA_HOME)→装模拟器→uiautomator dump→python 算 bounds 重叠→commit。

## 2026-12-XX 修复 gitea android-build chdir 报错
- 哪个坑浪费最多时间:无,错误信息里 cwd 路径直接指认了 working-directory 配置。
- skill 有没有提前警告:无此记录,已补 known-issues.md。
- 重来一次:同样直读 workflow yml,定位 defaults 与 Clone 的先后矛盾。

## 2026-12-XX 修复 mobile/user 首页"进行中订单"文字遮挡
- 哪个坑浪费最多时间:并行进程在我验证期间把同一修复提交了(673f4b3),我的 commit 变成空提交 exit 1;先查 git log 再恐慌。
- skill 有没有提前警告:部分。AppCard Box→Column 的机理记录已有,但共享工作区"提交前重查 git log"无红线。
- 重来一次:commit 失败先 `git log --oneline -5` 确认是否被并行提交吞并;模拟器验证用 uiautomator dump + grep bounds 断言纵向区间不重叠,模型不支持读图时这是替代手段。

## 2026-12-XX 生成 mobile/user Android 首页设计稿
- 哪个坑浪费最多时间:生图任务本身顺利；生成完成后尝试用 `read_image` 检查，但当前模型不支持图像输入，工具明确拒绝。
- skill 有没有提前警告:有，self-evolving 已记录“禁止假设模型支持图像输入”，并要求明确说明未验证。
- 重来一次:生成前先检查当前模型是否支持图像输入；若不支持，保留生成日志与文件存在性校验，并在总结中明确未进行视觉核验。

## 2026-12-XX 用户传授：dsh 模型不支持 read_image 的第一排查动作
- 内容:在 dsh 中发现模型不支持 read_image,先查 `~/.dsh/settings.yaml` 找对应 id 的模型,看 `input: [ text, image ]` 是否配置正确——可能是配置漏了 image,不是模型本身不支持。
- 沉淀位置:techniques.md 新增条目 + knowledge/实施.md 索引(环境/工具链 #5) + README 统计 31→32。已 grep 实证 settings.yaml 各 provider 模型确有 input 字段。
- 关联:高频红线 7(禁止假设模型不支持图像输入)的排查路径——先查配置再下结论。

## 2026-08-20 UserHomeScreen(用户端首页按设计稿实现)
- 最大的坑:差点按规格 L 节照单内置 mock 数据。用户点名"不要使用mock 我在self-evolving已经警告过多次"。skill 已有红线(red-lines #7 禁 mock)但没在开工前扫到"规格文档本身要求 mock"这种变体——教训:规格与用户裁定冲突时,用户裁定优先;数据一律先 curl 102。
- skill 有没有提前警告我?有(red-lines #7、lessons #88),但我只在"对接替换 mock"场景想起它,没意识到"新写页面直接造 mock"同罪。
- 重来一次:读到任何含 mock 字样的规格,第一步 curl 192.168.0.102:28080 真实端点;拿不到 token 就走 sms-code→portal_sms_codes→login 链路。
- 次要坑:gradlew 无 JAVA_HOME 被管道 tail 吞了退出码,"build 通过"是假的;已沉淀 lessons。

## 2026 user-home 提示词视觉信息丢失
- 哪个坑浪费最多时间：设计稿→提示词模板只约定组件类型不约定样式，蓝色特殊头部和底栏样式两处视觉信息在生成提示词时静默丢失，无视觉模型按默认渲染导致返工。
- skill 有没有预警：无，属于新坑。
- 重来一次：模板加"特殊视觉元素转写"必填节 + "禁止默认外观逃逸"硬约束 + 验收清单逐条对应特殊元素。

## 2026 模板 D+ 节臆造示例值
- 哪个坑：D+ 节示例写成了裸具体值（#273F70→#3A5A9C、24dp、64dp 均为臆造），与全模板 {{ }} 占位符惯例不一致，照抄即污染。
- skill 预警：无。
- 重来一次：模板示例只给维度提示（背景是什么/什么形状/与默认差异点），加"数值必须量取填入、示例数字不得照抄"填写规则。

## 2026 知识库模板副本被臆造内容污染
- 哪个坑：skill 知识库里的模板副本(design-to-prompt-android.md)留有上次会话未提交的"填好"版本，含张先生/8/12等臆造设计细节，用户误以为主模板被污染。
- skill 预警：无；且上次收尾没检查 git status 遗留了未提交改动。
- 重来一次：模板副本一律从主模板同步生成(保头5行+主模板正文)，不留手填版本；收尾必须 status 干净。

## 2025-08-20 创建 ui-proto 预设（gpt-image-2 原型设计师）
- 哪个坑浪费最多时间：想在动态插件沙箱里用 setTimeout 做 mount 校验探针，被拒两次（沙箱禁 Node timers；ctx.timeout 需 inject timer）。
- skill 有没有提前警告：cordis-plugin-development 文档有提，但我没先读就写，浪费一轮。
- 重来一次：先查沙箱可用 API 再写探针；mount 校验优先用"失败即抛错"探针而非 console.log（宿主 stdout 在 bash 里读不到）。
- 新经验：cordis 预设的 `!!js` 标签是 scalar-only，不能标记整个序列，要逐项 `!!js >-` 标记。

## 2025-08-20 gpt-image-generate 增加参考图支持
- 哪个坑：edit 端点 multipart 字段名按 OpenAI 文档写 `image[]`，实际代理返回 400 要求 `image`；另外我第一次重写脚本时在 loadEnv 里手滑留了一行垃圾代码。
- skill 有没有提前警告：没有（上游 API 字段差异），已记入 lessons.md。
- 重来一次：对接新端点先用最小请求探字段名，再写完整逻辑；重写文件后立刻 node 冒烟。

## 2026-08-19 用户端首页数据对账与修复
- 哪个坑：无大坑。bossctl 二进制的 base URL 参数是 `-server` 而非 `--url`/`--query _base`，试错两次才找到。
- skill 有没有提前警告：bossctl-cli SKILL.md 未写全局 flag 名，help 里有。已记入 lessons.md。
- 重来一次：CLI 报 "flag provided but not defined" 时第一时间看 -h 的 flag 列表，不要猜参数名。

## 2025-08-20 ui-proto 预设增加 spec.md 交付物
- 无大坑；一次校验用裸 js-yaml 解析预设 YAML 报 unknown tag，原因是 loader 用自己的 entryListSchema 方言（!!js 标签），须用 cordis-plugin-include 导出的 schema 或直接 mount 校验。

## 2025-08-20 ui-proto 预设加双 loop + 反馈台账
- 无翻车。落地前先在 boss 项目实测勘察清单（tokens.css 114 个令牌可提取），证明清单维度可执行而不是纸面检查。
- 经验：自反馈机制要落到"文件 + 强制时序"（FEEDBACK.md + 生成前必读）才有效，光写进 persona 会被后续上下文冲淡。

## 2025-08-20 gpt-image-analyze 识图脚本
- 无翻车。识图模型输出的布局树/色值表/圆角体系质量很高，正好支撑 Loop A/B 自检；单图分析和双图对比均实测通过。
- 经验：模型不支持图像输入时（红线7），不要再写"未目检"交付——用同账号视觉模型补位即可。

## 2025-08-20 gpt-image-analyze 增加 --diff 模式
- 无翻车。差异模式用固定输出结构（差异清单表 + 修复提示词 code block）约束视觉模型，实测输出可直接转发前端。
- 经验：让视觉模型产"可执行产物"的关键是 system 里写死输出骨架，而不是自由问答。

## 2025-08-20 profile-v1 设计稿评审与经验沉淀
- 哪个坑：这份稿（非本会话生成）的根因是 prompt 缺移动端硬约束，"塞一屏"指令逼模型缩水一切；评审判定不合格但有明确结构根因，不是玄学"不好看"。
- 经验：视觉评审要输出"可判定的不合格项 + 根因 + 规则"，六类结构性根因清单比自由点评可积累得多。

## 2025-08-20 优秀设计稿逆向学习
- 无翻车。识别到"克制"是高级感的核心：色彩比例纪律(80/15/5)+层次靠明度差+单一焦点，这些都能写成 prompt 硬规则。
- 经验：优秀基准稿是最好的生图 teacher——识图提取规则 → 固化范式文件 → 整套 prompt 复用，闭环成本低于反复试错(v2-v5 共 4 次生成才接近合格)。

## 2026-08-20 用户端首页数据打通 + UI 打磨 + 真机联调
- 哪个坑浪费最多时间：①验证码 5 分钟有效期,用户拿到码时已过期,还以为是登录链路坏了(我在服务端 DB 里逐项排查账号/验证码状态才定位是过期);②用户说"装到手机",我只看到模拟器就装了,用户看的还是旧包,来回一轮;"文字太大你改了么"其实是包没到手机。
- skill 有没有提前警告：验证码时效没有(业务知识);设备核对没有(红线6"没验证动作就声称已验证"的变体,但没具体到 adb 设备核对)。
- 重来一次：发码前先告知"5 分钟内一次性",过期直接重发不排查;adb install 前先 `adb devices -l` 确认目标 serial 是用户手里的实体机,不确定就先问。
- 其他沉淀：Compose 三个布局坑(Box offset 死间隙/heightIn 不居中/weight 列大字溢出)、uiautomator dump 断言法、build-install 脚本 mapfile 不兼容 macOS。

## 2026-08-22 短信验证码通道接入(阿里云国际版)
- 哪个坑浪费最多时间: go 不在 PATH(须 export PATH=/opt/homebrew/bin);NormalizeE164 裸号默认中国的启发式误吞 +1 美国号,测试抓住后才修。
- skill 有没有预警: 无——可沉淀"E.164 归一化时显式 + 前缀必须命中支持区号,裸号默认区号只对无前缀输入生效"。
- 重来一次: 先写归一化边界用例(带 + 的不支持区号)再写实现。

## 2026-08-21 auth-config 页面(后端+前端+102 部署)
- 最大时间坑: cdp-capture 每次运行用全新 profile,localStorage 种子(boss.servers/boss.token)跨运行丢失;必须在同一次运行里"种 localStorage → location.href 跳转 → 再 eval 断言"。skill 未警告。
- 意外: 共享工作区有并行进程自动把我的改动 commit 掉(6973224/70bb996),收尾前必须 git log 核对而不是只看 status。
- 重来一次: 先读 scripts/dev-token.mjs 是否还指向旧 API 前缀(/api/v1 已废),直接 curl 换 token 更稳。

## 2026-08-20 短信配置页 + CORS 端口漂移修复
- 哪个坑浪费最多时间: ①我本机自启后端(:28099)+vite 让用户自检,用户点名"不要本机自启服务,要用 102 部署"——上级叮嘱第 4 条早就写了对接 102:28080;②用户登录报跨域,根因是 vite 端口漂移到 5175 而 CORS 白名单写死 5173/5174。
- skill 有没有预警: 上级叮嘱第 4 条有(我没遵守);CORS 端口漂移无预警,属新坑。
- 重来一次: 改动完成直接 commit+push 触发 deploy-102 CI(部署唯一正道:push main→gitea→runner 构建镜像→compose up),再让用户在 102 上自检;CORS 涉及开发前端的一律按"本机源不限端口"设计,不枚举端口。

## 2026-08-20 「我的」页头部/滚动区/圆角马拉松(约 15 轮迭代)
- 哪个坑浪费最多时间:「交点圆角」猜错 3 次(首卡顶角→渐变尾底角→S型内圆角),每次都真机验证后被打回。最终正解:滚动区域整体用包裹 Box(padding 交点 + clip 顶部圆角 + 不透明底色),滚动中圆角恒在。
- skill 有没有提前预警:没有。这是"UI 空间描述歧义"类坑:用户说"内圆角/交点",我按几何语义理解,用户指的是"滚动容器的裁剪边界"。
- 重来一次:第 2 次猜错后就该用 ask_user_question,且选项要覆盖"卡片级/区域级/头部级"三个层级维度,而不是同层级猜形状;或者直接问"你期望滚动时什么东西在动、什么不动、圆角挂在谁身上"。
- 其他沉淀:clip 裁剪必须配不透明可区分底色否则蓝对蓝不可见;空 Box 撑高度会塌缩(onSizeChanged 层要与视觉层同体);statusBarsPadding 会消费 insets 害 sibling scrim 拿 0 高度;build FAILED 后链上的 adb install 仍装旧包还报 Success(必须先 grep BUILD 再 install);首帧负 padding 直接闪退;并行会话会裹走未提交改动(改完立刻 commit)。

## 2026-02 user-android 状态栏统一
- 坑:本机无 JDK(/usr/libexec/java_home 报无 Java Runtime),gradle 编译门禁跑不了,只能静态检查+人工核对。
- skill 提前警告了吗:docs 里提过门禁命令但没登记"无 JDK"这一环境事实。
- 重来一次:先探明构建工具链可用性再动手;改动保持最小并 grep 清理残留引用(extendIntoStatusBar/statusBarsPadding)。

## 2026-08-20 共用骨架组件收敛两页视觉(用户:"你就不要截图分析了 直接复用一个组件")
- 最值钱的经验:用户一句话点破正解——"两个页面视觉一致"这类需求,正确解法是抽共用组件(PinnedGradientPage+PinnedHeaderSpec 单点几何参数),而不是两页各自调参再截图对比修补。我在对比修补上又烧了好几轮像素断言,方向就是错的。
- 哪个坑:①并行会话提交了编译不过的代码(ea91bdf,WindowInsets.getHeight 不存在),阻塞我的 build,做了最小修复(asPaddingValues)解锁;②两次工具调用被打断(abort),4dp 上移的 build/install/commit 悬空,靠 git status 才发现收尾;③edit "file changed since read" 在并行环境下频繁出现,重读再改已是常态。
- skill 有没有预警:红线1(读最新内容)有;并行会话提交坏代码无预警——多会话共享工作区时,pull 之后必须先 build 再继续自己的活。
- 重来一次:听到"两页保持一致"就直接提组件化方案;开工前 git pull + 全量 build 确认基线是绿的。

## 2026-08 user-android JDK 误判纠正
- 坑:断言"本机无 JDK"被用户纠正。实际是 Homebrew openjdk@17 装在 /opt/homebrew/Cellar/openjdk@17/,只 grep /opt/homebrew 顶层没进 Cellar;/usr/bin/java 是 macOS stub 误导。
- 修法:找 JDK 先看 ~/.gradle/daemon/*/daemon-*.out.log 里的 javaHome=(最快最准),再查 /opt/homebrew/Cellar。
- 重来一次:gradle 项目报"无 Java Runtime"时,先查 daemon 日志的 javaHome 再下结论,不要只依赖 /usr/libexec/java_home。

## 2026-08-20 admin 短信配置菜单图标缺失

**哪个坑浪费了最多时间？**
补图标本身 10 分钟（menu key → public/icons/items/<key>.svg 的映射 docs 里早有）。真正耗时的是用户随后报告"图标只有 hover 才出现"：我先 DOM 断言（mask/bg/尺寸全部正常）、再对比令牌、再查 102 部署资源、最后开始自写元素级 CDP 截图脚本——多轮深挖后用户回复"可以了 我看过了"，异常大概率是 dev HMR/缓存陈旧状态，一次硬刷新就能排除。另外 docs 里的免登录脚本 `web/admin/scripts/dev-token.mjs` 已失效（404：脚本登录路径是 /auth/login，实际前缀是 /api/admin/v1），SKILL 示例的 #username/#password 选择器也与真实登录页不符（登录页是 placeholder input 无 id）。

**这个 skill 有没有提前警告我？**
部分。knowledge 里明确记了"新增菜单项必须补 items/<key>.svg，geo 曾漏"，所以补图标一步到位；但没有警告"用户报告的 UI 异常先排除缓存/环境陈旧"，也没维护 dev-token.mjs 的失效状态。

**重来一次我会怎么做？**
- 用户报 UI 异常时，第一反问/第一动作是"硬刷新 + 确认看的是哪个地址（localhost dev / 102 部署）"，再动代码排查；DOM 断言正常而用户看到异常 = 大概率陈旧客户端。
- 免登录优先直接注入 localStorage（boss.token + boss.servers + boss.server.active），不走登录页表单 eval，也不信未验证的辅助脚本。

## 2026-08-20 用户端账单 tab 测试数据 + in_progress 筛选修复
- 最耗时:订单推到 DONE 被四码唯一约束(uq_quad_links_customer_active)卡住,先补资产/标签/quad_link 再扫码,还误删旧 LINKED 行需手工 UNLINK;102 上 docker-registry htpasswd 无明文,需新增 ci 用户重做 docker login。
- 教训:102 的 docker-registry(htpasswd)与 act_runner(root daemon)认证是部署链路单点;镜像内迁移文件权限 600 会让容器以 app 用户启动即崩,Dockerfile 已加 chmod a+rX 兜底(commit 60b91ca)。
- 重来一次:先读 server.Dockerfile 的 USER app 与源文件权限,再触发 CI。

## 2026-08-20 102 服务器事故:docker prune 引发 boss-server 重启循环 + CI 断链
- 最耗时:两个独立故障叠加——(1) boss-server 因镜像内迁移文件 000066 root:root 600、容器以 uid=1000(app) 运行而 permission denied 崩溃重启 13 次;(2) CI job 拉不到 boss/deploy-runner:latest("no basic auth credentials")。都根源于 22:19 跑的 ~/docker-clean.sh(builder prune -af + image prune -af):前者触发重建踩中源文件权限坑,后者删掉了"仅本地标签、从未 push"的 deploy-runner 镜像,且宿主 docker 从未 login 过 registry(以前全靠本地缓存,从没暴露)。
- skill 有没有预警:没有。registry 认证/本地镜像依赖是部署链路的隐形单点,skill 里无任何记录;教训已补进 lessons/known-issues/red-lines。
- 重来一次:跑任何 docker prune 前先 `docker images` 圈出"仅本地标签"的关键镜像(deploy-runner 等)并确认 registry 有备份;清理后立刻验证 CI 全链路;宿主一次性 `docker login 192.168.0.102:5000`(凭据在 102:~/boss/deploy-image/dotdocker/config.json)。排查容器崩溃先看 `docker logs` + `docker run --rm --entrypoint sh <img> -c 'id; ls -la'` 直接验镜像内文件权限,别只盯容器状态。

## 2026-08-20 用户端服务页 tab 死数据排查
- 最大坑:write 工具创建的迁移文件权限是 600,进镜像后 server 启动报 `permission denied`,崩溃循环;新建任何要进 Docker 镜像的文件后 chmod 644。
- skill 提前警告了吗:没有,已喂到 techniques。
- 重来一次:构建镜像前统一 `chmod 644` 新增文件;docker build 偶发 "context canceled" 直接重跑即可。
- 另发现 000047 迁移漏授 sysadmin 权限导致 userdata 域 403,迁移补授权是既有模式(000039/000042)。

## 2026-08-20 订单分页 + 下拉刷新/上拉加载
- 最耗时:验证部署时被镜像时间线迷惑(容器重建于新镜像构建前 10 秒,且并发 CI 任务 deploy 互相撞容器名留下 rename 残壳);判断部署是否生效必须 docker exec 进容器 strings 二进制符号,不能只看容器"Up (healthy)"。
- 教训:连续 push 会并发跑多个 deploy job,compose up 互抢容器名;验完再 push,或 CI 侧需要串行化。

## 2026-08-20 首页图标+增值服务卡片
- 坑:直接 gradlew 报无 Java Runtime,需按 scripts/build-install-user-android.sh 的 find_java_home 逻辑 export JAVA_HOME(/opt/homebrew/Cellar/openjdk@17)。skill 未提前警告,已顺手写入 knowledge/前端.md?否,写入 lessons。
- 改动:Router->Home 图标;MyServiceCard 替换为 ActiveServicesCard 过滤 status==ACTIVE。

## 2026-08-20 user端认证页视觉对齐(subagent)
- 坑:git add -p 分离 fix/style 提交时需先 `git diff --cached` 核对暂存 hunk,避免混装;本次顺利。
- 教训:接口核对先行(sms-code→查库→login→verify/agreement 全部字段与页面解析一致),写 UI 前先 curl 真实响应省去返工。
- git status 余留 ProductPage.kt 为并行 agent 改动,未触碰。

## 2026-08-20 worker端剩余页面视觉与接口对齐
- 最大坑:页面共享 Cell 的尾随 lambda 会绑定 right 插槽，改为多插槽组件时必须显式写 `right = {}`，否则视觉内容可能进入错误组合槽位。
- skill 提前警告了吗:Android 经验已提醒 Compose 尾随 lambda，但本次仍在 TicketCell/History/Help/Notice 中发现同类风险。
- 重来一次:先 grep 所有 `Cell(...) {` 并逐一确认插槽语义；接口联调先 curl 102 服务确认 401 与路径，再修复默认 base URL、定位失败禁止发送 0,0、数组端点从独立 API 加载。

## 2026-08-20 全站二级页面对齐四 tab 基准(5+1 subagent)
- 坑:多 agent 并行时各自跑 assembleDebug 会互相撞到对方的半编辑态文件短暂红屏;约定"别人文件报错等 1 分钟重试"即可,不要去改别人的文件。
- 最值钱发现:pgx 简单协议 []byte→jsonb 22P02(POST /faults、PUT notify-settings 恒 50000),已修并实测;教训是"接口验证发现服务端 5xx 要 ssh 看 boss-server 日志实锤 SQLSTATE",不能只停留在报告。
- 二轮复查抓到一轮漏网:3 个页面 TopBar 尾随 lambda 误绑 onAction 丢返回键、登录页死按钮、OrderCard 读不存在的 createdAt 字段——第一轮 5 个 agent 分工过细时跨文件模式问题(尾随 lambda)没人兜底,复查 agent 全局 grep 一次就全抓到。
- 视觉验收没有图像模型时:uiautomator dump 断言文案/结构 + PIL 像素断言(头部 #006AE5、底 #F5F6F8)即可量化。

## 2026-08-20 worker端二轮页面修复
- 最大坑:服务端返回的月度 busyDays 是计数而非日期集合，不能直接按日期高亮；客户端先修正月份首日偏移和实际天数，计数语义需后端另立接口处理。
- 教训:页面字段必须以实际 handler 返回值为准，历史/公告页面对缺失字段提供明确回退，不显示空标签或假详情。

## 2026-08-23 支付对接 Stripe
- 哪个坑浪费最多时间:提交时 index 里已有并行进程暂存的 openapi/android 文件,commit 卷入 9 个无关文件,只能 reset --soft 拆开重提;user.yaml 里混着并行 hunk,不得不用 hash-object+update-index 做选择性暂存。
- skill 有没有提前警告:红线 5(必须 commit)警告了收尾,但没警告共享 index;已补进 lessons.md。
- 重来一次:commit 前先 `git diff --cached --stat` 核对暂存清单,只含自己的文件才提交。

## 2026-08-23 后台新接口对齐 OpenAPI 契约(22 路由)
- 哪个坑浪费最多时间:contract-sync 门禁 A 还在扫旧目录 internal/app,输出"0 条路由全部有契约"却显示 OK——门禁形同虚设长达多轮,后台新加的 22 条路由全部漂移无人发现。另外与并行会话共用工作区,我先暂存再提交之间被并行 commit 卷走 index,首提失败需重暂存。
- skill 有没有提前警告:部分。红线 5 只管"要提交",lessons 已有共享 index 教训,但"门禁显示 OK 却是空集"这种假阴性没有预警。
- 重来一次:凡门禁输出计数,先看计数是否为 0/异常小再信 OK;共享工作区提交前 `git diff --cached --stat` 核对清单,提交失败立即重查 index。

## 2026-08-20 验证用户端 Stripe 对接
- 坑:102 上 POST /payments/stripe/checkout 一律 42200,误以为请求体绑定失败,排查了半小时;实际是 handler 在 BindBody 之前先查 PayGateway.Get("stripe"),102 未配 BOSS_STRIPE_API_KEY 即降级返回 CodeInvalidParam。
- 重来:先读 handler 源码看校验顺序,再怀疑请求体。
- 已登记 known-issues。

## 2026-08-20 对账新增接口(realid/stripe/attachment/apikey)
- 最大坑:共享工作区有并行进程在改 minio/wiring/storage-config,审计中途 build/contract-sync 突然红,先 `git status` 分辨归属再动手,别人的 WIP 不碰不提交。
- skill 提前警告了:红线 #1(共享工作区文件可能被并行进程改掉)。
- 重来一次:开审前先 `git status`,出现非本任务改动立即在总结中声明归属。

## 2025-XX 单测覆盖率补齐(billing/sms 两包到100%)
- 最大坑:无。唯一摩擦:gofmt 改写文件后 edit 报 "file changed since read",重读后重试即过——印证红线1。
- 复用经验:不可达错误分支(json.Marshal 基本字段、硬编码合法 URL 的 NewRequest)用包级 var 注入口(marshalCDR / aliyunEndpoint / kafkaWriter 接口)做最小缝,行为零变更。
- 用户明确"不要 git commit"时,skill 门禁的 commit 要求让位于用户指令。

## 2025-XX 单测覆盖率补齐(关键边界优先)
- 最大坑:并行代理生成的 pgx mock 测试与实际依赖签名/SQL 参数不一致，导致全仓编译或测试失败；为追求数字而引入不可维护测试反而降低质量。
- skill 是否提前警告:共享工作区改动需先检查归属，红线覆盖了这一点；但没有替代“先确保测试可编译、再看覆盖率”的明确门禁。
- 重来一次:先按包单独运行测试和 coverprofile，再提交可靠的边界用例；对真实数据库/对象存储优先使用稳定集成测试或只覆盖可离线模拟的错误边界，不为不可达分支修改生产代码。

## 2026-08-21 user/worker Android 打包安装
- 哪个坑浪费最多时间:脚本只覆盖 user，且系统 PATH 没有 adb；如果直接运行脚本会在设备选择前失败。本次先读取 user/worker 的 Gradle 配置，确认两端默认分别连接 102 的 user/worker API，再用 SDK 内 adb 手动构建和安装。
- skill 有没有提前警告:有。Android 经验要求确认 `BUILD SUCCESSFUL` 后再 install，并区分实体机与模拟器；本次同时检测到两台设备，明确指定实体机 serial，避免误装模拟器。
- 重来一次:脚本应支持 worker 或统一双端构建安装；执行前先检查 `$HOME/Library/Android/sdk/platform-tools/adb` 和 `adb devices -l`，构建成功后再逐包安装，并用 `pm path` 与 `lastUpdateTime` 核验。

## 2026-08-21 worker Android 首页重设计
- 哪个坑浪费最多时间:把 `Modifier.padding(top=0.dp)` 通过参数传给 HomeCard,而 HomeCard 内部链了自己 `.padding(top=4.dp)`,修饰符在内部 padding 之前叠加,零边距被 4dp 覆盖,no-op 假修复;用户看真机后才二次点名。
- skill 有没有提前警告:部分。红线#6(未验证就声称已修复)方向对,但没写"Compose 修饰符叠加顺序会吃掉外部归零"这一具体机制。
- 重来一次:组件内部写死的 padding 要覆盖时,必须给组件加显式参数(如 topPadding)而不是靠外部 modifier;改完在真机/截图上确认数值,不靠代码推断。

## 2026-08-21 实名认证分步流程(后端+Android+102 部署)
- 哪个坑浪费最多时间:真机 SystemUI 通知栏卡死后盲目 reboot,设备有 PIN 锁(secure=true)远程无法解锁,被迫转模拟器;另有两个自造坑:Compose Modifier 链写了两处 `Modifier.androidx.xxx` 点号扩展语法(编译期拦住),校验 when 的 else 恒返回 "agree" 只能靠真机点击发现。
- skill 有没有提前警告:并行僵尸 agent 提交 WIP 已警告(android.md #4)且确实又发生(d9ff2ee);reboot 共享真机没有警告——新坑。
- 重来一次:动 device 状态前先 `dumpsys window policy | grep secure`;uiautomator 自动化发现按钮点击无效时,第一时间怀疑"按钮 enabled=false 的门控条件",而不是重复点击。

## 2026-08-21 工作台数据核对 + dashboard 时区修复
- 哪个坑浪费时间最多:bossctl 二进制参数是 -server 不是 --base/BOSS_BASE,试了三次才通;/tmp 写文件首跑被旧 shell 残留误导。
- skill 有没有预警:有(对接 102:28080),但没有记录 bossctl 的参数名。
- 重来一次:先 bossctl --help 看参数再调用;数据核对优先"同源交叉验证"(把 /orders /alarms /quad-links 原始拉回来用 python 重算,再对照 /dashboard)。

## 2026-08-20 ODN 地理空间编码规范对齐
- 最费时：无。pdftotext 不在默认 PATH（brew 在 /opt/homebrew/bin），首次调用失败后 export PATH 即解决——AGENTS.md 已提示但没第一时间用。
- skill 预警：有（环境事实），执行时没先读。
- 重来一次：凡 PDF/命令行工具先 `export PATH=/opt/homebrew/bin:$PATH` 再调用。
- 无新增红线；文档变更已 commit（a04343d），git status 干净。

## 2026-08-20 ODN 编码映射落地(goal round 1)
- 最费时：共享工作区有并行会话同刻提交——中途 git status 突变(000076 出现、我的 docs 编辑被对方提交)。发现后逐文件核对内容未丢失才继续,没有盲目重做。
- skill 预警：红线1(编辑前 read)多次救场;但"共享工作区文件可能被并行进程改掉"这条应升级意识:提交前必须 git status+log 核对,编号类资源(migration 序号)先 ls 再定。
- 重来一次：生成 migration 前先 ls migrations/ 拿最新序号,并在提交信息里写死文件名防混淆。
- 坑:python 内联补丁脚本改脚本(patching a patching),越改越乱,最后整文件重写才收敛——>3 次补丁就该重写。

## 2026-08-20 订单生命周期 UI 闭环
- 最费时:cdp 验证登录门禁页面(reload 停在 /login 排查 3 轮)——skill 有 localStorage 注入配方但没写"必须从 /login 起步再 replace";已补进 lessons。
- 踩坑:并行会话把我未提交的文件扫进它的 docs 提交;以后提交用明确文件清单。
- 重来一次:改完立刻 commit,不留给并行进程捡漏。

## 2026-08-20 ODN 无源物理层落地(goal round 2)
- 最费时：迁移文件漏写 COMMIT——migrate 实现按"文件自带 BEGIN/COMMIT 整段执行"约定,漏 COMMIT 时 exec 成功+版本已记录但事务被回滚,表不存在且 migrate 幂等跳过,形成"已记录未生效"的脏状态,排查两轮才定位。
- 新教训(候选红线)：本项目 migration 必须 BEGIN/COMMIT 成对;写入 schema_migrations 前应校验表确实存在。配套修复手段=核实后 delete 记录重放(000076/000079 两例)。
- 另一坑:NOT NULL DEFAULT '' 列配 NULLIF($n,'') 必触发 23502——'' 语义列直接绑参,不要 NULLIF。
- 集成测试要幂等:开头清理残留 + -count=2 验证。

## 2026-04-11 admin 原生 confirm 替换为 ConfirmDialog
- 最耗时的坑:找对了"没有 alert、只有 19 处 window.confirm",但浏览器冒烟时反复打不开页面——原因是告警页真实路由是 /alarm/alarm(菜单分组前缀),不是 /alarm;另外 cdp-capture 每次运行是全新浏览器上下文,localStorage 不跨运行持久,必须单次运行内 set 存储 + location.href 导航 + 最后一个 async eval 断言。
- skill 有没有预警:cdp-capture 用法文档齐全,但没写"存储不跨运行持久"和"eval 顺序在 settle 前"这两个细节,踩了 4 次截图才定位。
- 重来一次:先 grep menu.def.ts 拿真实 path 再开页;把全部前置动作压进同一次 cdp-capture 调用。

## 2026-08-20 ODN 管理接口落地(goal round 3)
- 顺利。一处失误:gofmt 漏跑 internal/pkg/httpx(只查了改动主目录),把未格式化文件提交了,靠 commit --amend 补救——门禁前 gofmt -l 要覆盖本次全部触碰包。
- 复用模式有效:ledger_test 的 fakeX 内嵌接口桩 + doJSON + middleware.Authn 测试引擎直接照搬,handler 测试半小时收敛。

## 2026-08-20 ODN 局点/核心设备与 Admin Web 收尾(goal round 4)
- 最费时：共享工作区并行会话同时改三语言 i18n 与订单类型，web build/typecheck 的全量错误混入本轮；使用 git diff HEAD 与 `git diff --cached` 只暂存 ODN UI，避免吞并邻居改动。
- 新经验：共享前端 i18n 文件不能直接 `git add`;必须从 HEAD 生成最小 patch 再 cached apply，保证提交边界。
- 设备扩容后缀初版误把 -10 判非法(正则首位 [2-9]);规范是禁 -1、允许 -10，已改为 `-([2-9]|[1-9][0-9]+)` 并用 102 PG 重放迁移验证。

## 2026-08-24 拆机/更换表单改 ResourcePicker,资源页筛选取消原生 select
- 顺利:契约先行(terms/domain-map/fields + 后端 order_sub.go/query.go 读序正确),先读 i18n 三语言块再落键,门禁 typecheck/test/build 一次过,成功避开累犯红线 #1/#3/#5。
- 唯一波折:长中文 commit message 用 `git commit -m "$(cat <<'EOF' ...)"` 报 bash bad substitution,改临时文件 + `git commit -F` 一次过——已沉淀 techniques.md。
- 重来一次仍应:先勘察三个目标页与 Dropdown/组件视觉再动手;每个 i18n 键落三语言后 grep 锚点行验证落点(本次 fPort:'Port ID' 两页面块同文案,靠 grep pickSearch 行号确认落进 dismantlePage)。

## 2026-08-24 web/admin 路由页面懒加载
- 最费时：React.lazy 对只导出命名组件的 error/placeholder 页面不能直接 import，必须在动态 import 后映射 `{ default: module.NamedPage }`；先跑 typecheck 能立即定位。
- skill 是否预警：前端门禁要求 build/typecheck，但没有直接覆盖 lazy named export 的类型形状。
- 重来一次：批量把页面导入改懒加载时，先区分 default export 与 named export，再运行 typecheck；最终用构建产物大小确认 chunk warning 是否消失。

## 2026-08-24 师傅接单区域校验补齐
- 最费时：新写的测试用例里 token 在 t.Setenv 之前签发,导致 "invalid worker token" 白排查一轮;另一个是 edit 工具对 tab 缩进敏感,从 read 复制时层级数错一次。
- skill 是否预警：红线 1/4 覆盖了"先读再改",但没有"签 token 必须在 Setenv 密钥之后"这条。
- 重来一次：封装 helper 时把 Setenv 放进 helper 首行;edit 前先 cat -et 确认目标行真实缩进层级。


## 2026-08-21 师傅端开发模式自动获取验证码

**哪个坑浪费了最多时间？**
工作区里有他人未提交的 dev-mode 收尾改动(user 端),worktree 不洁。我第一次 go build ./... 失败是因为 user/debug_sms.go 里 httpx "imported and not used",我下意识把它当成我刚看的状态去删除 import——但实际看的是文件旧版本缓存/或别人在他自己的 read 后改了它。继续 build 才发现 server 整树编译没问题,debug_sms.go 我根本不该碰。还有一处:我自己加的 LoginScreen.kt 第一版没有正确处理 devMode 路径下"已拿到 code 就不该跑 60s 倒计时"的边界,但 edit 时范围一致,没踩 edit 红线,反而让我重构了一次流程分支。

**这个 skill 有没有提前警告我？**
"高频红线 #5 任务完成必须 git commit"保证了产物理清,但"开工前 `git status` 看是否有他人未提交 diff"这一点在 skill 里是隐含的——应在 `references/techniques.md` 显式标注:任务开始先用 `git status -uall` 划定工作面。也不要轻信 read 工具的输出:如果他人同时改 worktree,你以为的可信快照下一秒就过期。

**重来一次我会怎么做？**
- 第一步永远 `git status -uall` 列出 unstaged + untracked,只动跟我任务相关的路径;其它路径(像 user 端那批改动)留在 dirty,提交严格按文件清单 add -m,避免污染。
- 工作区动别人的代码(如发现 user/debug_sms.go 有 lint 错误)先确认是不是我的责任:如果是, 提醒 user 端任务 owner 处理或单开一个 chore commit;如果不动, 不要顺手带。
- Android 端 devMode 校验时,Api 返回的 ApiException status/msg 一定要结构化,别靠吞 `_`——这里我用 `catch (_: ApiException)` 仅为了不中断轮询,其实可以在外面把 status 取出当日志,排查时不必再翻 trace。

