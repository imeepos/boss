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
- 场景:列表筛选需要 URL 可恢复但不能让 URL 变化覆盖即时交互。方法:首次 render 读取 `useQueryState/useQueryInt` 作为 `useState` 初值；事件处理器先更新本地 state，再调用 URL setter；不要在 render 中直接使用 query hook 返回值。
- 场景:验证自定义下拉确实选中。方法:在真实目标应用 DOM 中点击触发器和 `[role=option]`，随后用 `console.log("VERIFY:"+JSON.stringify({label, search:location.search, filteredCount}))` 输出触发器文本、URL 和筛选结果；不要把宿主 GUI 的 DOM 当业务页面验证。
- 场景:给定制设计系统项目（非 shadcn default theme）创建 shadcn-style UI 组件。方法:不走 `npx shadcn@latest add` CLI（生成代码用默认 `hsl(var(--primary))` 等 CSS 变量，与定制令牌不兼容），而是手动创建——参考 shadcn 官方组件的编码模式（forwardRef + cn() + cva variant），tailwind 类名改用项目已有的 `--color-brand-*`/`--shell-*` CSS 变量。标准流程：`src/lib/cn.ts`（已有）→ `src/components/ui/<name>.tsx`（基础组件，forwardRef + cn + 项目令牌）→ `src/components/business/<name>.tsx`（组合业务模式）→ 验证。shadcn 官方源码地址：https://github.com/shadcn-ui/ui/tree/main/apps/www/registry/default/ui。
- 场景:工具超时时排查原因。方法:不急着归因到网络，按顺序做排除：① 去掉管道（`| head`/`| grep`）重试看真实输出；② 检查是否在等交互输入（加 `-y`/`--yes` 或 `yes |`）；③ 检查目标 URL 是否可达（`curl -v --connect-timeout 5 <url>` 看连接耗时和 HTTP 状态码）；④ 检查 registry 配置（`npm config get registry` / `pnpm config get registry`）；⑤ 等足够长的时间确认不是慢而不是挂。

## bossctl CLI 模拟端到端业务流（组织链 + 三场景 + 账号落盘）

场景 → 用 CLI 工具完整模拟一条业务流（如"师傅注册→后台管理员审核→实名认证"），顺带验证接口可用性与数据关联，并让测试账号可复用。
怎么用 → 以 admin(sysadmin) 为唯一起点，全程 bossctl `--api-key` 驱动，五步：

1. **摸清接口**（动手前先查契约）：
   - `bossctl routes` 看全部端点（按模块 grep）；
   - 契约字段定义在 `api/openapi/admin/*.yaml`（如 `AccountInput` 的 roleCode/legalEntityId/deptId/postId）；42200 参数非法时先读请求 struct 再拼 JSON，整数 id 不要传字符串；
   - 权限归属查 `GET /menu-perms`：在 matrix.rows 里搜目标权限码（如 `menu:dispatch -> ['ops','sysadmin','technician']`），据此选建号角色。

2. **组织链数据**（审核人员来源，admin key 依次）：
   - `POST /legal-entities {"code","name"}` 建企业；
   - `POST /departments {"legalEntityId":<int>,"name"}` 建部门（int 传 id，字符串会 42200）；
   - `POST /posts {"deptId":<int>,"code","name","roles":[...]}` 建岗位（roles 全量绑定，未知码 42200）；
   - `POST /accounts {"username","password","realName","roleCode","legalEntityId","deptId","postId"}` 建审核账号（menu:account 仅 sysadmin 可建）；
   - `POST /api-keys {"subjectType":"account","subjectRef":<accountId>,"name"}` 签 API key——plainKey **只在创建时返回一次**，立即记录。

3. **业务侧关联数据**：`POST /worker-groups` 建班组 + `GET /regions` 找区域 id（师傅注册绑定 groupId+regionId）。

4. **三场景验证**（同一流程三种结果）：
   - 正常流：公开端点注册 → 审核员 approve → 生成主档 workerId → 提交实名 → verify PASS；
   - 冲突流：对已 APPROVED 再 approve → 42200（不幂等，二次操作被拒）；
   - 驳回流：注册 → reject 带意见 note（不生成主档）。
   每次操作后 GET 回读确认状态落库（APPROVED/REJECTED/PASS）。

5. **账号落盘 + 收尾**：所有产生的账号/密码/API key 当场写入 `.agents/skills/bossctl-cli/test-accounts.json` 并 `ls` 确认存在（不要只口头答应）；在 SKILL.md 写明存储位置。收尾跑 `go run ./scripts/check-contract-sync` 确认 A/B/C 全过 + `go test ./internal/domain/<域>/...`。

## check-contract-sync A 门禁的真实匹配机制（$ref 行）

场景 → A 检查报"路由已实现但契约未登记"，明明已在子文件 `api/openapi/admin/worker.yaml` 加了路径。
怎么用 → 该检查读的是**顶层** `api/openapi/admin.yaml` 里的紧凑 `$ref` 行（`  /worker-groups: { $ref: './admin/worker.yaml#/paths/~1worker-groups' }`），正则 `^  (/[^:\s]+):\s*\{?\s*\$ref` 只匹配**同一行带 $ref** 的路径，不递归子文件。新增路由要**同时**在顶层 admin.yaml 加 `$ref` 行（`~1` 编码 `/`，`{id}` 原样保留），保持 2 空格缩进；子文件是"契约详情"，顶层是"登记清单"。验证手法：往 `scripts/check-contract-sync/main.go` 的 collectSpecPaths 加临时 `fmt.Printf("DEBUG MATCH...")` 跑一次看真实匹配，再 revert。

## 并行 Agent 同工作区开发导致的编译失败识别

场景 → `go test ./internal/app/...` 报大量编译错误（重复声明 fakeX、API 签名不匹配、函数返回值数量不符），但自己没改过那些文件。
怎么用 → 先 `git status --short`：未跟踪新文件（`??`）+ `stat -f "%Sm"` 时间戳接近当前时间 = 另一个并行 Agent 正在同一工作区同时开发。如 `internal/app/http_provision_seed_test.go` 与我的 worker 改动无关，是他人正在写的内容。识别为并行工作而非自己回归：对比文件时间戳、git 是否 tracked、错误是否涉及自己未碰过的包。**不要修改他人正在写的文件**——只验证自己的领域包（如 `go test ./internal/domain/worker/...`），在总结里如实说明 app 包被并行改动暂时阻塞。

## 开网 12 环节 CLI 全流程置备蓝图（资源/端口/标签/四码/派单）

场景 → 要完整走通「下单→核查→预占→收费→派单→上门→开户」12 环节并落到订单 DONE，缺置备数据(渠道/产品/资源/端口/资产批次/标签/资产/四码/派单)任何一环后续就 404/409。
怎么用 → 按依赖序置备(可用 `POST /provision/...` 联调端点,menu:provision):
1. 渠道 `POST /provision/channels`(code 唯一)→ 拿 chanId。
2. 产品 `POST /products`(name/legalEntityId)→ 拿 offerId。
3. 资源 `POST /provision/resources`(addressId/legalEntityId)→ 拿 resId(**FK 关联必须用真实返回 id**)。研磨 4. 端口 `POST /provision/ports`(resourceId=上一步 id,addressId)。
4. 资产批次 `POST /provision/asset-batches`(code)→ 资产 `POST /provision/assets`(batchId)。
5. **标签绑定资产**:`POST /provision/tags` 填 `boundAssetId=<assetId>`+`status:"BOUND"`(否则扫码 MATCH 不了,见 known-issues)。
6. 四码 `POST /quad-links`(assetId/customerId/portId/addressId/legalEntityId)→ LINKED 状态由扫码驱动。
7. 下单 `POST /orders`(customerId/offerId/addressId/channelId)。
8. 推进:check-resource(2)→ reserve(3,自动占端口)→ charge(4,自动段 5-8)→ dispatch ticket + assign → scan-bind(9,需 EPC 对已绑资产)→ activate(10,自动段 11-12)→ 订单 stage 12/status DONE。
验证:订单 timeline 12 环节全 DONE、`GET /orders/{orderNo}` 的 stage/status、`GET /quad-links` 四码 LINKED、scan_logs 有 MATCH 行。端口占后保持 RESERVED(order_id 非空)是契约常态,不必强行置 USED。

## 为 customer/worker 主体签 API key 并绑定 bossctl 身份

场景 → 要给某个客户/师傅(非 account)保存可用凭证,或让 bossctl 直接以该主体身份调用其专属接口。
怎么用 → `POST /api-keys` 支持 `subjectType ∈ {account, worker, customer}` + `subjectRef=<主档 id>` + `name`;响应 `plainKey` 即明文密钥(只此一次返回)。`bossctl --api-key <plainKey> call GET /auth/me` 可自证身份(subjectType/subjectRef/name)。要长期用,把身份写进 `~/.bossctl/identities.json`(map: identity 名→key),之后 `bossctl --as <名> call ...`。注意 worker/customer 主体密钥**无菜单权限**(RBAC 恒拒),只能调其身份对应接口;customer 主档本无登录口令,此即默认鉴权方式。

## Compose 导航"幽灵跳转"的插桩定位法
场景 → 一次点击触发多次页面跳转/页面自己跳走,理论推演无法解释。
怎么用 → 在 NavHost 的 push/pop/switchTab 里临时加 `android.util.Log.d("NAVDBG", "push $s", Throwable("trace"))`,logcat -s NAVDBG 直接看:事件序列+完整调用栈。栈顶在 Recomposer.performRecompose/ComposableLambdaImpl.invoke = 组合期执行(渲染插槽误用);栈顶在 ClickableNode.handleUpEvent = 真实点击。定位后删插桩再提交。

## 真机 App 内部状态直查(免抓包)
场景 → 怀疑 token/配置没存上,抓包麻烦。
怎么用 → debug 包可 run-as:`adb shell run-as <pkg> cat /data/data/<pkg>/shared_prefs/<prefs>.xml`,空 `<map/>` 即没写过;配合 curl 同端点对照,两分钟分清"没存"vs"存了没用"。

## 真机自动化验证页面上报(adb)
场景 → 无 UI 自动化框架时在真机走登录/点按/断言。
怎么用 → `uiautomator dump /sdcard/ui.xml` + python 正则提取 text/bounds 算中心点 → `input tap x y` → 再 dump 断言标题文本;输入用 `input text`(非 ASCII 需先切输入法或用剪贴板);返回键 `input keyevent 4`。注意 dump 需在页面数据加载稳定后再取,加载中坐标会漂移。

- 场景:接口 404 / 自认为已注册的 gin 路由却 404。排查时第一动作用全路径 `lsof -nP -iTCP:<port> -sTCP:LISTEN`(macOS 的 lsof 在 /usr/sbin,默认 PATH 常没有)→ 列出占用该端口的全部进程;重点看是否 IPv4(127.0.0.1) 与 IPv6(*:port) 双绑同一端口。`curl 127.0.0.1:<port>` 走 IPv4 只命中 IPv4 监听者,若那是另一个陈旧进程,你会得到"假 404/假路由缺失"。发现是双绑时,`kill` 掉陈旧 IPv4 监听者,让 IPv6 wildcard 服务器也接收 IPv4 流量。2026-08-19 后端冒烟实证。
- 场景:go vet 报"missing method"(做不到接口)。给测试 fake 桩补方法时,一次性 `go vet ./...` 让编译器罗列全部缺失方法签名,再批量补,别逐个撞。2026-08-19。
- 场景:Go 单测 `x := helper(...)` 报"assignment mismatch"。凡是目标函数返回多值(如 auth.Sign 返回 (token,error)),一律 `x, _ := helper(...)`。2026-08-19。

## dsh 模型"不支持 read_image"先查配置再下结论

场景 → 在 dsh 中发现某模型调用 read_image 失败或被判定不支持图像输入(对应高频红线 7"禁止假设模型支持图像输入")。
怎么用 → 先检查配置文件 `~/.dsh/settings.yaml`:按 provider → models 找到对应 `id` 的模型条目,看 `input:` 字段是否正确声明了 `[ text, image ]`。缺 `image` 时 dsh 会直接不给该模型图像输入能力——是配置问题,不是模型本身不支持。改完配置重启会话生效。(2026-10 用户经验传授,已实证 settings.yaml 中各 provider 模型确有 `input: [ text, image ]` 字段)
- Android 真机 UI 验证(模型不能看图时):`adb exec-out uiautomator dump /dev/tty` + python 正则抽 text/bounds,断言单行高度、列内边界、元素间距;配合 `input tap x y` 可点开下拉/切 tab 再 dump。注意部分 Compose 节点(渐变头)不暴露。
- 本机工具链路径:go=/opt/homebrew/bin/go(默认 PATH 没有),JAVA_HOME=/opt/homebrew/opt/openjdk@17(gradle),adb=~/Library/Android/sdk/platform-tools/adb。bossctl 二进制全局 flag 是 -server 指定服务端地址。
- 从服务端 PG 取短信验证码明文:PG 映射在 192.168.0.102:25432(boss/boss/boss,sslmode=disable),查 portal_sms_codes(phone,scene);用临时 go 脚本+pgx 直连。

## 排查"用户说看不到但代码正常"的 UI 异常(2026-08-20)
- 场景:DOM 断言 computedStyle/mask/尺寸全部正常,用户却报告图标 hover 才出现。
- 手法:先让用户硬刷新 + 确认访问地址(dev HMR/缓存/部署版本差异是首因);确认仍异常再元素级像素对比(CDP Page.captureScreenshot + getBoundingClientRect clip),不要一上来深挖代码。
- 新增 migrations 文件后若镜像内服务读文件报 permission denied:write 工具默认 600,构建镜像前 `chmod 644`。
- 102 后端部署链路:docker context 102-remote(ssh imeepos@192.168.0.102)本地 build → push 192.168.0.102:5000/boss/server → `docker compose -f deployments/docker-compose.102.app.yml up -d --force-recreate server`;迁移由 server 启动自动执行。
- 本机无 JDK 也能编 Android:gradle 自带 JDK 在 ~/.gradle/jdks/eclipse_adoptium-17*/.../Contents/Home,`export JAVA_HOME=<该目录> ANDROID_HOME=~/Library/Android/sdk && ./gradlew compileDebugKotlin --rerun-tasks`;UP-TO-DATE 不算验证,必须 --rerun-tasks(2026-08-23 Stripe 支付页验证用)。
- 2026-08-21 真机点不准控件时: `adb shell uiautomator dump /sdcard/ui.xml && adb shell cat /sdcard/ui.xml | grep -o 'text="xxx"[^>]*bounds="[^"]*"'` 直接取 bounds 中心点，不猜坐标。
- git 提交长中文 message 不要用 `git commit -m "$(cat <<'EOF' ... EOF)"`(bash 报 bad substitution);写临时文件 `git commit -F <file>` 后删除。2026-08-24。
- 102 后端新部署链路(2026-08-21 实测):本地 git push gitea(192.168.0.102:222) → CI 自动 build+push 镜像(tag=commit SHA,latest 随之更新,约 1 分钟) → ssh 102 `cd ~/boss/deployments && docker compose -f docker-compose.102.app.yml pull server && up -d server`(报 loki logging plugin 错误但容器仍重建成功,以 docker inspect Image/StartedAt 为准) → curl 验证。
- 模拟器注入真实 token 联调:`adb shell am force-stop` 后 `adb shell "run-as <pkg> sh -c 'cat > /data/data/<pkg>/shared_prefs/<prefs>.xml'"` 写入信封 token;登录态用 `run-as <pkg> cat shared_prefs/<prefs>.xml` 核对,残留 mock-xxx token 一眼可辨。(2026-08-21)

## 任务开工先 `git status -uall` 划定"我的工作面"
场景 → monorepo 里有多端/多 agent 并行工作,worktree 已有他人未提交的 dirty 文件;你只该动自己负责的子目录,否则提交时会污染别人的改动、且看不出脏 diff 是谁先来的。
怎么用 → `git status -uall`(列出 unstaged + untracked)。把任务限定到目标子目录(mobile/worker/android/ 或 internal/httpapi/worker/),只 `git add <明确清单>`,他人的 diff 留给各自 owner 收口。如果发现清单里有他人刚遗留的小问题(如 import 缺失、lint),**别顺手带**,要么单开一个 chore commit,要么抛到群里给对应 owner。

## 开发模式验证码自动回填(不用再翻日志抄码)
场景 → 联调测试时频繁登录/注册,每次都要去 PG 查 portal_sms_codes 拿验证码再手动填入,繁琐。
怎么用 → 后端环境变量 `BOSS_DEBUG_SMS=1` 启动;Android App "我的" 页底部「开发选项」开关开启;点「获取验证码」后自动从 `/debug/sms-code?phone=&scene=` 拉明文回填。release 包永远不生效(前端 `BuildConfig.DEBUG=false` → `DevModeStore.isEnabled()` 恒 false)。后端未配 `BOSS_DEBUG_SMS=1` 时返回 404,Android 侧静默降级,不报错。
关键文件 → `internal/httpapi/user/debug_sms.go`(后端端点),`mobile/user/android/app/src/main/java/com/ymm/boss/user/api/DevModeStore.kt`(开关持久化),`util/DevSms.kt`(自动回填封装),`api/DebugApi.kt`(客户端端点)。verify 场景(实名认证)用 `debugOptionalAuthn` 中间件解析 JWT→从 portal_accounts 取 phone,不需要客户端显式传 phone。(2026-08-21)
20. CDP 单次运行内多步导航:eval 链中先 set localStorage,再 `location.href='/target'`(cdp-capture 每条 eval 后有 settle,后续 eval 落在新页面上),最后一条 eval 跑断言 IIFE,VERIFY 结果从 --logs 的 console 条目取(2026-08-2x importer 双主题验证)。
- 门禁(make check)红了先证伪"是不是我搞的":`git worktree add /tmp/base <开工前commit>` 在基线跑同一检查,失败集合相同即并行会话遗留债务,只修自己域+在总结如实声明,不替在场他人越界修(git show BASE:file | wc -l 对比行数可进一步定位)(2026-08-21 时区改造)。

## 远程(ssh/102)执行多行 SQL
场景 → ssh 到服务器在 docker 容器里跑 psql 多行 SQL。
怎么用 → 本地写 .sql 文件 → scp 到远程 → docker cp 进容器 → `docker exec <pg> psql -U boss -d boss -f /tmp/x.sql`,末尾带 SELECT 验证行数。禁止 ssh 单引号内嵌 heredoc:docker exec 会静默不执行(退出码 0 无输出),极易误判成功。

## PG 分区 default 积压导致 23514 拒建分区
场景 → `CREATE TABLE ... PARTITION OF` 报 `updated partition constraint for default partition would be violated`(SQLSTATE 23514),服务启动崩溃循环。
怎么用 → 事务内:暂存表(LIKE 母表)← default 中该范围行 → DELETE default 该范围 → 建分区 → 从暂存表 INSERT 回母表 → DROP 暂存表;修复代码见 internal/pkg/audit/partitions.go migrateDefaultRows(自愈路径)。
- 102 无短信凭据时通道降级 LogSender,验证码直接打在 boss-server 容器日志:POST sms-code 后 `docker logs boss-server --since 30s | grep 'sms\[dev\]'` 捞 code 即可 e2e 登录师傅端(2026-08-25 push_devices 冒烟);注意同 phone+scene 60s 冷却(42300)。

## cdp 截图指定主题/语言:URL 一次性覆盖优先于 --eval
场景 → 要给同一页拍 light/dark 双主题,或指定语言。
怎么用 → `cdp-capture.mjs "http://host/home?theme=dark" out.png`;lib/urlPrefs.ts 启动时把 ?theme=/?lang= 写入 localStorage 并抹除,首屏即正确;比 --eval setItem(页面加载后才执行、不重渲、需再 reload)省一步且无误差。

## git worktree 快速闭环(改代码防主分支污染)
场景 → AGENTS.md 禁止直接在主分支改代码时。
怎么用 → `git worktree add ../boss-<task> -b feat/<task>` → 在 worktree 内开发+门禁+commit → 主 checkout `git merge --no-ff` → `git worktree remove --force`(web/admin/node_modules 残留会挡普通 remove)→ `git branch -d`;最后 `git worktree list` + `git branch` 双确认干净。未跟踪文件(如设计稿 png)在主 checkout 直接 mv 进 worktree 即可带走。
- 2026-08-22 CDP 自动化点击 Dropdown 选项必须派发 `mousedown`(组件 onChange 挂在 onMouseDown,onClick 仅 preventDefault);触发器开浮层用 click,选项选择用 `el.dispatchEvent(new MouseEvent('mousedown',{bubbles:true}))`。
- 2026-08-22 cdp-capture 注入登录态后 SPA 不认:eval 设 localStorage 要与 `location.href=目标页` 同一条执行(先访问根路径设值再跳转),分开执行时应用已用空 token 启动重定向 /login。
- 2026-08-22 102 环境分工:28080=纯 API(直访 SPA 路由 404 page not found),admin GUI 在 5180 端口(compose boss-admin-web 容器);截图/联调一律打 5180。
- 2026-08-22 PG LIKE 中 `_` 是单字符通配符:精确前缀匹配写 `LIKE 'custom\_%'`(默认转义符反斜杠),或用 `starts_with(code,'custom_')` / 左等值。删除验收数据按主键等值删,不用模式。
- 场景:官方 API 文档是 SPA 抓不到、又不确定参数形态。做法:写 10 行 POP 签名脚本,缺啥参数补啥参数,让网关逐个报 "X is mandatory",3 轮内拼出必填参数表;再用明显非法值(如 ContentCode=test)读业务错误码(SMS_CONTENT_CODE_ILLEGAL)确认参数语义。

## 后台循环类功能验证:先看容器日志再查表
场景 → 部署后新循环(执行器/扫描/对账)该写表却没写,或怀疑循环没接上。
怎么用 → ① `docker ps --filter name=boss-server` 确认 Up 时间是新构建;② `docker logs boss-server --since 10m | grep <循环关键字>`(如 etl executor)看循环是否启动、有无 record running/finish 报错;③ 再查目标表。本次 RecordRun SQL 42725 就是日志先暴露,表空是结果不是原因。
- 2026-08-24 docker exec 进 postgres 查表用 heredoc 管道:`cat <<'EOF' | ssh imeepos@192.168.0.102 'docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atq'`;SQL 内单引号直接写,不要用 shell 转义(内联 \x27 会语法错)。

## 登录 admin API 拿 token 再验证业务端点
场景 → 需要带鉴权查 admin 接口(freshness/runs 等)。
怎么用 → `TOKEN=$(curl -s -X POST http://192.168.0.102:28080/api/admin/v1/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["token"])')` 然后 `curl -H "Authorization: Bearer $TOKEN" ...`。登录路径是 /auth/login(不是 /login),响应信封 data.token。

## ETL 新鲜度派单闭环核对(102 实机)
场景 → 验证"滞留派单与任务新鲜度一致、扫描无误派单"。
怎么用 → ① 查派单:`docker exec boss-infra-postgres-1 psql -U boss -d boss -c "select biz_id,status,count(*),max(updated_at) from compensation_tasks where biz_id like 'etl_freshness:%' group by 1,2"`;② 查新鲜度:`select job_key,last_status,last_run_at,now()-last_run_at from etl_job`;③ 判定一致:FRESH 任务不应有 OPEN 派单(CloseRecoveredETL 会自动关),无真实执行器的投影任务 last_status 恒 RUNNING/last_run_at NULL,其 OPEN 派单是真实滞留不是误派;④ 连续扫描周期核对:间隔 > ETL_AUTODISPATCH_INTERVAL_SECONDS(默认 5min)取两次快照,OPEN 行数与 created_at 不变即无重复派单(SubmitQualityViolations 按 bizId+OPEN 幂等)。实机证据:2026-08-24 17:42 UTC 恢复的 ar_aging_snapshot/metric_quality_scan 派单被自动 CLOSED,4 条投影任务 OPEN 派单与恒 RUNNING 台账一致。
