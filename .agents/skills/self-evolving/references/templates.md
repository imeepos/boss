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
