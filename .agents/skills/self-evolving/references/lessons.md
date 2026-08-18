# Lessons

<!-- 一条经验一行。格式：当 X 发生时，修复是 Y。skill 没提前警告我。 -->

- 当需要给"需登录的 Web 页面"截图且没有 Playwright 时，修复是系统 Chrome `--headless=new --remote-debugging-port` + Node>=22 全局 WebSocket 裸 CDP（脚本见 scripts/cdp-capture.mjs）。skill 没提前警告我。
- 当 CDP 截图要覆盖 localStorage 驱动的状态（如主题）时，修复是每个状态显式 `localStorage.setItem` 后 `Page.navigate` 重载再拍，或每次运行换全新 `--user-data-dir`；不复用上轮 profile。skill 没提前警告我。
- 当自动化填充 React 受控 input 时，修复是用 `Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set` + `dispatchEvent(new Event('input',{bubbles:true}))`，直接 `el.value=` 不触发 React 状态。skill 没提前警告我。
- 当要给描边色写死的第三方 SVG 图标换色（随 currentColor/主题）时，修复是 CSS `mask: url(...) center/contain no-repeat; background: currentColor`（alpha 遮罩，与原 SVG 色无关），而不是 `<img>`（无法改色）或逐个内联（量大）。skill 没提前警告我。
- 当悬浮按钮要钉在滚动容器可视区角落时，修复是 `position: fixed`（相对视口，配合底栏高度算 bottom），`position: sticky; bottom` + float 的组合对放置位置敏感、容易失效。skill 没提前警告我。
- 当 edit 的 old_string 覆盖到文件末尾时，修复是去掉 old_string/new_string 的结尾换行（文件可能无尾换行导致匹配失败）。skill 没提前警告我。
- 当页面表现异常（白屏/交互失效/数据不对）时，修复是先采浏览器 console 报错 + 网络请求清单再分析，而不是只看截图猜；`cdp-capture.mjs --logs out.json` 一次拿到两者（用户经验传授）。skill 没提前警告我。
- 当枚举切换器（语言/单位/主题）混在一排 ghost 图标按钮里时，修复是"图标按钮 + 自定义下拉浮层（role=listbox + 当前项打勾 + 点击外部收起）"，样式复用现有 token，明暗主题自动适配。skill 没提前警告我。
- 当顶栏要放退出登录时，修复是收进头像下拉（信息头 + 个人设置 + 危险色退出项，antd Pro 惯例），不在顶栏平铺独立退出按钮。skill 没提前警告我。
- 当页面出现第二个及以上下拉浮层时，修复是把浮层骨架（定位/背景/边框/阴影/选项行）写成多选择器共用样式，新菜单只写差异部分。skill 没提前警告我。
- 当侧栏菜单激活项可能滚出视野时，修复是 useEffect 监听 pathname，仅目标在可视区外时 `scrollTo({behavior:'smooth'})` 对齐到视野中央；在视野内则不滚。skill 没提前警告我。
- 当同一文件在本会话已被编辑过、又要做第二轮 edit 时，修复是先 Read 最新内容再拼 old_string，凭上一轮记忆拼会 not found。skill 没提前警告我。
- 当块级容器想占满父级宽度时，修复是不写 `width:100%`（display:block 的 auto 已撑满且 padding 内含），`width:100%`+padding 在无 border-box 重置的项目里必横向溢出。skill 没提前警告我。
- 当滚动容器（overflow-y:auto）意外出现横向滚动条时，修复是量 `scrollWidth > clientWidth` 定位溢出元素，再查 box-sizing/固定宽/白溢出（img、whitespace、flex 最小宽）。skill 没提前警告我。
- 当在深色/彩色表面放 button 时，修复是显式写 color 不指望继承（span 继承、button/input/select 不继承，UA 默认 buttontext 黑），否则深底黑字不可见。skill 没提前警告我。
- 当用户报"某主题下颜色不对"时，修复是同时检查全部主题同位置——本例深浅主题顶栏同为深色，深色主题其实同病只是未被注意。skill 没提前警告我。
- 当模型不支持读图（read_image 报错）时，修复是跳过自动核验继续推进修复，改用 --eval getComputedStyle 做程序化颜色验证，最后总结列出"请人类目测"清单（用户明确指示）。skill 没提前警告我。
