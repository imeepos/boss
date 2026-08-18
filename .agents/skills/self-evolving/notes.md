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




