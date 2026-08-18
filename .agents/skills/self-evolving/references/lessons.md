# Lessons

<!-- 一条经验一行。格式：当 X 发生时，修复是 Y。skill 没提前警告我。 -->

- 当需要给"需登录的 Web 页面"截图且没有 Playwright 时，修复是系统 Chrome `--headless=new --remote-debugging-port` + Node>=22 全局 WebSocket 裸 CDP（脚本见 scripts/cdp-capture.mjs）。skill 没提前警告我。
- 当 CDP 截图要覆盖 localStorage 驱动的状态（如主题）时，修复是每个状态显式 `localStorage.setItem` 后 `Page.navigate` 重载再拍，或每次运行换全新 `--user-data-dir`；不复用上轮 profile。skill 没提前警告我。
- 当自动化填充 React 受控 input 时，修复是用 `Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set` + `dispatchEvent(new Event('input',{bubbles:true}))`，直接 `el.value=` 不触发 React 状态。skill 没提前警告我。
- 当要给描边色写死的第三方 SVG 图标换色（随 currentColor/主题）时，修复是 CSS `mask: url(...) center/contain no-repeat; background: currentColor`（alpha 遮罩，与原 SVG 色无关），而不是 `<img>`（无法改色）或逐个内联（量大）。skill 没提前警告我。
- 当悬浮按钮要钉在滚动容器可视区角落时，修复是 `position: fixed`（相对视口，配合底栏高度算 bottom），`position: sticky; bottom` + float 的组合对放置位置敏感、容易失效。skill 没提前警告我。
- 当 edit 的 old_string 覆盖到文件末尾时，修复是去掉 old_string/new_string 的结尾换行（文件可能无尾换行导致匹配失败）。skill 没提前警告我。
- 当页面表现异常（白屏/交互失效/数据不对）时，修复是先采浏览器 console 报错 + 网络请求清单再分析，而不是只看截图猜；`cdp-capture.mjs --logs out.json` 一次拿到两者（用户经验传授）。skill 没提前警告我。
