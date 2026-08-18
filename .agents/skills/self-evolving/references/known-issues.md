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
