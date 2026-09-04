# Techniques

<!-- 排查技巧、工具命令、调试手法。格式：什么场景 → 怎么用。 -->

## 无 Playwright 时给 Web 页面（含需登录页）截图

场景 → 前端改动要可视化验证，环境无 playwright/puppeteer，但 macOS 有系统 Chrome。
怎么用 → 跑 `scripts/cdp-capture.mjs`（Node>=22，零依赖，自带 Chrome 启停）：

```bash
node scripts/cdp-capture.mjs http://localhost:5173/login /tmp/a.png \
  --eval '/* 可选:页面加载后执行的 JS,如填表登录 */' --settle 2500
```

## cdp-capture 异步登录后采集 DOM(2026-09-01 实证)

场景 → 用 --eval 做 fetch 登录再断言登录后页面(侧栏菜单/正文文本),直接跟一个采集 eval 会拿到空串——eval 是顺序立即执行的,fetch 的 .then 未完成、location.reload 未生效。
怎么用 → 采集断言放进 `new Promise(res=>setTimeout(()=>{console.log('KEY:',...);res()},3000~4000))` 让 eval 本身阻塞到登录完成;且 --logs 每次运行覆盖,重跑同一 out.json 不带 --logs 会丢之前的采集。选择器别依赖具体 class,`document.body.innerText.replace(/\n+/g,'|')` 全文前 600 字最稳。

## read_image 不回传视觉内容时的替代(2026-09-01 实证)

场景 → read_image 连续两次只返回 "has been processed" 而无图像内容(当前模型通道限制,与红线 7 同源)。
怎么用 → 不假装"看到了截图";改走 DOM 文本断言(上一条技巧)或 console 采集,结论里只写有真实证据支撑的判断。

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

## gitea actions 实机日志取证
场景 → 需要流水线"跳过/部署"分类的原文证据(UI 不便或需自动化)。
怎么用 → runner 侧 `docker logs gitea-runner --since 30m | grep task` 拿任务号;日志文件在 gitea 数据卷 `sudo find .../gitea_data/_data/actions_log/sker/boss -name <task>.log.zst`(目录按 task 号十六进制分桶,如 2605 在 2d/、2607 在 2f/),`sudo zstdcat` 后 grep "Runtime-affecting|Documentation"。任务运行中日志未落盘,需等完成。

## 新路由契约 A 检查红灯:登记源是 openapi yaml(2026-08-28 cms)
场景 → `make contract-sync` A 项报"路由已实现但契约未登记"。
怎么用 → 登记源是 `api/openapi/admin.yaml`(加 `$ref: './admin/<域>.yaml#/paths/...'` 引用)+ 新建 `api/openapi/admin/<域>.yaml`(抄 knowledge.yaml 形状);`cmd/bossctl/routes_admin.go` 只是 CLI 展示目录,加了它 A 检查照样红。写 handler 前先把 yaml 建好,一次过 A。

## pgxmock 单测两个形状坑(2026-08-28 cms)
场景 → 域 store 写 pgxmock 契约单测。
怎么用 → ① pool 接口类型名是 `pgxmock.PgxPoolIface`(不是 PgxPool);② SELECT 里的 `TO_CHAR(col,...)` 产出字符串列,mock 行必须喂字符串("2026-08-28 10:00"),喂 time.Time 报 destination kind 'string' not supported。

## 102 回放确认"新代码已上线"的可观测差异法(2026-08-28 cms)
场景 → push 后轮询端点,一直 200 但行为像旧代码。
怎么用 → 别用"端点可达"判定;制造只有新代码才有的行为差异再轮询:如 cms 修复后,创建即 PUBLISHED 的文章 publishedAt 非空才是新代码(try1 旧 try2 新,间隔 25s)。泛化:回放断言里必须包含至少一个"新代码专属可观测字段"。

## SPA SEO 元数据 DOM 断言(102 实机)
场景 → 验证异步详情页 SEO title/description/og。
怎么用 → cdp-capture `--eval "(() => ({title:document.title,description:document.querySelector('meta[name=description]')?.content,ogTitle:document.querySelector('meta[property=\\\"og:title\\\"]')?.content,ogImage:!!document.querySelector('meta[property=\\\"og:image\\\"]')}))()"`;同时 logs 断言 console errors=0。无需截图目测。

## nginx SPA 缓存头双断言
场景 → 验证 stale SPA 修复已部署。
怎么用 → `curl -sI http://192.168.0.102:5180/ | grep -i cache-control` 必须 no-cache;再从 `/home` HTML 取 `assets/index-*.js`,对 hash 资产 curl -I 必须 public, immutable。两侧一起断言,防误改。

## menu 权限 baseline 消化回放
场景 → 迁移补专属 menu:<key> 后确认已落库。
怎么用 → 先 `make contract-sync` E 必须显示 zero baseline;部署后 admin login → `/auth/me` 取 `permissionCodes`,逐项 grep 新码;只看到 menu.def 不等于 DB 已迁移。

## Tauri 内嵌静态资源验证(2026-09-01)
场景 → 验证 `tauri build` 产出的桌面二进制确实嵌入了最新的 `web/admin/dist` 静态资源。
怎么用 → ① 先确认 dist 入口文件名: `grep -o 'index-[^"]*\.js' web/admin/dist/index.html`;② 从二进制中 grep 该文件名: `strings target/<profile>/boss-desktop | grep -c "index-xxx"`(返回≥1 即嵌入成功);③ 可选查验资产路径总数: `strings binary | grep -oE "/assets/[A-Za-z0-9._-]+" | sort -u | wc -l` 与 dist 文件数对比。注意:assets 内容被 brotli 压缩,HTML 全文不会出现在 strings 中;入口文件名和路径 key 以明文出现。

## cdp-capture 免登录注入时序(2026-09-01)
场景 → 对 admin SPA 页面注入 token 后截图/断言 DOM。
怎么用 → 先从 `/login` 打开页面再注入 localStorage,然后 `location.href='/目标路径'` 导航;不要在目标页注入后 `location.reload()` —— reload 时序下 AuthGuard 先读旧态会弹回 /login。注入项:boss.token + boss.servers([{id,name,baseUrl}]) + boss.server.active(存 id)+ boss.theme。dev 页面用 `localhost:5199`(vite --strictPort),生产用 102:5180 同源代理;断言分组标题等计算样式直接 `--eval getComputedStyle` 返回 fontSize/color,无需读图(flash 模型无图像输入)。

## 幂等文件修改脚本模板(2026-09-22)
场景 → 用 python 往现有 TS/Go 文件插入字段/实体,防重复执行污染。
怎么用 → 每处插入先查标记:`if '  uniqueKey?: string[]' not in s:` 才插;每个 kind 块用 `if marker not in s.split(kind_line)[1].split('  {')[0]: continue` 防重;结尾打印 `s.count(标记)` 断言插入次数正确。写完立即 `git diff --stat | wc -l` + `wc -l <file>` 确认单次增量(本次死循环:输出每次都是 "entities ok",但文件从 140→1388 行)。
- Stripe 测试环境:webhook endpoint 可用 sk 直接调 POST /v1/webhook_endpoints 创建(url+enabled_events),whsec 创建响应一次返回;确认 PaymentIntent 需 return_url(账号启用重定向支付方式时);排查隧道时先验后端身份(/healthz + 未配置 webhook 的 503 特征),容器名不可信。
- 隧道后端身份三验:① /healthz 返回体;② 未配置端点的降级特征(如 stripe webhook 未配置=503 {"error":"stripe webhook not configured"});③ 401/错误文案在仓库 grep 是否命中——不命中即非自家服务。
- Stripe webhook endpoint 重建(REST,免 OAuth/CLI):POST /v1/webhook_endpoints(url=隧道+回调路径,enabled_events[]=payment_intent.succeeded/payment_intent.payment_failed)→ 响应 secret 即新 whsec;旧错路径 endpoint 可 DELETE。
- PaymentIntent API confirm 遇 400 要求 return_url:账号启用重定向型支付方式所致;测试确认卡加 `-d "return_url=https://example.com/pay-done"`。
- psql 经 ssh/嵌套 bash 执行 SQL:引号会被外层吃掉,`-c "..."` 里叠双引号必炸(column does not exist/syntax error);一律 `ssh host 'docker exec -i pg psql -U u -d d' <<'SQL' ... SQL` 单引号 heredoc 传 stdin,SQL 字符串字面量用单引号。
- portal E2E 客户映射套路:注册(合成负 id) → 查 live customers 必填列 → 建真实 customers 行 → UPDATE portal_accounts SET customer_id → 密码模式重登拿真实 id token。
- python/脚本插入结构化块(i18n/JSON/TS)时,锚点只包含"上一块块尾闭合行+下一键名",绝不把上一块的正文尾部行(如 testOk/testFail)带进锚点,否则整块插进上一块内部、闭合乱序;插完立刻 typecheck/渲染校验。
- 动态配置(Dynamic 类)缓存策略:先缓存配置本身,客户端按需懒建;与客户端构造无关的取值(如 WebhookSecret)不得被"缺主凭据"短路,否则误伤 webhook 路径。
- cdp-capture 注入 SPA 登录态:模块启动(localStorage 读取)先于 eval,直接 setItem 无效;须 eval 里先写 localStorage 再 location.href 重载到目标页,二次加载后才带 token。
- Stripe 托管收银台浏览器级测试驱动:cdp-capture 截图看不到时,写 CDP 脚本按 execution context 逐 frame
  evaluate(跨域 iframe 也能填):`Runtime.enable` 收集 executionContextCreated(frameId→contextId),对
  js.stripe.com 或 checkout.stripe.com 顶层 frame 用原生 setter+input 事件填 cardNumber/cardExpiry/
  cardCvc(测试卡 4242...),找 Pay 按钮 click,轮询 location.href 落出 checkout 域即支付成功。
  注意 checkout URL 的 #hash fragment 不能截断(截断报 CheckoutInitError)。
- Page.navigate 到完全相同的 URL 是 no-op(不重载脚本);要强制重载加 query 参数或 Page.reload。

## Android Compose + play-services 的最小依赖配置（Material3 BOM 2026.06+）

拉取 GPS 等 Google Play 服务 + 在 Kotlin 协程里 `.await()`，最小依赖组合：

```toml
# gradle/libs.versions.toml
[versions]
playLocation = "21.3.0"
coroutines = "1.10.2"

[libraries]
play-location = { group = "com.google.android.gms", name = "play-services-location", version.ref = "playLocation" }
kotlinx-coroutines-play-services = { group = "org.jetbrains.kotlinx", name = "kotlinx-coroutines-play-services", version.ref = "coroutines" }
```

```kotlin
// app/build.gradle.kts
implementation(libs.play.location)
implementation(libs.kotlinx.coroutines.play.services)
```

```kotlin
// 运行时位置
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import kotlinx.coroutines.tasks.await

@SuppressLint("MissingPermission")
suspend fun current(ctx: Context): Location? {
    val cts = CancellationTokenSource()
    return try {
        LocationServices.getFusedLocationProviderClient(ctx)
            .getCurrentLocation(Priority.PRIORITY_BALANCED_POWER_ACCURACY, cts.token).await()
            ?: LocationServices.getFusedLocationProviderClient(ctx).lastLocation.await()
    } finally { cts.cancel() }
}
```

**Manifest**：`ACCESS_FINE_LOCATION` + `ACCESS_COARSE_LOCATION`。**运行时权限**用 `ActivityResultContracts.RequestMultiplePermissions()`，在 Compose 里 `rememberLauncherForActivityResult` 调起。

## worktree 出现非自己创建的本地文件时

并行 worktree 残留的 0 字节临时文件（比如其他会话创建未跟踪的 `行政区划` 之类）：
- **不要删**（不是自己会话产物）
- `git worktree remove --force` 跳过清理，保留分支
- 在 `notes.md` 记录"曾出现 X 残留文件，未清理"
- CI 拥堵/未触发时的手动部署路径:本地 rsync 源码到 102 /tmp/boss-deploy-src(排除 .git/web 构建产物)→ ssh 上 docker build -t 192.168.0.102:5000/boss/server:<sha> -t ...:latest -f deployments/docker/server.Dockerfile . → docker push 两个 tag → docker rm -f boss-server boss-aaa boss-report → docker-compose -f deployments/docker-compose.102.app.yml -p boss-app up -d --force-recreate --remove-orphans(2026-08-27 实录;gitea clone 的 GITHUB_TOKEN 只在 runner 环境,ssh 会话拿不到)。
- bossctl 调用户端接口不要用 user: 前缀(CLI 映射 /api/v1,服务端实际 /api/user/v1),直接写完整路径 /api/user/v1/...(2026-08-27 实测 404)。
- 102 部署新二进制:本地 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build 后 scp,比在 102 编(go.toolchain 自动下载易超时)快且稳(2026-08-27 oltsim 实录)。
- 102 起长驻进程:setsid 二进制 ... > log 2>&1 < /dev/null & 再 disown,普通 nohup+& 经 ssh 会被会话收割(2026-08-27 实录)。
- 102 新服务开监听前先 ss -tln 查端口占用,避免撞 goproxy/其他服务;撞了就换高位端口(2026-08-27 oltsim 8081/8088 均被占,18099 才净)。

## 接口/页面对账手法:抓"接口有、页面无"(2026-08-26 实名代录)
场景:审查某业务流程两端是否闭环。做法:对后端路由文件(internal/httpapi/**)注册的每个端点路径,拿路径字面量去 web/mobile 前端源码 grep;零命中即"接口有页面无"(本次 POST /customers/:id/real-name 藏在 admin/customer.yaml 但无任何 apiFetch 调用)。反向同理:前端 apiFetch 的每个路径回 grep 后端路由注册,防端上写了死链。
## 需要登录的内页 CDP 冒烟(2026-09 用户详情重构)

场景:admin 前端内页需 token+serverConfig,直接访问会被 AuthGuard 弹回 /login。

用法:首个 eval 注入 `boss.token` + `boss.servers`(数组元素必须含 id)+ `boss.server.active` 后,**用 `location.href='/目标路由'` 显式导航**(reload 会停在 login);后续每个 eval 之间自带 settle,可完成"点击行按钮开抽屉→逐页签点击→innerText/th 枚举断言"的全链路 DOM 断言;双主题切换单独一次 eval 并用 Promise/setTimeout 等 ≥300ms 过渡结束再读 computed style。零 console 报错/失败请求检查交给 --logs。

## Tailwind utility 是否进了构建产物(2026-09)

场景:怀疑任意值类(bg-[var(--x)])没被生成时,vite dev 的 CSSOM 扁平遍历有 @layer 盲区,不要信。

用法:`grep -o ".\{0,60\}令牌名.\{0,80\}" dist/assets/*.css`——产物里有即已生成;要找"谁覆盖了背景"再做递归 cssRules walk(matches 按选择器过滤)。

## 菜单图标缺文件一键审计(2026-08-27,崩溃日志菜单空位)
- 场景:后台菜单某项无图标;MaskIcon 按 menu.def key 取 public/icons/items/<key>.svg,缺文件即 404+空白。
- 手法:`grep -o "key: '[^']*'" web/admin/src/router/menu.def.ts | cut -d"'" -f2 | sort > /tmp/want.txt && ls web/admin/public/icons/items/*.svg | xargs -n1 basename | sed 's/.svg//' | sort > /tmp/have.txt && comm -23 /tmp/want.txt /tmp/have.txt`
- 注意:macOS grep 无 -P;用 -o "key: 'xx'" + cut。新增同名资产前先查并行分支是否已占(`git branch -a | grep 关键词`)。

## 崩溃日志(或任何客户端上报)链路端到端验证(2026-08-27)
- 场景:管理页有数据为空,要判断"未对接"还是"真没数据"。
- 顺序:① curl 探路由(401=挂载,404=缺路由)→ ② ssh psql heredoc 查表迁移+行数 → ③ 用 test-accounts.json 里 worker key 走真实业务路径 POST 一条探针(app 带 probe 标记)→ ④ admin key 读回确认 JSON 形状 → ⑤ DELETE WHERE app='probe...' RETURNING id 清理并核对行数归零。
- 坑:链路三层(App/后端/DB)代码在 ≠ 线上有数据;功能合入时间 vs 镜像构建时间 vs 设备 APK 版本三方对表才解释得了 0 行。

## 幽灵 CSS 令牌全量审计(2026-09-25,实名审核页主题失效)

- 场景:页面用了 `var(--x)` 但 `--x:` 在全部 css 里无定义,var() 解析失败属性按 initial 渲染——主按钮白字透明底、徽章丢色,**亮暗主题双双坏**,且不报错、无 lint 拦截,极易上线数月无人察觉(本次 --color-brand-bg/--color-info/--color-warning 三令牌散布 5 个文件)。
- 手法(node 一行流):收集 dist 构建产物或源码里所有 `var(--` 引用名,减去所有 ` --xxx:` 定义名,差集即幽灵令牌。源码版:`grep -rhoE 'var\(--[a-z-]+' src --include='*.tsx' | sed 's/var(//' | sort -u > /tmp/use.txt; grep -rhoE '^  --[a-z0-9-]+:' src/theme/tokens.css src/styles.css | tr -d ' :' | sort -u > /tmp/def.txt; comm -23 /tmp/use.txt /tmp/def.txt`
- 注意:定义可能分散在 tokens.css/styles.css/组件自带 css 块三处,def 集要收全否则误报;tailwind 任意值类 `bg-[var(--x)]` 也走同一 grep 能覆盖。

## i18n 引用键门禁:规格 walker 测试(2026-09-26,用户详情抽屉首例)

- 场景:页面文案键走宽松 `Record<string,string>` 字典,typecheck 拦不住打错的引用键,线上直接露出英文键值。
- 手法:把页面规格(列/枚举映射/段名/页签名)抽成纯数据模块(detail-view.ts,无 React/i18n 依赖),vitest 里 `import 规格 + 三份 locale`,walker 自动收集全部引用键(col.k、enum map 值、section key、tab key、`d_` 主档键、抽屉字面量清单),断言三语言字典全覆盖;特殊段(非字典渲染)用共享常量标记并在 walker 跳过。
- 价值:新增列/枚举自动纳入,人为制造失配即红灯(本例首跑就抓出 usages 段名三语言全缺的存量静默 bug);文案键零硬编码有了机械对账面。
- 位置参照:`web/admin/src/pages/bss/user/detail-i18n.test.ts` + `detail-view.ts` 的 DRAWER_D_KEYS。

## 102 业务回归三层证据采集(2026-09-26,实名审核中心)

- 场景:修复 Go SQL/业务接口后需要确认 102 是否已部署且可回归。
- 固定顺序:① `curl /healthz` 记录进程存活;② POST `/api/admin/v1/auth/login` 记录登录 code;③ 带 token 请求目标列表和写接口,记录完整响应体。三层必须分开判定,`healthz=ok` 不覆盖业务 API 的授权失败。
- 判定模板:目标接口 `code=0` 才能继续业务断言;`LICENSE_REQUIRED` 是环境授权阻断;`403` 查权限迁移;`50000` 先取服务端日志/SQLSTATE;`40400` 查数据主体或路由。
- 对 SQL 修复必须同时保留 pgxmock 回归和真实路径结果;mock 只验证调用形状,不验证 PostgreSQL 列作用域、NULL 扫描和实际迁移。

## 枚举→展示文案映射前,先查真库取值域(2026-09-26,faults type 双口径)

- 场景:给某列写"枚举值→i18n 键"映射,契约文档已列出枚举,直接照抄开写。
- 坑:complaint-type-map.md 只登记了装维域 6 码(SINGLE_OUTAGE...),真库 complaints.type 还有一整套用户端落库值(`用户报障: no_internet|slow|ont_fault|other`、`用户投诉: attitude|...`),照抄上线即原样露出英文/复合前缀。
- 手法:动手前 `SELECT DISTINCT type FROM complaints` 对齐真实取值域;复合前缀在聚合 SQL 侧 `replace(type,'用户报障: ','')` 归一(对齐 service.go portalFaultTypeLabelFromStored 先例),前端映射只收干净值;未知值回退原文是兜底不是借口。

## worktree 复用主树 node_modules(pnpm store 不可达时)

- 场景:worktree 里跑前端门禁,`pnpm install` 撞全局 store-dir(如 /Volumes/sker 卷未挂载)EACCES。
- 手法:`ln -s /主树绝对路径/web/admin/node_modules worktree/web/admin/node_modules`(必须绝对路径,相对 ../.. 在该环境解析失败),之后直接调 `./node_modules/.bin/tsc|vitest|vite` 绕过 pnpm;门禁结果与主树一致。

## CDP 定位表单控件:语义锚点+作用域,禁盲选下标

- 场景:页面有多个 aria-haspopup="listbox"/aria-label 按钮(顶栏语言切换、菜单、业务下拉)时,querySelector('button[aria-label]') 或 .at(-1) 会点错。
- 手法:先 dump 候选 `[...document.querySelectorAll('button')].map((b,i)=>i+':'+b.textContent.trim())` 核对,再用组合锚点定位,如 `[...document.querySelectorAll('button[aria-haspopup="listbox"]')].find(b=>b.closest('label')?.textContent.includes('订阅事件'))`。

## CDP 双主题×双语言矩阵断言(2026-08-27,营销弹框适配)

- 场景:验收"组件/页面适配多主题多语言",模型不吃图、肉眼看像素不可复核。
- 手法:四组 capture( light/dark × zh/en|ms ),每组两发 DOM 断言当产物:
  主题断言 `getComputedStyle(input).backgroundColor/borderColor` 命中 tokens 实测值
  (light=#FFFFFF/#D7DDE7,dark=#10203F/rgba(255,255,255,.14),值源 theme/tokens.css);
  语言断言 `[role=dialog] [role=option]` 逐项文本等于当前 locale 译文。
- 工具:`scripts/cdp-admin-capture.mjs`(自动 login 取 token+两步注入+theme/lang 透传),
  断言经 `--eval` 走 stdout,不读图。改造前后各跑一遍,先证伪再修。
- 稳定选择器契约:Drawer 根=`aside[role=dialog]`,Dropdown 触发器=`button[aria-haspopup=listbox]`,
  选项=`[role=option]`;断言只准靠这些语义锚点,禁视觉坐标。

## boss.token 莫名消失排查树(2026-08-27)

- 症状:`?token=` 直访业务页必落 /login,probe `localStorage.getItem('boss.token')`=null,
  像"URL 参数没生效"。
- 排查序:① probe localStorage 三键(token/servers/active)定位丢的是哪个;
  ② token 丢 = AuthGuard(`layouts/AuthGuard.tsx`)启动预取 /auth/me 失败 catch 分支
  走 `adminLogout()` → `removeItem('boss.token')`——servers 未配置时 apiBaseUrl() 为空必失败;
  ③ 修复=servers 先于 token:先访任意页(如 /login)写 boss.servers+active,再带 token 跳目标页;
  ④ 全链路已封装 `scripts/cdp-admin-capture.mjs`,手写 eval 场景照 templates 模板 C/E。
- 铁律:boss.token 消失先怀疑代码内登出路径(adminLogout),不要怀疑 urlPrefs/浏览器存储。

## 新 worktree 装前端依赖:store-dir 抄主树 .modules.yaml

- 场景:worktree 里 `pnpm install` 撞全局 store-dir(/Volumes/sker 卷未挂载)EACCES;symlink 主树 node_modules 之外的另一条路。
- 手法:`grep storeDir 主树/web/admin/node_modules/.modules.yaml` 拿到实际 store 路径,再 `pnpm install --store-dir <该路径>` 完整安装(worktree 自带 node_modules,pnpm test/build 原生可跑,不依赖主树结构)。

## 「清单/下拉只有一项」反常调研双证据法(2026-08-27 实证:订阅事件选择器)

- 场景 → 用户报"XX 只有一个/很反常",对象是下拉选项、事件目录、清单类供给数据。
- 怎么用 → 五步定位:
  1. **grep 找数据源**:前端渲染哪个 API(如 `SubForm.tsx` → `GET /openplat/event-types`),后端该端点原样返回什么(`openplat.EventCatalog()`);渲染链路每层核对,排除前端过滤假象。
  2. **判定供给模式**:登记制(emit 侧落地才登记,如 `eventCatalog`)、配置制还是字典表——**登记制下"目录小"可能是如实反映而非缺陷**。
  3. **数 emit 侧调用点**:`grep -rn "Emit(" --include="*.go" internal/ | grep -v _test` 对准目标 emitter,核对「已发射事件 ⊆ 已登记事件」与「已登记 ⊆ 已发射」两个方向(后者漏登是 bug,前者缺口是覆盖度)。
  4. **102 运行时复核**:`curl -s -H "X-API-Key: $ADMIN_KEY" http://192.168.0.102:28080/api/admin/v1/openplat/event-types`(key 取 `.agents/skills/bossctl-cli/test-accounts.json`),与代码比对排除部署代差。
  5. **结论三档**:渲染链路 bug / 登记缺漏(发射了没登记) / 覆盖度缺口(发射侧就没挂)——前两档是代码缺陷直接修,第三档如实汇报交用户拍板(扩展事件=产品决策)。

## 门禁红归属判定:先回主树复跑 + 关键词范围断言(2026-08-27 实证:contract-sync 24 项红)

- 场景 → feature/worktree 分支跑 `make check`/`make contract-sync` 出红,而树里混着并行会话的改动。
- 怎么用 →
  1. **回主树复跑同一门禁**:同红 = 存量(并行会话引入),不碰不修,总结里注明归属;仅自己改动引入的红才属于本次修复范围。
  2. **修完做范围断言**:`make contract-sync 2>&1 | grep -ci "openplat"` 为 0,证明自己域内全绿,不背他人存量——用域关键词而不是数总数(总数随并行会话浮动)。
  3. **存量红落在本任务调研域内**:可顺手修,但必须压成独立小提交(零行为变更、可单独 revert),不埋进 feature 提交。

## 102 PG 容器定位与造数零残留核对(2026-08-27 实证)

- 场景 → 集成测试跑完真库(`BOSS_PG_TEST_DSN`)后核对无残留;或手工查 102 库本机无 psql。
- 怎么用 → 容器名不可猜(102 上并存 7 个 postgres),**按端口定位**:

```bash
# 1 定位容器:boss-infra-postgres-1 才是 25432;weibo-pro-postgres 占 5432 别混
ssh imeepos@192.168.0.102 'docker ps --format "{{.Names}} {{.Ports}}"' | grep 25432
# 2 残留计数:heredoc 传 stdin,SQL 字面量直接写单引号(红线 9a,不叠引号)
ssh imeepos@192.168.0.102 'docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tA' <<'SQL'
SELECT 'apps_left=' || count(*) FROM open_apps WHERE app_id LIKE 'op_app_insert_%';
SQL
```

- 断言 = 计数 0 才算清理闭环;测试 seed 一律用「语义前缀 + unixnano」专用命名(如 `op_app_insert_<nan>`),LIKE 前缀一查全中。

## grep 检索以短横线开头的 CSS token

- 场景 → 搜索 `--shell-*`、`--color-*` 等以 `-` 开头的变量名。
- 怎么用 → 优先使用 grep 工具；shell 中使用 `grep -e "--shell-input-border" file` 或 `grep -- "--shell-input-border" file`，否则模式会被当成 grep 选项。

## 前端部署确认"新代码已上线":远端 bundle grep 标记 + 本地 build hash 对照(2026-10-01)

- 场景:改完前端(web/admin)push main 触发 CI 部署 102,要确认 5180 上跑的是本地这份代码,而不是"有响应但旧 bundle"。
- 手法(三连,可脚本化成轮询):
  1. `curl -s http://192.168.0.102:5180/ | grep -o 'src="/assets/[^"]*"'` 取 index.html 引用的 bundle hash 名(assets 命中 immutable,hash 变=真的换了);
  2. `curl -s .../$bundle | grep -c '<新代码的标记文本>'`(如 zh-CN locale 里改动后的文案/新 key 名),≥1 即新代码在内;
  3. 对照:本地 `pnpm build` 产物的 dist/assets 里同名 hash 文件存在 = 部署的就是本地版本(bundle 哈希一致是铁证,比"有响应"硬)。
- 轮询骨架:每次 `index.html → bundle → grep 标记`,15s 间隔,超时 ~12min 判超时;不要拿"页面 200"当部署完成信号(nginx 常驻旧 index.html)。
- 关联:模板 B 的"制造可观测差异再轮询"是 API 层,本条是前端 bundle 层;修复前端后用户报"看不到变化"先走本条(红线 #2a)。

## react-router 版本 API 先查 d.ts 再动手(2026-10-01)

- 场景:要用 NavLink 的 `isActive` prop 做自定义激活,或依赖某 React Router API 行为。
- 手法:写码前 `grep -n "isActive" node_modules/.pnpm/react-router-dom@*/node_modules/react-router-dom/dist/index.d.ts`(或 dev 版 index.js 的 NavLink props 解构),确认当前版本(本项目 6.30.4)有没有该 API——6.30.4 已把 `isActive` 从 NavLink props 移除,只剩 `className/children/style` 函数收 `{isActive,isPending,isTransitioning}`;dist/index.js 的 `_excluded2` 数组列的就是被剥离的 props。
- 教训:按旧版 API 先写方案再回头翻源码浪费一轮;API 边界问题一律先查 installed 包类型/实现,不靠记忆。

## 双向关联表的孤儿诊断 SQL 模板(2026-08-27)

- 场景:用户报"两表数据没关联上",需要先量化是历史数据还是接口 bug。
- 手法:对双向外键表(A.x_id ↔ B.a_id)跑 6 个查询(以 assets↔tags 为例):

  ```sql
  -- 1) A 端有 x_id 且能反查到 B(正向可达)
  SELECT COUNT(*) FROM assets a JOIN tags t ON t.id = a.tag_id WHERE a.tag_id IS NOT NULL;
  -- 2) B 端有 a_id 且能反查到 A(反向可达)
  SELECT COUNT(*) FROM tags t JOIN assets a ON a.id = t.bound_asset_id WHERE t.bound_asset_id IS NOT NULL;
  -- 3) 双向完全一致(真关联)
  SELECT COUNT(*) FROM assets a JOIN tags t ON t.id = a.tag_id AND a.id = t.bound_asset_id WHERE a.tag_id IS NOT NULL;
  -- 4) A 端孤儿(x_id 指向不存在的 B)
  SELECT COUNT(*) FROM assets a WHERE a.tag_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tags WHERE id = a.tag_id);
  -- 5) B 端孤儿(a_id 指向不存在的 A)
  SELECT COUNT(*) FROM tags t WHERE t.bound_asset_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM assets WHERE id = t.bound_asset_id);
  -- 6) 单向不一致(A 有 B 无 / B 有 A 无)——通常是写入路径回填缺失
  SELECT COUNT(*) FROM assets a WHERE a.tag_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tags t WHERE t.id = a.tag_id AND t.bound_asset_id = a.id);
  SELECT COUNT(*) FROM tags t WHERE t.bound_asset_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM assets a WHERE a.tag_id = t.id AND a.id = t.bound_asset_id);
  ```
- 解读:1=2=3 时双向一致;4>0 是 A 端写入脏数据(基本不可能,FK 约束挡);5>0 是 B 端写入脏数据(同 FK);6>0 是写入路径缺回填——本次资产↔标签 124 条 B 单向孤儿即此因(CreateTag 完全无反向回填)。
- 教训:看代码前先量化"哪一侧有数据、哪一侧没数据",再针对性补写入路径;不要上来就写清理脚本(可能误删真数据)。
- 工具路径:102 真库 = `ssh imeepos@192.168.0.102 'docker exec -i boss-infra-postgres-1 psql -U boss -d boss' <<'SQL' ... SQL`,SQL 字符串字面量用单引号避免叠引号(红线 #9a)。
- cdp 断言「点击后才异步加载的抽屉/弹层」:cdp-admin-capture 的 --eval 不 await Promise,返回 {} / undefined;用同步 busy-wait(`(()=>{const end=Date.now()+3000;while(Date.now()<end){if(ready())break}})()`)在单个 eval 内等 DOM 就绪再返回断言文本。
- 验证"弹窗被遮罩挡住/无法点击"类 bug:cdp-admin-capture --eval 返回 `elementFromPoint(弹窗中心)` 的命中结果 + `getComputedStyle(el).zIndex`,命中遮罩=bug 复现,命中弹窗内部控件=修复生效;比截图更硬且不依赖模型读图能力(GLM 系不支持 read_image)。
- 临时回退工作区文件做红/绿对照后恢复:禁用 `git checkout -- <file>`(会把该文件全部未提交改动一起抹掉);用 `git stash push <file>` → 测试 → `git stash pop`,或 sed 双向改回。
- 102 真实环境 API 闭环测试:admin 用 X-API-Key: boss_a852...(test-accounts.json),worker 端同 header 用 worker 段 key(只认 subject=worker);SQL 直查 docker exec -i boss-infra-postgres-1 psql -U boss -d boss(容器名不是 pg,库名=user 也是 boss);部署验证看 docker images tag=commit sha + 容器 Up seconds。
- cdp-admin-capture 自定义端口 dev server:必须显式 `--base http://localhost:<port>`(默认 5173);否则采集落在不存在的端口,tokens/交互断言全空,像"代码没生效"。断言输出里先带 location.pathname/url 字段自证页面正确,再信后续数值。
- 冷启耗时测量(Android): `adb shell am start -W -S -n <pkg>/.MainActivity` 读 TotalTime;若 TotalTime=0 且 topResumedActivity 是 GrantPermissionsActivity,是运行时权限弹窗顶替了前台 Activity(启动 intent 被投递给顶层实例),先 `pm grant <pkg> <perm>` 再测(2026-08-27 user Android D10 实测)。
- Release 清单物证: `aapt2 dump` 对二进制 manifest 有时静默无输出,改用 build-tools 内 `aapt dump xmltree <apk> AndroidManifest.xml`(v1)可读(2026-08-27 cleartext 收敛物证实例)。
- 编辑 openapi yaml 新增 path 块后,跑一次依赖它的生成脚本(gen-bossctl-routes.mjs)并肉眼审查输出 diff——YAML 缩进错(如把 get 顶成 2 空格)会表现为生成器输出 summary 错位/路由丢失,一次暴露结构破坏。
- Android 构建前置检查(worktree 场景): local.properties 是 untracked 不进 worktree,从主树 cp;JAVA_HOME 用 /opt/homebrew/opt/openjdk@17;缺任一都只有 "SDK location not found / Unable to locate Java Runtime",与代码无关却最耗定位(2026-09-05 实例)。
- apitypes 信封 code 与 portalWorkerDo 的 _status 反序列化后是 int(非 float64),测试断言数值写 asNum(v) 兼容类型开关,别假定 JSON 数字全是 float64。
- 接手"只被正则/生成器消费过、从未被真解析器加载"的 YAML/OpenAPI 资产时,先写一次性全树扫描(逐文件 yaml.Unmarshal + 收集 paths/components 键 + 遍历全部 $ref 核对目标存在),一次列出语法错误/重复键/错位块/悬空引用全集;逐个报错改一版跑一版测试会拖十轮(2026-09-06 openapidoc 聚合器接入实测:admin/user/worker/open 共 10+ 存量坏点一次扫清)。
- `make check` 输出要用 tail 看完整结尾,不要 grep 自己预期的 "X OK" 标记——grep 模式外的门禁步骤(bossctl-routes-check 等)失败会被过滤吞掉,绿了假象直到合并后才炸(2026-09-06 实例)。
- Compose ExposedDropdownMenuBox(menuAnchor PrimaryNotEditable 挂在可编辑 TextField 上)真机行为:未聚焦字段首点=仅聚焦(不弹层),聚焦后再点=切换浮层;断言开层逻辑前先把字段弄到聚焦态(2026-08-28 MI 9 SE 实测)。
- Compose 字段值读不出时(uiautomator text 属性空)用浮层行内容反推值:候选=值的前缀命中项,浮层全量=值为空、仅一行命中=值为对应前缀;另清除钮 content-desc「清除」出现即值非空,都是免读文本的行为判别(2026-08-28 社区联想实测)。
- 真机验证"点建议行整体替换"免 IME 注入法:值空→点字段开全量浮层→直接点某行→清除钮出现+浮层收起即选择路径生效;绕开 MIUI 注入不稳(2026-08-28 实测)。

## 柜面收款轮(2026-08-29)——真实 UI 断言/镜像取证/卷隔离诊断

- cdp-capture 的 `--eval` 序列在 **location.href 跳转后仍按序执行**(CDP 会话存活,settle 等待新页),且每次 evaluate 的返回值直接打印到 stdout——组合出零新代码的真实 UI 断言:① 写 localStorage token ② 跳目标页 ③ 找到业务按钮 click ④ `JSON.stringify({键: innerText.includes(...)})` 返回断言对象;配合截图,人与机器都拿到证据(2026-08-29 柜面收款弹窗断言实证:{"客户下拉":true,"弹窗":true})。
- `df` 显示满、`du` 几乎为空 = **已删除文件句柄仍被进程持有**(docker 构建崩溃瞬时态典型);按 du 结论"空间充足"是误判,等待构建进程退出或重启 docker 释放,别急着清真实数据。
- 验证 ldflags `-X` 注入是否生效:对两个镜像分别 `docker run --rm --entrypoint sh <img> -c "strings 二进制 | grep -E ^格式\$ | sort -u"` 再 `comm -23` 求差,独有行即注入值;比翻 CI 配置可靠(2026-08-29 授权公钥取证实测)。
- **docker 卷按 compose 项目名隔离**:目录名=项目名=卷名前缀(CI 在 boss-app → 卷 boss-app_boss_license_data;手工在 deployments 跑 → 新建 deployments_* 空卷)。接手容器先 `docker inspect --format '{{.Mounts}}'` 核对卷身份;跨项目 up 会静默挂新空卷,持久化文件"消失"(2026-08-29 license.json 事故,postmortem 0010)。
- pgxmock 的 WillReturnRows 对 NULL 列不触发真驱动 Scan 校验——**可空列 Scan 缺陷(如 `Scan(&int64)` 撞 bill_id NULL)单测全绿,真库必炸**;制度:可空列一律 `SELECT COALESCE(col,0)` 再 Scan(充值流水退款事故实证)。

## 2026-08-29 验收/巡检脚本取 JSON 字段:先打印原文再接字段;sampleIds 先读 SQL 再下结论
- 一次性 shell+python 脚本先 dry 跑一轮只 `head -c 200` 打印响应原文,肉眼确认 envelope 形状({code,data,msg})后再写字段提取;envelope 取字段先 `d.get('data',d)`。
- 巡检/告警输出里的 `sampleIds`/`sample [N]` 是**命中行自身的主键**,不是被引用列的值;下结论前先读巡检 SQL(如 pg_patrol.go)确认 array_agg 的是哪一列。本轮 sample [100] 被误读成 customer_id,直查库才发现 100 是 lo_accounts.id。

## 2026-09-06 MCP/HTTP 客户端轮——WriteHeader 嗅探坑 + MCP stdio 手写子集
- Go net/http handler **显式调用 WriteHeader 后不再做内容嗅探**,Content-Type 缺省 text/plain(即使 body 是合法 JSON);判定响应形状时 content-type 命中与首字节嗅探({/[)应取"或",误判交给 json.Unmarshal 失败兜底,不要让 text/plain 短路 JSON 解析(bossmcp 业务错误信封漏解析实证)。
- MCP server 无需官方 SDK:**stdio 每行一条 JSON-RPC 2.0**,实现 initialize(回显客户端 protocolVersion)+ tools/list + tools/call + ping 即可被主流客户端连;通知(无 id)静默不回包;工具执行失败按规范走 result.isError=true 而非 RPC error;日志只准写 stderr(stdout 是协议通道)。cmd/bossmcp 是仓内可抄的完整参照。

## 2026-09-06 dsh 桥接外部 MCP server(bossmcp 实测)
- dsh 接外部 MCP 三件套:① profile 目录(~/.dsh/profiles/<name>/)package.json 的 dependencies 加 `"@deepseek-ai/dsh-mcp-client": "link:<vendor>/node_modules/@deepseek-ai/dsh-mcp-client"` 后 pnpm install;② cordis.patch.yml 写 `insert: [{id: mcp-boss, name: '@deepseek-ai/dsh-mcp-client', config: {serverName, transport: stdio, command, env, failOnStartupError}}]`;③ 无头单发 `dsh --profile <name> "任务"`。工具名 = `mcp__<serverName>__<原始名>`。
- **cordis id-targeted 覆盖是整体替换**:--patch 里按 id 覆盖某条目时 config 必须写全(只写变更字段会把其余字段抹掉,schema 校验直接红);没有深合并。
- dsh headless 单发默认模型走 settings.yaml agent-default-model,LLM key 经 `launchctl setenv` 注入 GUI 系进程;shell 里跑无头要手动 `BIGMODEL_API_KEY=$(launchctl getenv BIGMODEL_API_KEY)` 带上(2026-09-06 bossmcp 鉴权实测:有效 key 双端真数据,无效 key isError 原文 HTTP 401 invalid api key 透传到 agent)。

## 2026-09-06 dsh profile 运维——装依赖/HMR 重载
- 给运行中的 dsh profile 加依赖,**全量 pnpm install 大概率走不通**(npmmirror 缺历史版本元数据,--offline 也缺);最小侵入 = `ln -sfn <vendor>/node_modules/<pkg> <profile>/node_modules/<pkg>`,等价 link: 依赖,Node 按 realpath 向上解析不破坏既有依赖。
- mcp-client 的 HMR 只对**语义 diff** 重载(改 config 值),纯注释追加不触发;验证重载看 bossmcp 子进程 pid 是否变化(ps aux | grep bossmcp),重启后子进程自动从新二进制拉起。

## 2026-09-06 冒烟脚本轮——while 计数器与字段序
- `python | while read` 管道会让 while 进子shell,计数器改动全部丢失(退出码也对不上);用进程替换 `done < <(python ...)` 保持当前 shell。
- 断言输出协议定死"状态在前":首字段必须是 PASS/FAIL,壳侧 `case "$status"`,别让 tag 打头否则 PASS/FAIL 全被误读成 detail。

## 代码生成器输出必须过 go/format.Source(gofmt 门禁 vs 漂移门禁打架)

场景 → 生成 Go 代码的生成器(genrouteperms/routes_gen 等)渲染手写缩进,`gofmt -l` 会微调对齐,导致 make lint 红、再生成又让 --check 漂移门禁红,来回拉锯。
做法 → 渲染完统一 `format.Source(src)` 再写盘/比对:生成物与 gofmt 天然一致,两道门禁同时稳定。参照 scripts/genrouteperms/main.go render()。

## 长门禁的机械验收模式:HEAD 键控 rc（2026-08-30 上线审计轮）

场景 → 验收器（devloop_accept 等）内置等待窗口跑不完全量门禁;又必须防"陈旧绿"和自报通过。
做法 → 后台跑全量门禁,产物写 `.devloop/gates/<task>-$(git rev-parse --short HEAD).{log,rc}`;账本验收命令只做一件事:断言"当前 HEAD 的 rc 文件存在且 =0"。HEAD 前进旧结果自动失效必须重跑;日志随时人工抽查;目录加 `.git/info/exclude` 不污染 status。铁律:wrapper 退出码 0 不算数,门禁本体 rc 才算（本轮 T2 wrapper 0/门禁 2 的教训）。

## 子代理跨目录任务派发模式（2026-08-30）

场景 → worktree 协议要求目录在仓库外,但子代理沙箱 scope=主仓库目录且审批禁用,写不出去;`git worktree add` 会留下「分支已建/工作树缺失」半完成态。
做法 → 主会话先在工作区内建 worktree（`git worktree add .worktrees/<name> <branch>`,目录 info/exclude 排除）,再派发/续话给子代理并给出绝对路径;子代理大文件成品可写 /tmp,主会话 shasum 校验后 cp 进 worktree commit。

## 沙箱会话内 Go 编译与 worktree 形态（2026-08-30）

场景 → 会话沙箱只放行工作区内写入;go 默认缓存(`~/Library/Caches/go-build`、`~/go/pkg/mod`)全在工作区外,`go build` 报 operation not permitted;worktree 协议要求仓库外兄弟目录同样写不出。
做法 → ① `export GOCACHE=<工作区内已排除目录>/$USER-cache` 再编译(模块只读缓存可复用,stat cache 写失败提示非致命);② worktree 建仓库内注册形态 `git worktree add .wt/<name> -b <branch>` + `.git/info/exclude` 加 `.wt/`,commit 输出方括号确认分支名,收尾 `git worktree remove` + `git branch -d` 即净;worktree add 半完成态(分支在目录无)先 `git branch -D` 再重试。铁律:嵌套 worktree 必须是 git 注册的(有 .git 文件),手建同名目录再 cd 进去 commit 会静默落主树。

## 常驻服务落地 102 的端口与验证纪律（2026-08-30）

场景 → 102 是多会话共享机,18081/18082/18083 这类「顺延端口号」会在你 ss 检查和起服务之间被并行会话抢走;systemd 服务部分子监听失败时整体仍 active,假绿。
做法 → 选端口前 `ss -ltn` 全量列出已监听端口挑真空闲的(不要从被占端口顺延);服务启动后 journalctl -u <name> 必须逐行确认每个监听成功、curl 打通每个端口;telnet 类服务用 `/dev/tcp` 或真实客户端走一遍完整协议(login→命令→应答)。镜像化服务先查部署形态(compose 服务表/Dockerfile BINARIES/CI workflow),「二进制在镜像里但 compose 没这个服务」= 功能从未上线。
- worktree 无 node_modules 时(2026-09-01 产品绑定轮):`ln -s 主checkout绝对路径/node_modules node_modules`(仓库根)+ `ln -s 主checkout/web/admin/node_modules web/admin/node_modules`,pnpm typecheck/vitest/vite build 全部可用,免 pnpm install;用完随 worktree 一起 remove(链接是 ignored 不挡 remove)。
- make check D 项撞号裁决实操(2026-09-01):先 `git ls-tree main -- migrations/ | grep <号>` 确认自己已合并+已落库,再 `git ls-tree <对方分支> -- migrations/` 确认对方未合并——按「已合并者优先」判对方让号,session_link_send 发提醒附改名目标号,自己继续合并不被环境性 D FAIL 阻塞。
- 部署后验证三件套一轮过(2026-09-01):后台轮询 `healthz` 的 commit 字段等新 sha(取代人工猜)→ 用 test-accounts.json 的 admin api key curl 新 API 冒烟(正路径+错误路径都要打)→ `curl 5180` 拿 index.html 里 assets/index-*.js 再 grep 新 UI 文案确认前端指纹。

## 2026-09-01 部署后特征串验证必须扫懒加载分片(admin web)

- 场景:验证某前端特性是否已部署到 102(5180),往 bundle 里 grep 特征串(如新端点路径)。
- 坑:Vite 按页面懒加载分包,index-*.js 只是壳(440KB),页面代码在 `assets/<Page>-<hash>.js` 分片;只 grep index 会假 0 命中,误判「部署脑裂」白查一轮(本轮 boss-server 已新、admin-web 实际也已新)。
- 正解:先 `grep -o 'assets/[A-Za-z0-9_-]*\.js' index.js | sort -u` 拉分片清单,再逐片 grep 特征串;或直接用 `bash scripts/ops/verify-deploy.sh --expect-sha <sha>`(文件名一致性口径)+ 分片内容 grep 双确认。
- 注意 grep 结果里的文件名有语义:命中分片名可能与特性所在页面组件不同名(共享 chunk 按任一成员命名,如 customer 抽屉代码在 RegistrationQueueDrawer-*.js),文件名不像≠没部署。
查 102 上服务真实运行配置:先 ps 拿 pid,再 ssh 执行 tr '\0' '\n' < /proc/<pid>/environ 看 true env——systemctl status/cat 可能显示 inactive 或与实况不符(进程另有启动来源),部署文档也可能滞后(2026-09-03 验证有效)。

## DSH 宿主数据目录与持久化登记排查(2026-09-04 实证)

场景 → 验证 workspace_session_manage archiveSession 这类宿主侧操作是否真实生效;workspace_list/session_link_list 均不过滤归档态,GUI 截图受模型图像输入限制时不可依赖。
怎么用 → ① lsof -nP -iTCP:18181 -sTCP:LISTEN 拿 pid;② ps -p <pid> -wwE -o command= 看 DSH_HOME(18181 实例是 /Users/imeepos/.dsh/dsh012-clean,不是默认 ~/.dsh,后者是另一实例的旧数据);③ 直读 $DSH_HOME/storages/workspace.json 验收,结构 {unit, global:{initialized,workspaceIds,archivedSessionIds}, tables:{workspaces:{<id>:record}}},归档登记在 global.archivedSessionIds(跨工作区全局集合)。
注意 → archiveSession 响应的 archivedSessionIds 是全局登记表全集(含历史归档约 250 条),不是本次影响集;逐个归档 N 个会话就发 N 次调用,每次只带一个 sessionId,别试图一次传数组。
