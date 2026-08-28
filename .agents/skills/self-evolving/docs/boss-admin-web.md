# boss 项目速查手册（web/admin 前端）

> 只记"需要查证才知道"的事实；常识与 Read 即得的代码结构不记。
> 来源：2026-08-18 app shell / geo CRUD / geo 页重构任务期间的实际查证；事实漂移时以代码与 102 库实测为准。

## 冒烟登录

- 账号：`admin / admin123`，角色 `sysadmin`（由 `scripts/devseed/main.go` 写入 PG）
- 已部署后端：`http://192.168.0.102:28080`（vite dev 默认代理目标）
- 本地联调覆盖：`cd web/admin && BOSS_API_TARGET=http://127.0.0.1:18080 pnpm dev`
- mock 服务（`scripts/mock-admin.sh`，端口 8092）只有 user/worker 端登录，**不能**用于 admin 登录冒烟
- **dev 免登录**：~~`node web/admin/scripts/dev-token.mjs`~~ **已失效(HTTP 404,脚本仍按旧前缀 /auth/login 请求,2026-08-20 查证)**。现用:`curl -s http://192.168.0.102:28080/api/admin/v1/auth/login -X POST -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}'` 取 `data.token`,浏览器 localStorage 注入 `boss.token` + `boss.servers`(JSON 数组含 baseUrl) + `boss.server.active` 后直访目标页;登录页表单无 id/selector,不要走表单 eval
  - **2026-08-28 修正**:`boss.servers` 数组元素必须有 `id` 字段(string),`boss.server.active` 存的是该 **id 而非 name**;缺 id 会被 serverConfig.readStored 过滤掉 → 页面弹"未配置服务端"并被弹回登录页。可用注入:`localStorage.setItem('boss.servers',JSON.stringify([{id:'s102',name:'102',baseUrl:'http://192.168.0.102:28080'}]));localStorage.setItem('boss.server.active','s102')`
  - React 受控输入用 cdp eval 填值必须走原生 setter + input 事件:`Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(el,v); el.dispatchEvent(new Event('input',{bubbles:true}))`;注意 Dropdown 组件渲染的是 button 不是 input,querySelectorAll('input') 下标会跳过下拉位
  - **2026-08-27 修正(servers 先于 token)**:`?token=` 一次性覆盖只写 boss.token;若 boss.servers 未配置,AuthGuard 启动预取 /auth/me 网络失败 → adminLogout() **静默 removeItem('boss.token')** 并弹回 /login,表象是"token 参数没生效"。cdp 采集固定顺序:先访任意页(如 /login)eval 注入 boss.servers+boss.server.active,再 `location.href='/目标页?theme=dark&lang=en-US&token=<jwt>'`;boss.token 莫名消失先查这条登出路径

## 门禁

- `pnpm typecheck && pnpm test && pnpm build`（vitest 4 文件 20 用例；build 含 tsc --noEmit）

## 设计资料位置

- 外壳规格：`docs/admin/design-spec.md`（亮/暗 token、尺寸、断点，逐条可查）
- 品牌 token 源头：`docs/design/visual-design-prompts.md`
- 原型页面：`docs/admin/*.html`（jQuery 草稿，45 页）
- 原型图标：`docs/admin/icons/*.svg`（13 组）+ `docs/admin/icons/items/*.svg`（49 页），stroke 色写死 `#a6adb4`，换色须用 CSS mask

## 契约文档（AGENTS.md 强制，开工前必读）

- `docs/contract/terms.md`：订单 12 环节、状态枚举
- `docs/contract/domain-map.md`：能力域/Agent/internal 包/页面 四套命名对齐
- `docs/contract/fields.md`：页面列名 ↔ 字段名 ↔ 状态枚举

## 菜单图标体系（2026-08-18 查证）

- 菜单源：`web/admin/src/router/menu.def.ts`（13 组 49 项，key 唯一）
- 运行时图标在 `web/admin/public/icons/`：组图标 `<组id>.svg`（13 个），项图标 `items/<项key>.svg`（49 个）；新增菜单项必须同时补对应 SVG，否则侧栏该行无图标（geo 曾漏）
- 渲染走 `MaskIcon`（`layouts/icons.tsx`）：CSS mask + `background: currentColor`，颜色随所在按钮状态自适应，与 SVG 文件内写死的 stroke 色无关
- 顶栏 TopNav 组按钮也渲染组图标（size 16 + gap 6）；侧栏组图标 size 14、项图标 18

## 前端约定（本次确立）

- 主题：`localStorage('boss.theme')` = light/dark，`documentElement[data-theme]` 驱动 `src/theme/tokens.css` 变量；`index.html` 内联脚本首屏前写入防闪白
- 语言：`localStorage('boss.locale')`，三语言 zh-CN/en-US/ms-MY，新增文案须同步三个 locale 文件
- URL 一次性覆盖（`src/lib/urlPrefs.ts`）：`?theme=`、`?lang=`、`?token=`（dev 限定）启动时写入 localStorage 后 replaceState 抹除；权威存储仍是 localStorage
- 可视化验证：`.agents/skills/self-evolving/scripts/cdp-capture.mjs`（零依赖 CDP 截图）
- 硬性规则（用户明令）：任何组件/页面必须同时考虑多主题与多语言——文案一律走 i18n（types.ts + 三份 locale 闭环），颜色一律走 `tokens.css` 变量/主题 token，禁止 JSX 内硬编码文案或色值；交付前 grep 裸字符串与裸色值

## 不可更改事实（登录/主题/组件契约，2026-08-27 查证源码后固化）

- **AuthGuard 会清 token**：`src/layouts/AuthGuard.tsx` 启动预取 `/auth/me`，任何失败（含 servers 未配置导致 apiBaseUrl 为空的网络错）走 catch → `adminLogout()` → `removeItem('boss.token')` 并 `<Navigate to="/login">`。所以 boss.servers 必须先于受保护页启动存在，`?token=` 单独直访业务页必被弹回且 token 消失
- **无 vite 代理通道**：`api/client.ts` 注释明令"禁止再引入 vite 代理通道"；全部请求 = `apiBaseUrl()`(来自 boss.servers) + `API_PREFIX(/api/admin/v1)` 直连，102 后端已配 CORS。vite.config 只有 port 5173，不配 proxy
- **urlPrefs 边界**：`?theme=|?lang=` 双端生效，`?token=` 仅 `import.meta.env.DEV`（生产剥离不写入）；权威存储是 localStorage，URL 只一次性覆盖
- **表单输入唯一入口** `components/ui/input.tsx`（Input）：h-8 w-full 全 shell-input-* 令牌含 focus/disabled 态；裸 `<input className="w-full">` 暗色下 UA 默认样式，适配任务一律替换
- **断言稳定选择器契约**：Drawer 根 = `aside[role=dialog]`（标题在其 h3/aria-label）；Dropdown 触发器 = `button[aria-haspopup=listbox]`（渲染的是 button 不是 input）；选项 = `[role=option]`。CDP 断言只准用语义锚点
- **主题令牌只有两块**：`src/theme/tokens.css` 仅 `:root[data-theme='light']`/`'dark'` 两个定义块；引用任何 var 前先 grep 该文件确认双主题都有，styles.css 的 `--color-*` 是不分主题的静态色
- **免登录采集脚本**：`scripts/cdp-admin-capture.mjs` 已封装 login 取 token + servers/token 注入 + theme/lang 透传（用法见 templates 模板 K）

## geo 域速查（2026-08-18 查证）

- 页面 `/base/geo`（国家+区划双页签）；API 前缀 `/api/admin/v1/geo/*`（全站统一 `/api/admin/v1`，见 serverConfig.ts API_PREFIX），门禁 `menu:geo`（仅 sysadmin，迁移 000039 授予）
- "删"= 软删除：`PUT /geo/{countries,subdivisions}/:code/active {"active":false}`，契约 fields.md 1.5.1 规定停用码保留不物理删
- 国家关联属性整体替换：`PUT /geo/countries/:code/attrs`，请求体 `timeZones: string[]`、`callingCodes: string[]`、`currencies: {currency,isPrimary,minorUnit}[]`（字段类型以 `internal/domain/geo/geo.go` struct 为准）
- 译名 locale 用 BCP-47 风格 `zh-Hans`/`en`（种子数据口径），不是 `zh-CN`
- 102 库有停用态冒烟数据：国家 `T1`、区划 `T1-TS`（is_active=false，勿清理也勿复用为主数据）
- 前端样板：主题化 CRUD 页全套见 `web/admin/src/pages/base/geo/`（antd Pro 布局 + 双主题），通用右侧抽屉 `web/admin/src/components/Drawer.tsx`

## 102 库与权限架构事实（2026-08-18 查证）

- `schema_migrations.version` 是 TEXT（如 `000039_geo_menu`），不是整型；手工补迁移后须 INSERT 记录版本
- 权限模型：`HasPermission` 只查 `role_permissions` 显式行（internal/domain/user/pg.go），**sysadmin 没有隐式全权**——新菜单必须同时有"permissions 行 + role_permissions 授权"才会放行
- DB 直连（本机无 psql 时）：`host=192.168.0.102 port=25432 user=boss password=boss dbname=boss`（与 configs/config.example.yaml 同源）；用 /tmp 临时 go 程序 + pgx
- 冒烟账号 admin 的 account_id=103（102 库）

- 102 远程环境（CI 自动部署）：`http://192.168.0.102:28080`，冒烟账号 `admin / admin123`（sysadmin，accountId=103；2026-08-21 login 接口实测可用）。口令权威=仓库 `deployments/app.env` 的 `BOSS_ADMIN_PASSWORD=admin123`，与 `docker-compose.102.app.yml`、`scripts/devseed/main.go` 三处一致；app.env 已入库系内网私有仓库裁定，转公网前必须移出。geo 维护页在 基础配置→国家与行政区划（menu:geo）。

## 页面状态与分页约定（2026-08-18 geo 首例）

- 列表页"刷新后条件不变"：`src/lib/useQueryState.ts`（useQueryState 字符串 / useQueryInt 正整数），底层 useSearchParams + replace 不产生历史；值为 fallback/空时删参保持 URL 干净
- 分页组件：`src/components/Pagination.tsx`（i18n 文案由调用方传入，geo block 的 prev/next/perPage）
- 语义区分：lib/urlPrefs.ts 是"一次性 URL 覆盖写 localStorage 后抹除"，与页面持久状态方案相反

## 自研通用组件约定（2026-08-18 geo 分页/下拉沉淀）

- `src/components/Dropdown.tsx`：全站唯一下拉实现（触发器 + 浮层 listbox + 描边 SVG 打勾 + 点击外部收起）；禁止新增原生 `<select>`；CountryForm 内两个遗留原生 select 待替换
- `src/components/Pagination.tsx`：对齐 antd 规范（首末页恒显 + 当前±2 + 省略号、区间文案 rangeText、size changer、>10 页出现跳转、aria-current）；文案由调用方 i18n 传入（geo block 的 prev/next/perPage/rangeText/jumpText/pageUnit）
- 图标规格：描边 SVG，viewBox 24 / stroke 1.8-2 / round cap / currentColor，显示 14px；禁止文字字形当图标
- 主题：自研组件按 geo.css 模式自带 `:root[data-theme='light'/'dark']` 组件级令牌块；引用任何 var(--x) 前先 grep theme/tokens.css + styles.css 确认存在（曾引用不存在的 --shell-bg 静默翻车）
- 可复用外壳令牌（双主题）：--shell-card-bg / --shell-card-border / --shell-content-text / --shell-group-title / --shell-menu-hover-bg / --shell-fab-bg（亮藏青/暗金）/ --shell-fab-bg-icon；focus 描边 --color-border-focus（styles.css，不分主题）

## 招商入驻域速查(2026-08-22 查证,迁移 000098)

- 公开申请页 `/partner/apply`(免登录,登录页有入口);admin 审核页 `/org/partner`(menu:partner,org 组)
- 企业工作台组 `/partner/{home,staff,orders}`:仅 partner_admin/partner_staff 可见;sysadmin 不见该组(无 legal_entity 归属)
- 审核通过 = 建 legal_entities(P-<信用码>)+ 管理账号 pt_<信用码后8位>(role partner_admin,legal_entity_id 绑定);初始口令 12 位随机,仅审核响应/弹窗展示一次
- 102 冒烟数据(勿清理也勿复用):legal_entity 7 "Smoke Partner Co"(信用码 SMOKE123456),账号 pt_KE123456(口令见会话记录)/pt_smoke_staff1;申请 1=已通过,2=已驳回,3=已通过(CDPVERIFY9)
- 102 admin 口令实为 admin/admin123(compose environment BOSS_ADMIN_PASSWORD 权威;deployments/app.env 的 Boss-admin-2026 已过时)
- admin-web 前端部署在 http://192.168.0.102:5180(nginx 同源代理 /api/);CI deploy-102 的 compose 常把容器留在 Created,需 ssh 上去 docker start
- git remote 名是 `gitea`(ssh://git@192.168.0.102:222/sker/boss.git),没有 origin;push gitea main 触发 CI(后端 28080 与前端 5180 一起出新构建)
- 部署验证(2026-09-22 装维队):push 后查 gitea actions 最新 run——DB `action_run` 按 `index` 排序,status 3=被更新 push 取代(非成功),deploy 成功会重建镜像(tag=commit sha)并重启容器;并行会话推进 main 会取代我的 run,等最新 main 的 run 完成再验接口。验证后端新路由:`curl http://192.168.0.102:28080/api/admin/v1/<新路径>` 未部署=404 page not found,部署后=401/业务信封。

## 用户详情抽屉冒烟数据(2026-09-26 查证,102 库)

- 路由 `/bss/user`(menu key user);详情抽屉数据源 `GET /users/{customerId}`
- 客户 213(采购经理·王):富数据——addons 10 条(>默认上限 5,可验"查看全部/收起")、orders 45、faults 3、complaints 1、plans 1(ACTIVE)、addresses 4
- 客户 215(王经理):仅 orders 4 条,其余段全空——空态断言对象;orders 4 ≤ 5 可验"无展开按钮"
- complaints 同源双口径:faults 段=报障(`用户报障: ` 前缀已 strip 成 no_internet/slow/...+装维 6 码)、complaints 段=投诉(`用户投诉: ` 前缀保留),见 fields.md §2.1.1
- 当前 harness 模型(deepseek-v4-flash)不接受 read_image,冒烟一律 DOM 断言(innerText/querySelector)+ --logs 查 console/网络,截图仅供人工复核

## 侧栏激活与 react-router 事实(2026-10-01 查证源码后固化)

- 依赖版本:react-router-dom **6.30.4**(pnpm .pnpm 目录);该版 **NavLink 已无 `isActive` prop**(d.ts 的 NavLinkProps 只剩 children/className/style/caseSensitive/end/viewTransition),className 函数收 `{isActive,isPending,isTransitioning}`
- NavLink 默认 `end=false` 前缀匹配:访问 `/boss/site/cats` 时 `/boss/site`(官网内容)也被判 active → 兄弟菜单双击亮
- 侧栏激活判定统一走 `router/menu.def.ts` 的 `isNavActive(to,pathname)`(精确路径激活;深层路由自身是其它菜单项完整路径不高亮父项);Sidebar 用 Link + 显式 `aria-current={active?'page':undefined}`,不靠 NavLink 前缀匹配
- 同源前缀兄弟项:官网内容 `/boss/site` ↔ 官网分类 `/boss/site/cats`;`/bss/marketing` ↔ `/bss/marketing-recon`(均已按 isNavActive 精确化)

## 部署验证事实(2026-10-01 实证)

- 前端确认上线:远端 `index.html` 的 bundle hash 名 ↔ 本地 `pnpm build` 产物 dist/assets 同名 hash;两者一致=部署的就是本地版本;再 grep bundle 内新标记文本双重确认
- 102:5180 nginx assets 命中 immutable(max-age=31536000),bundle 名带 hash,index.html 已 no-cache——修完前端 push 后看 index.html 引用的 hash 是否更新

## 遗留缺口(2026-10-01 查证,勿与已修项混淆)

- `/boss/site`(官网内容列表,sitePage)的 `columns` 数组仍是**裸字段标识符**(title/slug/category/status/publishedAt/version),zh-CN 界面下列头裸英文——siteCatsPage 与 knowledgePage 均已本地化(knowledge 页由 feat/knowledge-i18n-theme 修复,columns ['标识码','标题','内容','状态','版本']),sitePage 待同样处理

## API 文档页(/base/apidocs,2026-09-06 接入)

- 页面:GET /docs/openapi?portal=admin|user|worker|open 取聚合契约,Swagger UI 渲染;四端 tab 切换
- 免登录采集:cdp-admin-capture --path /base/apidocs 即可(sysadmin 默认可见 channel 组)
- 端点验收:scripts/ops/apidocs-acceptance.sh(envelope/四端无外部 $ref/401 门禁/非法 portal)
- 注意:102 license 失效时该页接口与其他业务接口一样 403 LICENSE_REQUIRED;恢复路径=release-platform 铸码(ops-renewal-30d batch 先例)→ /license/activate 兹码,见 adopted/2026-09-06-api-docs-openapidoc.md
