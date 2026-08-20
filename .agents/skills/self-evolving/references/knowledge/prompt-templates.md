# 提示词模板与指南速查（设计稿转提示词 / Goal 编写）

> 本文件是索引：三份可复用的提示词模板/指南，原文在 `docs/` 下（只增不改，用时以原文为准）。
> 收录日期：2025-12（从 docs/notes/goal-prompt-guide.md、docs/design/design-to-prompt-template*.md 沉淀）。

---

## 1. 设计稿 → 页面布局提示词（Web 通用版）

- 原文：`docs/design/design-to-prompt-template.md`
- 何时用：拿到 Web/H5 视觉设计稿（PNG/Figma 导出/截图），要产出交给编码 AI 生成页面布局的提示词。
- 流程：read_image 量取 → 填 A–N 节模板（骨架树/区域规格/色板 Token/字阶/间距栅格/组件规格/交互态/响应式断点/数据契约/资产/验收清单）→ cdp-capture.mjs 截图对比迭代。
- 核心原则：量化一切（px/色值/字重），禁止"大一点""偏上"这类模糊词；差异反馈写"应为 X 而非 Y"。
- 本项目专用约束块（粘贴到 B 节）在原文 §3：tokens.css 令牌、Dropdown.tsx 禁原生 select、字段名查 docs/contract/fields.md。

## 2. 设计稿 → 页面布局提示词（Android/Compose 版）

- 原文：`docs/design/design-to-prompt-template-android.md`
- 何时用：移动端设计稿 → Compose 页面提示词（mobile/user、mobile/worker）。
- 与 Web 版差异：px→dp/sp 换算（1080px 稿 ÷3，750px 稿 ÷2，提示词禁止裸 px）、无 hover 只有按压涟漪、edge-to-edge insets、material3 组件、真机 adb 验证。
- 关键红线：颜色落 ui/theme/Color.kt + 深色成对值，禁止页面内硬编码 Color(0xFF...)；onClick 显式命名传参；自维护导航栈配 BackHandler；触控目标 ≥48×48dp。
- 验证：./gradlew assembleDebug + adb install + uiautomator dump 断言（区别于 Web 的 cdp-capture）。

## 3. DSH Goal 提示词（objective）编写指南

- 原文：`docs/notes/goal-prompt-guide.md`
- 何时用：建 create_goal 前先判断该不该用（长线、跨轮次、需自动续行才建），以及怎么写 objective。
- 六条硬规则：写成可判定的完成状态不写动作清单；单一目标；完成标准必须含验证动作；写明阻塞条件；说明为什么长线；显式设 max_goal_rounds。
- 模板：【目标】【完成标准】【背景/为什么长线】【约束】【阻塞条件】【轮次上限】，原文 §3 可复制。
- 运行期：complete 需逐条判据有验证证据；blocked 需同一条件持续 ≥3 轮且原因具体；方向变更走 edit 不靠聊天纠正。

---

## 开工前检索

- 要把设计稿变成页面提示词 → 先读原文 §2 量取检查清单再动笔。
- 要建 Goal → 先过原文 §1 判断表 + §4 反例对照。
- 相关关键词：设计稿、提示词、objective、create_goal、dp 换算、量取。
