# PP1-A 组织与基础配置域页面打磨 · 验收证据(2026-09-05)

- 分支:feat/pp1a-org-base(worktree /Users/imeepos/ext512/ymm-001/boss-pp1a-org-base,自 main@4313a313 切出)
- 负责会话:PP1-A 定点打磨 | 变更:23 files,+656/-683,8 commits(证据文档为其后第 9 提交)
- 契约依据:docs/contract/terms.md、domain-map.md、fields.md、data-relations.md(V1.5)、docs/admin/picker-guide.md

## 0. 机械验收(全绿)

| 项 | 命令 | 结果 |
|---|---|---|
| 门禁 | cd web/admin && pnpm typecheck && TZ=Asia/Shanghai pnpm test && pnpm build | typecheck 0 错;test 69 files/423 tests 全过;build OK 4.9s |
| select 红线 | git grep -n '<select' -- web/admin/src/pages/org web/admin/src/pages/base | 0 命中 |
| alert 红线 | git grep -nE '(^|[^.a-zA-Z])alert\(' -- 同上 | 0 命中 |
| 拆分行数 | wc -l 四个拆分产物 | 157/182/155/211,全部 ≤300 |
| 造数清理 | 102 psql 实查 | 探针账号 945 已物理删除,0 rows 残留 |

## 1. 定点清单逐项处置

### 清单1 原生 select 红线 —— 已修复
- org/company/index.tsx:taxJurisdiction/taxChannel 两处原生 select 换 components/pickers/SimplePicker
  (静态 options + emptyLabel 可空项,ariaLabel 走页面 i18n)。commit 625ddd88。
- 全站 select 红线 grep 复扫 org/base:0 命中。

### 清单2 超 300 行拆分 —— 已拆分(157+182 / 155+211)
- base/authconfig/index.tsx(307→157):三块编辑抽屉 + SecretInput + ConfigFooter 移入
  Drawers.tsx(AuthConfigDrawers,182),页面状态经 props 传入,零行为变更。878b5fa9。
- base/importer/EntityImportPanel.tsx(339→155):导入逻辑抽为 useEntityImport.ts hook(211),
  面板只留视图。569a0933。
- 函数长度:逻辑函数全部 ≤60(run≈40、readFile≈19、registerTask≈16);AuthConfigDrawers/
  EntityImportPanel 为 JSX 组件函数(109/139 行),与全站既有惯例一致(CompanyPage/StaffOrgPage
  等存量组件同量级),本次未强拆,如实登记。

### 清单3 内联 style 收敛 —— 已清零(4 页点名 + 4 页残留)
- base/audit(19 处):PageHead/ToolbarButton/Badge/ActionLink/Drawer 重写,手搓 modal 换
  Drawer,裸色值(#888/#1677ff/#e54545/#f0f5ff 等)全部改令牌。
- base/crashlogs(8 处):令牌类表格 + ErrorBanner/EmptyState;堆栈面板改主题中性底色;
  顺手修 map 内 Fragment 缺 key(React key 告警源)。
- login(5 处):字面量清零;服务端选择器迁 SimplePicker;品牌面用 --color-brand-navy-950/
  --color-text-secondary 等静态色板(不随主题翻转,防暗色白字);inputStyle/buttonStyle 为
  auth-shell 共享常量(范围外文件,只引用不修改)。
- partner/apply(7 处):字面量与本地 CSSProperties 常量(含裸色 #3D7EFF/#7C8799)全部转
  --color-text-link/--color-text-secondary/--color-danger 等令牌类。
- 追加清零:menuperm(2)、AccountForm(1)、error(2)、placeholder(1,#888 裸色值)。
  843c55e0、7dcea04f。

### 清单4 删除类操作确认核实 —— 1 处补确认,6 处核实通过

| 文件 | 结论 |
|---|---|
| org/staff/OrgTree.tsx | 删除按钮委托 index.tsx delDept/delPost,已有 useConfirm(danger),核实通过 |
| profile/index.tsx | 无删除路径(仅 PUT profile / change-password),非删除路径 |
| backup/BackupDrawers.tsx | 无删除路径(新建 POST / 恢复 POST 为追加语义;删除入口在 index.tsx 已带确认) |
| base/geo/CountryPanel.tsx | 面板无删除;真删在 CountryDetail removeName(删译名),已有 useConfirm,核实通过 |
| base/importer/TaskList.tsx | 只读任务历史,无删除 |
| partner/staff/index.tsx | 仅启停(PUT status),无删除 |
| home/index.tsx | 无删除 |
| org/openplat/AppDetail.tsx | delSub(DELETE /openplat/subscriptions/123)为真删且无确认、失败静默,本次修复:补 useConfirm(danger) + catch 置错误条;requeue 同补 catch。aa7ca837 |

### 清单5 建/删能力对齐(102 实测)—— 全对齐,1 项决策登记
以 api/openapi/admin/*.yaml 为契约源,102(192.168.0.102:28080,admin/admin123)实测:
- DELETE /accounts/999999999 返回 HTTP200 信封 40400「资源不存在」(路由真实);
  建探针号 id=945 后 DELETE 返回 code 0 ok=true,但 GET /accounts 复核该号仍在且 status=0,
  即 102 的 DELETE /accounts 实为软删,可观测效果与页面既有「停用」(PUT status=0)完全等价。
  决策:不在 base/account、org/staff MemberPanel 另设物理删除按钮(避免同效双按钮),维持
  停用入口;如需真删请后端先明确口径。探针数据已 psql 物理清理。
- org/staff:departments/posts 的 PUT+DELETE、accounts PUT,对齐(删除入口在组织树);
- profile/ApiKeySection:POST /api-keys + DELETE /api-keys/{id}(吊销已带确认),对齐;
- org/openplat:POST /apps、PUT /{id}/status、订阅 GET/POST/DELETE、投递 GET/requeue、
  test-event POST 全对齐(契约无 app 编辑/删除端点,页面亦无,无越权按钮);
- org/company EntityStaffDialogs+Panel:POST staff、PUT password、PUT status,对齐;
- org/department/DeptForm、org/post/PostForm:POST/PUT,对齐(删除入口在组织树);
- org/menuperm RoleManagerCard:POST/PUT/DELETE /roles,对齐(内置角色只读,删除带确认);
- base/address/AddressNodeDrawer:POST/PUT,对齐(删除入口在地址树,已带确认;102 DELETE
  /addresses/999999999 返回 40400,路由真实)。

### 清单6 列表骨架 business 组件核对 —— 2 迁移、2 豁免

| 页面 | 结论 |
|---|---|
| base/account/index.tsx | 手搓 table 换 business/DataTable,行操作迁 ActionLinks 体系;停用手搓 modal 换 useConfirm。4e20caa3 |
| backup/index.tsx | 手搓 table 换 business/DataTable(列渲染含 StatusTag/下载/删除)。4e20caa3 |
| news/index.tsx | 非列表页(公开文章详情 /news/:slug),DataTable 不适用,豁免 |
| base/address/index.tsx | 树形懒加载视图非平铺列表,已用 ErrorBanner/EmptyState/ToolbarButton/ui Badge,豁免 |

### 需求 B 关联关系(account 详情样板)
- base/account DetailDrawer 已呈现归属链:角色(roleName)/法人主体(legalEntityName)/
  部门(deptName)/岗位(postName)/区域范围(regionScope)+ 手机号,与 data-relations 2.1
  accounts 归属(roles/legal_entity/dept/post/region_scope)逐项对齐,取不到显示「—」,
  核实通过,无需改动。

## 2. CDP 实截冒烟(102 生产 http://192.168.0.102:5180,免登录注入)

工具:.agents/skills/self-evolving/scripts/cdp-admin-capture.mjs(自动 login+token 注入)。
说明:102 为部署基线(本分支未部署),本次冒烟验证触达页面与真实后端的数据链路与
console 卫生;本分支无新增/删减接口调用(AppDetail 确认弹窗为纯前端交互),契约与基线一致。

| 页面 | 截图 | console err/warn | network >=400 | DOM 断言 |
|---|---|---|---|---|
| /org/company | /tmp/pp1a-shots/_org_company.png (149KB) | 0 | 0 | path 正确、非登录页、内容渲染 |
| /base/audit | /tmp/pp1a-shots/_base_audit.png (198KB) | 0 | 0 | 同上 |
| /base/crashlogs | /tmp/pp1a-shots/_base_crashlogs.png (73KB) | 0 | 0 | 同上 |
| /base/account | /tmp/pp1a-shots/_base_account.png (188KB) | 0 | 0 | 同上 |
| /base/backup | /tmp/pp1a-shots/_base_backup.png (201KB) | 0 | 0 | 同上 |
| /org/openplat | /tmp/pp1a-shots/_org_openplat.png (152KB) | 0 | 0 | 同上 |

执行模型不支持图像输入,截图以文件尺寸 + DOM 断言 + 日志三重佐证;「未目检像素」如实说明。

## 3. 范围外遗留债务登记(只登记不修)

1. bss/user/filter.test.ts 时区敏感:createdAtCell 断言依赖本地时区,TZ=America/Los_Angeles
   必红、TZ=Asia/Shanghai 全绿。属 bss 域,范围外;建议用例固定 TZ 或注入格式化器。
2. 102 DELETE /accounts 软删口径:删除后 GET 列表仍返回 status=0 行;产品语义上与停用
   重合,后端若希望保留「删除」能力应改为过滤已删行或物理删,前置条件见清单5。
3. 102 DELETE /openplat/subscriptions/{不存在 id} 返回 50000 内部错误:应为 40400,
   后端 error mapping 债务。
4. business/SearchBar placeholder 硬编码「搜索...」:共享组件(范围外),建议接 i18n。
5. 数据驱动内联样式豁免说明:address 树缩进 paddingLeft depth*20、home 图表动态
   s.color、TopNav var(--home-nav-bg)(home.css 独立令牌体系)为动态值/独立体系,非裸
   色值,保留。
6. auth-shell.tsx / auth-ads.tsx(pages 根,范围外):品牌壳层自有 CSSProperties 常量
   含裸色值(#FFFFFF/#4E5664 等),login/apply 仅引用;如需收敛须单开任务改壳层。

## 4. 提交清单(时间序即 merge 顺序)

    625ddd88 fix(org): replace native select with SimplePicker in company form
    878b5fa9 refactor(base): 拆分 authconfig 页面满足单文件行数红线
    569a0933 refactor(base): 拆分 EntityImportPanel 满足单文件行数红线
    843c55e0 refactor(base): 收敛 audit/crashlogs/login/partner-apply 内联样式
    aa7ca837 fix(openplat): 应用详情删订阅补确认弹窗,失败路径留痕
    fa5f361c chore(i18n): 登记 openplat.delSubConfirm 词条(三语+类型,append-only)
    4e20caa3 refactor(base): account 列表迁 DataTable,停用确认换 ConfirmDialog
    7dcea04f style(base): 清零范围页残留内联 style 字面量

i18n 变更为集中登记文件(locale×3 + types.ts),已按协议压成独立 append-only 小提交
(fa5f361c);menu.def.ts/App.tsx 无路由变更,零改动。
