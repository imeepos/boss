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
