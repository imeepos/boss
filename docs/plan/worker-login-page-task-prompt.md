# 任务提示词：师傅端 Android 登录页 —— 设计对齐与开发收尾

> 整段复制给执行代理执行。执行代理无历史会话上下文，本文件自包含；仓库纪律以根目录 `AGENTS.md` 与 `mobile/worker/android/DEV-GUIDE.md` 为准（两者冲突时以 AGENTS.md 为准）。

## 目标

在 `ymm-001/boss` 仓库的 `mobile/worker/android`（师傅端 Android App）完成登录页的设计对齐与开发收尾：以设计稿与实现规格为视觉与功能基准，核验既有实现、补齐缺口，全链路对接 102 部署环境真实后端，验收全部通过后按仓库协议合并进 main 并完成清理。

## 现状与问题（勘察结论，开工后先复核再动手）

1. 登录页已有完整初版并已接线，**不是从零开发**：
   - 页面与共享组件：`mobile/worker/android/app/src/main/java/com/ymm/boss/worker/ui/` 下 `LoginScreen.kt`（双模式：验证码/密码；渐变品牌区、悬浮白卡、协议勾选、通栏主按钮、入驻入口）与 `AuthForm.kt`（登录表单组件体系）。
   - 已接线：`AppRoot.kt` 的 `Screen.Login`（登录成功进首页、未授权自动回登录、协议页与入驻页跳转）。
   - 多语言：`res/values`、`values-en`、`values-ms` 三语各 20 条 `auth_*` 文案。
   - 接口：`api/WorkerApi.kt` 的 `AuthApi`（发码/登录/登出），统一响应信封与错误文案已有约定。
2. 设计基准为本任务新产出：`mobile/worker/design/worker-login-states-v1.png` 与同名 `.spec.md`（逐项视觉与状态规格）；同源参照根目录 `designs/login-register-states-v2.spec.md` 与 `docs/worker/UI-SPEC.md`。主题色以代码真值 `ui/theme/Color.kt` 为准，与文档冲突时以代码真值为准。
3. 构建现状：main 分支 `assembleDebug` 编译通过（已实测）。本机 gradle wrapper 联网下载分发版会 SSL 超时失败，缓存目录已有解压好的 8.14.3 分发版，构建应绕开 wrapper 下载环节；Java 环境用 `/opt/homebrew/opt/openjdk@17`。
4. 联调环境：102 部署环境 worker 端接口真实可达端口为 `28080`（前缀 `/api/worker/v1`，未鉴权请求返回 401 属预期）；构建配置 debug 默认端口与其不一致，联调前先验证接口可达并按既有机制覆盖（见 `mobile/worker/android/app/build.gradle.kts`）。测试师傅账号：王测试 `13800001001`（APPROVED）；验证码须先触发发码、再从 102 数据库 `portal_sms_codes` 查取（5 分钟一次性，顺序颠倒报"未签发"）。

## 起止边界

- 起点：在**独立 worktree** 中基于最新 main 新建功能分支，先反向同步 main，再动手。
- 终点：全部验收通过、分支合并进 main、分支与 worktree 清理完毕、工作区干净。
- 范围：仅允许改动登录页相关源码与文案（`LoginScreen.kt`、`AuthForm.kt`、三语 `strings.xml` 的 `auth_*` 文案），以及为修复登录缺陷确有必要的最小改动；其余文件一律只读。
- 时限：当天完成合并，不过夜。

## 验收标准（全部满足才完成）

A. 构建：`assembleDebug` 成功；既有单元测试全部通过（与登录无关的既有失败须附证据说明）。
B. 视觉：对照设计稿与 spec 逐维度核验——品牌区渐变、主色、卡片圆角、间距节奏（4 的倍数）、输入行 ≥48dp、主按钮 ≥44dp、辅助文字 ≥12sp 且对比度 ≥4.5:1、占位文字可辨；不得引入稿中不存在的突兀元素。
C. 功能（对照 spec 状态清单逐项）：
   1. 验证码登录全链路：输入手机号 → 获取验证码（倒计时/防重复点击/失败提示）→ 登录成功进首页，后续接口保持已登录状态（重启 App 仍有效）；
   2. 密码登录：密码输入、明文切换、登录与错误提示；
   3. 协议勾选门控：未勾选主按钮禁用；《服务协议》可跳转协议页；
   4. 加载态与防重复提交、空输入与手机号格式校验；
   5. 入驻入口跳入驻页；登录态失效自动回登录页；
   6. 三语文案完整无缺漏，无新增硬编码中文。
D. 规范：单文件 ≤300 行、单函数 ≤60 行；无 emoji 图标；提交信息 `type(scope): subject` 且正文写机理。
E. 提交纪律：按"勘察 → 修复/增强 → 联调"分批，一次提交一个可独立陈述的变更，禁止一锅端；配套测试与文案随主变更同提交。
F. 收尾（硬性）：① 推送到远端 → ② 回主树 ff 合并 → ③ 移除 worktree → ④ 删除本地与远端分支；合并前在 worktree 内反向合并 main 解冲突；ff 失败时回 worktree rebase 后重试，**严禁删除未合并的 worktree**；收尾后确认提交在 main、git 状态干净、分支与 worktree 无残留。

## 明确不做

- 不重新设计视觉：以 `mobile/worker/design/worker-login-states-v1.png` 为基准，不再另起炉灶；稿中光斑/波浪等装饰按 spec 简化，不逐像素复刻。
- 不改共享文件：`Nav.kt / AppRoot.kt / Widgets.kt / Load.kt / Api.kt / WorkerApi.kt / MainActivity.kt / theme/`；确需改动先如实报告、等待确认。
- 不做新功能：不做"忘记密码"（后端无对应能力，师傅账号由平台分配）、不做注册页（入驻页已有）、不做第三方登录、不做深色模式、不新增依赖/SDK。
- 不 mock 数据：联调一律 102 真实环境；接口异常如实报告，不得用假数据冒充验收通过。
- 不动其他模块：`mobile/user`、`docs/worker`（H5）、`web/admin` 等一律不改。

## 执行纪律

- 使用独立 worktree 工作，禁止直接在主分支改动，禁止覆盖他人未提交改动。
- 分批修改、每批测试并 commit；无需逐步请示；遇阻塞（后端不可达、接口异常、账号缺失、权限不足等）如实报告并附证据。
- 完成后报告：改动文件清单、提交列表、验收逐条结果、未验证项、合并提交号与清理确认。
