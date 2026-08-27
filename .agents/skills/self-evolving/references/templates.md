# 固定模板（解题思路 + 回放脚本骨架）

> 2026-08-28 固化。来源:cms 域端到端交付与两轮 102 回放缺陷修复。
> 模板是骨架不是法律:按域裁剪,但步骤顺序与"每步验证"的纪律不变。

## 模板 A:新增后台业务域端到端 checklist

按序执行,每项完成即验证提交(type(scope) 规范):

1. **契约先行**:`docs/contract/fields.md` 加一节(页面列↔字段↔枚举三列对齐)、
   `domain-map.md` 登记、`terms.md` 登记状态枚举;不可逆裁定写
   `docs/notes/adopted/<date>-<topic>.md`(why + 放弃了什么)。
2. **迁移**:查号(`ls migrations | tail` + 跨未合并分支 `git ls-tree`)→ up/down 成对;
   `make contract-sync` 的 D 项会机械拦撞号。
3. **权限码**:菜单页必须同步登记 `permissions` + `role_permissions`(sysadmin),
   先例 000039 geo_menu / 000135 cms_menu;漏掉 = 102 回放 403。
4. **域包**:`internal/domain/<x>/` model+validate+PGStore;错误 sentinel 三件套
   (NotFound/Taken/Invalid),在 `internal/pkg/httpx/error.go` 登记映射;
   INSERT 与 UPDATE 对同一状态字段的副作用必须对称。
5. **路由**:handlers + `register<X>Routes` 挂 root.go;**A 检查源是
   `api/openapi/admin.yaml` + `api/openapi/admin/<x>.yaml`**,不是 bossctl catalog;
   `cmd/bossctl/routes_admin.go` 是 CLI 展示层,两处都要加。
6. **前端中央登记**:menu.def.ts / App.tsx 懒加载分支 / i18n types+三语言,
   压成独立小提交,不埋进页面实现。
7. **页面**:抄最近的同型页(如 pages/boss/site);下拉一律 Dropdown(带 ariaLabel);
   分页文案 `pagerTexts(t.pages.company)`。
8. **门禁**:`make check`(go test+lint+contract-sync) + web 三件套
   (`pnpm --config.verifyDepsBeforeRun=false run typecheck|test|build`;
   worktree 内先 `pnpm install --prefer-offline`,node_modules 不随 worktree 走)。
9. **真机回放**:push gitea main 触发 CI 部署 102,用模板 B 回放。
10. **收尾四步**:push 分支 → 主树 `merge --ff-only` → `worktree remove` → `branch -d` + 删远端。

## 模板 B:102 真机 CRUD 回放脚本骨架

```bash
BASE=http://192.168.0.102:28080/api/admin/v1
TOKEN=$(curl -s $BASE/auth/login -X POST -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])')
# 1 创建(观察 code;403=权限码迁移漏,40900=唯一冲突,42200=校验)
ID=$(curl -s $BASE/<res> -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{...}' | python3 -c 'import sys,json;d=json.load(sys.stdin);print(d["data"]["id"] if d["code"]==0 else d)')
# 2 状态副作用断言:列表回读关键派生列(如 publishedAt/version),空=INSERT/UPDATE 副作用不对称
# 3 负路径:重复键 409 / 不存在 404 / 未授权 403 / 公开端匿名读泄露检查
# 4 清理:DELETE 或恢复原状态;测试数据不留脏
```

要点:部署是异步的,**端点可达 ≠ 新代码**;确认换新的办法是制造可观测差异
(带新字段/新语义的请求)再轮询,每 20-25s 一次。

## 模板 C:SPA 鉴权页截图验证(cdp-capture token 注入)

登录页无表单 selector,别走 eval 填表;在 /login 注入 localStorage 再跳转:

```bash
node .agents/skills/self-evolving/scripts/cdp-capture.mjs \
  'http://192.168.0.102:5180/login' out.png --settle 4500 --logs out.json \
  --eval "localStorage.setItem('boss.token','$TOKEN')" \
  --eval "localStorage.setItem('boss.servers',JSON.stringify([{name:'102',baseUrl:'http://192.168.0.102:28080'}]))" \
  --eval "localStorage.setItem('boss.server.active','102')" \
  --eval "location.href='/boss/<page>'"
```

验证不看像素看证据:--logs 里目标 API `status: 200` + console errors 为 0 即页面
数据链路通;整页 --settle 后导航可能超时,产物 png/logs 已落盘可继续分析。

## 模板 D:公开(免鉴权)端点设计三原则

1. 只读、最小投影:handler 里手动 gin.H 挑字段,不透出管理列(version/author/ownerId)。
2. 不泄露存在性:非公开状态一律统一 404,不区分"不存在"与"未发布"。
3. 挂靠既有先例:admin 前缀 public 子路由(registerPartnerPublicRoutes 模式),
   不新开无鉴权路由组。

## 模板 E:前端 UI 改动 cdp 调试固定模板(2026-08-29 固化,来源:表单抽屉化批量重构)

> 模板 C 的注入有缺漏(见下方修正);本模板是它的升级版,覆盖"填表→点按钮→断言"全链路。

**第 1 步 免登录注入(修正模板 C:servers 元素必须带 `id`,`boss.server.active` 存 id 而非 name,
缺 id 会被 `serverConfig.readStored` 静默过滤 → 弹"未配置服务端"并弹回登录页,极易误判为 token 失效):**

```bash
TOKEN=$(curl -s http://192.168.0.102:28080/api/admin/v1/auth/login -X POST \
  -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])')
node .agents/skills/self-evolving/scripts/cdp-capture.mjs 'http://localhost:<port>/' out.png \
  --eval "localStorage.setItem('boss.token','$TOKEN');localStorage.setItem('boss.servers',JSON.stringify([{id:'s102',name:'102',baseUrl:'http://192.168.0.102:28080'}]));localStorage.setItem('boss.server.active','s102');location.href='/<路由>'" \
  --settle 3500
```

**第 2 步 React 受控输入填值(直接 `el.value=` 不触发 onChange,必须原生 setter + input 事件):**

```js
const s = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set
s.call(el, '文本'); el.dispatchEvent(new Event('input', { bubbles: true }))
```

坑:Dropdown 组件渲染的是 **button 不是 input**,`querySelectorAll('input')` 的下标会跳过下拉位
——填表前先打印每个 input 的 placeholder 对齐下标,再动手(2026-08-28 券模板抽屉实测踩过)。

**第 3 步 交互断言(DOM 证据优先于截图像素):**

```js
// 抽屉/弹框打开断言(role=dialog 是项目 Drawer 组件的稳定契约)
document.querySelector('aside[role=dialog] h3')?.textContent   // 期望抽屉标题
// 提交后断言:抽屉关闭 + 数据出现在列表
'dialog=' + (document.querySelector('aside[role=dialog]') ? 'open' : 'closed')
 + ' | row=' + document.body.innerText.includes('<新数据关键字>')
```

**第 4 步 失败定位:** 加 `--logs out.json`,先看 console errors 与失败请求响应体;
负路径顺带验证(故意漏填必填 → 断言出现"请补全必填项"类文案)。

## 模板 F:解题思路固定模板(通用排障八步,2026-08-29 固化)

> 任何"东西不工作/要加新能力"的任务套这个骨架;顺序不可换,每步有产物。

1. **复现并固化**:最小可重复的命令/URL/操作;不能稳定复现先别改代码。
2. **取证不求猜**:console/--logs/HTTP 状态码/response body/curl 复打;一次拿全。
3. **读源头对契约**:行为冲突时以 `docs/contract/*` 为准;实现疑点直接读源码
   (如 serverConfig.readStored 过滤逻辑),不靠记忆拼 API。
4. **定位到唯一根因**:能一句话说清"X 因为 Y";说不清就回到 2。
5. **最小修改**:只动根因半径内的代码;顺手重构=另开任务。
6. **门禁**:typecheck + test + build;改动页面必须有模板 E 的 DOM 断言,不写"已验证"空话。
7. **立刻存档**:小簇 commit(type(scope): subject,正文写 why);worktree 内编辑完即提交,
   不留无 commit 文件过夜。
8. **反思回喂**:坑进 recidivism/lessons,可复用手法进 techniques,固定流程进本文件。

## 模板 G:admin 表单抽屉化布局固定模板(2026-08-29 固化,来源:13 处平铺表单重构)

> 全局约定:**表单一律 Drawer/Dialog,页面不平铺 form**(登录/公开申请页/富文本编辑器/搜索筛选工具栏除外)。

**列表页新建型**(先例 `pages/bss/marketing/coupons.tsx`):

- 列表卡片工具栏右侧 `+ 新建` 主按钮(`ToolbarButton primary`),点开 `Drawer`;
- 抽屉 footer 固定三件:取消(plain 类)+ 提交(primary 类);提交中 `disabled={busy}` 文案切换;
- 提交成功:关抽屉 + 重置表单 + `load()`;校验错误用 `ErrorBanner` 显示在抽屉表单下方。

**配置页摘要型**(先例 `pages/base/pushconfig/index.tsx`):

- 页面卡片 = 只读摘要:标题 + 启停徽标(Badge success/default)+ 每字段一行
  `label(w-32 灰) + value`(密钥只显"已配置",绝不回显);右上 `编辑` primary 按钮;
- 抽屉内保留原字段/开关 + 试发/试核控件(放抽屉底部 border-t 分隔区),footer 取消/保存。

**按钮/间距速查(上级叮嘱的机械化)**:

- 主按钮:`bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] h-8 px-4 text-[13px]`;
- 次按钮:`border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] ... h-8 px-4 text-[13px]`;
- 按钮文字居中,上下左右留够边距;元素间距 4 的倍数,语义亲密的用小档(4/8),分组用大档(12/16);
- 图标与周围文字比例不能失衡,不使用 emoji 图标;
- 下拉一律 `components/Dropdown.tsx`(禁原生 select),输入框统一 shell-input-* 令牌类。

**输入框组件化补充(2026-08-27 修正,来源:营销弹框适配)**:

- 抽屉/弹框表单输入一律 `<Input>`(`components/ui/input.tsx`:h-8 w-full 全令牌,双主题自适应,
  含 focus ring 与 disabled 态),**不要手写 `className="w-full"` 裸 input**——暗色下渲染 UA 默认样式,
  color-scheme:dark 兜底观感仍与 tokens 体系割裂;审计 grep:`'<input className="w-full"'`;
- 字段包装统一 `FormField`(label+必填星+hint+error),提示文案 11px 走 `--shell-group-title`;
- 选项型表单值(枚举下拉)的 options 数组禁模块级硬编码 label,组件内由 i18n 派生:
  `const opts = TYPE_VALUES.map(v => ({ value: v, label: labels[v] }))`,列表列映射复用同一数组。

## 模板 H:Stripe/支付通道 E2E 验收模板(真实 102 + 真实渠道测试账号)

适用:任何"收单发起 → 渠道收款 → webhook 回调落账"闭环验收。纪律:走真实渠道与真实回调,
不 mock;造数前缀统一(acc_/BILL-E2E-xxx/2099-xx 独立账期),验收后按序清理,孤儿巡检门禁兜底。

```bash
#!/usr/bin/env bash
# 用法:BASE / SK(渠道密钥) / WH(回调签名密钥) 经环境变量注入,不落盘不 commit。
set -u
BASE=${BASE:-http://192.168.0.102:28080/api/user/v1}
SK=${SK:?STRIPE secret key}; WH=${WH:?webhook signing secret}
TS=$(date +%s); PHONE="138$((RANDOM%9+1))$((RANDOM%89999999))"   # E.164 +86,否则 42200
# 1 渠道就绪探针:未配置应 503/降级特征,已配置走到验签(400)= env 生效
curl -s -X POST $BASE/webhooks/stripe -d '{}' -o /dev/null -w 'probe=%{http_code}\n'
# 2 门户账号:注册(合成负 id)→ 建真实 customers 行 → UPDATE portal_accounts 改指 → 密码重登
#   (自定义 customers 必填列先查 live information_schema 防迁移漂移)
# 3 插 UNPAID 账单(period 2099-xx,避开 uq(customer_id,period) 撞真实出账)
# 4 POST /payments/stripe/intent {billNo,amount} → intentId/payNo(账单归属校验 404/已缴 409)
# 5 渠道确认:curl -u $SK: -X POST /v1/payment_intents/$PI/confirm \
#     -d payment_method_data[type]=card -d payment_method_data[card][token]=tok_visa \
#     -d "return_url=https://example.com/pay-done"   # 账号启用重定向方式时必带,否则 400
# 6 轮询 ~7s:payments 行 method=card status=SUCCESS、bills 置 PAID、pay_no 唯一
# 7 幂等:同 payload 用 WH 自签(t.v1 = HMAC-SHA256(WH, "$t.$payload"))重投 → 200 且行数不变
# 8 失败路径:tok_chargeDeclined → payment_intent.payment_failed → FAILED 行 + 账单仍 UNPAID
# 9 无账单(充值)意图:渠道直建 intent,metadata 带 pay_no/customer_id → SUCCESS 行 bill_id NULL
#    且 customer_id 必填(双空=0)
# 10 清理按序:payments(bill_id 归属兜底)-> bills -> customers -> addresses -> portal_accounts
#     -> portal_sms_codes;最后跑孤儿巡检门禁
```

要点:webhook endpoint 凭 SK 用 REST 建(POST /v1/webhook_endpoints),whsec 创建时一次返回;
快速隧道 URL 重启即变,换 URL 需重建 endpoint(流程见 adopted note 2026-08-26)。

## 模板 I:admin API 50000 内部错误排障固定模板(2026-09-26 固化,来源:实名审核中心 PASS 500)

> 前置:模板 F 已走到第 4 步"定位到唯一根因";本模板专治 Go 后端信封 `{code:50000,msg:"内部错误"}`
> (= `internal/pkg/httpx/error.go` RespondErr 的 default 分支:sentinel 未登记或底层 SQL/Scan 报错)。

1. **复打取证**:对同一接口 `curl -i` 完整响应体 + `X-Request-Id`;页面截图不算证据。
2. **定位错误通道**:grep 路由找到 handler;50000 只有两种来源——领域 sentinel 没在 error.go 登记,
   或 pgx 返回了未被 errors.Is 覆盖的错误(ErrNoRows 包装/SQLSTATE)。
3. **数据态核对先行**:用列表接口或直查权威表确认触发行的关联主体是否真实存在;
   聚合表的软引用行(verifications.subject_id 等,无 FK)合法允许指向已删除主档,先排除数据态再怀疑代码。
4. **底层归类**:`QueryRow(...).Scan` 报 `pgx.ErrNoRows` = SELECT 匹配 0 行,是可预期分支不是崩溃;
   其余错误解包看 SQLSTATE(42703 列作用域/22P02 类型/42P18 占位符)。
5. **对照映射表**:error.go 是否已有该领域 NotFound→40400 通道(customer.ErrCustomerNotFound 等);
   没有就补 errors.Is 分支,让语义错误返回 40400/42200,绝不静默落 500。
6. **最小修复**:SQL 错改 SQL、缺映射补映射;同提交附 pgxmock 回归(期望正则照抄同文件既有锚点),
   先 `go test ./...` 全量再合并。
7. **合并即验证**:push 分支 → 主树 `git fetch gitea main` + `merge --ff-only` → push main 触发 CI。
8. **真实复核走三层**(techniques「102 业务回归三层证据采集」):/healthz → 登录 → 目标接口;
   目标层返回 `LICENSE_REQUIRED` 属环境授权阻断,如实记录,不修改代码绕授权、不谎报回归通过。

## 模板 J:页面双主题双语言适配固定模板(2026-08-27 固化,来源:营销与积分规则弹框适配)

> 适用:任何"页面/组件没适配多主题或多语言"的任务;硬性规则=文案全走 i18n、颜色全走 tokens.css
> 变量,禁止 JSX 硬编码文案或色值。顺序不可换,第 1 步先证伪再修。

1. **证伪取证**:cdp 打开目标页,断言当前缺陷证据(暗色下 input computed style 非 tokens 值 /
   非 zh 语言下 option 文本仍是中文)——先有证据再动代码。
2. **审计 grep 三连**(改动前圈定全部违规点,一次修完):
   裸中文 `grep -n "[一-龥]" *.tsx`(排除注释行)、裸色值 `grep -n "#[0-9a-fA-F]\{3,6\}"`、
   裸输入框 `grep -n '<input className="w-full"'`。
3. **文案 i18n 闭环**:新 key 四处同步——`i18n/types.ts`(漏了 tsc TS2353 抓) +
   `locales/zh-CN|en-US|ms-MY.ts` 三语言;模块级常量数组的 label 一律改为组件内由 `m` 派生
   (见模板 G 输入框组件化补充),列表列展示复用同一数组,不另写第二份映射。
4. **颜色令牌化**:裸 input 换 `ui/Input`;引用任何 var(--x) 前 grep `theme/tokens.css` +
   `styles.css` 确认双主题都有定义(幽灵令牌审计见 techniques)。
5. **门禁**:`pnpm typecheck && pnpm test && pnpm build`。
6. **矩阵回归**:cdp 四组 capture(light/dark × zh/en|ms),断言模板见
   techniques「CDP 双主题双语言矩阵断言」——主题看 computed style 命中 tokens 值,
   语言看 `[role=option]`/列文本等于当前 locale 译文;工具 `scripts/cdp-admin-capture.mjs`。
7. **存档**:一个页面域一个 commit(fix(web-admin): ...适配多主题与多语言),正文写根因
   (裸 input=UA 默认样式、label 硬编码)与验证证据。

## 模板 K:boss admin 免登录采集脚本用法(cdp-admin-capture,2026-08-27 固化)

> 模板 C/E 的脚本化封装:自动 login 取 token + servers/token 两步注入 + theme/lang URL 透传,
> dev(5173/自起 vite)与生产(102:5180)通用;断言/截图产物语义与 cdp-capture 完全一致。

```bash
# 冒烟目标页(dark 主题英文),追加任意断言 eval,返回值打印 stdout
node .agents/skills/self-evolving/scripts/cdp-admin-capture.mjs out.png \
  --path /bss/marketing --theme dark --lang en-US \
  --base http://localhost:5173 \
  --eval "const b=[...document.querySelectorAll('button')].find(x=>/^\+/.test(x.textContent.trim()));b&&b.click()" \
  --eval "JSON.stringify({dialog:!!document.querySelector('[role=dialog]')})"
# 手动指定 token(免一次 login 请求):--token <jwt>;生产环境 --base http://192.168.0.102:5180
```

要点:token 经 localStorage 注入而非 `?token=`(后者 devOnly,生产被剥离);
`servers 先于受保护页启动`是硬前提,脚本已内置;坏 theme/lang 秒退 exit 2。

## 模板 L:开放平台新增可订阅 Webhook 业务事件(2026-08-27 固化,来源:订阅事件调研+测试事件投递修复)

> 登记制目录:emit 侧落地后才登记,未登记不进目录、不可被集成方订阅;
> 目录唯一事实源 `internal/domain/openplat/events.go` eventCatalog。按序执行:

1. **emit 侧**:业务域定义事件类型常量 + 负载 struct(参照 `internal/domain/order/stage_hook.go`
   的 `StageEventType`/`StageEventPayload`),经 Emitter 接口发射,幂等键业务化(如 `orderNo:stage:N`);
   业务域不 import openplat(避免域耦合),app 装配层注入(`ord.SetStageNotifier(app.OpenWebhook)` 模式;
   守护测试跨域 import 发射方包,测试向 import 无环)。
2. **目录登记**:eventCatalog 追加 `{Type: "x.y.z", Description: "英文基准文案"}`
   (多语言展示由 admin 前端 i18n 承担);`events_test.go` 的
   `TestEventCatalogCoversEmittedEvents` 补守护断言(目录必须覆盖已发射事件)。
3. **匹配语义核对**(本步是测试事件空转 bug 的出生地):业务事件走 `Emit`——按事件类型精确匹配;
   测试事件自检走 `EmitToApp`——按应用匹配全部启用订阅(`InsertAppDeliveries`)。
   新增匹配语义 = `WebhookStore` 接口加方法 + `fakeWebhookStore` 补桩
   (`go vet ./...` 让编译器罗列缺失方法,一次补全,别撞一个补一个)。
4. **真实 PG 集成回归**(`webhook_pg_*_integration_test.go` 模式):`BOSS_PG_TEST_DSN` 未设即 skip;
   seed 用语义前缀 + unixnano 专用命名(`op_<语义>_<nan>`,open_apps.app_id 仅 UNIQUE 无 CHECK);
   t.Cleanup 按依赖逆序删 deliveries → subscriptions → apps;收尾 ssh psql 按前缀计数 = 0
   (容器按端口定位,见 techniques「102 PG 容器定位」)。
5. **契约**:`api/openapi/admin/openplat.yaml` 登记路径 + 顶层 `admin.yaml` 加同行 `$ref`
   (checker A 项只匹配带 `$ref` 的行);`fields.md` 订阅事件行同步。
6. **验证组合**:fake 单测(参数透传/载荷序列化)+ 真库集成(匹配维度/幂等重放/停用过滤)+
   `make contract-sync` 域关键词范围断言;新增事件本身无需迁移(复用 open_webhook_deliveries)。

## 模板 M:侧栏导航激活 + 列头 i18n 双修固定模板(2026-10-01 固化,来源:官网分类激活连带高亮)

> 适用:侧栏菜单"访问子项父项也高亮"或"列表页列头裸英文"。顺序不可换,先取证再改。

1. **取证**:cdp 打开目标路由,断言当前缺陷证据——`Array.from(document.querySelectorAll('nav a[aria-current]')).map(a=>a.getAttribute('href'))`
   应只有当前项,实际多出父项(`["/boss/site","/boss/site/cats"]`=双击亮实锤);列头 `[...document.querySelectorAll('thead th')].map(t=>t.textContent)`
   在 zh-CN 下应显示中文,实际英文=columns 存了字段标识符。
2. **改激活判定**:在 `router/menu.def.ts`(与 KEY_BY_PATH 同文件)加 `isNavActive(to,pathname)` 纯函数:
   精确路径激活 + 深层路由仅当其不在 KEY_BY_PATH 时算同页;Sidebar 从 NavLink 改 Link + 显式 `aria-current`,
   className 直接拼 active 类(react-router 6.30.4 NavLink 已无 isActive prop,别按旧 API 写)。
3. **配单测**:menu.def.test.ts 锁四组——精确匹配、深层非菜单项(编辑页高亮父项)、深层是菜单项(不高亮父项)、兄弟前缀项互不高亮。
4. **列头 i18n**:三语 locale 的 `columns` 数组直接放译文标签(如 ['标识码','名称','排序','启用']),不放字段名;
   同步 i18n/types.ts(columns: string[] 已有则不改);页面 `<th key={x}>{x}</th>` 原样渲染即得三语。
5. **门禁**:`pnpm typecheck && pnpm test && pnpm build`(单测含模板 M 第 3 步)。
6. **矩阵回归**:cdp 断言——目标路由 nav 仅当前项、/boss/site/new 仍高亮官网内容、三语列头(zh/en/ms 各拍一次)、暗色无 console 报错。
7. **部署确认**:push → CI → 用 techniques「前端部署确认新代码已上线」(远端 bundle grep 标记 + 本地 build hash 对照),再对 102:5180 重拍第 6 步。
8. **存档**:导航激活与 i18n 是两个独立可 revert 的提交(fix(admin): 侧栏激活精确化 / fix(admin): 列头三语),不混装。
