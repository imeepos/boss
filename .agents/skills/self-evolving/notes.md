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
