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
