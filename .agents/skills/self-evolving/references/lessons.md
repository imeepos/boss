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
- 当在新 bash 会话里跑 go/docker 等命令报 command not found 时，修复是先 `export PATH=/opt/homebrew/bin:$PATH`（brew 工具全在此），不要结论"工具没装"。skill 之前未警告。
- 当给域 Service 接口（如 user.Service）追加方法时，修复是同一提交内同步补齐测试桩（fakeUser 等），编译错误清单就是桩清单。skill 没提前警告我。
- 当初始化超管/首个账号时，修复是走「启动引导 + ON CONFLICT DO NOTHING」：密码只从环境变量注入、绝不写进迁移或种子文件、已存在不覆盖（防重启重置密码）。skill 没提前警告我。
- 当部署 compose 需要密钥时，修复是 env_file 管 secrets（app.env 不入库，加 .gitignore），environment 段只留非密默认值；记住 compose 优先级 environment > env_file，要覆盖默认值得两处一起动。skill 没提前警告我。
- 当 sysadmin 角色被菜单门禁拒（403 no permission:menu:x）时，修复是先查远端库 `schema_migrations` 最新版本对比 `ls migrations/*.up.sql`——本项目权限全是 role_permissions 显式行，sysadmin 无隐式全权，迁移漏跑（漏插权限/授权）是首要嫌疑。skill 没提前警告我。
- 当手测 API 收到 42200 参数非法时，修复是先读后端请求 struct 再拼 JSON（如 geo attrs 的 timeZones 是 string[] 而非对象数组），不凭直觉猜字段类型。skill 没提前警告我。
- 当本机没有 psql 却要查/改远端 PG 时，修复是 /tmp 临时 go 程序 + pgx 直连 DSN（configs/config.example.yaml 有现成连接串），迁移文件是纯 SQL 可整文件 Exec。skill 没提前警告我。
- 当新增数据库迁移文件时,修复是先 `ls migrations/*.up.sql | tail -5` 确认真实最大编号——`ls | head` 截断列表曾让我险些撞号 000031(已被 order_no_seq 占用);代码注释里的迁移号(internal/domain/order/pg.go)也要 grep 交叉验证。
- 当 CI 全新 clone 后 compose up 报 env file not found 时,修复是 workflow 里从 example 生成 env 文件、密钥从 gitea repo secret 注入、缺失即 fail fast——被 .gitignore 的文件在无人值守环境必然缺失。
- 当验证 CI 部署结果时,修复是看 actions 日志或比对镜像 tag(GITHUB_SHA),curl healthz 只证明"有容器活着"——部署在 compose up 前失败时旧容器照常应答 ok。
- 当本机无 docker/psql 而要验证 SQL 迁移时,修复是 sqlglot(pip install --user sqlglot)按 postgres 方言 parse 全文件拦语法错;约束语义仍需真实 PG。
- 当 Go 工具链不在 PATH 时,修复是 export PATH=/opt/homebrew/bin:$PATH(AGENTS.md 已声明 brew 在此);go build 失败先查这个再怀疑代码。
