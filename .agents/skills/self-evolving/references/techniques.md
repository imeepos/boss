# Techniques

<!-- 排查技巧、工具命令、调试手法。格式：什么场景 → 怎么用。 -->

## 无 Playwright 时给 Web 页面（含需登录页）截图

场景 → 前端改动要可视化验证，环境无 playwright/puppeteer，但 macOS 有系统 Chrome。
怎么用 → 跑 `scripts/cdp-capture.mjs`（Node>=22，零依赖，自带 Chrome 启停）：

```bash
node scripts/cdp-capture.mjs http://localhost:5173/login /tmp/a.png \
  --eval '/* 可选:页面加载后执行的 JS,如填表登录 */' --settle 2500
```

## 浏览器 console + 网络请求采集(截图的"为什么"层)

场景 → 页面白屏/行为不符/接口报错,截图只能看到"结果",定位"原因"必须看浏览器 console 报错和实际发出的网络请求(用户经验传授:这是调试分析最有价值的信息)。
怎么用 → 同一脚本加 `--logs`,console 输出、未捕获异常、浏览器级报错、全部 HTTP 请求(方法/URL/状态码,失败请求附响应体前 2000 字符)落成 JSON:

```bash
node scripts/cdp-capture.mjs http://localhost:5173/ /tmp/page.png \
  --logs /tmp/page-logs.json --eval '/* 登录等操作 */'
```

要点:
- CDP 事件不是应答,要在 WebSocket 消息分发里专门接 `msg.method` 分支(`Runtime.consoleAPICalled` / `Runtime.exceptionThrown` / `Log.entryAdded` / `Network.requestWillBeSent|responseReceived|loadingFailed`)。
- 先 `Runtime.enable` + `Log.enable` + `Network.enable` 才有事件流;requestId 是请求三元组的关联键。
- 失败请求(status>=400 或 errorText)用 `Network.getResponseBody {requestId}` 抓响应体——排 4xx/5xx 最直接的证据;成功请求只记方法/URL/状态,避免日志爆炸。
- `console.warn` 在 `Runtime.consoleAPICalled` 里的 type 是 `warning` 不是 `warn`,过滤时要同时收。

要点：
- Chrome 二进制：`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
- 启动参数：`--headless=new --remote-debugging-port=0 --user-data-dir=$(mktemp -d)`；端口 0 让系统分配，从 stderr 的 `DevTools listening on ws://...` 读实际端口，避免端口冲突与状态泄漏。
- CDP 通道：`fetch http://127.0.0.1:<port>/json` 拿 page target 的 `webSocketDebuggerUrl`，全局 `WebSocket` 直连，自增 id 收发 JSON。
- 关键 CDP 方法：`Page.enable` / `Page.navigate` / `Runtime.evaluate`（`awaitPromise+returnByValue`）/ `Emulation.setDeviceMetricsOverride` / `Page.captureScreenshot`（返回 base64）。

## React 受控输入框的自动化填充

场景 → CDP/控制台里给 React 表单的 input 赋值。
怎么用 → 用 native setter 绕过 React 的 value 追踪：

```js
const set = (el, v) => {
  Object.getOwnPropertyDescriptor(Object.getPrototypeOf(el), 'value').set.call(el, v)
  el.dispatchEvent(new Event('input', { bubbles: true }))
}
```

## 描边色写死的 SVG 图标按主题换色

场景 → 第三方/原型 SVG（stroke 写死灰色）要在亮/暗/激活态显示不同颜色。
怎么用 → CSS 遮罩：`background: currentColor; mask: url("icon.svg") center/contain no-repeat`（-webkit-mask 同理），颜色完全由 `color` 控制；适合图标放 public 目录、URL 动态拼接的场景。

## boss 项目 admin 前端冒烟账号与后端

场景 → web/admin 要登录冒烟（本次截图验证靠它）。
怎么用 → 种子账号 `admin / admin123`（scripts/devseed/main.go 写入，角色 sysadmin）；已部署后端 `http://192.168.0.102:28080`；本地联调 `cd web/admin && BOSS_API_TARGET=http://127.0.0.1:18080 pnpm dev`（默认代理 102 部署）。

## boss 后端 API 冒烟测试路径（无 psql 环境）

场景 → 检查 admin 页面对应后端接口（如 /base/geo → /api/v1/geo/*）。
怎么用 → 前端 5173 的 vite 代理已把 /api 转发到 102 部署，直接对 `http://localhost:5173/api/v1/...` 发 curl 即测真实链路：
1. `POST /api/v1/auth/login {"username":"admin","password":"admin123"}` 拿 token；
2. 带 `Authorization: Bearer <token>` 逐接口增删改查；
3. 403 时先查远端库迁移是否漏跑（DSN: host=192.168.0.102 port=25432 user=boss password=boss dbname=boss）；
4. geo 域"删"=软删除（PUT .../active {"active":false}），测试数据留停用态即可，勿物理删。

## 手工补跑漏掉的数据库迁移（无 psql/migrate CLI）

场景 → 远端库 schema_migrations 落后于 migrations/ 目录（如 403 定位到漏跑）。
怎么用 → /tmp 临时 go 程序 + pgx：`db.Exec(ctx, string(读入的.sql整文件))`（迁移含 BEGIN/COMMIT 可整文件执行），成功后 `INSERT INTO schema_migrations(version) VALUES ('0000XX_名称')`；version 列是 TEXT。

## boss 超管引导与部署 env 管理速查

场景 → 生产环境初始化超级管理员 / 真实部署注入密钥。
怎么用 → 超管：`BOSS_ADMIN_USERNAME`（默认 admin）+ `BOSS_ADMIN_PASSWORD`（空=跳过）经启动引导幂等创建 sysadmin（internal/domain/user/bootstrap.go，EnsureSuperAdmin）；已存在静默跳过。部署：`cd deployments && cp app.env.example app.env` 填 JWT_SECRET 与超管密码，compose 用 `env_file: [app.env]`，密钥不进 yml；验证用 `docker compose -f docker-compose.102.app.yml config | grep BOSS_`。本地裸跑：`cp .env.example .env && set -a; source .env; set +a`（项目无 godotenv，config.Load 只读进程环境）。

## 迁移编号与 CI 密钥注入速查

场景 → 新增 SQL 迁移 / gitea workflow 部署需要 env 密钥文件。
怎么用 → 迁移编号:`ls migrations/*.up.sql | tail -5` 取真实最大号 +1;别信截断列表,也别与代码注释引用的编号冲突。CI 密钥:workflow 在 compose up 前加 Prepare 步骤,`cp app.env.example app.env` 后用 `sed` 注入 `${{ secrets.BOSS_JWT_SECRET }}`;secret 为空即 `exit 1` 并在日志里写明去仓库 Settings→Secrets 配置。部署是否生效:`docker ps` 看容器镜像 tag 是否等于 GITHUB_SHA,或直接看 actions 步骤状态。

## boss admin 页面主题化改造样板(antd Pro 惯例 × 品牌令牌)

场景 → 把写死亮色的 admin 页面重构成亮暗双主题 + antd Pro 布局(以 /base/geo 为样板)。
怎么用 → 布局骨架:页头(20px 标题+12px 描述)→ antd 风格页签(文字+金色 2px 底指示线)→ geo-card 卡片(工具栏+表格+右对齐合计)。主题三原则:
1. 零写死色值,一律 `--shell-*`(card/heading/content-text/side-border)与 `--color-*` 令牌;
2. 页级新色(表头底/行 hover/输入框)在页面 css 里加 `:root[data-theme='light'/'dark']` 自定义 `--geo-*` 块,值照抄 docs/admin/design-spec.md §4.2;
3. 主操作按钮复用 `--shell-fab-bg`(亮=品牌蓝/暗=品牌金),天然主题自适应。
新建/编辑走右侧 Drawer(src/components/Drawer.tsx 通用件),表单用两列 grid+label+必填 `*`;状态列用语义 Tag(启用绿/停用灰)。i18n 加 key 记得 4 处同步(types.ts + 3 locale)。
样板文件:web/admin/src/pages/base/geo/{index.tsx,CountryPanel.tsx,CountryForm.tsx,CountryDetail.tsx,SubdivisionPanel.tsx,geo.css}。

## CDP 程序化主题/交互验证(模型不能读图时的替代)

场景 → 页面改造后需验证双主题渲染与交互,但模型不支持 read_image。
怎么用 → cdp-capture 一条 eval 链完成"登录→设主题→跳页→断言":

```bash
node .agents/skills/self-evolving/scripts/cdp-capture.mjs \
  http://localhost:5173/login /tmp/x.png --logs /tmp/x.json \
  --eval '(async()=>{ /* native setter 填表登录 */ await sleep(1500);
    localStorage.setItem("boss.theme","dark"); location.href="/base/geo" })()' \
  --eval '(async()=>{ await sleep(2500);
    const cs=(s,p)=>{const el=document.querySelector(s);return el?getComputedStyle(el)[p]:"(none)"};
    /* 点开抽屉/页签等交互 */ await sleep(600);
    console.log("VERIFY:"+JSON.stringify({theme:document.documentElement.dataset.theme, card:cs(".geo-card","backgroundColor"), ...}))
  })()'
```

断言点:卡片/表头/文字/主按钮的计算色,`.length` 行数,`!!document.querySelector(".dvr-panel")` 弹层开合;结果靠 `console.log("VERIFY:...")` 进 --logs 再 grep;失败请求与 console error 同份日志可直接查。

## 判断"本次部署真生效"的第三个探针

场景 → 无 docker 访问权,healthz/迁移版本都可能来自旧容器。
怎么用 → 挑一个"只有新代码+新 env 才会产生"的可观测副作用当探针,如本次的"bootstrap 新建 admin 账号":/tmp 临时 go 程序 + pgx 直连查 accounts WHERE username='admin' 是否出现/created_at 是否刷新。比 curl 更硬,比镜像 tag 比对更省事。

- 场景:自研组件要对齐 antd/Pro 规范。手法:`curl -sL https://raw.githubusercontent.com/ant-design/ant-design/master/components/<name>/index.zh-CN.md` 拿一手 API/设计说明(何时使用/默认值/Token),博客教程只做线索。
- 场景:组件在某一主题下样式不对但无报错。手法:`grep -rn -- "--令牌名" src/theme/ src/styles.css src/pages/*/[页面].css` 确认令牌真的定义过——CSS 变量缺失静默走 fallback,是最典型的双主题静默失败。
- 场景:开工写 UI 组件前。手法:先 grep references/lessons.md + red-lines.md 的相关关键词(select/图标/主题/分页),旧教训按场景检索,不凭记忆。
- 场景:开工数据库/基础设施类任务前。手法:同样先 grep lessons.md 关键词(psql/DSN/PG/迁移)——本轮"临时 go + pgx 直连远端库"在 lesson 28 早有正解,没回看等于重新发明;另 grep configs/*.yaml 找现成 DSN,再问用户"库在哪",不要直接 brew 装本地 PG。
- 场景:验证 SQL 迁移 down/up 双向可执行且不污染共享库。手法:/tmp 临时 go 程序 + pgx,脚本剥掉迁移文件内的 `BEGIN;`/`COMMIT;` 行,外层 conn.Begin() 依次 Exec(down 内容→断言行数→up 内容→断言行数)后 Commit;全程一个事务,中途任何错 Rollback 零副作用(2026-08-19 PH PSGC 43769 节点回环验证实例)。
- 场景:核对"必须真实"的行政区划类数据。手法:三层校验——① 结构完整性(孤儿父节点=0、level=父+1、全节点有 en 名);② 总数对官方口径(PSA PSGC 2025-07:18 大区/82 省/150 市/1493 镇/42011 Barangay);③ 抽查易错点(宿务市 80 Barangay、BARMM 下 5 省、2024 新设 NIR 含 Bacolod)。
- 场景:批量删数据/设计测试自清理,需要 FK 依赖拓扑序。命令:`SELECT conrelid::regclass || ' -> ' || confrelid::regclass FROM pg_constraint WHERE contype='f' AND confrelid IN ('addresses'::regclass, ...);` 逐层下钻(引用表→再被引用表),孙→子→父排删除序;自引用表(parent_id)按 level 自底向上分批删。
- 场景:本机没装 postgresql 但装了 libpq(brew),psql 不在 PATH。命令:`export PATH=/opt/homebrew/opt/libpq/bin:$PATH && PGPASSWORD=boss psql -h 192.168.0.102 -p 25432 -U boss -d boss ...`;go 也同理 /opt/homebrew/bin。
- 场景:验证测试自清理是否真闭环(单看 PASS 不够)。方法:BOSS_PG_TEST_DSN 指真库 `go test -run TestE2E... -v`(grep cleanup 看尽力而为日志),跑完 psql 按 `path::text ~ '^(e2e|w8|an[0-9]|g[0-9])'` 等前缀计数,残留必须为 0。
- 场景:定位"垃圾数据从哪来"。方法:按 `split_part(path::text,'.',1)` 分组 + created_at 日期,前缀对 grep 测试文件(`grep -rn "E2E测试市\|分析楼栋\|g%d" internal --include=*_test.go`),十分钟内锁定污染源。
