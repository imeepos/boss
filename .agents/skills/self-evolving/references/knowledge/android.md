# Android 经验索引

> mobile/worker 与 mobile/user 两个 Android 端（Kotlin + Jetpack Compose）。开工前先扫一遍标题。

---

## 1. Compose 组件 / 导航

| # | 来源 | 要点 |
|---|------|------|
| 1 | known-issues（2026-08-20） | Compose 尾随 lambda 绑渲染插槽 → 组合期执行导航：多参数组件末位是 `@Composable` 插槽（如 Cell 的 right）时，裸尾随 lambda 必绑错位且编译器不报错；动作一律 `onClick = {...}` 显式命名传 |
| 2 | techniques（2026-08-20） | 导航幽灵跳转插桩法：push/switchTab 临时加 `Log.d(tag, msg, Throwable())`，栈顶在 `Recomposer.performRecompose` = 渲染期执行，在 `ClickableNode.handleUpEvent` = 真实点击；定位后删插桩再提交 |
| 3 | lessons（2026-08-19） | edge-to-edge 下顶栏被状态栏遮挡致返回键点不到：根布局加 `windowInsetsPadding(WindowInsets.safeDrawing)`；自维护导航栈必须配 `BackHandler(enabled = stack.size > 1) { pop() }`，否则系统返回直接退出 App |

## 2. 网络 / 登录 / 数据

| # | 来源 | 要点 |
|---|------|------|
| 1 | known-issues（2026-08-20） | 登录成功但全部请求 401：mock 平铺 `{token}` vs 真实 `{code,data:{token}}` 信封，顶层取到空串 setToken 存空；先 run-as 读 shared_prefs 验 token 落盘，再 curl 同端点对照 |
| 2 | lessons（2026-08-20） | 对接真实后端替换 mock 时，先 curl 关键端点核对响应结构（信封/字段层级）再写解析 |
| 3 | lessons（2026-08-20） | 后端验证码只落库不发短信（未接网关）：curl 触发发码 → 临时 go+pgx 查 `portal_sms_codes` 拿真码（5 分钟有效一次性）；测试师傅账号在 `.agents/skills/bossctl-cli/test-accounts.json` |

## 3. 真机 / adb 验证

| # | 来源 | 要点 |
|---|------|------|
| 1 | techniques（2026-08-20） | 真机 adb 自动化验证：`uiautomator dump` 取 text/bounds 算中心点 → `input tap` → 再 dump 断言标题；返回键 `input keyevent 4`；数据加载稳定后再取坐标（加载中会漂移） |
| 2 | techniques（2026-08-20） | 真机 App 内部状态直查（免抓包）：debug 包 `adb shell run-as <pkg> cat shared_prefs/<prefs>.xml`，空 `<map/>` 即没写过 |
| 3 | lessons（2026-08-20） | 真机与电脑时间对不上先 `adb shell date` 对时区差（本例差 9 小时），再比对 `dumpsys package <pkg> | grep lastUpdateTime` 判断 APK 是否被覆盖安装 |
| 4 | lessons（2026-08-20） | 并行 agent 共享真机：装完 APK 用 lastUpdateTime 确认没被覆盖再下结论 |

## 4. 共享工作区协作（Android 端高发）

| # | 来源 | 要点 |
|---|------|------|
| 1 | red-lines（2026-08-20） | 修复验证通过后立即 git commit——未提交的工作区会被并行僵尸进程 git checkout 回退 |
| 2 | known-issues（2026-08-20 subagent 条目） | 僵尸 subagent 中断后仍写盘/擅自 commit+push/覆盖安装旧 APK，收尾前 sleep 后再 git status 复核 |

## 5. 开工前 grep 关键词

写 Android 代码前，用这些关键词检索 lessons.md + known-issues.md + red-lines.md + techniques.md：

```
Compose, 尾随lambda, onClick, right, 插槽, BackHandler, insets, statusBarsPadding,
导航, push, pop, switchTab, 401, token, data.token, 信封, shared_prefs, run-as,
offset, 死间隙, heightIn, 居中, weight 列溢出, maxLines, bounds 断言, 下拉 DropdownMenu, LangStore,
clip 圆角, 不透明底色, 滚动区裁剪, insets 消费, 负 padding 闪退, BUILD SUCCESSFUL 再 install,
uiautomator, input tap, keyevent, lastUpdateTime, 真机, adb, 验证码, sms-code,
portal_sms_codes, emulator, 10.0.2.2, adb reverse
```

> 拿到移动端设计稿要写 Compose 页面提示词时，先看 [prompt-templates.md](prompt-templates.md)（Android 版模板 + px→dp 换算规则）。

| 5 | lessons（2026-08-20） | 新页面数据一律直连 102 真实服务(192.168.0.102:28080),禁止 mock 样例数据——规格文档写"mock"也不执行(用户多次点名);user 端 token 走 sms-code→查库→login | 
| 6 | lessons（2026-08-20） | gradlew 报无 Java Runtime → export JAVA_HOME=/opt/homebrew/opt/openjdk@17;管道接 tail 会吞退出码 |
| 7 | lessons（2026-08-20） | 模型不能读图时：screencap+PIL 像素断言(渐变色/底栏高度/选中蓝像素数)+uiautomator dump 文案断言,可量化完成视觉验收 |
| 8 | lessons（2026-08-20） | Compose Modifier.offset 只移视觉不缩布局槽,压卡要套整个容器,单卡套 offset 给后续卡留死间隙 |
| 9 | lessons（2026-08-20） | Text 挂 heightIn(min) 当按钮文字不居中:最小高度给外层 Box(contentAlignment=Center) |
| 10 | lessons（2026-08-20） | weight 均分列大字号数值必须按最长内容校验+maxLines=1,32sp 在三列必溢出变形 |
| 11 | techniques（2026-08-20） | uiautomator dump 抓不到渐变头等未暴露语义的 Compose 文本;能抓到的用 bounds 数值断言单行/间距 |
| 12 | techniques（2026-08-20） | 取服务端验证码明文:PG 192.168.0.102:25432 boss/boss,查 portal_sms_codes;5 分钟一次性 |
| 13 | known-issues（2026-08-20） | build-install-user-android.sh mapfile 不兼容 macOS bash3.2,手动 adb install 绕过 |
| 14 | lessons（2026-08-20） | 滚动区整体圆角恒在:包裹 Box(padding 交点+clip 顶角+不透明底色),内部 Column 只滚动;clip 无底色则蓝对蓝不可见 |
| 15 | lessons（2026-08-20） | statusBarsPadding 消费 insets→后绘 sibling 取 statusBars 得 0,提前 asPaddingValues 量取;首帧负 padding 闪退,coerceAtLeast(0) |
| 16 | lessons（2026-08-20） | UI 空间词(内圆角/交点)被打回一次后必须问,选项按卡片级/区域级/头部级分层 |
| 17 | known-issues（2026-08-20） | build FAILED 后 && 链 adb install 装旧包报 Success;先判定 BUILD SUCCESSFUL 再 install |
| 18 | lessons（2026-08-20） | 多页视觉一致=抽共用组件+单点参数对象(PinnedGradientPage/PinnedHeaderSpec),别各页调参对比修补 |
| 19 | lessons（2026-08-20） | 多会话仓库:pull 后先 assemble 验基线绿;别人坏提交最小修复解锁 |
- Compose 修饰符叠加:内部 `.padding(top=4)` 会覆盖外部传入的 `padding(top=0)`;组件内写死的间距要可归零,必须暴露显式参数(2026-08-21 worker 首页 OverviewCard 实锤,见 lessons.md)
| 20 | techniques（2026-08-21） | 开发模式验证码自动回填:后端 `/debug/sms-code?phone=&scene=` 查 portal_sms_codes;Android 侧 `DevModeStore`(SharedPreferences) + `DebugApi` + `devAutoFillSms` 封装;release 包 `BuildConfig.DEBUG=false` 永不生效;102 机配 `BOSS_DEBUG_SMS=1` 环境变量 |
| 21 | lessons（2026-08-21） | build.gradle.kts 默认端口与真实服务不一致时(`8080` vs `28080`),用 `-PbossBaseUrl=...` 覆盖;建 feature 前先 `grep -n "BOSS_BASE_URL" build.gradle.kts` 确认 debug 端口 |
| 22 | lessons（2026-08-21） | dev-mode 降级:后端 BOSS_DEBUG_SMS=0 时 /debug/sms-code 404,Android 侧 `devAutoFillSms` catch Exception 返回 false,不填入验证码也不报错——正确降级行为 |
| 23 | techniques（2026-08-21） | 开发模式开关持久化:SharedPreferences 读写(`boss_user_dev` prefs + `boss_user_dev_mode` key),与 LangStore/TokenStore 同源但不同 prefs 文件,避免 token 被清时连带丢失开发配置 |
| 24 | techniques（2026-08-25） | gradle wrapper 分发版缓存缺 `.ok` 标记时,每次构建都联网 forceFetch 下载失败(SSL read timeout, networkTimeout=10000);直接用 `~/.gradle/wrapper/dists/gradle-8.14.3-bin/<hash>/gradle-8.14.3/bin/gradle` 绕开下载环节;构建退出码禁止经管道 tail 取(吞码假 EXIT=0),重定向日志文件后单独读 |
